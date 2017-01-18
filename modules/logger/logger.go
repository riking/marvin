package logger

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"

	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/slack/rtm"
	"github.com/riking/marvin/util"
)

func init() {
	marvin.RegisterModule(NewLoggerModule)
}

const Identifier = "logger"

type LoggerModule struct {
	team  marvin.Team
	cache *cache.Cache
}

func NewLoggerModule(t marvin.Team) marvin.Module {
	mod := &LoggerModule{team: t}
	mod.cache = cache.New(5*time.Minute, 30*time.Minute)
	return mod
}

func (mod *LoggerModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *LoggerModule) Load(t marvin.Team) {
	t.DB().MustMigrate(Identifier, 1479767598, sqlMigrate1)
	t.DB().SyntaxCheck(
		sqlInsertMessage,
		sqlEditMessage,
		sqlQueryChannel,
		sqlGetLastMessage,
	)
}

func (mod *LoggerModule) Enable(t marvin.Team) {
	t.OnEvent(Identifier, "message", mod.OnMessage)
	go mod.BackfillAll()
	t.HandleHTTP("/logs", http.HandlerFunc(mod.LogsIndex))
}

func (mod *LoggerModule) Disable(t marvin.Team) {
	t.OffAllEvents(Identifier)
}

// ---

const (
	sqlMigrate1 = `
	CREATE TABLE module_logger_logs (
		id SERIAL PRIMARY KEY,
		channel     varchar(15) NOT NULL,   -- slack.ChannelID
		timestamp   varchar(20) NOT NULL,   -- slack.MessageTS
		msg_user    varchar(15) NOT NULL,   -- slack.UserID
		text        TEXT        NOT NULL,
		raw         JSONB       NOT NULL,
		edited_user varchar(15) DEFAULT NULL, -- slack.UserID
		edited_ts   varchar(20) DEFAULT NULL, -- slack.MessageTS
		edited_raw  JSONB       DEFAULT NULL,

		UNIQUE(channel, timestamp)
	)`

	// $1 = channel $2 = timestamp $3 = user $4 = text $5 = original
	sqlInsertMessage = `
	INSERT INTO module_logger_logs
	(channel, timestamp, msg_user, text, raw)
	VALUES ($1, $2, $3, $4, $5::jsonb)
	ON CONFLICT DO NOTHING`

	// $1 = channel $2 = timestamp $3 = text $4 = editor $5 = edit_event_ts $6 = edit.original
	sqlEditMessage = `
	UPDATE module_logger_logs
	SET text = $3, edited_user = $4, edited_ts = $5, edited_raw = $6
	WHERE channel = $1 AND timestamp = $2`

	// $1 = channel $2 = before_ts $3 = limit
	sqlQueryChannel = `
	SELECT timestamp, msg_user, text, raw, edited_raw IS NULL as was_edited, edited_raw
	FROM module_logger_logs
	WHERE channel = $1
	AND COALESCE(timestamp < $2, TRUE)
	ORDER BY timestamp DESC
	LIMIT $3`

	// $1 = channel
	sqlGetLastMessage = `
	SELECT MAX(timestamp)
	FROM module_logger_logs
	WHERE channel = $1
	LIMIT 1`
)

// ---

func (mod *LoggerModule) OnMessage(_rtm slack.RTMRawMessage) {
	switch _rtm.Subtype() {
	case "message_changed", "message_deleted":
		return
	}

	stmt, err := mod.team.DB().Prepare(sqlInsertMessage)
	if err != nil {
		util.LogError(errors.Wrap(err, "prepare"))
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		string(_rtm.ChannelID()), string(_rtm.MessageTS()),
		string(_rtm.UserID()), string(_rtm.Text()), string(_rtm.Original()))
	if err != nil {
		util.LogError(errors.Wrap(err, "insert"))
		return
	}
}

func (mod *LoggerModule) getHistory(method string, channel slack.ChannelID, stmt *sql.Stmt) ([]json.RawMessage, error) {
	form := url.Values{
		"count": []string{"40"},
	}
	form.Set("channel", string(channel))

	row := stmt.QueryRow(string(channel))
	var lastSeenTS sql.NullString
	err := row.Scan(&lastSeenTS)
	if err == sql.ErrNoRows || !lastSeenTS.Valid {
		lastSeenTS.String = "0"
		lastSeenTS.Valid = true
	} else if err != nil {
		return nil, errors.Wrapf(err, "Backfill database err")
	}

	// PostRaw is used because we're unmarshalling a large response body into `json.RawMessage`s.
	// Best to keep the json work to a minimum.
	resp, err := mod.team.SlackAPIPostRaw(method, form)
	if err != nil {
		return nil, errors.Wrapf(err, "Slack API %s error", method)
	}

	var response struct {
		slack.APIResponse
		Messages []json.RawMessage `json:"messages"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	resp.Body.Close()
	if err != nil {
		return nil, errors.Wrapf(err, "Slack API %s json decode error", method)
	}
	if !response.OK {
		return nil, errors.Wrapf(response, "Slack API %s error", method)
	}
	if len(response.Messages) > 0 {
		fmt.Println("[Backfill]", channel, len(response.Messages), "recent messages")
	} else {
		fmt.Println("[Backfill]", channel, "no recent messages")
	}
	return response.Messages, nil
}

func (mod *LoggerModule) BackfillAll() {
	if mod.team.TeamConfig().IsDevelopment {
		return // do not backfill in development
	}

	stmt, err := mod.team.DB().Prepare(sqlGetLastMessage)
	if err != nil {
		util.LogError(errors.Wrap(err, "backfill database error"))
		return
	}
	defer stmt.Close()

	publicList := mod.listChannels()
	for _, v := range publicList {
		messages, err := mod.getHistory("channels.history", v, stmt)
		if err != nil {
			util.LogError(errors.Wrapf(err, "could not backfill logs for %s", v))
			return
		}
		c := mod.saveBackfillData(v, messages)
		if c != 0 {
			util.LogGood(fmt.Sprintf("Backfilled %d messages from %s", c, v))
		}
	}
	groupList := mod.listGroups()
	for _, v := range groupList {
		messages, err := mod.getHistory("groups.history", v, stmt)
		if err != nil {
			util.LogError(errors.Wrapf(err, "could not backfill logs for %s", v))
			return
		}
		c := mod.saveBackfillData(v, messages)
		if c != 0 {
			util.LogGood(fmt.Sprintf("Backfilled %d messages from %s", c, v))
		}
	}
	mpimList := mod.listMPIMs()
	for _, v := range mpimList {
		messages, err := mod.getHistory("mpim.history", v, stmt)
		if err != nil {
			util.LogError(errors.Wrapf(err, "could not backfill logs for %s", v))
			return
		}
		c := mod.saveBackfillData(v, messages)
		if c != 0 {
			util.LogGood(fmt.Sprintf("Backfilled %d messages from %s", c, v))
		}
	}
	imList := mod.listIMs()
	for _, v := range imList {
		messages, err := mod.getHistory("im.history", v, stmt)
		if err != nil {
			util.LogError(errors.Wrapf(err, "could not backfill logs for %s", v))
			return
		}
		c := mod.saveBackfillData(v, messages)
		if c != 0 {
			util.LogGood(fmt.Sprintf("Backfilled %d messages from %s", c, v))
		}
	}
}

func (mod *LoggerModule) listChannels() []slack.ChannelID {
	c := mod.team.GetRTMClient().(*rtm.Client)
	c.MetadataLock.RLock()
	defer c.MetadataLock.RUnlock()

	ids := make([]slack.ChannelID, len(c.Channels))
	for i := range c.Channels {
		ids[i] = c.Channels[i].ID
	}
	return ids
}

func (mod *LoggerModule) listGroups() []slack.ChannelID {
	c := mod.team.GetRTMClient().(*rtm.Client)
	c.MetadataLock.RLock()
	defer c.MetadataLock.RUnlock()

	ids := make([]slack.ChannelID, len(c.Groups))
	for i := range c.Groups {
		ids[i] = c.Groups[i].ID
	}
	return ids
}

func (mod *LoggerModule) listMPIMs() []slack.ChannelID {
	c := mod.team.GetRTMClient().(*rtm.Client)
	c.MetadataLock.RLock()
	defer c.MetadataLock.RUnlock()

	ids := make([]slack.ChannelID, len(c.Mpims))
	for i := range c.Mpims {
		ids[i] = c.Mpims[i].ID
	}
	return ids
}

func (mod *LoggerModule) listIMs() []slack.ChannelID {
	c := mod.team.GetRTMClient().(*rtm.Client)
	c.MetadataLock.RLock()
	defer c.MetadataLock.RUnlock()

	ids := make([]slack.ChannelID, len(c.Ims))
	for i := range c.Ims {
		ids[i] = c.Ims[i].ID
	}
	return ids
}

func (mod *LoggerModule) saveBackfillData(channel slack.ChannelID, messages []json.RawMessage) (totalAdded int64) {
	var msgInterestingFields struct {
		User slack.UserID    `json:"user"`
		TS   slack.MessageTS `json:"ts"`
		Text string          `json:"text"`
	}
	stmt, err := mod.team.DB().Prepare(sqlInsertMessage)
	if err != nil {
		util.LogError(errors.Wrap(err, "prepare"))
		return 0
	}
	defer stmt.Close()

	for _, msgRaw := range messages {
		err = json.Unmarshal([]byte(msgRaw), &msgInterestingFields)
		if err != nil {
			util.LogError(errors.Wrap(err, "unmarshal"))
			return
		}
		r, err := stmt.Exec(string(channel),
			string(msgInterestingFields.TS),
			string(msgInterestingFields.User),
			string(msgInterestingFields.Text),
			[]byte(msgRaw))
		if err != nil {
			util.LogError(err)
			return
		}
		c, err := r.RowsAffected()
		if err != nil {
			util.LogError(err)
			return
		}
		totalAdded += c
	}
	return totalAdded
}
