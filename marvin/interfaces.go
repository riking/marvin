package marvin

import (
	"net/http"
	"net/url"

	"gopkg.in/ini.v1"

	"github.com/riking/homeapi/marvin/database"
	"github.com/riking/homeapi/marvin/slack"
)

type ModuleID string

type SendMessage interface {
	SendMessage(channelID slack.ChannelID, message string) (slack.MessageTS, slack.RTMRawMessage, error)
	SendComplexMessage(channelID slack.ChannelID, message url.Values) (slack.MessageTS, error)
}

type ModuleConfig interface {
	Get(key, defaultValue string) (string, error)
	Set(key, value string) error
	Add(key, defaultValue string)
}

type TeamConfig struct {
	TeamDomain   string
	ClientID     string
	ClientSecret string
	VerifyToken  string
	DatabaseURL  string
	UserToken    string
	LogChannel   slack.ChannelID
}

func LoadTeamConfig(sec *ini.Section) *TeamConfig {
	c := &TeamConfig{}
	c.TeamDomain = sec.Key("TeamDomain").String()
	c.ClientID = sec.Key("ClientID").String()
	c.ClientSecret = sec.Key("ClientSecret").String()
	c.VerifyToken = sec.Key("VerifyToken").String()
	c.DatabaseURL = sec.Key("DatabaseURL").String()
	c.UserToken = sec.Key("UserToken").String()
	c.LogChannel = slack.ChannelID(sec.Key("LogChannel").String())
	return c
}

type SlashCommand interface {
	SlashCommand(t Team, req slack.SlashCommandRequest) slack.SlashCommandResponse
}

type SubCommand interface {
	Handle(t Team, args *CommandArguments) CommandResult
	Help(t Team, args *CommandArguments) CommandResult
}

type SubCommandFunc func(t Team, args *CommandArguments) CommandResult

type CommandRegistration interface {
	RegisterCommand(name string, c SubCommand)
	RegisterCommandFunc(name string, c SubCommandFunc, help string) SubCommand
	UnregisterCommand(name string)
}

type HTTPDoer interface {
	Do(*http.Request) (http.Response, error)
}

type Team interface {
	// Domain returns the leftmost component of the Slack domain name.
	Domain() string
	DB() *database.Conn
	TeamConfig() *TeamConfig
	ModuleConfig(mod ModuleID) ModuleConfig

	BotUser() slack.UserID

	// EnableModules loads every module and attempts to transition them to
	// the state listed in the configuration.
	EnableModules()

	// DependModule places the instance of the requested module in the
	// given pointer.
	//
	// If the requested module is already enabled, the pointer is
	// filled immediately and the function returns 1. If the requested
	// module has errored, the pointer is left alone and the function
	// returns -2.
	//
	// During loading, when the requested module has not been enabled
	// yet, the function returns 0 and remembers the pointer. If the
	// requested module is not known, the function returns -1.
	DependModule(self Module, dependencyID ModuleID, ptr *Module) int
	GetModule(modID ModuleID) Module

	SendMessage
	ReactMessage(msgID slack.MessageID, emojiName string) error
	SlackAPIPost(method string, form url.Values) (*http.Response, error)

	ArchiveURL(msgID slack.MessageID) string

	OnEveryEvent(mod ModuleID, f func(slack.RTMRawMessage))
	OnEvent(mod ModuleID, event string, f func(slack.RTMRawMessage))
	OnNormalMessage(mod ModuleID, f func(slack.RTMRawMessage))
	OffAllEvents(mod ModuleID)

	CommandRegistration
	DispatchCommand(args *CommandArguments) CommandResult

	ReportError(err error, source ActionSource)

	ChannelName(channel slack.ChannelID) (string)
	UserName(user slack.UserID) (string)
	GetIM(user slack.UserID) (slack.ChannelID, error)
	PublicChannelInfo(channel slack.ChannelID) (*slack.Channel, error)
	PrivateChannelInfo(channel slack.ChannelID) (*slack.Channel, error)
}

type ShockyInstance interface {
	TeamConfig(teamDomain string) TeamConfig
	ModuleConfig(team TeamConfig) ModuleConfig
	DB(team TeamConfig) *database.Conn

	SendChannelSlack(team Team, channel string, message slack.OutgoingSlackMessage)
	SendPrivateSlack(team Team, user string, message slack.OutgoingSlackMessage)

	RegisterSlashCommand(c SlashCommand)
}
