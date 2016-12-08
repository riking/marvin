package weblogin

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/antonlindstrom/pgstore"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
)

type API interface {
	marvin.Module

	GetUser(w http.ResponseWriter, r *http.Request) slack.UserID
	StartLogin(w http.ResponseWriter, r *http.Request)
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
			RedirectURL: "https://marvin.riking.org/oauth/callback",
			Scopes:      []string{"channels:read", "groups:read", "identity.basic"},
		},
		store: nil,
	}

	return mod
}

func (mod *WebLoginModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *WebLoginModule) Load(t marvin.Team) {
	store, err := pgstore.NewPGStoreFromPool(
		t.DB(),
		[]byte(t.TeamConfig().CookieSecret1), []byte(t.TeamConfig().CookieSecret2),
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
}

func (mod *WebLoginModule) Enable(team marvin.Team) {
	team.HandleHTTP("/oauth/start", mod.StartLogin)
	team.HandleHTTP("/oauth/callback", mod.OAuthCallback)
}

func (mod *WebLoginModule) Disable(team marvin.Team) {
}

// ---

var (
	cookieLoginTmp = "login"
	keyOauthNonce  = "login-nonce"
	keyAfterLogin  = "login-redirect"

	cookieLongTerm = "slack"
	keyUserID      = "slack-id"
	keyUserToken   = "slack-token"
)

const shortTermMaxAge = 10 * time.Minute / time.Second
const longTermMaxAge = 6 * 30 * 24 * time.Hour / time.Second

func (mod *WebLoginModule) getSession(w http.ResponseWriter, r *http.Request, name string) (*sessions.Session, bool) {
	sess, err := mod.store.Get(r, name)
	if err != nil {
		if err, ok := err.(securecookie.Error); ok {
			if err.IsDecode() {
				sess.Options.MaxAge = -1
				sess.Save(r, w)
				util.LogBadf("Cookie decode error: %s", err)
				http.Error(w, "invalid cookies, please login again", http.StatusBadRequest)
				return nil, false
			}
		}

		util.LogBadf("Cookie error: %s", err)
		http.Error(w, fmt.Sprintf("cookie error: %s", err), http.StatusInternalServerError)
		return nil, false
	}

	if name == cookieLoginTmp {
		sess.Options.MaxAge = shortTermMaxAge
	} else if name == cookieLongTerm {
		sess.Options.MaxAge = longTermMaxAge
	}

	return sess, true
}

func (mod *WebLoginModule) StartLogin(w http.ResponseWriter, r *http.Request) {
	loginSession, ok := mod.getSession(w, r, cookieLoginTmp)
	if !ok {
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "bad form encoding", http.StatusBadRequest)
		return
	}

	var bytes [16]byte
	rand.Read(bytes[:])
	nonce := base64.URLEncoding.EncodeToString(bytes[:])
	loginSession.Values[keyOauthNonce] = nonce

	if after := r.Form.Get("redirect_url"); after != "" {
		loginSession.Values[keyAfterLogin] = after
	}

	redirectURL := mod.oauthConfig.AuthCodeURL(nonce)
	loginSession.Save(r, w)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (mod *WebLoginModule) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	loginSession, ok := mod.getSession(w, r, cookieLoginTmp)
	if !ok {
		return
	}

	// Burn nonce (this deletes record from database)
	loginSession.Options.MaxAge = -1
	loginSession.Save(r, w)

	if loginSession.IsNew {
		http.Error(w, "nonce expired", http.StatusBadRequest)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "bad form encoding", http.StatusBadRequest)
		return
	}

	slackErr := r.Form.Get("error")
	if slackErr != "" {
		http.Error(w, "slack error: "+slackErr, http.StatusBadRequest)
		return
	}

	nonceExpect, anyStored := loginSession.Values[keyOauthNonce]
	nonceGot := r.Form.Get("state")
	if nonceExpect != nonceGot || !anyStored {
		http.Error(w, "nonce mismatch", http.StatusBadRequest)
		return
	}

	code := r.Form.Get("code")
	token, err := mod.oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, fmt.Sprintf("bad token/could not contact Slack: %s", err), http.StatusBadRequest)
		return
	}

	form := url.Values{}
	form.Set("token", token.AccessToken)

	var resp struct {
		User struct {
			ID slack.UserID `json:"id"`
		} `json:"user"`
		Team struct {
			ID slack.TeamID `json:"id"`
		} `json:"team"`
	}

	err = mod.team.SlackAPIPostJSON("users.identity", form, &resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("bad token/could not contact Slack: %s", err), http.StatusBadRequest)
		return
	}
	if resp.Team.ID != mod.team.TeamID() {
		http.Error(w, fmt.Sprintf("Wrong Slack team! This is only available for %s.slack.com",
			mod.team.TeamConfig().TeamDomain), http.StatusBadRequest)
		return
	}

	permSession, ok := mod.getSession(w, r, cookieLongTerm)
	if !ok {
		return
	}
	permSession[keyUserID] = resp.User.ID
	permSession[keyUserToken] = token.AccessToken
	permSession.Save(r, w)

	redirectURL, ok := loginSession.Values[keyAfterLogin]
	if ok {
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
	w.Write([]byte(`You are now logged in, redirecting to the homepage...`))
}
