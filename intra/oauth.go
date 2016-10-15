package intra

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type SSOHelper struct {
	PayloadB64 string
	Payload    url.Values
	Nonce      string
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
	fmt.Println(string(payloadForm), "->", h.Payload)
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
	session, err := cookieStore.Get(r, cookieKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	sso, err := SSORequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	session.Values["nonce"] = sso.Nonce
	session.Save(w, r)
	w.Write([]byte("nonce: "))
	w.Write([]byte(sso.Nonce))
}

func HTTPStartOauth(w http.ResponseWriter, r *http.Request) {

}

type intraCredentials struct {
	AccessToken string  `json:"access_token"`
	TokenType   string  `json:"token_type"`
	ExpiresIn   float64 `json:"expires_in"`
	Scope       string  `json:"scope"`
	CreatedAt   float64 `json:"created_at"`
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

func HTTPOauthCallback(w http.ResponseWriter, r *http.Request) {
	session, err := cookieStore.Get(r, cookieKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err := r.ParseForm()
	if err != nil {
		err = errors.Wrap(err, "bad form parameters")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req, err := http.NewRequest("POST", "https://api.intra.42.fr/oauth/token", nil)
	if err != nil {
		panic(err)
	}
	req.PostForm = make(url.Values)
	req.PostForm.Set("grant_type", "client_credentials")
	req.PostForm.Set("client_id")
	req.PostForm.Set("client_secret", intraSecret.Get())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		errStr := fmt.Sprintf("Could not contact Intra\n%s", errors.Wrap(err, "post /oauth/token"))
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Println(body, err)
}
