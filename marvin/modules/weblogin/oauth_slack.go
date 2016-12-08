package weblogin

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
)

var (
	cookieLoginTmp = "login"
	keyOauthNonce  = "login-nonce"
	keyAfterLogin  = "login-redirect"

	cookieLongTerm = "slack"
	keyUserID      = "slack-id"
	keyUserToken   = "slack-token"
	keyUserScope   = "slack-scope"
)

const shortTermMaxAge = 10 * time.Minute
const longTermMaxAge = 6 * 30 * 24 * time.Hour

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
		sess.Options.MaxAge = int(shortTermMaxAge / time.Second)
	} else if name == cookieLongTerm {
		sess.Options.MaxAge = int(longTermMaxAge / time.Second)
	}

	return sess, true
}

func (mod *WebLoginModule) OAuthSlackStart(w http.ResponseWriter, r *http.Request) {
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

	redirectURL := mod.oauthConfig.AuthCodeURL(nonce, oauth2.SetAuthURLParam("team", string(mod.team.TeamID())))
	err = loginSession.Save(r, w)
	if err != nil {
		util.LogError(err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (mod *WebLoginModule) OAuthSlackCallback(w http.ResponseWriter, r *http.Request) {
	loginSession, ok := mod.getSession(w, r, cookieLoginTmp)
	if !ok {
		return
	}

	// Burn nonce (this deletes record from database)
	loginSession.Options.MaxAge = -1
	err := loginSession.Save(r, w)
	if err != nil {
		util.LogError(err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if loginSession.IsNew {
		http.Error(w, "nonce expired", http.StatusBadRequest)
		return
	}

	err = r.ParseForm()
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
	tokenScope := token.Extra("scope").(string)

	form := url.Values{}
	form.Set("token", token.AccessToken)

	var resp struct {
		UserID slack.UserID `json:"user_id"`
		TeamID slack.TeamID `json:"team_id"`
	}

	err = mod.team.SlackAPIPostJSON("auth.test", form, &resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("bad token/could not contact Slack: %s", err), http.StatusBadRequest)
		return
	}
	if resp.TeamID != mod.team.TeamID() {
		util.LogBad("(http) Bad team id, got", resp.TeamID, "wanted", mod.team.TeamID())
		http.Error(w, fmt.Sprintf("Wrong Slack team! This is only available for %s.slack.com",
			mod.team.TeamConfig().TeamDomain), http.StatusBadRequest)
		return
	}

	permSession, ok := mod.getSession(w, r, cookieLongTerm)
	if !ok {
		return
	}
	permSession.Values[keyUserID] = string(resp.UserID)
	permSession.Values[keyUserToken] = token.AccessToken
	permSession.Values[keyUserScope] = tokenScope
	err = permSession.Save(r, w)
	if err != nil {
		util.LogError(err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	redirectURL, ok := loginSession.Values[keyAfterLogin].(string)
	if ok {
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
	w.Write([]byte(`You are now logged in, redirecting to the homepage...`))
}

func (mod *WebLoginModule) GetUser(w http.ResponseWriter, r *http.Request) (slack.UserID, error) {
	permSession, ok := mod.getSession(w, r, cookieLongTerm)
	if !ok {
		return "", errors.Errorf("Bad cookie data")
	}
	if permSession.IsNew {
		return "", nil
	}
	uid, ok := permSession.Values[keyUserID].(string)
	if !ok {
		return "", nil
	}
	return slack.UserID(uid), nil
}

func (mod *WebLoginModule) GetUserToken(w http.ResponseWriter, r *http.Request) (string, error) {
	permSession, ok := mod.getSession(w, r, cookieLongTerm)
	if !ok {
		return "", errors.Errorf("Bad cookie data")
	}
	if permSession.IsNew {
		return "", nil
	}
	token, ok := permSession.Values[keyUserToken].(string)
	if !ok {
		return "", nil
	}
	return token, nil
}
