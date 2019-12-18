package marvin

import (
	"crypto/sha256"
	"fmt"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/hkdf"
	"gopkg.in/ini.v1"

	"github.com/riking/marvin/slack"
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
	Controllers     []slack.UserID
	ChannelPrefix   slack.ChannelID
	IsDevelopment   bool
	IsReadOnly   	bool
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
	c.ChannelPrefix = slack.ChannelID(sec.Key("ChannelPrefix").String())
	c.IsDevelopment, _ = sec.Key("IsDevelopment").Bool()
	c.IsReadOnly, _ = sec.Key("IsReadOnly").Bool()

	var controllerKey = sec.Key("Controller").String()
	var split = strings.Split(controllerKey, ",")
	c.Controllers = make([]slack.UserID, len(split))
	for uid := range c.Controllers {
		c.Controllers[uid] = slack.UserID(split[uid])
	}

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

func (t *TeamConfig) IsController(user slack.UserID) bool {
	for id := range t.Controllers {
		if t.Controllers[id] == user {
			return true
		}
	}
	return false
}

// This checks if the channel where a factoid / command
// invocation is coming from one of our own channels.
// It will ignore the message otherwise.
func (t *TeamConfig) CheckChannelName(chanName string) bool {
	if len(t.ChannelPrefix) > 0 {
		if !strings.HasPrefix(chanName, string("#"+t.ChannelPrefix)) {
			fmt.Printf("Unapproved channel! %s\n", chanName, t.ChannelPrefix)
			// Unapproved public channel.
			return false
		}
	}
	return true
}

// GetSecretKey expands the CookieSecretKey value using the 'purpose' parameter as a salt.
// An example value for 'purpose' would be "csrf protection".
func (t *TeamConfig) GetSecretKey(purpose string, p []byte) (n int, err error) {
	kdf := hkdf.New(sha256.New,
		[]byte(t.CookieSecretKey),
		[]byte(purpose), []byte(purpose))
	return kdf.Read(p)
}
