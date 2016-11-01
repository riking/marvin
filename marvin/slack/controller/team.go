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
	modules    []marvin.Module

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
	result := t.panicDispatch(args)
	return result
}

func (t *Team) panicDispatch(args *marvin.CommandArguments) (err error) {
	defer func() {
		rec := recover()
		if rec != nil {
			if recErr, ok := rec.(error); ok {
				err = recErr
			} else if recStr, ok := rec.(string); ok {
				err = errors.Errorf(recStr)
			} else {
				panic(errors.Errorf("Unrecognized panic object type=[%T] val=[%#v]", rec, rec))
			}
		}
	}()
	err = t.commands.Handle(t, args)
	return err
}

func (t *Team) Help(args *marvin.CommandArguments) error {
	return t.commands.Help(t, args)
}

// ---

func (t *Team) SendMessage(channel slack.ChannelID, message string) (slack.MessageTS, slack.RTMRawMessage, error) {
	msg, err := t.client.SendMessage(channel, message)
	if err != nil {
		return "", msg, err
	}
	return msg.MessageTS(), msg, err
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

func (t *Team) ReactMessage(channel slack.ChannelID, msgID slack.MessageTS, emojiName string) error {
	form := url.Values{
		"name":      []string{emojiName},
		"channel":   []string{string(channel)},
		"timestamp": []string{string(msgID)},
	}
	resp, err := t.SlackAPIPost("reactions.add", form)
	var response struct {
		slack.APIResponse
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return errors.Wrap(err, "failed to decode json from reactions.add")
	}
	if !response.OK {
		return errors.Wrap(response.APIResponse, "Error calling reactions.add")
	}
	return nil
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

// ---

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

// ---

func (t *Team) ArchiveURL(channel slack.ChannelID, msg slack.MessageTS) string {
	splitTS := strings.Split(string(msg), ".")
	stripTS := "p" + splitTS[0] + splitTS[1]
	if channel[0] == 'D' {
		return fmt.Sprintf("https://%s.slack.com/archives/%s/%s",
			t.teamConfig.TeamDomain, channel, stripTS)
	}
	if channel[0] == 'G' {
		info, err := t.PrivateChannelInfo(channel)
		if err != nil || info.IsMultiIM() {
			return fmt.Sprintf("https://%s.slack.com/archives/%s/%s",
				t.teamConfig.TeamDomain, channel, stripTS)
		} else {
			return fmt.Sprintf("https://%s.slack.com/archives/%s/%s",
				t.teamConfig.TeamDomain, info.Name, stripTS)
		}
	}
	if channel[0] == 'C' {
		info, err := t.PublicChannelInfo(channel)
		if err != nil {
			panic(errors.Wrap(err, "could not get info about public channel"))
		}
		return fmt.Sprintf("https://%s.slack.com/archives/%s/%s",
			t.teamConfig.TeamDomain, info.Name, stripTS)
	}
	panic(errors.Errorf("Invalid channel id '%s' passed to ArchiveURL", channel))
}

// ---

func (t *Team) EnableModules() error {
	var modList []marvin.Module

	for _, constructor := range marvin.AllModules() {
		mod := constructor(t)
		modList = append(modList, mod)
	}
	for _, v := range modList {
		v.RegisterRTMEvents(t)
	}
	t.modules = modList
	return nil
}
