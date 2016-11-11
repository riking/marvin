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
	mod := &AutoInviteModule{team: t}
	return mod
}

func (mod *AutoInviteModule) Identifier() marvin.ModuleID {
	return Identifier
}

var (
	_ on_reaction.API = &on_reaction.OnReactionModule{}
)

func (mod *AutoInviteModule) Load(t marvin.Team) {
	var _ marvin.Module = mod.onReact

	t.DependModule(mod, on_reaction.Identifier, &mod.onReact)
}

func (mod *AutoInviteModule) Enable(t marvin.Team) {
	mod.onReactAPI().RegisterHandler(mod, Identifier)
	t.RegisterCommandFunc("make-invite", marvin.SubCommandFunc(mod.PostInvite),
		"`make-invite` posts a message to another channel that functions as a private channel invitation. "+
			"Any team member can react to the message to be added to the private channel."+
			"\n"+usage,
	)
}

func (mod *AutoInviteModule) Disable(t marvin.Team) {
	t.UnregisterCommand("make-invite")
}

func (mod *AutoInviteModule) onReactAPI() on_reaction.API {
	if mod.onReact != nil {
		return mod.onReact.(on_reaction.API)
	}
	return nil
}

// ---

type PendingInviteData struct {
	InviteTargetChannel slack.ChannelID
}

func (mod *AutoInviteModule) OnReaction(evt *on_reaction.ReactionEvent, customData []byte) error {
	var data PendingInviteData

	util.LogGood("Reaction from", mod.team.UserName(evt.UserID), "emoji", evt.EmojiName, "in", mod.team.ChannelName(evt.ChannelID))
	if !evt.IsAdded {
		return nil
	}
	if evt.UserID == mod.team.BotUser() {
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
	resp, err := mod.team.SlackAPIPost("groups.invite", form)
	if err != nil {
		imChannel, err := mod.team.GetIM(evt.UserID)
		if err == nil {
			mod.team.SendMessage(imChannel, "Sorry, an error occured. Try again later?")
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
		imChannel, err := mod.team.GetIM(evt.UserID)
		if err == nil {
			mod.team.SendMessage(imChannel, fmt.Sprintf("Sorry, an error occured: %s", response.Error()))
		}
		return errors.Wrap(response, "Could not invite to channel")
	}
	util.LogGood("Invited", mod.team.UserName(evt.UserID), "to", mod.team.ChannelName(data.InviteTargetChannel))
	return nil
}

const defaultInviteText = `%v has invited everybody to the #%s channel%s%s
:point_down: Click here to be added!`

const andSaid = ", saying:\n>"
const defaultEmoji = `white_check_mark`

// TODO - support timeouts
const usage = "Usage: `@marvin make-invite` [emoji = :white_check_mark:] <send_to = #channel> [message]"

type postInviteResult struct {
	MsgID slack.MessageID
	Emoji string
}

func (mod *AutoInviteModule) PostInvite(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	util.LogDebug("PostInvite", args.Arguments)

	inviteTarget := args.Source.ChannelID()
	if inviteTarget == "" || inviteTarget[0] != 'G' {
		return marvin.CmdFailuref(args, "Command must be used from a private channel.").WithNoEdit()
	}
	privateChannel, err := t.PrivateChannelInfo(inviteTarget)
	if err != nil {
		return marvin.CmdError(args, err, "Could not retrieve information about the channel")
	}
	if privateChannel.IsMultiIM() {
		return marvin.CmdFailuref(args, "You cannnot invite users to a multi-party IM.").WithNoEdit()
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

	// Handle edits

	if args.IsEdit {
		prev := args.PreviousResult.Args.ModuleData.(postInviteResult)
		form := url.Values{
			"ts":      []string{string(prev.MsgID.MessageTS)},
			"channel": []string{string(prev.MsgID.ChannelID)},
			"as_user": []string{"true"},
			"text":    []string{msg},
			"parse":   []string{"client"},
		}
		resp, err := mod.team.SlackAPIPost("chat.update", form)
		if err != nil {
			return marvin.CmdError(args, err, "Error editing message")
		}
		resp.Body.Close()
		if prev.Emoji != emoji {
			form := url.Values{
				"name":      []string{prev.Emoji},
				"channel":   []string{string(prev.MsgID.ChannelID)},
				"timestamp": []string{string(prev.MsgID.MessageTS)},
			}
			go mod.team.ReactMessage(prev.MsgID, emoji)
			mod.team.SlackAPIPost("reactions.remove", form)
			prev.Emoji = emoji
			args.SetModuleData(prev)
		}
		return marvin.CmdSuccess(args, fmt.Sprintf("Message updated: %s", t.ArchiveURL(prev.MsgID))).WithEdit()
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
	args.SetModuleData(postInviteResult{MsgID: msgID, Emoji: emoji})
	err = mod.onReactAPI().ListenMessage(msgID, Identifier, dataBytes)
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
	return marvin.CmdSuccess(args, fmt.Sprintf("Message posted: %s", t.ArchiveURL(msgID))).WithEdit()
}
