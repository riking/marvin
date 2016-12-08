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
	t.DB().MustMigrate(Identifier, 1481226823, sqlMigrate1)
	t.DB().SyntaxCheck(sqlInsert)
}

func (mod *AutoInviteModule) Enable(t marvin.Team) {
	mod.onReactAPI().RegisterHandler(mod, Identifier)
	t.RegisterCommandFunc("make-invite", marvin.SubCommandFunc(mod.PostInvite), inviteHelp)
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

const (
	sqlMigrate1 = `
	CREATE TABLE module_invites (
		id SERIAL PRIMARY KEY,
		invited_channel varchar(10) NOT NULL,
		inviting_user   varchar(10) NOT NULL,
		inviting_ts     varchar(20) NOT NULL,
		msg_channel     varchar(10) NOT NULL,
		msg_ts          varchar(20) NOT NULL,
		msg_emoji       varchar(200) NOT NULL,
		msg_text        TEXT
	)`

	sqlInsert = `
	INSERT INTO module_invites
	(invited_channel, inviting_user, inviting_ts, msg_channel, msg_ts, msg_emoji, msg_text)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
)

// ---

const inviteHelp = "`@marvin make-invite [:emoji:] <#channel> [message]` posts a message to another " +
	"channel that functions as a private channel invitation.\n" +
	"Any team member can react to the message to be added to the private channel you sent the command from."

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
	var response struct {
		AlreadyInGroup bool `json:"already_in_group"`
	}
	form := url.Values{
		"channel": []string{string(data.InviteTargetChannel)},
		"user":    []string{string(evt.UserID)},
	}
	err = mod.team.SlackAPIPostJSON("groups.invite", form, &response)
	if err != nil {
		imChannel, err := mod.team.GetIM(evt.UserID)
		if err == nil {
			mod.team.SendMessage(imChannel, "Sorry, an error occured. Try again later?")
		}
		return errors.Wrap(err, "invite to group")
	}
	util.LogGood("Invited", mod.team.UserName(evt.UserID), "to", mod.team.ChannelName(data.InviteTargetChannel))
	return nil
}

const defaultInviteText = `%v has invited everybody to the #%s channel%s%s
:point_down: Click here to be added!`

const andSaid = ", saying:\n>"
const defaultEmoji = `white_check_mark`

type postInviteResult struct {
	MsgID      slack.MessageID
	Emoji      string
	TargetName string
}

func (mod *AutoInviteModule) PostInvite(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	util.LogDebug("PostInvite", args.Arguments)

	if len(args.Arguments) < 1 {
		return marvin.CmdUsage(args, inviteHelp)
	}

	inviteTarget := args.Source.ChannelID()
	if inviteTarget == "" || inviteTarget[0] != 'G' {
		return marvin.CmdFailuref(args, "Command must be used from the private channel you want to invite people to.").WithNoEdit().WithSimpleUndo()
	}
	privateChannel, err := t.PrivateChannelInfo(inviteTarget)
	if err != nil {
		return marvin.CmdError(args, err, "Could not retrieve information about the channel")
	}
	if privateChannel.IsMultiIM() {
		return marvin.CmdFailuref(args, "You cannnot invite users to a multi-party IM.").WithNoEdit().WithSimpleUndo()
	}

	usage := func() marvin.CommandResult {
		return marvin.CmdUsage(args, inviteHelp)
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
		prev, ok := args.PreviousResult.Args.ModuleData.(postInviteResult)
		if !ok {
			return marvin.CmdFailuref(args, "Bad edit data").WithNoEdit().WithNoUndo()
		}
		args.SetModuleData(prev)
		form := url.Values{
			"ts":      []string{string(prev.MsgID.MessageTS)},
			"channel": []string{string(prev.MsgID.ChannelID)},
			"as_user": []string{"true"},
			"text":    []string{msg},
			"parse":   []string{"client"},
		}
		err := mod.team.SlackAPIPostJSON("chat.update", form, nil)
		if err != nil {
			return marvin.CmdError(args, err, "Error editing message")
		}
		if prev.Emoji != emoji {
			form := url.Values{
				"name":      []string{prev.Emoji},
				"channel":   []string{string(prev.MsgID.ChannelID)},
				"timestamp": []string{string(prev.MsgID.MessageTS)},
			}
			mod.team.SlackAPIPostJSON("reactions.remove", form, nil)
			go mod.team.ReactMessage(prev.MsgID, emoji)
			prev.Emoji = emoji
		}
		return marvin.CmdSuccess(args, fmt.Sprintf("Message updated: %s", t.ArchiveURL(prev.MsgID))).WithEdit().WithCustomUndo()
	}
	if args.IsUndo {
		prev, ok := args.PreviousResult.Args.ModuleData.(postInviteResult)
		if !ok {
			return marvin.CmdFailuref(args, "Bad edit data").WithNoEdit().WithNoUndo()
		}
		args.SetModuleData(prev)

		form := url.Values{
			"name":      []string{prev.Emoji},
			"channel":   []string{string(prev.MsgID.ChannelID)},
			"timestamp": []string{string(prev.MsgID.MessageTS)},
		}
		util.LogIfError(mod.team.SlackAPIPostJSON("reactions.remove", form, nil))
		form = url.Values{
			"ts":      []string{string(prev.MsgID.MessageTS)},
			"channel": []string{string(prev.MsgID.ChannelID)},
			"as_user": []string{"true"},
			"text":    []string{fmt.Sprintf("(Invite to %s retracted)", prev.TargetName)},
			"parse":   []string{"client"},
		}
		err := mod.team.SlackAPIPostJSON("chat.update", form, nil)
		if err != nil {
			return marvin.CmdError(args, err, "Error editing message")
		}
		return marvin.CmdSuccess(args, fmt.Sprintf("Invite successfully cancelled. %s", t.ArchiveURL(prev.MsgID))).WithNoEdit().WithNoUndo()
	}

	var callbackData PendingInviteData
	callbackData.InviteTargetChannel = inviteTarget
	callbackBytes, err := json.Marshal(callbackData)
	if err != nil {
		return marvin.CmdError(args, err, "error marshal callback")
	}

	util.LogDebug("sending invite to", messageChannel, "text:", msg)
	ts, _, err := t.SendMessage(messageChannel, msg)
	if err != nil {
		return marvin.CmdError(args, err, "Couldn't send message")
	}
	msgID := slack.MsgID(messageChannel, ts)
	args.SetModuleData(postInviteResult{MsgID: msgID, Emoji: emoji, TargetName: privateChannel.Name})
	err = mod.onReactAPI().ListenMessage(msgID, Identifier, callbackBytes)
	if err != nil {
		// Failed to save, delete the message
		form := url.Values{
			"ts":      []string{string(msgID.MessageTS)},
			"channel": []string{string(msgID.ChannelID)},
			"as_user": []string{"true"},
			"text":    []string{fmt.Sprintf("(Invite to %s cancelled due to internal error)", inviteTarget)},
			"parse":   []string{"client"},
		}
		util.LogIfError(t.SlackAPIPostJSON("chat.delete", form, nil))
		return marvin.CmdError(args, err, "Error saving message")
	}
	err = t.ReactMessage(msgID, emoji)
	if err != nil {
		return marvin.CmdError(args, err,
			"Couldn't post sample reaction (the message should still work)")
	}
	return marvin.CmdSuccess(args, fmt.Sprintf("Message posted: %s", t.ArchiveURL(msgID))).WithEdit().WithCustomUndo()
}
