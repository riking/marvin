package weblogin

import (
	"crypto/aes"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/antonlindstrom/pgstore"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
)

type API interface {
	marvin.Module

	// GetCurrentUser gets the current User object for the request's cookies.
	// If there is no logged in user, this method returns (nil, nil).
	// An error is returned only in case of corrupt cookie data.
	GetCurrentUser(w http.ResponseWriter, r *http.Request) (*User, error)

	// GetUserBySlack gets the User object for the given Slack user.
	// If no associated Slack account is found, ErrNoSuchUser is returned.
	GetUserBySlack(slackID slack.UserID) (*User, error)
	// GetUserByIntra gets the User object for the given Intra username.
	// If no associated Intra account is found, ErrNoSuchUser is returned.
	GetUserByIntra(login string) (*User, error)

	// StartSlackURL returns the URL to redirect to to start Slack authentication.
	StartSlackURL(returnURL string, extraScopes ...string) string
	// StartSlackURL returns the URL to redirect to to start Intra authentication.
	StartIntraURL(returnURL string, extraScopes ...string) string

	// HTTPError renders a formatted error page.
	HTTPError(w http.ResponseWriter, r *http.Request, err error)
}

var _ API = &WebLoginModule{}

// ---

func init() {
	marvin.RegisterModule(NewWebLoginModule)
}

const Identifier = "weblogin"

type WebLoginModule struct {
	team             marvin.Team
	slackOAuthConfig oauth2.Config
	IntraOAuthConfig oauth2.Config
	store            sessions.Store

	authTokenMap  map[string]authNonceValue
	authTokenLock sync.Mutex
}

func NewWebLoginModule(t marvin.Team) marvin.Module {
	mod := &WebLoginModule{
		team: t,
		slackOAuthConfig: oauth2.Config{
			ClientID:     t.TeamConfig().ClientID,
			ClientSecret: t.TeamConfig().ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://slack.com/oauth/authorize",
				TokenURL: "https://slack.com/api/oauth.access",
			},
			RedirectURL: t.AbsoluteURL("/oauth/slack/callback"),
			Scopes:      []string{"BOGUS_VALUE"},
		},
		IntraOAuthConfig: oauth2.Config{
			ClientID:     t.TeamConfig().IntraUID,
			ClientSecret: t.TeamConfig().IntraSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://api.intra.42.fr/oauth/authorize",
				TokenURL: "https://api.intra.42.fr/oauth/token",
			},
			RedirectURL: t.AbsoluteURL("/oauth/intra/callback"),
			Scopes:      []string{},
		},
		store:        nil,
		authTokenMap: make(map[string]authNonceValue),
	}

	return mod
}

func (mod *WebLoginModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *WebLoginModule) Load(t marvin.Team) {
	keyBytes, err := hex.DecodeString(t.TeamConfig().CookieSecretKey)
	if err != nil || len(keyBytes) != aes.BlockSize {
		panic(errors.Errorf("CookieSecretKey must be a %d-byte hex string", aes.BlockSize))
	}

	store, err := pgstore.NewPGStoreFromPool(
		t.DB().DB,
		keyBytes, keyBytes,
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

	mod.team.DB().MustMigrate(Identifier, 1482049049, sqlMigrateUser1, sqlMigrateUser2, sqlMigrateUser3)
	mod.team.DB().SyntaxCheck(
		sqlLoadUser,
		sqlNewUser,
		sqlUpdateIntra,
		sqlUpdateSlack,
		sqlLookupUserBySlack,
		sqlLookupUserByIntra,
	)
}

func (mod *WebLoginModule) Enable(team marvin.Team) {
	team.Router().HandleFunc("/oauth/slack/start", mod.OAuthSlackStart)
	team.Router().HandleFunc("/oauth/slack/callback", mod.OAuthSlackCallback)
	team.Router().HandleFunc("/oauth/altslack/start", mod.OAuthAltSlackStart)
	team.Router().HandleFunc("/oauth/intra/start", mod.OAuthIntraStart)
	team.Router().HandleFunc("/oauth/intra/callback", mod.OAuthIntraCallback)

	team.Router().HandleFunc("/", mod.ServeRoot)
	team.Router().PathPrefix("/assets/").HandlerFunc(mod.ServeAsset)
	team.Router().HandleFunc("/session/csrf.json", mod.ServeCSRF)
	team.Router().NotFoundHandler = http.HandlerFunc(mod.Serve404)

	team.RegisterCommandFunc("web-authenticate", mod.CommandWebAuthenticate, "Used for assosciating a intra login with a slack name.")
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			mod.janitorAuthToken()
		}
	}()
}

func (mod *WebLoginModule) Disable(team marvin.Team) {
}

// ---

// ---

func (mod *WebLoginModule) StartURL() string {
	return mod.team.AbsoluteURL("/oauth/slack/start")
}
