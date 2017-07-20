package factoid

import (
	"html/template"
	"net/http"
	"regexp"

	"github.com/pkg/errors"

	"github.com/riking/marvin"
	"github.com/riking/marvin/modules/weblogin"
	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/util"
)

func (mod *FactoidModule) registerHTTP() {
	mod.team.HandleHTTP("/factoids", http.HandlerFunc(mod.HTTPListFactoids))
	mod.team.HandleHTTP("/factoids/_/{name}", http.HandlerFunc(mod.HTTPShowFactoid))
	mod.team.HandleHTTP("/factoids/{channel}/{name}", http.HandlerFunc(mod.HTTPShowFactoid))
}

var tmplListFactoids = template.Must(weblogin.LayoutTemplateCopy().Parse(string(weblogin.MustAsset("templates/factoid-list.html"))))

type bodyList struct {
	List []*Factoid
	team marvin.Team
}

type bodyShow struct {
	Layout  *weblogin.LayoutContent
	Factoid *Factoid
	team    marvin.Team
	History []Factoid
}

func (d bodyList) Team() marvin.Team { return d.team }
func (d bodyShow) Team() marvin.Team { return d.team }

func (mod *FactoidModule) HTTPListFactoids(w http.ResponseWriter, r *http.Request) {
	lc, err := weblogin.NewLayoutContent(mod.team, w, r, weblogin.NavSectionFactoids)
	if err != nil {
		mod.team.GetModule(weblogin.Identifier).(weblogin.API).HTTPError(w, r, err)
		return
	}

	err = r.ParseForm()
	if err != nil {
		http.Error(w, "Bad form data: "+err.Error(), http.StatusBadRequest)
	}

	scopeChannel := r.Form.Get("channel")

	list, err := mod.ListFactoidsWithInfo(r.Form.Get("q"), slack.ChannelID(scopeChannel))
	if err != nil {
		mod.team.GetModule(weblogin.Identifier).(weblogin.API).HTTPError(w, r, err)
		return
	}

	lc.BodyData = bodyList{
		List: list,
		team: mod.team,
	}
	util.LogIfError(
		tmplListFactoids.ExecuteTemplate(w, "layout", lc))
}

var tmplShowFactoid = template.Must(weblogin.LayoutTemplateCopy().Parse(string(weblogin.MustAsset("templates/factoid-info.html"))))

var rgxShowFactoid = regexp.MustCompile(`/factoids/(C[A-Z0-9]+|_)/([^/]*)`)

func (mod *FactoidModule) HTTPShowFactoid(w http.ResponseWriter, r *http.Request) {
	lc, err := weblogin.NewLayoutContent(mod.team, w, r, weblogin.NavSectionFactoids)
	if err != nil {
		mod.team.GetModule(weblogin.Identifier).(weblogin.API).HTTPError(w, r, err)
		return
	}

	m := rgxShowFactoid.FindStringSubmatch(r.URL.Path)
	if m == nil {
		err = errors.Errorf("Bad URL format")
		mod.team.GetModule(weblogin.Identifier).(weblogin.API).HTTPError(w, r, err)
		return
	}

	scopeChannel := m[1]
	if scopeChannel == "_" {
		scopeChannel = ""
	}
	name := m[2]

	finfo, err := mod.GetFactoidInfo(name, slack.ChannelID(scopeChannel), false)
	if err == ErrNoSuchFactoid {
		mod.team.GetModule(weblogin.Identifier).(weblogin.API).HTTPError(w, r, nil)
		return
	} else if err != nil {
		mod.team.GetModule(weblogin.Identifier).(weblogin.API).HTTPError(w, r, err)
		return
	}

	history, err := mod.GetFactoidHistory(name, slack.ChannelID(scopeChannel))
	if err != nil {
		mod.team.GetModule(weblogin.Identifier).(weblogin.API).HTTPError(w, r, err)
		return
	}

	lc.BodyData = bodyShow{
		Factoid: finfo,
		team:    mod.team,
		History: history,
		Layout:  lc,
	}

	util.LogIfError(
		tmplShowFactoid.ExecuteTemplate(w, "layout", lc))
}
