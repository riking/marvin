package factoid

import (
	"html/template"
	"net/http"

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
	}{
		List: list,
	}
	util.LogIfError(
		tmplListFactoids.ExecuteTemplate(w, "layout", lc))
}

var tmplShowFactoid = template.Must(weblogin.LayoutTemplateCopy().Parse(`
{{define "content"}}
Coming soon
{{end}}`))

func (mod *FactoidModule) HTTPShowFactoid(w http.ResponseWriter, r *http.Request) {
	lc, err := weblogin.NewLayoutContent(mod.team, w, r, weblogin.NavSectionFactoids)
	if err != nil {
		mod.team.GetModule(weblogin.Identifier).(weblogin.API).HTTPError(w, r, err)
		return
	}

	util.LogIfError(
		tmplShowFactoid.ExecuteTemplate(w, "layout", lc))
}
