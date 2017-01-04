package weblogin

import (
	"crypto/rand"
	"encoding/base64"
	"html/template"
	"net/http"
	"time"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/util"
)

type authNonceValue struct {
	UserID  int64
	Expires time.Time
}

func (mod *WebLoginModule) newAuthToken(u *User) string {
	var randBytes [16]byte
	n, _ := rand.Reader.Read(randBytes[:])
	if n != 16 {
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

	var data struct {
		Layout          *LayoutContent
		LoginIntraFirst bool
		AlreadyComplete bool
		RandomToken     string
	}
	data.Layout = lc
	lc.BodyData = data

	if u == nil || u.IntraLogin == "" {
		data.LoginIntraFirst = true
	} else if u.SlackUser != "" {
		data.AlreadyComplete = true
	} else {
		data.RandomToken = mod.newAuthToken(u)
	}

	util.LogIfError(tmplLoginAltSlack.Execute(w, lc))
}

func (mod *WebLoginModule) CommandWebAuthenticate(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if len(args.Arguments) != 1 {
		return marvin.CmdFailuref(args, "Please only use this command as instructed on the website.")
	}
	token := args.Pop()
	userID := mod.findAuthToken(token)
	if userID == -1 {
		return marvin.CmdFailuref(args, "Please only use this command as instructed on the website.")
	}

	user, err := mod.GetUserByID(userID)
	if err != nil {
		return marvin.CmdError(args, err, "Error loading user info; you were not logged in.")
	}

	slackUser, _ := mod.team.UserInfo(args.Source.UserID())
	err = user.UpdateSlack(args.Source.UserID(), slackUser.Name, "", []string{})
	if err != nil {
		return marvin.CmdError(args, err, "Error saving user info; you were not logged in.")
	}
	return marvin.CmdSuccess(args, "You have been logged in. Return to https://marvin.riking.org .")
}
