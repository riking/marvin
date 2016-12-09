package factoid

import (
	"html/template"
	"net/http"
	"regexp"

	"github.com/pkg/errors"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/modules/weblogin"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
)

func (mod *FactoidModule) registerHTTP() {
	mod.team.HandleHTTP("/factoids", http.HandlerFunc(mod.HTTPListFactoids))
	mod.team.HandleHTTP("/factoids/_/{name}", http.HandlerFunc(mod.HTTPShowFactoid))
	mod.team.HandleHTTP("/factoids/{channel}/{name}", http.HandlerFunc(mod.HTTPShowFactoid))
}

var tmplListFactoids = template.Must(weblogin.LayoutTemplateCopy().Parse(string(weblogin.MustAsset("templates/factoid-list.html"))))

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

	lc.BodyData = struct {
		List []*Factoid
		Team marvin.Team
	}{
		List: list,
		Team: mod.team,
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

	factoid, err := mod.GetFactoidInfo(name, slack.ChannelID(scopeChannel), false)
	if err == ErrNoSuchFactoid {
		mod.team.GetModule(weblogin.Identifier).(weblogin.API).HTTPError(w, r, nil)
		return
	} else if err != nil {
		mod.team.GetModule(weblogin.Identifier).(weblogin.API).HTTPError(w, r, err)
		return
	}

	lc.BodyData = struct {
		Factoid *Factoid
		Team    marvin.Team
	}{
		Factoid: factoid,
		Team:    mod.team,
	}

	util.LogIfError(
		tmplShowFactoid.ExecuteTemplate(w, "layout", lc))
}
