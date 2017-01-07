package weblogin

import (
	"html/template"
	"net/http"

	"github.com/pkg/errors"

	"github.com/riking/homeapi/marvin/util"
)

var tmplLogout = template.Must(LayoutTemplateCopy().Parse(string(MustAsset("templates/logged-out.html"))))

func (mod *WebLoginModule) DestroySession(w http.ResponseWriter, r *http.Request) {
	user, err := mod.GetCurrentUser(w, r)
	if err != nil {
		mod.HTTPError(w, r, errors.Wrap(err, "Error determining login state"))
		return
	}
	if user == nil {
		w.WriteHeader(200)
		return
	}

	lc, _ := NewLayoutContent(mod.team, w, r, NavSectionInvite)

	err = user.Destroy()
	if err != nil {
		mod.HTTPError(w, r, errors.Wrap(err, "Could not complete logout"))
		return
	}

	util.LogIfError(tmplLogout.Execute(w, lc))
}
