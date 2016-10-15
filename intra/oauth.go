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

	"github.com/pkg/errors"
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
	h.PayloadB64 = r.Form.Get("payload")
	enc := base64.URLEncoding
	payloadForm, err := enc.DecodeString(strings.TrimSpace(r.Form.Get("payload")))
	if err != nil {
		return nil, errors.Wrap(err, "invalid b64 encoding")
	}
	fmt.Println(payloadForm)
	payload, err := url.ParseQuery(string(payloadForm))
	if err != nil {
		return nil, errors.Wrap(err, "invalid b64 payload")
	}
	h.Payload = payload
	sigBytes, err := hex.DecodeString(r.Form.Get("sig"))
	if err != nil {
		return nil, errors.Wrap(err, "invalid hex encoding")
	}
	if !h.IsValid(sigBytes) {
		return nil, errors.Wrap(err, "invalid signature")
	}
	h.Nonce = payload.Get("nonce")
	if h.Nonce == "" {
		return nil, errors.Wrap(err, "nonce missing")
	}
	return h, nil
}

func (h SSOHelper) IsValid(sig []byte) bool {
	mac := hmac.New(sha256.New, []byte(getSSOSecret()))
	mac.Write([]byte(h.PayloadB64))
	expectedSig := mac.Sum(nil)
	return hmac.Equal(expectedSig, sig)
}

//func (h SSOHelper) Build(params url.Values) (url.Values, error) {
//}

var ssoSecret string
var ssoSecretLock sync.Mutex

func getSSOSecret() string {
	ssoSecretLock.Lock()
	defer ssoSecretLock.Unlock()
	if ssoSecret != "" {
		return ssoSecret
	}
	content, err := ioutil.ReadFile(`/tank/www/discourse_sso_secret`)
	if err != nil {
		fmt.Println("Unable to read SSO secret:", err)
		return "XXX"
	}
	ssoSecret = strings.TrimSpace(string(content))
	return ssoSecret
}

type stringWriter interface {
	WriteString(string) (int, error)
}

func HTTPDiscourseSSO(w http.ResponseWriter, r *http.Request) {
	sso, err := SSORequest(r)
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	if sso != nil {
		w.Write([]byte("nonce: "))
		w.Write([]byte(sso.Nonce))
	} else {
		w.Write([]byte("??? sso object was nil"))
	}
}
