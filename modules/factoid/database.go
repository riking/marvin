package factoid

import (
	"database/sql"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
)

const (
	FactoidNameMaxLen = 75

	sqlMigrate1 = `CREATE TABLE module_factoid_factoids (
		id                SERIAL PRIMARY KEY,
		name              TEXT NOT NULL,
		channel_only      varchar(10) DEFAULT NULL, -- null OR slack.ChannelID
		rawtext           TEXT NOT NULL,

		last_set_user     varchar(10) NOT NULL, -- slack.UserID
		last_set_channel  varchar(10) NOT NULL, -- slack.ChannelID
		last_set_ts       varchar(20) NOT NULL, -- slack.MessageTS
		last_set          timestamptz NOT NULL, -- time.Time

		locked            boolean NOT NULL DEFAULT FALSE,
		forgotten         boolean NOT NULL DEFAULT FALSE
	)`

	sqlMigrate2 = `
	CREATE INDEX factoid_get ON module_factoid_factoids
	(name, channel_only, last_set, forgotten)
	WHERE forgotten = FALSE`

	// $1 = name $2 = scopeChannel
	sqlGetFactoid = `
	SELECT rawtext, last_set_user
	FROM module_factoid_factoids
	WHERE name = $1 AND ((channel_only IS NULL AND $2 IS NULL) OR channel_only = $2)
	AND forgotten = FALSE
	ORDER BY channel_only DESC, last_set DESC
	LIMIT 1`

	// $1 = name $2 = scopeChannel $3 = includeForgotten
	sqlFactoidInfo = `
	SELECT id, rawtext, channel_only, last_set_user, last_set_channel, last_set_ts, last_set, locked, forgotten
	FROM module_factoid_factoids
	WHERE name = $1 AND ((channel_only IS NULL AND $2 IS NULL) OR channel_only = $2)
	AND ($3 OR forgotten = FALSE)
	ORDER BY channel_only DESC, last_set DESC
	LIMIT 1`

	// $1 = name $2 = scopeChannel
	sqlFactoidHistory = `
	SELECT id, name, rawtext, channel_only, last_set_user, last_set_channel, last_set_ts, last_set, locked, forgotten
	FROM module_factoid_factoids
	WHERE name = $1 AND ((channel_only IS NULL AND $2 IS NULL) OR channel_only = $2)
	ORDER BY channel_only DESC, last_set DESC
	-- no LIMIT`

	// $1 = name $2 = scopeChannel $3 = source $4 = userid $5 = msg_chan $6 = msg_ts
	sqlMakeFactoid = `
	INSERT INTO module_factoid_factoids
	(name, channel_only, rawtext, last_set_user, last_set_channel, last_set_ts, last_set)
	VALUES
	($1,   $2,           $3,      $4,            $5,               $6, CURRENT_TIMESTAMP)`

	// $1 = nameMatch $2 = scopeChannel
	sqlListMatches = `
	SELECT DISTINCT name, channel_only IS NOT NULL
	FROM module_factoid_factoids
	WHERE name LIKE '%' || $1 || '%'
	AND ((channel_only IS NULL AND $2 IS NULL) OR channel_only = $2)
	AND (forgotten = FALSE)
	GROUP BY name, channel_only`

	// $1 = nameMatch $2 = scopeChannel $3 = includeForgotten
	sqlListMatchesWithInfo = `
	WITH names AS (
		SELECT MAX(id) id, name, channel_only
		FROM module_factoid_factoids
		WHERE name LIKE '%' || $1 || '%'
		AND ((channel_only IS NULL AND $2 IS NULL) OR channel_only = $2 OR '_ANY' = $2)
		AND ($3 OR forgotten = FALSE)
		GROUP BY name, channel_only
	)
	SELECT f.id, f.name, f.rawtext, f.channel_only, f.last_set_user, f.last_set_channel, f.last_set_ts, f.last_set, f.locked, f.forgotten
	FROM names
	INNER JOIN module_factoid_factoids f ON names.id = f.id
	ORDER BY channel_only DESC, last_set DESC
	`

	// $1 = isLocked $2 = dbID
	sqlLockFactoid = `
	UPDATE module_factoid_factoids
	SET locked = $1
	WHERE id = $2`

	// $1 = isForgotten $2 = dbID
	sqlForgetFactoid = `
	UPDATE module_factoid_factoids
	SET forgotten = $1
	WHERE id = $2`
)

var ErrNoSuchFactoid = errors.Errorf("Factoid does not exist")

type Factoid struct {
	Mod *FactoidModule

	IsBareInfo   bool
	DbID         int64
	FactoidName  string
	RawSource    string
	ScopeChannel slack.ChannelID

	LastUser      slack.UserID
	LastChannel   slack.ChannelID
	LastMessage   slack.MessageTS
	LastTimestamp time.Time

	IsLocked    bool
	IsForgotten bool

	tokenize sync.Once
	tokens   []Token
}

func (mod *FactoidModule) doMigrate(t marvin.Team) {
	t.DB().MustMigrate(Identifier, 1478236994, sqlMigrate1, sqlMigrate2)
	t.DB().MustMigrate(Identifier, 1484348222, sqlMigrate3)
}

func (mod *FactoidModule) doSyntaxCheck(t marvin.Team) {
	t.DB().SyntaxCheck(
		sqlGetFactoid,
		sqlFactoidInfo,
		sqlMakeFactoid,
		sqlListMatches,
		sqlListMatchesWithInfo,
		sqlLockFactoid,
		sqlForgetFactoid,

		sqlFDataGetOne,
		sqlFDataGetAll,
		sqlFDataMakeTempTable,
		//sqlFDataSetFromTemp,
		sqlFDataSet,
	)
}

func (mod *FactoidModule) GetFactoidInfo(name string, channel slack.ChannelID, withForgotten bool) (*Factoid, error) {
	var result = new(Factoid)
	result.Mod = mod
	result.FactoidName = name
	result.IsBareInfo = false

	stmt, err := mod.team.DB().Prepare(sqlFactoidInfo)
	if err != nil {
		return nil, errors.Wrap(err, "Database error")
	}
	defer stmt.Close()

	if channel == "_" {
		channel = ""
	}
	var scopeChannel = sql.NullString{Valid: channel != "", String: string(channel)}

	row := stmt.QueryRow(name, scopeChannel, withForgotten)
	err = row.Scan(
		&result.DbID, &result.RawSource, &scopeChannel,
		(*string)(&result.LastUser), (*string)(&result.LastChannel), (*string)(&result.LastMessage),
		&result.LastTimestamp,
		&result.IsLocked, &result.IsForgotten,
	)
	if err == sql.ErrNoRows {
		// retry without channel scope
		if scopeChannel.Valid {
			return mod.GetFactoidInfo(name, "", withForgotten)
		}
		return nil, ErrNoSuchFactoid
	} else if err != nil {
		return nil, errors.Wrap(err, "Database error")
	}

	if scopeChannel.Valid {
		result.ScopeChannel = slack.ChannelID(scopeChannel.String)
	} else {
		result.ScopeChannel = ""
	}

	return result, nil
}

func (mod *FactoidModule) GetFactoidBare(name string, channel slack.ChannelID) (*Factoid, error) {
	var result = new(Factoid)
	result.Mod = mod
	result.FactoidName = name
	result.IsBareInfo = true

	stmt, err := mod.team.DB().Prepare(sqlGetFactoid)
	if err != nil {
		return nil, errors.Wrap(err, "Database error")
	}
	defer stmt.Close()

	if channel == "_" {
		channel = ""
	}
	var scopeChannel = sql.NullString{Valid: channel != "", String: string(channel)}

	row := stmt.QueryRow(name, scopeChannel)
	err = row.Scan(&result.RawSource, (*string)(&result.LastUser))
	if err == sql.ErrNoRows {
		// retry without channel scope
		if scopeChannel.Valid {
			return mod.GetFactoidBare(name, "")
		}
		return nil, ErrNoSuchFactoid
	} else if err != nil {
		return nil, errors.Wrap(err, "Database error")
	}
	return result, nil
}

func (mod *FactoidModule) GetFactoidHistory(name string, channel slack.ChannelID) ([]Factoid, error) {
	stmt, err := mod.team.DB().Prepare(sqlFactoidHistory)
	if err != nil {
		return nil, errors.Wrap(err, "Database error")
	}
	defer stmt.Close()

	if channel == "_" {
		channel = ""
	}
	var scopeChannel = sql.NullString{Valid: channel != "", String: string(channel)}

	rows, err := stmt.Query(name, scopeChannel)
	if err != nil {
		return nil, errors.Wrap(err, "Database error")
	}

	var resAry []Factoid

	for rows.Next() {
		result := Factoid{
			Mod: mod,
		}
		err = rows.Scan(
			&result.DbID, &result.FactoidName, &result.RawSource, &scopeChannel,
			(*string)(&result.LastUser), (*string)(&result.LastChannel), (*string)(&result.LastMessage),
			&result.LastTimestamp,
			&result.IsLocked, &result.IsForgotten,
		)
		if err != nil {
			return nil, errors.Wrap(err, "Database error")
		}
		resAry = append(resAry, result)
	}
	if rows.Err() != nil {
		return nil, errors.Wrap(err, "Database error")
	}
	if scopeChannel.Valid && len(resAry) == 0 {
		// retry without channel scope
		return mod.GetFactoidHistory(name, "")
	}
	return resAry, nil
}

// FillInfo transforms a bare FactoidInfo into a full FactoidInfo.
func (fi *Factoid) FillInfo(channel slack.ChannelID) error {
	if !fi.IsBareInfo {
		return nil
	}
	newInfo, err := fi.Mod.GetFactoidInfo(fi.FactoidName, channel, false)
	if err != nil {
		return err
	}
	fi.IsBareInfo = false

	fi.DbID = newInfo.DbID
	fi.FactoidName = newInfo.FactoidName
	fi.ScopeChannel = newInfo.ScopeChannel
	fi.IsForgotten = newInfo.IsForgotten
	fi.IsLocked = newInfo.IsLocked
	fi.LastChannel = newInfo.LastChannel
	fi.LastMessage = newInfo.LastMessage
	fi.LastUser = newInfo.LastUser
	fi.LastTimestamp = newInfo.LastTimestamp
	fi.RawSource = newInfo.RawSource
	return nil
}

func (mod *FactoidModule) SaveFactoid(name string, channel slack.ChannelID, rawSource string, source marvin.ActionSource) error {
	if len(name) > FactoidNameMaxLen {
		return errors.Errorf("Factoid name is too long (%d > %d)", len(name), FactoidNameMaxLen)
	}
	if strings.ContainsAny(name, " \n/\"") {
		return errors.Errorf("Factoid name contains prohibited characters (space, newline, forward slash, double quote)")
	}
	stmt, err := mod.team.DB().Prepare(sqlMakeFactoid)
	if err != nil {
		return errors.Wrap(err, "Database error")
	}
	defer stmt.Close()

	scopeChannel := sql.NullString{Valid: channel != "", String: string(channel)}

	_, err = stmt.Exec(
		name, scopeChannel, rawSource,
		string(source.UserID()), string(source.ChannelID()), string(source.MsgTimestamp()),
	)
	if err != nil {
		return errors.Wrap(err, "Database error")
	}
	return nil
}

func (mod *FactoidModule) ListFactoids(match string, channel slack.ChannelID) (channelOnly, global []string, err error) {
	if len(match) > FactoidNameMaxLen {
		return nil, nil, errors.Errorf("Factoid name is too long (%d > %d)", len(match), FactoidNameMaxLen)
	}
	stmt, err := mod.team.DB().Prepare(sqlListMatches)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Database error")
	}
	defer stmt.Close()

	scopeChannel := sql.NullString{Valid: channel != "", String: string(channel)}

	cursor, err := stmt.Query(
		match, scopeChannel,
	)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Database error")
	}
	var name string
	var isChannelScope bool

	for cursor.Next() {
		err = cursor.Scan(&name, &isChannelScope)
		if err != nil {
			return nil, nil, errors.Wrap(err, "Database error")
		}
		if isChannelScope {
			channelOnly = append(channelOnly, name)
		} else {
			global = append(global, name)
		}
	}
	if cursor.Err() != nil {
		return nil, nil, errors.Wrap(cursor.Err(), "Database error")
	}
	return channelOnly, global, nil
}

func (mod *FactoidModule) ListFactoidsWithInfo(match string, channel slack.ChannelID) ([]*Factoid, error) {
	if len(match) > FactoidNameMaxLen {
		return nil, errors.Errorf("Factoid name is too long (%d > %d)", len(match), FactoidNameMaxLen)
	}
	stmt, err := mod.team.DB().Prepare(sqlListMatchesWithInfo)
	if err != nil {
		return nil, errors.Wrap(err, "Database error")
	}
	defer stmt.Close()

	scopeChannel := sql.NullString{Valid: channel != "", String: string(channel)}

	cursor, err := stmt.Query(
		match, scopeChannel, false,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Database error")
	}
	var result *Factoid
	var list []*Factoid

	for cursor.Next() {
		result = new(Factoid)
		err = cursor.Scan(&result.DbID, &result.FactoidName, &result.RawSource, &scopeChannel,
			(*string)(&result.LastUser), (*string)(&result.LastChannel), (*string)(&result.LastMessage),
			&result.LastTimestamp,
			&result.IsLocked, &result.IsForgotten,
		)
		if err != nil {
			return nil, errors.Wrap(cursor.Err(), "Database error")
		}
		list = append(list, result)
	}
	if cursor.Err() != nil {
		return nil, errors.Wrap(cursor.Err(), "Database error")
	}
	return list, nil
}

func (mod *FactoidModule) ForgetFactoid(dbID int64, isForgotten bool) error {
	stmt, err := mod.team.DB().Prepare(sqlForgetFactoid)
	if err != nil {
		return errors.Wrap(err, "Database error")
	}
	defer stmt.Close()

	_, err = stmt.Exec(isForgotten, dbID)
	if err != nil {
		return errors.Wrap(err, "Database error")
	}
	return nil
}

func (mod *FactoidModule) LockFactoid(dbID int64, isLocked bool) error {
	stmt, err := mod.team.DB().Prepare(sqlLockFactoid)
	if err != nil {
		return errors.Wrap(err, "Database error")
	}
	defer stmt.Close()

	_, err = stmt.Exec(isLocked, dbID)
	if err != nil {
		return errors.Wrap(err, "Database error")
	}
	return nil
}
