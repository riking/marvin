// The controller package implements the Team type.
package controller

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"

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

	httpMux   *mux.Router
	httpStrip string
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

	t := &Team{
		teamConfig: cfg,
		client:     nil, // ConnectRTM()
		db:         db,
		commands:   marvin.NewParentCommand(),
		modules:    nil,
		confMap:    make(map[marvin.ModuleID]*DBModuleConfig),
		httpMux:    mux.NewRouter(),
	}

	u, err := url.Parse(cfg.HTTPURL)
	if err != nil {
		return nil, err
	}
	if u.Path != "" && u.Path != "/" {
		t.httpStrip = u.Path
	}

	return t, nil
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

func (t *Team) TeamID() slack.TeamID {
	return t.client.AboutTeam.ID
}

func (t *Team) GetRTMClient() interface{} {
	return t.client
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
	fmt.Printf("[%s] [@marvin] %s\n", t.ChannelName(channel), message)
	msg, err := t.client.SendMessage(channel, message)
	if err != nil {
		return "", msg, err
	}
	return msg.MessageTS(), msg, err
}

func (t *Team) SendComplexMessage(channelID slack.ChannelID, form url.Values) (slack.MessageID, slack.RTMRawMessage, error) {
	form.Set("channel", string(channelID))
	form.Set("token", t.teamConfig.UserToken)
	form.Set("as_user", "true")

	var response struct {
		TS      slack.MessageTS     `json:"ts"`
		Channel slack.ChannelID     `json:"channel"`
		Message slack.RTMRawMessage `json:"message"`
	}
	err := t.SlackAPIPostJSON("chat.postMessage", form, &response)
	if err != nil {
		return slack.MessageID{}, nil, err
	}
	return slack.MsgID(response.Channel, response.TS), response.Message, nil
}

func (t *Team) ReactMessage(msgID slack.MessageID, emojiName string) error {
	form := url.Values{
		"name":      []string{emojiName},
		"channel":   []string{string(msgID.ChannelID)},
		"timestamp": []string{string(msgID.MessageTS)},
	}
	return t.SlackAPIPostJSON("reactions.add", form, nil)
}

func (t *Team) SlackAPIPostRaw(method string, form url.Values) (*http.Response, error) {
	var url string
	if strings.HasPrefix(method, "https://slack.com") {
		url = method
	} else {
		url = fmt.Sprintf("https://slack.com/api/%s", method)
	}

	// Allow custom tokens
	if form.Get("token") == "" {
		form.Set("token", t.teamConfig.UserToken)
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(form.Encode()))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "marvin-slackbot (+https://github.com/riking/homeapi/tree/shocky)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (t *Team) SlackAPIPostJSON(method string, form url.Values, result interface{}) error {
	var rawResponse json.RawMessage
	var slackResponse slack.APIResponse

	resp, err := t.SlackAPIPostRaw(method, form)
	if err != nil {
		util.LogBadf("Slack API %s error: %s", method, err)
		return errors.Wrapf(err, "Slack API %s: contact Slack", method)
	}
	err = json.NewDecoder(resp.Body).Decode(&rawResponse)
	resp.Body.Close()
	if err != nil {
		util.LogBadf("Slack API %s error: %s", method, err)
		return errors.Wrapf(err, "Slack API %s: decode json", method)
	}
	err = json.Unmarshal(rawResponse, &slackResponse)
	if err != nil {
		util.LogBadf("Slack API %s error: %s", method, err)
		return errors.Wrapf(err, "Slack API %s: decode json", method)
	}
	if !slackResponse.OK {
		err = slackResponse
		util.LogBadf("Slack API %s error: %s", method, err)
		util.LogBadf("Form for %s: %v", method, form)
		return errors.Wrapf(err, "Slack API %s", method)
	}

	// Early return - no result needed
	if result == nil {
		util.LogDebug("Slack API", method, "success", slackResponse)
		return nil
	}

	err = json.Unmarshal(rawResponse, result)
	if err != nil {
		util.LogBadf("Slack API %s error: %s", method, err)
		return errors.Wrapf(err, "Slack API %s: decode json", method)
	}
	util.LogDebug("Slack API", method, "success", result)
	return nil
}

// ---

func (t *Team) OnEveryEvent(mod marvin.ModuleID, f func(slack.RTMRawMessage)) {
	t.client.RegisterRawHandler(mod, f, rtm.MsgTypeAll, nil)
}

func (t *Team) OnEvent(mod marvin.ModuleID, event string, f func(slack.RTMRawMessage)) {
	t.client.RegisterRawHandler(mod, f, event, nil)
}

func (t *Team) OnSpecialMessage(mod marvin.ModuleID, msgSubtype []string, f func(slack.RTMRawMessage)) {
	t.client.RegisterRawHandler(mod, f, "message", msgSubtype)
}

var _filterNoSubgroup = []string{""}

func (t *Team) OnNormalMessage(mod marvin.ModuleID, f func(slack.RTMRawMessage)) {
	t.client.RegisterRawHandler(mod, f, "message", _filterNoSubgroup)
}

func (t *Team) OffAllEvents(mod marvin.ModuleID) {
	t.client.UnregisterAllMatching(mod)
}

// ---

func (t *Team) ConnectHTTP(l net.Listener) {
	go func() {
		err := http.Serve(l, t.httpMux)
		if err != nil {
			util.LogError(err)
		}
		os.Exit(4)
	}()
}

// HandleHTTP must be called as follows:
//
//   team.HandleHTTP("/links/", module)
func (t *Team) HandleHTTP(folder string, handler http.Handler) *mux.Route {
	return t.httpMux.Handle(folder, http.StripPrefix(t.httpStrip, handler))
}

func (t *Team) Router() *mux.Router {
	return t.httpMux
}

// MakeURL takes a (non-rooted) path to the webserver and makes it absolute.
func (t *Team) AbsoluteURL(path string) string {
	return fmt.Sprintf("%s%s", t.teamConfig.HTTPURL, path)
}

// ---

func (t *Team) ArchiveURL(msgID slack.MessageID) string {
	channel := msgID.ChannelID
	if channel[0] == 'D' {
		return slack.ArchiveURL(t.teamConfig.TeamDomain, "", msgID)
	}
	if channel[0] == 'G' {
		info, err := t.PrivateChannelInfo(channel)
		if err != nil || info.IsMultiIM() {
			return slack.ArchiveURL(t.teamConfig.TeamDomain, "", msgID)
		} else {
			return slack.ArchiveURL(t.teamConfig.TeamDomain, info.Name, msgID)
		}
	}
	if channel[0] == 'C' {
		info, err := t.PublicChannelInfo(channel)
		if err != nil {
			panic(errors.Wrap(err, "could not get info about public channel"))
		}
		return slack.ArchiveURL(t.teamConfig.TeamDomain, info.Name, msgID)
	}
	panic(errors.Errorf("Invalid channel id '%s' passed to ArchiveURL", channel))
}

func (t *Team) ReportError(err error, source marvin.ActionSource) {
	fmt.Fprintf(os.Stderr, "[ERR] From %v: %+v", source, err)
}
