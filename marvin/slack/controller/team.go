package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"sync"

	"os"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/database"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/slack/rtm"
	"github.com/riking/homeapi/marvin/util"
)

type Team struct {
	teamConfig *marvin.TeamConfig
	client     *rtm.Client
	db         *database.Conn
	commands   *marvin.ParentCommand

	modulesLock sync.Mutex
	modules     []*moduleStatus

	confLock sync.Mutex
	confMap  map[marvin.ModuleID]*DBModuleConfig
}

func NewTeam(cfg *marvin.TeamConfig) (*Team, error) {
	db, err := database.Dial(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	err = MigrateModuleConfig(db)
	if err != nil {
		return nil, err
	}

	return &Team{
		teamConfig: cfg,
		client:     nil, // ConnectRTM()
		db:         db,
		commands:   marvin.NewParentCommand(),
		modules:    nil,
		confMap:    make(map[marvin.ModuleID]*DBModuleConfig),
	}, nil
}

func (t *Team) ConnectRTM(c *rtm.Client) {
	t.client = c
}

func (t *Team) EnableModules() {
	t.ModuleConfig("modules").(*DBModuleConfig).DefaultsLocked = true
	t.ModuleConfig("blacklist").(*DBModuleConfig).DefaultsLocked = true

	t.constructModules()
	t.loadModules()
	t.enableModules()
}

func (t *Team) Domain() string {
	return t.teamConfig.TeamDomain
}

func (t *Team) TeamConfig() *marvin.TeamConfig {
	return t.teamConfig
}

func (t *Team) DB() *database.Conn {
	return t.db
}

func (t *Team) ModuleConfig(ident marvin.ModuleID) marvin.ModuleConfig {
	st := t.GetModuleStatus(ident)
	if st == nil {
		if ident != "modules" && ident != "blacklist" {
			return nil
		}
	}

	t.confLock.Lock()
	defer t.confLock.Unlock()
	conf, ok := t.confMap[ident]
	if ok {
		return conf
	}
	conf = newModuleConfig(t, ident)
	t.confMap[ident] = conf
	return conf
}

func (t *Team) BotUser() slack.UserID {
	return t.client.Self.ID
}

// ---

func (t *Team) RegisterCommand(name string, c marvin.SubCommand) {
	t.commands.RegisterCommand(name, c)
}

func (t *Team) RegisterCommandFunc(name string, c marvin.SubCommandFunc, help string) marvin.SubCommand {
	return t.commands.RegisterCommandFunc(name, c, help)
}

func (t *Team) UnregisterCommand(name string) {
	t.commands.UnregisterCommand(name)
}

func (t *Team) DispatchCommand(args *marvin.CommandArguments) marvin.CommandResult {
	var result marvin.CommandResult
	err := util.PCall(func() error {
		result = t.commands.Handle(t, args)
		return nil
	})
	if err != nil {
		return marvin.CmdError(args, err, "Runtime error")
	}
	return result
}

func (t *Team) Help(args *marvin.CommandArguments) marvin.CommandResult {
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

func (t *Team) ReactMessage(msgID slack.MessageID, emojiName string) error {
	form := url.Values{
		"name":      []string{emojiName},
		"channel":   []string{string(msgID.ChannelID)},
		"timestamp": []string{string(msgID.MessageTS)},
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
	util.LogDebug("Slack API request", method, form)

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
		return resp, errors.Wrapf(err, "Slack API %s", method)
	}
	return resp, nil
}

// ---

func (t *Team) OnEveryEvent(mod marvin.ModuleID, f func(slack.RTMRawMessage)) {
	t.client.RegisterRawHandler(mod, f, rtm.MsgTypeAll, nil)
}

func (t *Team) OnEvent(mod marvin.ModuleID, event string, f func(slack.RTMRawMessage)) {
	t.client.RegisterRawHandler(mod, f, event, nil)
}

var _filterNoSubgroup = []string{""}

func (t *Team) OnNormalMessage(mod marvin.ModuleID, f func(slack.RTMRawMessage)) {
	t.client.RegisterRawHandler(mod, f, "message", _filterNoSubgroup)
}

func (t *Team) OffAllEvents(mod marvin.ModuleID) {
	t.client.UnregisterAllMatching(mod)
}

// ---

func (t *Team) ArchiveURL(msgID slack.MessageID) string {
	splitTS := strings.Split(string(msgID.MessageTS), ".")
	stripTS := "p" + splitTS[0] + splitTS[1]

	channel := msgID.ChannelID
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

func (t *Team) ReportError(err error, source marvin.ActionSource) {
	fmt.Fprintf(os.Stderr, "[ERR] From %v: %+v", source, err)
}
