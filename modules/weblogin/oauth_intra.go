package weblogin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/oauth2"

	"github.com/riking/marvin/intra"
	"github.com/riking/marvin/util"
)

func (mod *WebLoginModule) StartIntraURL(returnURL string, extraScopes ...string) string {
	form := url.Values{}
	if returnURL != "" {
		form.Set("redirect_url", returnURL)
	}
	if len(extraScopes) > 0 {
		form.Set("scope", strings.Join(extraScopes, " "))
	}
	uri, err := url.Parse(mod.team.AbsoluteURL("/oauth/intra/start"))
	if err != nil {
		panic(err)
	}
	uri.RawQuery = form.Encode()
	return uri.String()
}

func (mod *WebLoginModule) OAuthIntraStart(w http.ResponseWriter, r *http.Request) {
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

	var scopes []string
	if len(r.Form.Get("scope")) > 0 {
		scopes = strings.Split(r.Form.Get("scope"), " ")
	}
	scopes = append(scopes, "public")
	scope := strings.Join(scopes, " ")

	targetURL := mod.IntraOAuthConfig.AuthCodeURL(nonce,
		oauth2.SetAuthURLParam("scope", scope),
	)
	err = loginSession.Save(r, w)
	if err != nil {
		util.LogError(errors.Wrap(err, "could not create oauth cookie"))
		http.Error(w, fmt.Sprintf("Internal error: %s", err), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, targetURL, http.StatusSeeOther)
}

func (mod *WebLoginModule) OAuthIntraCallback(w http.ResponseWriter, r *http.Request) {
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

	intraErr := r.Form.Get("error")
	if intraErr != "" {
		mod.HTTPError(w, r, errors.Errorf("Intra OAuth failed: %s", intraErr))
		return
	}

	nonceExpect, anyStored := loginSession.Values[cookieKeyNonce]
	nonceGot := r.Form.Get("state")
	if nonceExpect != nonceGot || !anyStored {
		http.Error(w, "nonce mismatch", http.StatusBadRequest)
		return
	}

	token, err := mod.IntraOAuthConfig.Exchange(context.Background(), r.Form.Get("code"))
	if err != nil {
		util.LogError(err)
		http.Error(w, fmt.Sprintf("could not contact Intra: %s", err), http.StatusInternalServerError)
		return
	}

	cl := intra.Client(mod.IntraOAuthConfig, token)
	var response struct {
		Login  string `json:"login"`
		Campus []struct {
			ID int `json:"id"`
		} `json:"campus"`
	}
	const fremontCampusID = 7
	httpResp, err := cl.GetJSON("/v2/me", &response)
	fmt.Println(httpResp.Status, httpResp.Header)
	if err != nil {
		err = errors.Wrap(err, "contacting intra")
		util.LogError(err)
		http.Error(w, fmt.Sprintf("could not contact Intra: %s", err), http.StatusInternalServerError)
		return
	}

	fmt.Println(response)
	ok := false
	for _, v := range response.Campus {
		if v.ID == fremontCampusID {
			ok = true
		}
	}
	if !ok {
		mod.HTTPError(w, r, errors.Errorf("Sorry, Marvin is only available to the Fremont campus."))
		return
	}

	var user *User
	user, err = mod.GetUserByIntra(response.Login)
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

	err = user.UpdateIntra(response.Login, token, []string{"public"}) // TODO scopes
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
