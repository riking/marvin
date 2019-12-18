package autoinvite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/riking/marvin"
	"github.com/riking/marvin/modules/on_reaction"
	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/util"
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
	t.DB().MustMigrate(Identifier, 1482202815, sqlMigrate2, sqlMigrate3)
	t.DB().MustMigrate(Identifier, 1482215299, sqlMigrate4)
	t.DB().SyntaxCheck(sqlInsert, sqlFindInvite, sqlRevokeInvite)
}

func (mod *AutoInviteModule) Enable(t marvin.Team) {
	mod.onReactAPI().RegisterHandler(mod, Identifier)
	mod.team.OnEvent(Identifier, "reaction_added", mod.OnRawReaction)
	t.RegisterCommandFunc("make-invite", mod.PostInvite, inviteHelp)
	t.RegisterCommandFunc("revoke-invite", mod.CmdRevokeInvite, revokeHelp)
	t.RegisterCommandFunc("mass-invite", CmdMassInvite, usageMass)
	mod.registerHTTP()
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
		valid           boolean NOT NULL,
		-- public       boolean NOT NULL DEFAULT true,
		invited_channel varchar(10) NOT NULL,
		inviting_user   varchar(10) NOT NULL,
		inviting_ts     varchar(20) NOT NULL,
		msg_channel     varchar(10) NOT NULL,
		msg_ts          varchar(20) NOT NULL,
		msg_emoji       varchar(200) NOT NULL,
		msg_text        TEXT
	)`

	sqlMigrate2 = `CREATE INDEX index_module_invites_on_message
			ON module_invites (msg_channel, msg_ts, invited_channel)`
	sqlMigrate3 = `CREATE INDEX index_module_invites_on_channel
			ON module_invites (invited_channel, valid)`
	sqlMigrate4 = `ALTER TABLE module_invites ADD COLUMN public boolean NOT NULL DEFAULT true`

	sqlInsert = `
	INSERT INTO module_invites
	(valid, public, invited_channel, inviting_user, inviting_ts, msg_channel, msg_ts, msg_emoji, msg_text)
	VALUES (true, $1, $2, $3, $4, $5, $6, $7, $8)`

	sqlFindInvite = `
	SELECT invited_channel, valid
	FROM module_invites
	WHERE msg_channel = $1 AND msg_ts = $2`

	sqlRevokeInvite = `
	UPDATE module_invites
	SET valid = false
	WHERE invited_channel = $1
	RETURNING msg_channel, msg_ts, msg_emoji`

	sqlListInvites = `
	SELECT invited_channel, inviting_user, inviting_ts, msg_text
	FROM module_invites
	WHERE valid = true AND public = true AND ($1 = '' OR invited_channel = $1)
	ORDER BY inviting_ts DESC`
)

// ---

const (
	inviteHelp = "`@marvin make-invite [:emoji:] <#channel> [message]` posts a message to another " +
		"channel that functions as a private channel invitation.\n" +
		"Any team member can react to the message to be added to the private channel you sent the command from."
	revokeHelp = "`@marvin revoke-invite` revokes all standing invitations to the channel you sent the command from."
)

func (mod *AutoInviteModule) OnRawReaction(rtm slack.RTMRawMessage) {
	var msg struct {
		User       slack.UserID `json:"user"`
		TargetUser slack.UserID `json:"item_user"`
		Item       struct {
			Type    string          `json:"type"`
			Channel slack.ChannelID `json:"channel"`
			TS      slack.MessageTS `json:"ts"`
		}
		EventTS slack.MessageTS `json:"event_ts"`
	}
	rtm.ReMarshal(&msg)

	if msg.TargetUser != mod.team.BotUser() {
		return
	}
	if msg.User == mod.team.BotUser() {
		return
	}
	if msg.Item.Type != "message" {
		return
	}

	stmt, err := mod.team.DB().Prepare(sqlFindInvite)
	if err != nil {
		util.LogError(errors.Wrap(err, "prepare"))
	}
	defer stmt.Close()

	row := stmt.QueryRow(string(msg.Item.Channel), string(msg.Item.TS))
	var targetChannelStr string
	var isValid bool
	err = row.Scan(&targetChannelStr, &isValid)
	if err == sql.ErrNoRows {
		return
	} else if err != nil {
		util.LogError(err)
		return
	}

	if !isValid {
		channel, _ := mod.team.GetIM(msg.User)
		mod.team.SendMessage(channel, "That invitation has been deleted.")
		return
	}

	var response struct {
		AlreadyInGroup bool `json:"already_in_group"`
	}
	form := url.Values{
		"channel": []string{string(targetChannelStr)},
		"user":    []string{string(msg.User)},
	}
	err = mod.team.SlackAPIPostJSON("groups.invite", form, &response)
	if err != nil {
		util.LogError(err)
		return
	}
	if response.AlreadyInGroup {
		util.LogGood("Invite skipped:", mod.team.UserName(msg.User), "already in", mod.team.ChannelName(slack.ChannelID(targetChannelStr)))
		return
	}
	util.LogGood("Invited", mod.team.UserName(msg.User), "to", mod.team.ChannelName(slack.ChannelID(targetChannelStr)))
}

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
	util.LogGood("(Old) Invited", mod.team.UserName(evt.UserID), "to", mod.team.ChannelName(data.InviteTargetChannel))
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
	if mod.team.TeamConfig().IsReadOnly {
		return marvin.CmdFailuref(args, "Marvin is currently on read only.")
	}
	util.LogDebug("PostInvite", args.Arguments)

	if args.Source.AccessLevel() < marvin.AccessLevelController {
		return marvin.CmdFailuref(args, "This command has been restricted to bot controller only.")
	}

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
	messageChannel := t.ResolveChannelName(arg)
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

	ts, _, err := t.SendMessage(messageChannel, msg)
	if err != nil {
		return marvin.CmdError(args, err, "Couldn't send message")
	}
	msgID := slack.MsgID(messageChannel, ts)
	args.SetModuleData(postInviteResult{MsgID: msgID, Emoji: emoji, TargetName: privateChannel.Name})

	err = mod.SaveInvite(args, msgID, emoji, customMsg)

	//err = mod.onReactAPI().ListenMessage(msgID, Identifier, callbackBytes)
	_ = callbackBytes
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

func (mod *AutoInviteModule) SaveInvite(
	args *marvin.CommandArguments,
	sentMsgId slack.MessageID,
	emoji, text string) error {
	stmt, err := mod.team.DB().Prepare(sqlInsert)
	if err != nil {
		return errors.Wrap(err, "prepare")
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		bool(sentMsgId.ChannelID[0] == 'C'),
		string(args.Source.ChannelID()), string(args.Source.UserID()), string(args.Source.MsgTimestamp()),
		string(sentMsgId.ChannelID), string(sentMsgId.MessageTS),
		emoji, text,
	)
	if err != nil {
		return errors.Wrap(err, "insert")
	}
	return nil
}

func (mod *AutoInviteModule) CmdRevokeInvite(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if mod.team.TeamConfig().IsReadOnly && args.Source.AccessLevel() < marvin.AccessLevelAdmin {
		return marvin.CmdFailuref(args, "Marvin is currently on read only.")
	}

	if args.Source.AccessLevel() < marvin.AccessLevelController {
		return marvin.CmdFailuref(args, "This command has been restricted to bot controller only.")
	}
	stmt, err := mod.team.DB().Prepare(sqlRevokeInvite)
	if err != nil {
		return marvin.CmdError(args, err, "database error")
	}
	defer stmt.Close()

	rows, err := stmt.Query(string(args.Source.ChannelID()))
	if err != nil {
		return marvin.CmdError(args, err, "database error")
	}

	var channel, ts, emoji string
	var count int64
	for rows.Next() {
		err = rows.Scan(&channel, &ts, &emoji)
		if err != nil {
			util.LogError(err)
			continue
		}

		form := url.Values{
			"name":      []string{emoji},
			"channel":   []string{channel},
			"timestamp": []string{ts},
		}
		err = mod.team.SlackAPIPostJSON("reactions.remove", form, nil)
		if err, ok := errors.Cause(err).(slack.APIResponse); ok {
			if err.SlackError == "no_reaction" {
				continue
			}
		}
		if err != nil {
			util.LogError(err)
			continue
		}
		form = url.Values{
			"channel": []string{channel},
			"ts":      []string{ts},
			"as_user": []string{"true"},
			"text":    []string{"(Invite deleted)"},
			"parse":   []string{"client"},
		}
		util.LogIfError(mod.team.SlackAPIPostJSON("chat.update", form, nil))
		count++
	}

	return marvin.CmdSuccess(args, fmt.Sprintf("%d invite messages revoked.", count))
}
