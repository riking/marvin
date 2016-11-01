package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/database"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/slack/rtm"
)

type Team struct {
	teamConfig *marvin.TeamConfig
	client     *rtm.Client
	db         *database.Conn

	commands marvin.ParentCommand
}

func NewTeam(cfg *marvin.TeamConfig) (*Team, error) {
	db, err := database.Dial(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	err = marvin.MigrateModuleConfig(db)
	if err != nil {
		return nil, err
	}

	return &Team{
		teamConfig: cfg,
		db:         db,
		commands:   marvin.NewParentCommand(),
	}, nil
}

func (t *Team) Connect(c *rtm.Client) {
	t.client = c

	//for _, v := range c.Ims {
	//
	//}
}

func (t *Team) Domain() string {
	return t.teamConfig.TeamDomain
}

func (t *Team) TeamConfig() *marvin.TeamConfig {
	return t.teamConfig
}

func (t *Team) DB() *database.Conn {
	panic("Not implemented")
	return nil // TODO
}

func (t *Team) ModuleConfig() marvin.ModuleConfig {
	panic("Not implemented")
	// TODO - needs DB()
	return nil
}

func (t *Team) BotUser() slack.UserID {
	return t.client.Self.ID
}

// ---

func (t *Team) RegisterCommand(name string, c marvin.SubCommand) {
	t.commands.RegisterCommand(name, c)
}

func (t *Team) UnregisterCommand(name string, c marvin.SubCommand) {
	t.commands.UnregisterCommand(name, c)
}

func (t *Team) DispatchCommand(args *marvin.CommandArguments) error {
	return t.commands.DispatchCommand(t, args)
}

func (t *Team) Help(args *marvin.CommandArguments) error {
	return t.commands.Help(marvin.Team(t), args)
}

// ---

func (t *Team) SendMessage(channel slack.ChannelID, message string) (slack.MessageTS, error) {
	msg, err := t.client.SendMessage(channel, message)
	if err != nil {
		return "", err
	}
	return msg.Timestamp(), err
}

func (t *Team) SendComplexMessage(channelID slack.ChannelID, message url.Values) (slack.MessageTS, error) {
	message.Set("channel", string(channelID))
	message.Set("token", t.teamConfig.UserToken)
	message.Set("as_user", "true")

	resp, err := t.SlackAPIPost(`https://slack.com/api/chat.postMessage`, message)
	if err != nil {
		return "", errors.Wrap(err, "post slack chat.postMessage")
	}
	var response struct {
		slack.APIResponse
		TS      slack.MessageTS `json:"ts"`
		Channel slack.ChannelID `json:"channel"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", errors.Wrap(err, "decode json")
	}
	resp.Body.Close()
	if !response.OK {
		return "", response.APIResponse
	}
	return response.TS, nil
}

func (t *Team) SlackAPIPost(method string, form url.Values) (*http.Response, error) {
	fmt.Println("[DEBUG]", "Slack API request", method, form)

	var url string
	if strings.HasPrefix(method, "https://slack.com") {
		url = method
	} else {
		url = fmt.Sprintf("https://slack.com/api/%s", method)
	}
	form.Set("token", t.teamConfig.UserToken)

	req, err := http.NewRequest("POST", url, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "marvin-slackbot (+https://github.com/riking/homeapi/tree/shocky)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return resp, errors.Wrapf(err, "Error calling slack.%s", method)
	}
	return resp, nil
}

func (t *Team) SubmitLateSlashCommand(responseURL string, resp slack.SlashCommandResponse) {

}

func (t *Team) OnEveryEvent(unregisterID string, f func(slack.RTMRawMessage)) {
	t.client.RegisterRawHandler(unregisterID, f, rtm.MsgTypeAll, nil)
}

func (t *Team) OnEvent(unregisterID string, event string, f func(slack.RTMRawMessage)) {
	t.client.RegisterRawHandler(unregisterID, f, event, nil)
}

var _filterNoSubgroup = []string{""}

func (t *Team) OnNormalMessage(unregisterID string, f func(slack.RTMRawMessage)) {
	t.client.RegisterRawHandler(unregisterID, f, "message", _filterNoSubgroup)
}

func (t *Team) OffAllEvents(unregisterID string) {
	t.client.UnregisterAllMatching(unregisterID)
}

func (t *Team) PrivateChannelInfo(channel slack.ChannelID) (*slack.Channel, error) {
	// TODO caching
	form := url.Values{"channel": []string{string(channel)}}
	resp, err := t.SlackAPIPost("groups.info", form)
	if err != nil {
		return nil, err
	}
	var response struct {
		slack.APIResponse
		Group slack.Channel `json:"group"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, errors.Wrap(err, "decode json")
	}
	resp.Body.Close()
	if !response.OK {
		return nil, response.APIResponse
	}
	return &response.Group, nil
}

func (t *Team) GetIM(user slack.UserID) (slack.ChannelID, error) {
	// TODO caching
	form := url.Values{"user": []string{string(user)}}
	resp, err := t.SlackAPIPost("im.open", form)
	if err != nil {
		return "", err
	}
	var response struct {
		slack.APIResponse
		Channel struct {
			ID slack.ChannelID `json:"id"`
		} `json:"channel"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", errors.Wrap(err, "decode json")
	}
	resp.Body.Close()
	if !response.OK {
		return "", response.APIResponse
	}
	return response.Channel.ID, nil
}
