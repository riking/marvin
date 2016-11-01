package autoinvite

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"regexp"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
	"sync"
)

func init() {
	marvin.RegisterModule(NewAutoInviteModule)
}

const Identifier = "autoinvite"

type AutoInviteModule struct {
	team marvin.Team

	listLock sync.Mutex
	list []PendingInvite
}

func NewAutoInviteModule(t marvin.Team) marvin.Module {
	aim := &AutoInviteModule{team: t}
	t.RegisterCommand("make-invite", marvin.SubCommandFunc(aim.PostInvite))
	return aim
}

func (aim *AutoInviteModule) Identifier() string {
	return Identifier
}

func (aim *AutoInviteModule) Unregister(t marvin.Team) {
	t.UnregisterCommand("make-invite", marvin.SubCommandFunc(aim.PostInvite))
}

func (aim *AutoInviteModule) RegisterRTMEvents(t marvin.Team) {
	t.OnEvent(Identifier, "reaction_added", aim.OnReaction)
}

type PendingInvite struct {
	TargetChannel slack.ChannelID
	MsgChannel slack.ChannelID
	TS slack.MessageTS
}

func (aim *AutoInviteModule) OnReaction(rtm slack.RTMRawMessage) {
	iUID, ok := rtm["item_user"].(string)
	if !ok {
		return
	}
	if iUID != aim.team.BotUser() {
		return
	}
}

const defaultInviteText = `<@%s> has invited everybody to the #%s channel%s%s
:point_down: Click here to be added!`

const andSaid = ", saying:\n>"
const defaultEmoji = `white_check_mark`

var channelMentionRgx = regexp.MustCompile(`<#(C[A-Z0-9]+)\|([a-z0-9_-]+)>`)

func (aim *AutoInviteModule) PostInvite(t marvin.Team, args *marvin.CommandArguments) error {

	fmt.Println("[DEBUG]", "PostInvite", args.Arguments)

	inviteTarget := args.Source.ChannelID()
	if inviteTarget == "" || inviteTarget[0] != 'G' {
		return marvin.CmdErrorf(args, "Command must be used from a private channel.")
	}
	channel, err := t.PrivateChannelInfo(inviteTarget)
	if err != nil {
		return errors.Wrap(err, "Could not retrieve information about the channel")
	}
	if channel.IsMultiIM() {
		return marvin.CmdErrorf(args, "You cannnot invite users to a multi-party IM.")
	}

	usage := func() error {
		return marvin.CmdErrorf(args, "Usage: `@marvin make-invite` [`emoji` = :white_check_mark:] <`send_to` = #boardgame>")
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
	match := channelMentionRgx.FindStringSubmatch(arg)
	if match == nil {
		return usage()
	}
	messageTarget := slack.ChannelID(match[1])

	customMsg := strings.TrimSpace(strings.Join(args.Arguments, " "))
	msg := ""
	if customMsg != "" {
		msg = fmt.Sprintf(defaultInviteText, args.Source.UserID(), channel.Name, andSaid, customMsg)
	} else {
		msg = fmt.Sprintf(defaultInviteText, args.Source.UserID(), channel.Name, ".", "")
	}



	fmt.Println("[DEBUG]", "sending invite to", messageTarget, "text:", msg)
	_, myRTM, err := t.SendMessage(messageTarget, msg)
	if err != nil {
		return errors.Wrap(err, "Failed to send message")
	}
	err = t.ReactMessage(messageTarget, myRTM.MessageTS(), emoji)
	if err != nil {
		return err
	}
	return nil
}
