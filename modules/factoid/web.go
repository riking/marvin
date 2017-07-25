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
	"fmt"
)

func (mod *FactoidModule) registerHTTP() {
	r := mod.team.Router()
	r.Path("/factoids").HandlerFunc(mod.HTTPListFactoids)
	r.Path("/factoids/_/{name}").HandlerFunc(http.HandlerFunc(mod.HTTPShowFactoid))
	r.Path("/factoids/{channel}/{name}").HandlerFunc(http.HandlerFunc(mod.HTTPShowFactoid))
	r.Methods("POST").Path("/factoids/_/{name}/edit").HandlerFunc(mod.HTTPEditFactoid)
	r.Methods("POST").Path("/factoids/{channel}/{name}/edit").HandlerFunc(mod.HTTPEditFactoid)
}

var tmplListFactoids = template.Must(weblogin.LayoutTemplateCopy().Parse(string(weblogin.MustAsset("templates/factoid-list.html"))))

type bodyList struct {
	List []*Factoid
	team marvin.Team
}

type bodyShow struct {
	Layout          *weblogin.LayoutContent
	Factoid         *Factoid
	team            marvin.Team
	ScopeChannelURL string
	History         []Factoid
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
		Factoid:         finfo,
		team:            mod.team,
		History:         history,
		ScopeChannelURL: m[1],
		Layout:          lc,
	}

	util.LogIfError(
		tmplShowFactoid.ExecuteTemplate(w, "layout", lc))
}

func (mod *FactoidModule) HTTPEditFactoid(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	lc, err := weblogin.NewLayoutContent(mod.team, w, r, weblogin.NavSectionFactoids)
	if err != nil {
		http.Error(w, `{"ok": false, "message": "bad login/cookies"}`, 401)
		return
	}

	m := rgxShowFactoid.FindStringSubmatch(r.URL.Path)
	if m == nil {
		http.Error(w, `{"ok": false, "message": "bad URL"}`, 404)
		return
	}

	scopeChannel := slack.ChannelID(m[1])
	if scopeChannel == "_" {
		scopeChannel = ""
	}
	factoidName := m[2]

	if lc.CurrentUser == nil {
		http.Error(w, `{"ok": false, "message": "must log in"}`, 403)
		return
	}

	actionSource := weblogin.ActionSourceWeb{Team: mod.Team(), User: lc.CurrentUser}

	r.ParseMultipartForm(-1)
	fmt.Println(r.Form)
	factoidSource := r.Form.Get("raw")
	if factoidSource == "" {
		http.Error(w, `{"ok": false, "message": "new raw not provided"}`, 422)
		return
	}
	fmt.Println("web factoid set by", lc.CurrentUser.IntraLogin, "set", factoidName, "to", factoidSource)

	prevFactoidInfo, err := mod.GetFactoidInfo(factoidName, scopeChannel, false)
	if err == ErrNoSuchFactoid {
		// make a pseudo value that passes all the checks
		prevFactoidInfo = &Factoid{IsLocked: false, ScopeChannel: ""}
	} else if err != nil {
		http.Error(w, fmt.Sprintf(`{"ok": false, "message": "internal server error: %v"}`, err), 500)
		return
	}

	if prevFactoidInfo.IsLocked {
		if scopeChannel != "" {
			// Overriding a locked global with a local is OK
			if prevFactoidInfo.ScopeChannel != "" {
				http.Error(w, `{"ok": false, "message": "Factoid is locked"}`, 403)
			}
		} else {
			http.Error(w, `{"ok": false, "message": "Factoid is locked"}`, 403)
		}
	}

	// Attempt parse
	fi := Factoid{
		Mod:       mod,
		RawSource: factoidSource,
	}
	err = util.PCall(func() error {
		fi.Tokens()
		return nil
	})
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"ok": false, "message": "Bad syntax: %v"}`, err), 422)
		return
	}

	util.LogGood("Saving factoid", factoidName, "-", factoidSource)
	err = mod.SaveFactoid(factoidName, scopeChannel, factoidSource, actionSource)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"ok": false, "message": "Could not save factoid: %v"}`, err), 500)
	}

	fmt.Fprint(w, `{"ok": true}`)
}
