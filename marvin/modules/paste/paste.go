package paste

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"sync"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/util"
)

type API interface {
	marvin.Module

	CreatePaste(content string) (int64, error)
	GetPaste(id int64) (string, error)
	GetURL(id int64) string
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
}

func NewPasteModule(t marvin.Team) marvin.Module {
	mod := &PasteModule{
		team:         t,
		pasteContent: make(map[int64]string),
	}
	return mod
}

func (mod *PasteModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *PasteModule) Load(t marvin.Team) {
	t.DB().MustMigrate(Identifier, 1479357009, sqlMigrate1)
	t.DB().SyntaxCheck(sqlAddPaste, sqlGetPaste)
}

func (mod *PasteModule) Enable(team marvin.Team) {
	team.HandleHTTP("/p/", mod)
}

func (mod *PasteModule) Disable(team marvin.Team) {
}

// ---

const (
	sqlMigrate1 = `CREATE TABLE module_paste_data (id SERIAL PRIMARY KEY, content TEXT)`

	// $1 = content
	// id sql.NullInt64
	sqlAddPaste = `INSERT INTO module_paste_data (content) VALUES ($1)
			RETURNING id`

	// $1 = id
	sqlGetPaste = `SELECT content FROM module_paste_data WHERE id = $1`
)

const idBase = 36

// ---

func (mod *PasteModule) GetPaste(id int64) (string, error) {
	var content string
	found := false
	mod.pasteLock.Lock()
	content, found = mod.pasteContent[id]
	mod.pasteLock.Unlock()
	if found {
		return content, nil
	}

	stmt, err := mod.team.DB().Prepare(sqlGetPaste)
	if err != nil {
		return "", err
	}
	defer stmt.Close()
	row := stmt.QueryRow(id)
	err = row.Scan(&content)
	if err != nil {
		return "", err
	}
	return content, nil
}

func (mod *PasteModule) CreatePaste(content string) (int64, error) {
	stmt, err := mod.team.DB().Prepare(sqlAddPaste)
	if err != nil {
		return -1, err
	}
	defer stmt.Close()
	row := stmt.QueryRow(content)
	var id int64
	err = row.Scan(&id)
	if err != nil {
		return -1, err
	}

	mod.pasteLock.Lock()
	mod.pasteContent[id] = content
	mod.pasteLock.Unlock()
	return id, nil
}

// ---

var allowedRequest = regexp.MustCompile(`/p/([0-9a-z]+)`)

func (mod *PasteModule) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m := allowedRequest.FindStringSubmatch(r.URL.Path)
	if m == nil || r.Method != "GET" {
		http.Error(w, "acceptable: GET /p/:id\nid: int", http.StatusNotFound)
		http.NotFound(w, r)
		return
	}
	id, err := strconv.ParseInt(m[1], idBase, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("id must be an int, got %s", m[1]), http.StatusBadRequest)
		return
	}
	content, err := mod.GetPaste(id)
	if err != nil {
		util.LogError(err)
		http.Error(w, fmt.Sprintf("error: %s", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	fmt.Fprint(w, content)
}

func (mod *PasteModule) GetURL(id int64) string {
	idStr := strconv.FormatInt(id, idBase)
	return mod.team.AbsoluteURL(fmt.Sprintf("/p/%s", idStr))
}
