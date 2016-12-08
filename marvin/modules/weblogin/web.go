package weblogin

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
)

const (
	NavSectionHome     = "Home"
	NavSectionFactoids = "Factoids"
	NavSectionInvite   = "Invite"
)

var NavbarContent = []struct {
	Name string
	URL  string
}{
	{Name: NavSectionFactoids, URL: "/factoids"},
	{Name: NavSectionInvite, URL: "/invite"},
}

type LayoutContent struct {
	Team        marvin.Team
	Title       string
	CurrentUser *slack.User

	NavbarCurrent     string
	NavbarItemsCustom interface{}

	BodyData interface{}
}

// NewLayoutContent will always succeed, but may leave some fields unfilled when err != nil.
func NewLayoutContent(team marvin.Team, w http.ResponseWriter, r *http.Request, navSection string) (*LayoutContent, error) {
	var err error
	var userID slack.UserID
	var user *slack.User = nil

	userID, err = team.GetModule(Identifier).(*WebLoginModule).GetUser(w, r)
	if userID != "" {
		var err error
		user, err = team.UserInfo(userID)
		err = errors.Wrap(err, "getting user info for layout")
	}
	return &LayoutContent{
		Team:          team,
		NavbarCurrent: navSection,
		CurrentUser:   user,
	}, err
}

func (w *LayoutContent) NavbarItems() interface{} {
	if w.NavbarItemsCustom != nil {
		return w.NavbarItemsCustom
	}
	return NavbarContent
}

var tmplLayout = template.Must(template.New("layout").Parse(string(MustAsset("layout.html.tmpl")))).Funcs(tmplFuncs)

var tmplFuncs = template.FuncMap{
	"user": func(team marvin.Team, userID slack.UserID) (*slack.User, error) {
		return team.UserInfo(slack.UserID(userID))
	},
	"channel_name": func(team marvin.Team, channelID slack.ChannelID) string {
		return team.ChannelName(slack.ChannelID(channelID))
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

var tmplHome = template.Must(LayoutTemplateCopy().Parse(`
{{define "content"}}
<div class="container">
<div class="page-header"><h1>Home</h1></div>
<p>Homepage content</p>
<p>Goes here</p>
</div>
{{end}}`))

func (mod *WebLoginModule) ServeRoot(w http.ResponseWriter, r *http.Request) {
	lc, err := NewLayoutContent(mod.team, w, r, "Home")
	if err != nil {
		mod.HTTPError(w, r, err)
		return
	}

	lc.BodyData = nil
	util.LogIfError(tmplHome.ExecuteTemplate(w, "layout", lc))
}
