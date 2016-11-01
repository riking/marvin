package marvin

import (
	"database/sql"
	"net/http"
	"net/url"

	"gopkg.in/ini.v1"

	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/database"
)

type SendMessage interface {
	SendMessage(channelID slack.ChannelID, message string) (slack.MessageTS, error)
	SendComplexMessage(channelID slack.ChannelID, message url.Values) (slack.MessageTS, error)
}

type ModuleConfig interface {
	Get(key string) (string, error)
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
}

func LoadTeamConfig(sec *ini.Section) *TeamConfig {
	c := &TeamConfig{}
	c.TeamDomain = sec.Key("TeamDomain").String()
	c.ClientID = sec.Key("ClientID").String()
	c.ClientSecret = sec.Key("ClientSecret").String()
	c.VerifyToken = sec.Key("VerifyToken").String()
	c.DatabaseURL = sec.Key("DatabaseURL").String()
	c.UserToken = sec.Key("UserToken").String()
	return c
}

type SlashCommand interface {
	SlashCommand(t Team, req slack.SlashCommandRequest) slack.SlashCommandResponse
}

type SubCommand interface {
	Handle(t Team, args *CommandArguments) error
}

type SubCommandFunc func(t Team, args *CommandArguments) error

func (f SubCommandFunc) Handle(t Team, args *CommandArguments) error {
	return f(t, args)
}

type CommandRegistration interface {
	RegisterCommand(name string, c SubCommand)
	UnregisterCommand(name string, c SubCommand)
}

type HTTPDoer interface {
	Do(*http.Request) (http.Response, error)
}

type Team interface {
	Domain() string
	DB() *database.Conn
	TeamConfig() *TeamConfig
	ModuleConfig() ModuleConfig

	BotUser() slack.UserID

	SendMessage
	SlackAPIPost(method string, form url.Values) (*http.Response, error)
	SubmitLateSlashCommand(responseURL string, resp slack.SlashCommandResponse)

	OnEveryEvent(unregisterID string, f func(slack.RTMRawMessage))
	OnEvent(unregisterID string, event string, f func(slack.RTMRawMessage))
	OnNormalMessage(unregisterID string, f func(slack.RTMRawMessage))
	OffAllEvents(unregisterID string)

	CommandRegistration
	DispatchCommand(args *CommandArguments) error

	GetIM(user slack.UserID) (slack.ChannelID, error)
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
