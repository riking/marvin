package marvin

import (
	"fmt"
	"net"
	"os"
	"strings"

	"gopkg.in/ini.v1"

	"github.com/riking/homeapi/marvin/slack"
)

// TeamConfig is loaded from the config.ini file.
type TeamConfig struct {
	TeamDomain      string
	ClientID        string
	ClientSecret    string
	CookieSecretKey string
	IntraUID        string
	IntraSecret     string
	DatabaseURL     string
	UserToken       string
	LogChannel      slack.ChannelID
	HTTPListen      string
	HTTPURL         string
	Controller      slack.UserID
	IsDevelopment   bool
}

func LoadTeamConfig(sec *ini.Section) *TeamConfig {
	c := &TeamConfig{}
	c.TeamDomain = sec.Key("TeamDomain").String()
	c.ClientID = sec.Key("ClientID").String()
	c.ClientSecret = sec.Key("ClientSecret").String()
	c.CookieSecretKey = sec.Key("CookieSecretKey").String()
	c.IntraUID = sec.Key("IntraUID").String()
	c.IntraSecret = sec.Key("IntraSecret").String()
	c.DatabaseURL = sec.Key("DatabaseURL").String()
	c.UserToken = sec.Key("UserToken").String()
	c.HTTPListen = sec.Key("HTTPListen").String()
	c.HTTPURL = sec.Key("HTTPURL").String()
	c.LogChannel = slack.ChannelID(sec.Key("LogChannel").String())
	c.Controller = slack.UserID(sec.Key("Controller").String())
	c.IsDevelopment, _ = sec.Key("IsDevelopment").Bool()

	if c.HTTPURL == "__auto" {
		hostname, err := os.Hostname()
		if err != nil {
			return c
		}
		idx := strings.Index(hostname, ".")
		_, port, err := net.SplitHostPort(c.HTTPListen)
		if err != nil {
			return c
		}
		c.HTTPURL = fmt.Sprintf("http://%s:%s", hostname[:idx], port)
	}
	return c
}
