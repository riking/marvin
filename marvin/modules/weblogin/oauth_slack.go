package weblogin

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
)

var (
	cookieLoginTmp    = "oauth"
	cookieKeyNonce    = "nonce"
	cookieKeyRedirect = "redirect"
	cookieKeyScope    = "scope"

	ErrBadCookie = errors.New("Bad cookie data")
)

const shortTermMaxAge = 10 * time.Minute
const longTermMaxAge = 6 * 30 * 24 * time.Hour

func (mod *WebLoginModule) getSession(w http.ResponseWriter, r *http.Request, name string) (*sessions.Session, error) {
	sess, err := mod.store.Get(r, name)
	if err != nil {
		if err, ok := err.(securecookie.Error); ok {
			if err.IsDecode() {
				sess.Options.MaxAge = -1
				sess.Save(r, w)
				util.LogBadf("Cookie decode error: %s", err)
				http.Error(w, "invalid cookies, please login again", http.StatusBadRequest)
				return nil, ErrBadCookie
			}
		}

		util.LogBadf("Cookie error: %s", err)
		http.Error(w, fmt.Sprintf("cookie error: %s", err), http.StatusInternalServerError)
		return nil, ErrBadCookie
	}

	if name == cookieLoginTmp {
		sess.Options.MaxAge = int(shortTermMaxAge / time.Second)
	} else if name == cookieLongTerm {
		sess.Options.MaxAge = int(longTermMaxAge / time.Second)
	}

	return sess, nil
}

func (mod *WebLoginModule) StartSlackURL(returnURL string, extraScopes ...string) string {
	form := url.Values{}
	if returnURL != "" {
		form.Set("redirect_url", returnURL)
	}
	if len(extraScopes) > 0 {
		form.Set("scope", strings.Join(extraScopes, " "))
	}

	relURL := "/oauth/slack/start"
	if len(extraScopes) == 0 {
		relURL = "/oauth/altslack/start"
	}
	uri, err := url.Parse(mod.team.AbsoluteURL(relURL))
	if err != nil {
		panic(err)
	}
	uri.RawQuery = form.Encode()
	return uri.String()
}

func (mod *WebLoginModule) OAuthSlackStart(w http.ResponseWriter, r *http.Request) {
	loginSession, err := mod.getSession(w, r, cookieLoginTmp)
	if err != nil {
		return
	}

	err = r.ParseForm()
	if err != nil {
		http.Error(w, "bad form encoding", http.StatusBadRequest)
		return
	}

	var bytes [16]byte
	rand.Read(bytes[:])
	nonce := base64.URLEncoding.EncodeToString(bytes[:])
	loginSession.Values[cookieKeyNonce] = nonce

	if after := r.Form.Get("redirect_url"); after != "" {
		loginSession.Values[cookieKeyRedirect] = after
	}

	scopes := strings.Split(r.Form.Get("scope"), " ")
	scopes = append(scopes, "identify")

	targetURL := mod.slackOAuthConfig.AuthCodeURL(nonce,
		oauth2.SetAuthURLParam("team", string(mod.team.TeamID())),
		oauth2.SetAuthURLParam("scope", strings.Join(scopes, " ")),
	)
	err = loginSession.Save(r, w)
	if err != nil {
		util.LogError(errors.Wrap(err, "could not create oauth cookie"))
		http.Error(w, fmt.Sprintf("Internal error: %s", err), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, targetURL, http.StatusSeeOther)
}

func (mod *WebLoginModule) OAuthSlackCallback(w http.ResponseWriter, r *http.Request) {
	loginSession, err := mod.getSession(w, r, cookieLoginTmp)
	if err != nil {
		return
	}

	doneRedirectURL, _ := loginSession.Values[cookieKeyRedirect].(string)

	// Burn nonce (this deletes record from database)
	loginSession.Options.MaxAge = -1
	err = loginSession.Save(r, w)
	if err != nil {
		util.LogError(errors.Wrap(err, "oauth: clearing session"))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		w.Write([]byte(`<br><a href="/oauth/slack/start">Start Over</a>`))
		return
	}

	if loginSession.IsNew {
		http.Error(w, "nonce expired", http.StatusBadRequest)
		w.Write([]byte(`<br><a href="/oauth/slack/start">Start Over</a>`))
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

	nonceExpect, anyStored := loginSession.Values[cookieKeyNonce]
	nonceGot := r.Form.Get("state")
	if nonceExpect != nonceGot || !anyStored {
		http.Error(w, "nonce mismatch", http.StatusBadRequest)
		return
	}

	// Nonce passed, exchange code for an auth token
	code := r.Form.Get("code")
	token, err := mod.slackOAuthConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, fmt.Sprintf("bad token/could not contact Slack: %s", err), http.StatusBadRequest)
		return
	}

	// Contact Slack via auth.test to get the full token scope, user ID, team ID

	form := url.Values{}
	form.Set("token", token.AccessToken)

	var response struct {
		slack.APIResponse
		UserID   slack.UserID `json:"user_id"`
		UserName string       `json:"user"`
		TeamID   slack.TeamID `json:"team_id"`
	}

	// PostRaw is used to get the X-OAuth-Scopes header
	resp, err := mod.team.SlackAPIPostRaw("auth.test", form)
	if err != nil {
		util.LogError(err)
		http.Error(w, fmt.Sprintf("bad token/could not contact Slack: %s", err), http.StatusBadRequest)
		return
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	resp.Body.Close()
	if err != nil {
		util.LogBadf("Slack API auth.test error: %s", err)
		http.Error(w, fmt.Sprintf("bad token/could not contact Slack: %s", err), http.StatusBadRequest)
		return
	}
	if !response.APIResponse.OK {
		err = response.APIResponse
		util.LogBadf("Slack API auth.test error: %s", err)
		http.Error(w, fmt.Sprintf("bad token/could not contact Slack: %s", err), http.StatusBadRequest)
		return
	}

	// Verify that they actually logged into the correct team
	// User IDs are NOT unique across Slack teams
	if response.TeamID != mod.team.TeamID() {
		util.LogBad("(http) Bad team id, got", response.TeamID, "wanted", mod.team.TeamID())
		http.Error(w, fmt.Sprintf("Wrong Slack team! This is only available for %s.slack.com",
			mod.team.TeamConfig().TeamDomain), http.StatusBadRequest)
		// asynchronously revoke the token
		go mod.team.SlackAPIPostJSON("auth.revoke", form, nil)
		return
	}

	// The whole reason we're using PostRaw: the X-OAuth-Scopes header
	authorizedScopes := strings.Split(resp.Header.Get("X-OAuth-Scopes"), ",")
	for i, v := range authorizedScopes {
		authorizedScopes[i] = strings.TrimSpace(v)
	}

	// Check for existing User
	var user *User
	user, err = mod.GetUserBySlack(response.UserID)
	if err != nil && err != ErrNoSuchUser {
		util.LogError(err)
		http.Error(w, fmt.Sprintf("database error: %s", err), http.StatusBadRequest)
		return
	}
	if user == nil {
		user, err = mod.GetOrNewCurrentUser(w, r)
		if err != nil {
			util.LogError(err)
			http.Error(w, fmt.Sprintf("error getting current user: %s", err), http.StatusBadRequest)
			return
		}
	}

	err = user.UpdateSlack(response.UserID, response.UserName, token.AccessToken, authorizedScopes)
	if err != nil {
		err = errors.Wrap(err, "error saving login data")
		util.LogError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = user.Login(w, r)
	if err != nil {
		err = errors.Wrap(err, "error saving login cookie")
		util.LogError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if doneRedirectURL != "" {
		http.Redirect(w, r, doneRedirectURL, http.StatusFound)
		return
	}
	http.Redirect(w, r, mod.team.AbsoluteURL("/"), http.StatusFound)
	w.Write([]byte(`You are now logged in, redirecting to the homepage...`))
}
