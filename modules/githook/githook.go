package githook

import (
	"crypto/hmac"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/util"
)

const Identifier = "githook"

func init() {
	marvin.RegisterModule(NewGithookModule)
}

type GithookModule struct {
	team marvin.Team
}

func NewGithookModule(t marvin.Team) marvin.Module {
	mod := &GithookModule{
		team: t,
	}
	return mod
}

func (mod *GithookModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *GithookModule) Load(t marvin.Team) {
	t.DB().MustMigrate(Identifier, 1516095152, sqlMigrate1, sqlMigrate2)
	t.DB().SyntaxCheck(
		sqlGetRepoSecret,
		sqlGetDestinations,
		sqlGetSubscriptions,
		sqlInsertRepo,
		sqlInsertSubscription,
		sqlDeleteSubscription,
		sqlDeleteRepo,
		sqlDeleteUnverifiedRepo,
		sqlStampLastUsed,
	)
}

func (mod *GithookModule) Enable(team marvin.Team) {
	mod.team.HTTPMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/github/hook/") {
				mod.HandleHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
}

func (mod *GithookModule) Disable(t marvin.Team) {
}

const (
	sqlMigrate1 = `
	CREATE TABLE module_githook_repos (
		id         SERIAL PRIMARY KEY,
		repo_name  text,
		secret     text,
		created_by varchar(10),
		created_at timestamptz default(CURRENT_TIMESTAMP),
		last_used  timestamptz default(NULL),

		UNIQUE(repo_name)
	)`
	sqlMigrate2 = `
	CREATE TABLE module_githook_configs (
		id         SERIAL PRIMARY KEY,
		repo_id    int 		REFERENCES module_githook_repos(id) ON DELETE CASCADE,
		channel    varchar(10),
		created_by varchar(10),
		created_at timestamptz default(CURRENT_TIMESTAMP),

		UNIQUE(repo_id, channel)
	)`

	// $1 = name
	sqlGetRepoSecret = `SELECT id, secret FROM module_githook_repos WHERE repo_name = $1`

	// $1 = id
	sqlGetDestinations = `SELECT channel FROM module_githook_configs WHERE repo_id = $1`

	// $1 = channel
	sqlGetSubscriptions = `
	SELECT r.repo_name, c.created_by
	FROM module_githook_configs c
	LEFT JOIN module_githook_repos r
		ON r.id = c.repo_id
	WHERE c.channel = $1`

	// $1 = name $2 = secret $3 = userid
	sqlInsertRepo = `
	INSERT INTO module_githook_repos
	(repo_name, secret, created_by)
	VALUES ($1, $2, $3)`

	// $1 = repo id $2 = channel $3 = userid
	sqlInsertSubscription = `
	INSERT INTO module_githook_configs
	(repo_id, channel, created_by)
	VALUES ($1, $2, $3)`

	// $1 = channel id $2 = name
	sqlDeleteSubscription = `
	DELETE FROM module_githook_configs
	WHERE channel = $1
	AND repo_id = (SELECT id FROM module_githook_repos WHERE repo_name = $2)`

	// $1 = name
	sqlDeleteRepo = `
	DELETE FROM module_githook_repos
	WHERE repo_name = $1`

	sqlDeleteUnverifiedRepo = `
	DELETE FROM module_githook_repos
	WHERE last_used IS NULL
	AND created_at < (CURRENT_TIMESTAMP - INTERVAL '1 month')`

	// $1 = id
	sqlStampLastUsed = `
	UPDATE module_githook_repos
	SET last_used = CURRENT_TIMESTAMP
	WHERE id = $1`
)

// Handle Github webhooks.
func (mod *GithookModule) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	eventType := r.Header.Get("X-Github-Event")
	if eventType == "" {
		w.WriteHeader(400)
		fmt.Fprintln(w, "GitHub event deliveries only!")
		return
	}

	repoNameURL := strings.TrimPrefix(r.URL.Path, "/github/hook/")
	util.LogDebug("githook: got delivery for", repoNameURL)
	repoLocalID, secret, err := mod.recognizeRepo(repoNameURL)
	if err == sql.ErrNoRows {
		w.WriteHeader(404)
		util.LogDebug("githook: Repository not configured.")
		fmt.Fprintln(w, "no config for", repoNameURL, "\nplease register and retry delivery")
		return
	} else if err != nil {
		w.WriteHeader(500)
		util.LogBad("githook: recognizeRepo() error:", err)
		fmt.Fprintln(w, "internal server error", err)
		return
	}

	hookPayload, err := mod.decodeBody(r, secret)
	if err != nil {
		w.WriteHeader(400)
		util.LogBad("githook: bad request:", err)
		fmt.Fprintln(w, "bad request:", err)
		return
	}
	// verification passed, this is from Github
	go mod.stampLastUsed(repoLocalID)

	destinations, err := mod.getDestinations(repoLocalID)
	if err != nil {
		w.WriteHeader(500)
		util.LogBad("githook: getDestinations() error:", err)
		fmt.Fprintln(w, "internal server error", err)
		return
	}
	if len(destinations) == 0 {
		w.WriteHeader(200)
		util.LogDebug("githook: decode successful, but nowhere to send")
		fmt.Fprintln(w, "Hook valid, but no destinations found")
		return
	}

	var msg slack.OutgoingSlackMessage

	switch eventType {
	case "push":
		msg = mod.RenderPush(hookPayload)
	}

	if msg.Text != "" || len(msg.Attachments) > 0 {
		for _, v := range destinations {
			mod.team.SendComplexMessage(v, msg)
		}
	}

	w.WriteHeader(200)
	util.LogDebug("githook: successful delivery")
}

func (mod *GithookModule) recognizeRepo(name string) (repoLocalID int, secret string, err error) {
	// id, secret, created_at, last_used
	stmt, err := mod.team.DB().Prepare(sqlGetRepoSecret)
	if err != nil {
		return -1, "", err
	}
	defer stmt.Close()
	row := stmt.QueryRow(name)

	err = row.Scan(&repoLocalID, &secret)
	if err != nil {
		return -1, "", err
	}
	return repoLocalID, secret, nil
}

func (mod *GithookModule) decodeBody(r *http.Request, secret string) (v interface{}, err error) {
	mac := hmac.New(sha1.New, []byte(secret))
	hashedBody := io.TeeReader(r.Body, mac)

	var hookPayload interface{}
	err = json.NewDecoder(hashedBody).Decode(&hookPayload)
	if err != nil {
		return nil, err
	}
	// Consume any trailing data (newlines...)
	io.Copy(ioutil.Discard, hashedBody)

	hubSig := r.Header.Get("X-Hub-Signature")
	expectedHMAC, err := hex.DecodeString(strings.TrimPrefix(hubSig, "sha1="))
	if err != nil {
		return nil, errors.Errorf("Missing signature")
	}
	if !hmac.Equal(expectedHMAC, mac.Sum(nil)) {
		// return nil, errors.Errorf("Bad signature %s %s", hex.EncodeToString(expectedHMAC), hex.EncodeToString(mac.Sum(nil)))
	}
	return hookPayload, nil
}

func (mod *GithookModule) getDestinations(repoID int) ([]slack.ChannelID, error) {
	stmt, err := mod.team.DB().Prepare(sqlGetDestinations)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	var result []slack.ChannelID
	rows, err := stmt.Query(repoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var str string
		err = rows.Scan(&str)
		if err != nil {
			return nil, err
		}
		result = append(result, slack.ChannelID(str))
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return result, nil
}

func (mod *GithookModule) stampLastUsed(repoID int) error {
	stmt, err := mod.team.DB().Prepare(sqlStampLastUsed)
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(repoID)
	if err != nil {
		return err
	}
	return nil
}
