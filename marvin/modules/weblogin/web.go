package weblogin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/csrf"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
)

const (
	NavSectionHome     = "Home"
	NavSectionFactoids = "Factoids"
	NavSectionInvite   = "Channels"
	NavSectionLogs     = "Logs"
	NavSectionUser     = "User"
)

var NavbarContent = []struct {
	Name string
	URL  string
}{
	{Name: NavSectionFactoids, URL: "/factoids"},
	{Name: NavSectionInvite, URL: "/invites"},
	{Name: NavSectionLogs, URL: "/logs"},
}

type LayoutContent struct {
	Team        marvin.Team
	WLMod       *WebLoginModule
	Title       string
	CurrentURL  string
	CurrentUser *User

	slackUser *slack.User

	NavbarCurrent     string
	NavbarItemsCustom interface{}

	BodyData interface{}
}

// NewLayoutContent will always succeed, but may leave some fields unfilled when err != nil.
func NewLayoutContent(team marvin.Team, w http.ResponseWriter, r *http.Request, navSection string) (*LayoutContent, error) {
	var err error
	var user *User

	wlMod := team.GetModule(Identifier).(*WebLoginModule)
	user, err = wlMod.GetCurrentUser(w, r)
	return &LayoutContent{
		Team:          team,
		WLMod:         wlMod,
		NavbarCurrent: navSection,
		CurrentURL:    r.URL.RequestURI(),
		CurrentUser:   user,
	}, err
}

func (w *LayoutContent) NavbarItems() interface{} {
	if w.NavbarItemsCustom != nil {
		return w.NavbarItemsCustom
	}
	return NavbarContent
}

func (w *LayoutContent) SlackUser() (*slack.User, error) {
	if w.slackUser != nil {
		return w.slackUser, nil
	}
	if w.CurrentUser == nil {
		return nil, nil
	}
	if w.CurrentUser.SlackUser == "" {
		return nil, nil
	}
	var err error
	w.slackUser, err = w.Team.UserInfo(w.CurrentUser.SlackUser)
	return w.slackUser, err
}

func (w *LayoutContent) StartSlackURL(extraScopes ...string) string {
	return w.WLMod.StartSlackURL(w.CurrentURL, extraScopes...)
}

func (w *LayoutContent) StartIntraURL(extraScopes ...string) string {
	return w.WLMod.StartIntraURL(w.CurrentURL, extraScopes...)
}

func (w *LayoutContent) DCurrentUser() User {
	if w.CurrentUser != nil {
		return *w.CurrentUser
	}
	return User{}
}

var tmplReltime = template.Must(template.New("reltime").Parse(`<span class="reltime" title="{{.RFC3339}}">{{.Relative}}</span>`))
var tmplLayout = template.Must(template.New("layout").Parse(string(MustAsset("layout.html")))).Funcs(tmplFuncs)

var tmplFuncs = template.FuncMap{
	"user": func(team marvin.Team, userID slack.UserID) (*slack.User, error) {
		return team.UserInfo(slack.UserID(userID))
	},
	"channel_name": func(team marvin.Team, channelID slack.ChannelID) string {
		return team.ChannelName(slack.ChannelID(channelID))
	},
	"reltime": func(t time.Time) (template.HTML, error) {
		var buf bytes.Buffer
		data := struct {
			RFC3339  string
			Relative string
		}{
			RFC3339:  t.Format(time.RFC3339),
			Relative: humanize.Time(t),
		}
		err := tmplReltime.Execute(&buf, data)
		if err != nil {
			return "", err
		}
		return template.HTML(buf.String()), nil
	},
}

func LayoutTemplateCopy() *template.Template {
	return template.Must(tmplLayout.Clone())
}

// ---

func (mod *WebLoginModule) ServeAsset(w http.ResponseWriter, r *http.Request) {
	b, err := Asset(strings.TrimPrefix(r.URL.Path, "/"))
	if err != nil {
		fmt.Println("asset missing", r.URL.Path)
		mod.Serve404(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, ".css") {
		w.Header().Set("Content-Type", "text/css")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Write(b)
}

var tmplError = template.Must(LayoutTemplateCopy().Parse(`
{{define "content"}}
<div style="display:flex;justify-content:center;align-items:center;">
<div style="width: 300px">
<h1>Whoops!</h1>
{{- if . }}
  <p>It seems like something went a bit wrong. Here's the info:</p>
  <p>{{.}}</p>
  <pre style="overflow-y:scroll;max-height:500px"><code>{{printf "%+v" .}}</code></pre>
{{ else }}
  <p>Can't seem to find the page you're looking for. Maybe you took a wrong turn?</p>
{{ end -}}
</div>
</div>
{{end}}`))

func (mod *WebLoginModule) HTTPError(w http.ResponseWriter, r *http.Request, err error) {
	lc, _ := NewLayoutContent(mod.team, w, r, "404")
	lc.BodyData = err
	lc.Title = "Oops! - Marvin"

	util.LogIfError(tmplError.ExecuteTemplate(w, "layout", lc))
}

func (mod *WebLoginModule) Serve404(w http.ResponseWriter, r *http.Request) {
	_, err := NewLayoutContent(mod.team, w, r, "404")
	if err != nil {
		mod.HTTPError(w, r, err)
		return
	}
	mod.HTTPError(w, r, nil)
}

var tmplHome = template.Must(LayoutTemplateCopy().Parse(string(MustAsset("templates/home.html"))))

func (mod *WebLoginModule) ServeRoot(w http.ResponseWriter, r *http.Request) {
	lc, err := NewLayoutContent(mod.team, w, r, "Home")
	if err != nil {
		mod.HTTPError(w, r, err)
		return
	}

	lc.BodyData = nil
	util.LogIfError(tmplHome.ExecuteTemplate(w, "layout", lc))
}

func (mod *WebLoginModule) ServeCSRF(w http.ResponseWriter, r *http.Request) {
	var jsonData struct {
		Token string `json:"token"`
	}
	jsonData.Token = csrf.Token(r)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, must-revalidate, no-cache, private")
	_ = json.NewEncoder(w).Encode(jsonData)
}
