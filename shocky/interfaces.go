package shocky

import (
	"database/sql"
	"net/http"

	"github.com/riking/homeapi/shocky/slack"
)

type SendMessage interface {
	SendChannel(channelID, message string)
	SendPrivate(userID, message string)
	SendChannelSlack(channelID string, message slack.Message)
	SendPrivateSlack(userID string, message slack.Message)
}

type ModuleConfig interface {
	Get(key string)
	Set(key, value string)
	Add(key, defaultValue string)
}

type TeamConfig struct {
	TeamDomain   string
	ClientID     string
	ClientSecret string
	VerifyToken  string
	DBName       string
	UserToken    string
}

type Module interface {
	Setup(i ShockyInstance)
	WantOnMessage() bool
	OnMessage(m slack.IncomingMessage) error
	WantOnReaction() bool
	OnReaction(m slack.IncomingReaction) error
}

type SlashCommand interface {
	SlashCommand(t Team, req slack.SlashCommandRequest) slack.SlashCommandResponse
}

type SubCommand interface {
	Do(t Team)
}

type Team interface {
	Domain() string
	DB() *sql.DB
	TeamConfig() TeamConfig
	ModuleConfig() ModuleConfig
	SendMessage
	HTTPClient() http.Client
}

type ShockyInstance interface {
	TeamConfig(teamDomain string) TeamConfig
	ModuleConfig(team TeamConfig) ModuleConfig
	DB(team TeamConfig) *sql.DB

	SendChannel(team Team, channel, message string)
	SendPrivate(team Team, user, message string)
	SendChannelSlack(team Team, channel string, message slack.Message)
	SendPrivateSlack(team Team, user string, message slack.Message)

	RegisterModule(m Module)
	RegisterSlashCommand(c SlashCommand)
	RegisterCommand(c SubCommand)

	SubmitLateSlashCommand(responseURL string, resp slack.SlashCommandResponse)
}
