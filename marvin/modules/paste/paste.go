package paste

import (
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strconv"
	"sync"

	"github.com/pkg/errors"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/util"
)

type API interface {
	marvin.Module

	CreatePaste(content string) (int64, error)
	GetPaste(id int64) (string, error)
	URLForPaste(id int64) string

	CreateLink(content string) (int64, error)
	GetLink(id int64) (string, error)
	URLForLink(id int64) string
}

var _ API = &PasteModule{}

// ---

func init() {
	marvin.RegisterModule(NewPasteModule)
}

const Identifier = "paste"

type PasteModule struct {
	team marvin.Team

	pasteLock    sync.Mutex
	pasteContent map[int64]string
	linkContent  map[int64]string
}

func NewPasteModule(t marvin.Team) marvin.Module {
	mod := &PasteModule{
		team:         t,
		pasteContent: make(map[int64]string),
		linkContent:  make(map[int64]string),
	}
	return mod
}

func (mod *PasteModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *PasteModule) Load(t marvin.Team) {
	t.DB().MustMigrate(Identifier, 1479357009, sqlMigrate1)
	t.DB().MustMigrate(Identifier, 1483845740, sqlMigrate2)
	t.DB().SyntaxCheck(sqlAddPaste, sqlGetPaste, sqlAddLink, sqlGetLink)
}

func (mod *PasteModule) Enable(team marvin.Team) {
	team.HandleHTTP("/p/", mod)
	team.HandleHTTP("/l/", mod)
}

func (mod *PasteModule) Disable(team marvin.Team) {
}

const idBase = 36

var allowedRequest = regexp.MustCompile(`/(p|l)/([0-9a-z]+)`)

func (mod *PasteModule) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := allowedRequest.FindStringSubmatch(r.URL.Path)
	if m == nil || r.Method != "GET" {
		http.Error(w, "acceptable: GET /p/:id\nid: int", http.StatusNotFound)
		http.NotFound(w, r)
		return
	}
	isPaste := m[1] == "p"
	isLink := m[1] == "l"
	id, err := strconv.ParseInt(m[2], idBase, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("id must be an int, got %s", m[1]), http.StatusBadRequest)
		return
	}

	if isPaste {
		content, err := mod.GetPaste(id)
		if err != nil {
			util.LogError(err)
			http.Error(w, fmt.Sprintf("error: %s", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		fmt.Fprint(w, content)
	} else if isLink {
		redirect, err := mod.GetLink(id)
		if err != nil {
			util.LogError(err)
			http.Error(w, fmt.Sprintf("error: %s", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Location", redirect)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusFound)
		fmt.Fprint(w, "<script>document.location = '")
		template.JSEscape(w, []byte(redirect))
		fmt.Fprint(w, "';")
	} else {
		util.LogError(errors.Errorf("unknown url type"))
		w.WriteHeader(404)
	}
}

func (mod *PasteModule) URLForPaste(id int64) string {
	idStr := strconv.FormatInt(id, idBase)
	return mod.team.AbsoluteURL(fmt.Sprintf("/p/%s", idStr))
}

func (mod *PasteModule) URLForLink(id int64) string {
	idStr := strconv.FormatInt(id, idBase)
	return mod.team.AbsoluteURL(fmt.Sprintf("/l/%s", idStr))
}
