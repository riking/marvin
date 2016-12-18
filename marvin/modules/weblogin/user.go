package weblogin

import (
	"database/sql"
	"net/http"

	"encoding/json"

	"github.com/pkg/errors"
	"github.com/riking/homeapi/marvin/slack"
	"golang.org/x/oauth2"
)

const (
	sqlMigrateUser1 = `
	CREATE TABLE web_users (
		id                  serial primary key,
		slack_uid           varchar(12) null, -- slack.UserID
		slack_name          varchar(64) null,
		slack_token         text        null,
		slack_scopes        jsonb       default '[]',
		intra_user          varchar(64) null,
		intra_token         json        null,
		intra_scopes        jsonb       default '[]'
	)`

	sqlMigrateUser2 = `CREATE UNIQUE INDEX web_users_slack ON web_users (slack_uid)`
	sqlMigrateUser3 = `CREATE UNIQUE INDEX web_users_intra ON web_users (intra_user)`

	sqlLoadUser = `
	SELECT id,
		slack_uid, slack_name, slack_token, slack_scopes,
		intra_user, intra_token, intra_scopes
	FROM web_users
	WHERE id = $1`

	sqlNewUser = `
	INSERT INTO web_users (id, slack_uid, intra_user)
	VALUES (DEFAULT, NULL, NULL)
	RETURNING id`

	sqlLookupUserBySlack = `SELECT id FROM web_users WHERE slack_uid = $1`
	sqlLookupUserByIntra = `SELECT id FROM web_users WHERE intra_user = $1`

	sqlUpdateSlack = `
	UPDATE web_users
	SET slack_uid = $2, slack_name = $3, slack_token = $4, slack_scopes = $5
	WHERE id = $1`

	sqlUpdateIntra = `
	UPDATE web_users
	SET intra_user = $2, intra_token = $3, intra_scopes = $4
	WHERE id = $1`

	sqlDestroyUser = `DELETE FROM web_users WHERE id = $1`
)

const (
	cookieLongTerm = "user"
	cookieKeyUID   = "id"
)

var ErrNoSuchUser = errors.New("That user does not exist.")
var ErrNotLoggedIn = errors.New("You are not logged in.")

// A User contains possibly a logged-in Slack user, and possibly a logged-in Intra user.
type User struct {
	mod *WebLoginModule
	ID  int64

	SlackUser   slack.UserID
	SlackName   string
	SlackToken  string
	SlackScopes []string

	IntraLogin  string
	IntraToken  *oauth2.Token
	IntraScopes []string
}

func (u *User) HasScopeSlack(scope string) bool {
	for _, v := range u.SlackScopes {
		if v == scope {
			return true
		}
	}
	return false
}

func (mod *WebLoginModule) GetUserByID(uid int64) (*User, error) {
	stmt, err := mod.team.DB().Prepare(sqlLoadUser)
	if err != nil {
		return nil, errors.Wrap(err, "get user: prepare")
	}
	defer stmt.Close()

	row := stmt.QueryRow(uid)
	u := &User{
		mod: mod,
		ID:  uid,
	}
	var idMatch int64
	var slackUser, slackName, slackToken sql.NullString
	var intraLogin sql.NullString
	var slackScopes, intraToken, intraScopes []byte

	err = row.Scan(&idMatch,
		&slackUser, &slackName, &slackToken, &slackScopes,
		&intraLogin, &intraToken, &intraScopes,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNoSuchUser
	} else if err != nil {
		return nil, errors.Wrap(err, "get user: query")
	}

	// Done, have valid user
	if slackToken.Valid {
		u.SlackUser = slack.UserID(slackUser.String)
		u.SlackName = slackName.String
		u.SlackToken = slackToken.String
		err = json.Unmarshal(slackScopes, &u.SlackScopes)
		if err != nil {
			return nil, errors.Wrap(err, "get user: unmarshal slack scopes")
		}
	}
	if intraLogin.Valid {
		u.IntraLogin = intraLogin.String
		err = json.Unmarshal(intraToken, &u.IntraToken)
		if err != nil {
			return nil, errors.Wrap(err, "get user: unmarshal intra scopes")
		}
		err = json.Unmarshal(intraScopes, &u.IntraScopes)
		if err != nil {
			return nil, errors.Wrap(err, "get user: unmarshal intra scopes")
		}
	}
	return u, nil

}

func (mod *WebLoginModule) _getCurrentUser(create bool, w http.ResponseWriter, r *http.Request) (*User, error) {
	sess, err := mod.getSession(w, r, cookieLongTerm)
	if err != nil {
		return nil, err
	}

	uidI, haveUID := sess.Values[cookieKeyUID]
	if !haveUID {
		// not logged in
		if !create {
			return nil, nil
		} else {
			return &User{
				mod: mod,
				ID:  -1,
			}, nil
		}
	}
	var uid = uidI.(int64)

	u, err := mod.GetUserByID(uid)
	if err == ErrNoSuchUser {
		if !create {
			return nil, nil
		} else {
			return &User{
				mod: mod,
				ID:  -1,
			}, nil
		}
	}
	return u, err
}

// GetCurrentUser looks up the current user based on the request.
// If there is no currently logged in user, it returns a nil *User instead of an error.
func (mod *WebLoginModule) GetCurrentUser(w http.ResponseWriter, r *http.Request) (*User, error) {
	return mod._getCurrentUser(false, w, r)
}

// GetOrNewCurrentUser looks up the current user based on the request.
// If there is no currently logged in user, it returns a new User object with a pseudo-id of -1.
func (mod *WebLoginModule) GetOrNewCurrentUser(w http.ResponseWriter, r *http.Request) (*User, error) {
	return mod._getCurrentUser(true, w, r)
}

// GetUserBySlack checks for an existing user with the given Slack user ID.
func (mod *WebLoginModule) GetUserBySlack(slackID slack.UserID) (*User, error) {
	stmt, err := mod.team.DB().Prepare(sqlLookupUserBySlack)
	if err != nil {
		return nil, errors.Wrap(err, "lookup user: prepare")
	}
	defer stmt.Close()

	row := stmt.QueryRow(string(slackID))
	var id sql.NullInt64
	err = row.Scan(&id)
	if err == sql.ErrNoRows {
		return nil, ErrNoSuchUser
	} else if err != nil {
		return nil, errors.Wrap(err, "lookup user: query")
	}
	if !id.Valid {
		return nil, ErrNoSuchUser
	}
	return mod.GetUserByID(id.Int64)
}

// GetUserByIntra checks for an existing user with the given Intra login name.
func (mod *WebLoginModule) GetUserByIntra(login string) (*User, error) {
	stmt, err := mod.team.DB().Prepare(sqlLookupUserByIntra)
	if err != nil {
		return nil, errors.Wrap(err, "lookup user: prepare")
	}
	defer stmt.Close()

	row := stmt.QueryRow(string(login))
	var id sql.NullInt64
	err = row.Scan(&id)
	if err == sql.ErrNoRows {
		return nil, ErrNoSuchUser
	} else if err != nil {
		return nil, errors.Wrap(err, "lookup user: query")
	}
	if !id.Valid {
		return nil, ErrNoSuchUser
	}
	return mod.GetUserByID(id.Int64)
}

// Login writes an auth cookie. This cannot be used with a User object not yet saved to the database (ID == -1).
func (u *User) Login(w http.ResponseWriter, r *http.Request) error {
	if u.ID == -1 {
		return ErrNotLoggedIn
	}

	sess, err := u.mod.getSession(w, r, cookieLongTerm)
	if err != nil {
		return err
	}

	sess.Values[cookieKeyUID] = int64(u.ID)
	sess.Save(r, w)
	return nil
}

// Destroy removes the User's row in the database.
func (u *User) Destroy() error {
	stmt, err := u.mod.team.DB().Prepare(sqlDestroyUser)
	if err != nil {
		return errors.Wrap(err, "users.destroy prepare")
	}
	defer stmt.Close()

	_, err = stmt.Exec(u.ID)
	if err != nil {
		return errors.Wrap(err, "users.destroy exec")
	}
	return nil
}

// UpdateSlack saves the given slack token to the database, and creates a new row in the database if necessary (id == -1).
func (u *User) UpdateSlack(uid slack.UserID, name, token string, scopes []string) error {
	u.SlackUser = uid
	u.SlackName = name
	u.SlackToken = token
	u.SlackScopes = scopes

	scopeBytes, err := json.Marshal(u.SlackScopes)
	if err != nil {
		return errors.Wrap(err, "update slack data: marshal")
	}

	tx, err := u.mod.team.DB().Begin()
	if err != nil {
		return errors.Wrap(err, "update slack data: begin")
	}

	stmt, err := tx.Prepare(sqlUpdateSlack)
	if err != nil {
		return errors.Wrap(err, "update slack data: prepare")
	}
	defer stmt.Close()

	if u.ID == -1 {
		row := tx.QueryRow(sqlNewUser)
		err = row.Scan(&u.ID)
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "update slack data: create user row")
		}
	}

	_, err = stmt.Exec(u.ID, string(u.SlackUser), u.SlackName, u.SlackToken, scopeBytes)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "update slack data: update")
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "update slack data: commit")
	}
	return nil
}

// UpdateIntra saves the given intra token to the database, and creates a new row in the database if necessary (id == -1).
func (u *User) UpdateIntra(login string, token *oauth2.Token, scopes []string) error {
	u.IntraLogin = login
	u.IntraToken = token
	u.IntraScopes = scopes

	scopeBytes, err := json.Marshal(u.IntraScopes)
	if err != nil {
		return errors.Wrap(err, "update slack data: marshal")
	}
	tokenBytes, err := json.Marshal(u.IntraToken)
	if err != nil {
		return errors.Wrap(err, "update slack data: marshal")
	}

	tx, err := u.mod.team.DB().Begin()
	if err != nil {
		return errors.Wrap(err, "update intra data: begin")
	}

	stmt, err := tx.Prepare(sqlUpdateIntra)
	if err != nil {
		return errors.Wrap(err, "update intra data: prepare")
	}
	defer stmt.Close()

	if u.ID == -1 {
		row := tx.QueryRow(sqlNewUser)
		err = row.Scan(&u.ID)
		if err != nil {
			tx.Rollback()
			return errors.Wrap(err, "update intra data: create user row")
		}
	}

	_, err = stmt.Exec(u.ID, u.IntraLogin, tokenBytes, scopeBytes)
	if err != nil {
		tx.Rollback()
		return errors.Wrap(err, "update intra data: update")
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "update intra data: commit")
	}
	return nil
}
