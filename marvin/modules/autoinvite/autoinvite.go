package autoinvite

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/modules/on_reaction"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
)

func init() {
	marvin.RegisterModule(NewAutoInviteModule)
}

const Identifier = "autoinvite"

type AutoInviteModule struct {
	team marvin.Team

	onReact marvin.Module
}

func NewAutoInviteModule(t marvin.Team) marvin.Module {
	aim := &AutoInviteModule{team: t}
	return aim
}

func (aim *AutoInviteModule) Identifier() marvin.ModuleID {
	return Identifier
}

var (
	_ on_reaction.API = &on_reaction.OnReactionModule{}
)

func (aim *AutoInviteModule) Load(t marvin.Team) {
	var _ marvin.Module = aim.onReact

	t.DependModule(aim, on_reaction.Identifier, &aim.onReact)
}

func (aim *AutoInviteModule) Enable(t marvin.Team) {
	aim.onReactAPI().RegisterHandler(aim, Identifier)
	t.RegisterCommandFunc("make-invite", marvin.SubCommandFunc(aim.PostInvite),
		"`make-invite` posts a message to another channel that functions as a private channel invitation. "+
			"Any team member can react to the message to be added to the private channel."+
			"\n"+usage,
	)
}

func (aim *AutoInviteModule) Disable(t marvin.Team) {
	t.UnregisterCommand("make-invite")
}

func (aim *AutoInviteModule) onReactAPI() on_reaction.API {
	if aim.onReact != nil {
		return aim.onReact.(on_reaction.API)
	}
	return nil
}

// ---

type PendingInviteData struct {
	InviteTargetChannel slack.ChannelID
}

func (aim *AutoInviteModule) OnReaction(evt *on_reaction.ReactionEvent, customData []byte) error {
	var data PendingInviteData

	util.LogGood("Reaction from", aim.team.UserName(evt.UserID), "emoji", evt.EmojiName, "in", aim.team.ChannelName(evt.ChannelID))
	if !evt.IsAdded {
		return nil
	}
	if evt.UserID == aim.team.BotUser() {
		return nil
	}

	err := json.Unmarshal(customData, &data)
	if err != nil {
		return errors.Wrap(err, "unmarshal json")
	}
	form := url.Values{
		"channel": []string{string(data.InviteTargetChannel)},
		"user":    []string{string(evt.UserID)},
	}
	resp, err := aim.team.SlackAPIPost("groups.invite", form)
	if err != nil {
		imChannel, err := aim.team.GetIM(evt.UserID)
		if err == nil {
			aim.team.SendMessage(imChannel, "Sorry, an error occured. Try again later?")
		}
		return errors.Wrap(err, "invite to group")
	}
	var response slack.APIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	resp.Body.Close()
	if err != nil {
		return errors.Wrap(err, "decode slack API response")
	}
	if !response.OK {
		// TODO logging
		imChannel, err := aim.team.GetIM(evt.UserID)
		if err == nil {
			aim.team.SendMessage(imChannel, fmt.Sprintf("Sorry, an error occured: %s", response.Error()))
		}
		return errors.Wrap(response, "Could not invite to channel")
	}
	util.LogGood("Invited", aim.team.UserName(evt.UserID), "to", aim.team.ChannelName(data.InviteTargetChannel))
	return nil
}

const defaultInviteText = `%v has invited everybody to the #%s channel%s%s
:point_down: Click here to be added!`

const andSaid = ", saying:\n>"
const defaultEmoji = `white_check_mark`

// TODO - support timeouts
const usage = "Usage: `@marvin make-invite` [emoji = :white_check_mark:] <send_to = #channel> [message]"

func (aim *AutoInviteModule) PostInvite(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	util.LogDebug("PostInvite", args.Arguments)

	inviteTarget := args.Source.ChannelID()
	if inviteTarget == "" || inviteTarget[0] != 'G' {
		return marvin.CmdFailuref(args, "Command must be used from a private channel.")
	}
	privateChannel, err := t.PrivateChannelInfo(inviteTarget)
	if err != nil {
		return marvin.CmdError(args, err, "Could not retrieve information about the channel")
	}
	if privateChannel.IsMultiIM() {
		return marvin.CmdFailuref(args, "You cannnot invite users to a multi-party IM.")
	}

	usage := func() marvin.CommandResult {
		return marvin.CmdUsage(args, usage)
	}

	if len(args.Arguments) < 1 {
		return usage()
	}

	// Command passed validation

	emoji := defaultEmoji
	arg := args.Pop()
	if len(arg) > 0 && arg[0] == ':' {
		emoji = strings.Trim(arg, ":")
		arg = args.Pop()
	}
	messageChannel := slack.ParseChannelID(arg)
	if messageChannel == "" {
		return usage()
	}

	customMsg := strings.TrimSpace(strings.Join(args.Arguments, " "))
	msg := ""
	if customMsg != "" {
		msg = fmt.Sprintf(defaultInviteText, args.Source.UserID(), privateChannel.Name, andSaid, customMsg)
	} else {
		msg = fmt.Sprintf(defaultInviteText, args.Source.UserID(), privateChannel.Name, ".", "")
	}

	var data PendingInviteData
	data.InviteTargetChannel = inviteTarget
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return marvin.CmdError(args, err, "error marshal json")
	}

	util.LogDebug("sending invite to", messageChannel, "text:", msg)
	ts, _, err := t.SendMessage(messageChannel, msg)
	if err != nil {
		return marvin.CmdError(args, err, "Couldn't send message")
	}
	msgID := slack.MsgID(messageChannel, ts)
	err = aim.onReactAPI().ListenMessage(msgID, Identifier, dataBytes)
	if err != nil {
		// Failed to save, delete the message
		form := url.Values{
			"ts":      []string{string(msgID.MessageTS)},
			"channel": []string{string(msgID.ChannelID)},
			"as_user": []string{"true"},
		}
		slack.SlackAPILog(t.SlackAPIPost("chat.delete", form))
		return marvin.CmdError(args, err, "Error saving message")
	}
	err = t.ReactMessage(msgID, emoji)
	if err != nil {
		return marvin.CmdError(args, err,
			"Couldn't post sample reaction (the message should still work)")
	}
	return marvin.CmdSuccess(args, fmt.Sprintf("Message posted: %s", t.ArchiveURL(msgID)))
}
