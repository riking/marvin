package weblogin

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/antonlindstrom/pgstore"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"crypto/aes"
	"encoding/hex"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
)

type API interface {
	marvin.Module

	// GetUser returns an empty string if the user has not authed with Slack.
	// An error is returned only in case of corrupt cookie data.
	GetUser(w http.ResponseWriter, r *http.Request) (slack.UserID, error)
	// Returns an empty string if the user has not authed with Slack.
	// An error is returned only in case of corrupt cookie data.
	GetUserToken(w http.ResponseWriter, r *http.Request) (string, error)
	StartURL() string

	HTTPError(w http.ResponseWriter, r *http.Request, err error)
}

var _ API = &WebLoginModule{}

// ---

func init() {
	marvin.RegisterModule(NewWebLoginModule)
}

const Identifier = "weblogin"

type WebLoginModule struct {
	team        marvin.Team
	oauthConfig oauth2.Config
	store       sessions.Store
}

func NewWebLoginModule(t marvin.Team) marvin.Module {
	mod := &WebLoginModule{
		team: t,
		oauthConfig: oauth2.Config{
			ClientID:     t.TeamConfig().ClientID,
			ClientSecret: t.TeamConfig().ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://slack.com/oauth/authorize",
				TokenURL: "https://slack.com/api/oauth.access",
			},
			RedirectURL: t.AbsoluteURL("/oauth/slack/callback"),
			Scopes:      []string{"identify", "groups:read"},
		},
		store: nil,
	}

	return mod
}

func (mod *WebLoginModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *WebLoginModule) Load(t marvin.Team) {
	b, err := hex.DecodeString(t.TeamConfig().CookieSecretKey)
	if err != nil || len(b) != aes.BlockSize {
		panic(errors.Errorf("CookieSecretKey must be a %d-byte hex string", aes.BlockSize))
	}

	store, err := pgstore.NewPGStoreFromPool(
		t.DB().DB,
		b, b,
	)
	if err != nil {
		panic(errors.Wrap(err, "Could not setup session store"))
	}
	uri, err := url.Parse(t.TeamConfig().HTTPURL)
	if err != nil {
		panic(err)
	}
	store.Options.HttpOnly = true
	store.Options.Domain = uri.Host
	if uri.Path != "" {
		store.Options.Path = uri.Path
	}
	if strings.HasPrefix(t.TeamConfig().HTTPURL, "https") {
		store.Options.Secure = true
	}
	mod.store = store
}

func (mod *WebLoginModule) Enable(team marvin.Team) {
	team.HandleHTTP("/oauth/slack/start", http.HandlerFunc(mod.OAuthSlackStart))
	team.HandleHTTP("/oauth/slack/callback", http.HandlerFunc(mod.OAuthSlackCallback))
	team.HandleHTTP("/", http.HandlerFunc(mod.ServeRoot))
	team.Router().PathPrefix("/assets/").HandlerFunc(mod.ServeAsset)
	team.Router().NotFoundHandler = http.HandlerFunc(mod.Serve404)
}

func (mod *WebLoginModule) Disable(team marvin.Team) {
}

// ---

const (
	sqlMigrate1 = `
	CREATE TABLE slack_login_tokens (
		id serial primary key,
		userID      varchar(10), -- slack.UserID
		username    varchar(64),
		token       varchar(60), -- slack.LoginToken
		scopes      text[],
		created_at  timestamp with zone,
	)`
)

// ---

func (mod *WebLoginModule) StartURL() string {
	return mod.team.AbsoluteURL("/oauth/slack/start")
}
