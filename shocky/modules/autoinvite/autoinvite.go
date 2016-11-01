package autoinvite

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/riking/homeapi/shocky"
	"github.com/riking/homeapi/shocky/slack"
)

func init() {
	shocky.RegisterModule(NewAutoInviteModule)
}

const Identifier = "autoinvite"

type AutoInviteModule struct {
}

func NewAutoInviteModule(t shocky.Team) shocky.Module {
	aim := &AutoInviteModule{}
	t.RegisterCommand("make-invite", shocky.SubCommandFunc(aim.PostInvite))
	return aim
}

func (aim *AutoInviteModule) Identifier() string {
	return Identifier
}

func (aim *AutoInviteModule) Unregister(t shocky.Team) {
	t.UnregisterCommand("make-invite", shocky.SubCommandFunc(aim.PostInvite))
}

func (aim *AutoInviteModule) RegisterRTMEvents(t shocky.Team) {
}

const defaultInviteText = `Click here to be added to the %s channel!`
const defaultEmoji = `white_check_mark`

func (aim *AutoInviteModule) PostInvite(t shocky.Team, args *shocky.CommandArguments) error {
	inviteTarget := args.Source.ChannelID()
	if inviteTarget == "" || inviteTarget[0] != 'G' {
		return shocky.CmdErrorf(args, "Command must be used from a private channel.")
	}
	channel, err := t.PrivateChannelInfo(inviteTarget)
	if err != nil {
		return errors.Wrap(err, "Could not retrieve information about the channel")
	}
	if channel.IsMultiIM() {
		return shocky.CmdErrorf(args, "You cannnot invite users to a multi-party IM.")
	}

	if len(args.Arguments) < 2 {
		// TODO - allow choice of emoji
		return shocky.CmdErrorf(args, "Usage: `@shocky make-invite <send_to = #boardgame> [message = %s]",
			fmt.Sprintf(defaultInviteText, channel.Name))
	}

	// Command passed validation

	messageTarget := slack.ChannelID(args.Arguments[1]) // XXX
	emoji := defaultEmoji

	contents := strings.TrimSpace(strings.Join(args.Arguments[2:], " "))
	if len(contents) == 0 {
		contents = fmt.Sprintf(defaultInviteText, channel.Name)
	}
	ts, err := t.SendMessage(messageTarget, contents)
	if err != nil {
		return errors.Wrap(err, "Failed to send message")
	}
	form := url.Values{
		"name":      []string{emoji},
		"timestamp": []string{string(ts)},
	}
	resp, err := t.SlackAPIPost("reactions.add", form)
	var response struct {
		slack.APIResponse
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return errors.Wrap(errors.Wrap(err, "failed to decode json"), "Failed to add example reaction")
	}
	if !response.OK {
		return errors.Wrap(response.APIResponse, "Failed to add example reaction")
	}
	return nil
}
