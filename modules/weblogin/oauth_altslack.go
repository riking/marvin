package weblogin

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/riking/marvin"
	"github.com/riking/marvin/util"
)

type authNonceValue struct {
	UserID  int64
	Expires time.Time
}

func (mod *WebLoginModule) newAuthToken(u *User) string {
	var randBytes [32]byte
	n, _ := rand.Reader.Read(randBytes[:])
	if n != 32 {
		return "ERROR"
	}
	str := base64.RawURLEncoding.EncodeToString(randBytes[:])

	mod.authTokenLock.Lock()
	defer mod.authTokenLock.Unlock()
	mod.authTokenMap[str] = authNonceValue{
		UserID:  u.ID,
		Expires: time.Now().Add(20 * time.Minute),
	}
	return str
}

func (mod *WebLoginModule) findAuthToken(token string) (userID int64) {
	mod.authTokenLock.Lock()
	item, ok := mod.authTokenMap[token]
	mod.authTokenLock.Unlock()
	if !ok {
		return -1
	}
	if item.Expires.Before(time.Now()) {
		return -1
	}
	return item.UserID
}

func (mod *WebLoginModule) janitorAuthToken() {
	mod.authTokenLock.Lock()
	defer mod.authTokenLock.Unlock()

	now := time.Now()
	for key := range mod.authTokenMap {
		if mod.authTokenMap[key].Expires.Before(now) {
			delete(mod.authTokenMap, key)
		}
	}
}

var tmplLoginAltSlack = template.Must(LayoutTemplateCopy().Parse(string(
	MustAsset("templates/login-altslack.html"))))

func (mod *WebLoginModule) OAuthAltSlackStart(w http.ResponseWriter, r *http.Request) {
	lc, err := NewLayoutContent(mod.team, w, r, "Account")
	if err != nil {
		mod.HTTPError(w, r, err)
		return
	}
	u, err := mod.GetCurrentUser(w, r)
	if err != nil {
		return
	}
	r.ParseForm()

	var data struct {
		Layout          *LayoutContent
		LoginIntraFirst bool
		AlreadyComplete bool
		RandomToken     string
		RedirectURL     string
		RedirectB64     string
	}
	data.Layout = lc

	if u == nil || u.IntraLogin == "" {
		data.LoginIntraFirst = true
	} else if u.SlackUser != "" {
		data.AlreadyComplete = true
	} else {
		data.RandomToken = mod.newAuthToken(u)
	}
	data.RedirectURL = r.Form.Get("redirect_url")
	data.RedirectB64 = base64.RawURLEncoding.EncodeToString([]byte(data.RedirectURL))
	if data.RedirectB64 == "" {
		data.RedirectB64 = "==="
	}

	if data.AlreadyComplete && data.RedirectURL != "" {
		if !strings.HasPrefix(data.RedirectURL, "/") {
			http.Error(w, "SecurityError: off-site redirect", http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, data.RedirectURL, http.StatusFound)
		return
	}
	lc.BodyData = data
	util.LogIfError(tmplLoginAltSlack.Execute(w, lc))
}

func (mod *WebLoginModule) CommandWebAuthenticate(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if len(args.Arguments) != 2 {
		return marvin.CmdFailuref(args, "Please only use this command as instructed on the website.")
	}
	token := args.Pop()
	redirectB64 := args.Pop()
	redirectURLBytes, err := base64.RawURLEncoding.DecodeString(redirectB64)
	redirectURL := string(redirectURLBytes)
	if redirectB64 == "===" {
		redirectURL = "/"
	} else if err != nil {
		util.LogWarn(err)
		return marvin.CmdFailuref(args, "Please only use this command as instructed on the website.")
	}

	userID := mod.findAuthToken(token)
	if userID == -1 {
		util.LogWarn("bad auth token")
		return marvin.CmdFailuref(args, "Please only use this command as instructed on the website.")
	}

	user, err := mod.GetUserByID(userID)
	if err != nil {
		util.LogWarn(err)
		return marvin.CmdError(args, err, "Error loading user info; you were not logged in.")
	}

	slackUser, _ := mod.team.UserInfo(args.Source.UserID())
	err = user.UpdateSlack(args.Source.UserID(), slackUser.Name, "", []string{})
	if err != nil {
		return marvin.CmdError(args, err, "Error saving user info; you were not logged in.")
	}
	return marvin.CmdSuccess(args, fmt.Sprintf("You have been logged in. Return to %s .",
		mod.team.AbsoluteURL(redirectURL)))
}
