package intra

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type SSOHelper struct {
	PayloadB64 string
	Payload    url.Values
	Nonce      string
	ReturnURL  string
}

func SSORequest(r *http.Request) (*SSOHelper, error) {
	h := &SSOHelper{}
	err := r.ParseForm()
	if err != nil {
		return nil, errors.Wrap(err, "invalid POST form")
	}
	h.PayloadB64 = r.Form.Get("sso")
	enc := base64.URLEncoding
	payloadForm, err := enc.DecodeString(strings.TrimSpace(h.PayloadB64))
	if err != nil {
		return nil, errors.Wrap(err, "invalid b64 encoding")
	}
	payload, err := url.ParseQuery(string(payloadForm))
	if err != nil {
		return nil, errors.Wrap(err, "invalid b64 payload")
	}
	h.Payload = payload
	sigBytes, err := hex.DecodeString(r.Form.Get("sig"))
	if err != nil {
		return nil, errors.Wrap(err, "invalid hex encoding")
	}
	if !h.IsValid([]byte(r.Form.Get("sso")), sigBytes) {
		return nil, errors.Errorf("invalid signature")
	}
	h.Nonce = payload.Get("nonce")
	if h.Nonce == "" {
		return nil, errors.Errorf("nonce missing from request")
	}
	h.ReturnURL = payload.Get("return_sso_url")
	return h, nil
}

func (h *SSOHelper) IsValid(payload, sig []byte) bool {
	mac := hmac.New(sha256.New, []byte(ssoSecret.Get()))
	mac.Write(payload)
	expectedSig := mac.Sum(nil)
	return hmac.Equal(expectedSig, sig)
}

type secretFile struct {
	Filename string
	content  string
	lock     sync.Mutex
}

func (f *secretFile) Get() string {
	f.lock.Lock()
	defer f.lock.Unlock()
	if f.content != "" {
		return f.content
	}
	content, err := ioutil.ReadFile(f.Filename)
	if err != nil {
		fmt.Println("Unable to read SSO secret:", err)
		return "XXX"
	}
	f.content = strings.TrimSpace(string(content))
	return f.content
}

var (
	ssoSecret = secretFile{
		Filename: `/tank/www/keys/discourse_sso_secret`,
	}
	intraSecret = secretFile{
		Filename: `/tank/www/keys/intra_oauth`,
	}
	cookieSecret = secretFile{
		Filename: `/tank/www/keys/oauth_cookies`,
	}
)

var cookieStore = sessions.NewCookieStore([]byte(cookieSecret.Get()))

const cookieKey = `intra-oauth`

func HTTPDiscourseSSO(w http.ResponseWriter, r *http.Request) {
	sso, err := SSORequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	redirURL := oauthConfig.AuthCodeURL(sso.Nonce, oauth2.SetAuthURLParam("response_type", "code"))
	http.Redirect(w, r, redirURL, http.StatusFound)
}

var oauthConfig = oauth2.Config{
	ClientID:     "00d2a4918d470c47c08448c37fdd170793c2a94320f7971981d461be028f2a35",
	ClientSecret: intraSecret.Get(),
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://api.intra.42.fr/oauth/authorize",
		TokenURL: "https://api.intra.42.fr/oauth/token",
	},
	RedirectURL: "https://home.riking.org/oauth/callback",
	Scopes:      []string{"public"},
}

type IntraUser struct {
	ID          int    `json:"id"`
	Email       string `json:"email"`
	Login       string `json:"login"`
	DisplayName string `json:"displayname"`
	ImageURL    string `json:"image_url"`
	IsStaff     bool   `json:"staff?"`
	// Which computer you're on right now
	//Location    string `json:"location"`
	CursusUsers []struct {
		Cursus struct {
			ID int `json:"id"`
		} `json:"cursus"`
	} `json:"cursus_users"`
	Campus []struct {
		ID int `json:"id"`
	}
}

const discourseBase = "http://42.riking.org"

func HTTPOauthCallback(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := r.ParseForm()
	if err != nil {
		err = errors.Wrap(err, "bad form parameters")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	token, err := oauthConfig.Exchange(ctx, r.Form.Get("code"))
	if err != nil {
		http.Error(w, errors.Wrap(err, "exchanging token").Error(), http.StatusServiceUnavailable)
		return
	}
	client := oauthConfig.Client(ctx, token)
	req, err := http.NewRequest("GET", "https://api.intra.42.fr/v2/me", nil)
	if err != nil {
		panic(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, errors.Wrap(err, "contacting Intra").Error(), http.StatusServiceUnavailable)
		return
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		http.Error(w, errors.Wrap(err, "could not read from Intra").Error(), http.StatusInternalServerError)
		return
	}

	var user IntraUser
	err = json.NewDecoder(bytes.NewReader(respBody)).Decode(&user)
	if err != nil {
		http.Error(w, errors.Wrap(err, "could not read JSON").Error(), http.StatusInternalServerError)
		return
	}
	sso := make(url.Values)
	sso.Set("nonce", r.Form.Get("state"))
	sso.Set("name", user.DisplayName)
	sso.Set("username", user.Login)
	sso.Set("email", user.Email)
	sso.Set("external_id", strconv.Itoa(user.ID))
	sso.Set("avatar_url", fmt.Sprintf("https://cdn.intra.42.fr/users/medium_%s.jpg", user.Login))
	if user.IsStaff {
		sso.Set("moderator", "true")
	} else {
		sso.Set("moderator", "false")
	}

	payload := sso.Encode()
	b64Payload := base64.URLEncoding.EncodeToString([]byte(payload))

	mac := hmac.New(sha256.New, []byte(ssoSecret.Get()))
	mac.Write([]byte(b64Payload))
	sig := mac.Sum(nil)
	hexSig := hex.EncodeToString(sig)

	ssoValues := make(url.Values)
	ssoValues.Set("sso", b64Payload)
	ssoValues.Set("sig", hexSig)
	url, err := url.Parse(fmt.Sprintf("%s/session/sso_login", discourseBase))
	if err != nil {
		panic(err)
	}
	url.RawQuery = ssoValues.Encode()
	http.Redirect(w, r, url.String(), http.StatusFound)
}
