package logger

import (
	"github.com/pkg/errors"
	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
)

func init() {
	marvin.RegisterModule(NewLoggerModule)
}

const Identifier = "logger"

type LoggerModule struct {
	team marvin.Team
}

func NewLoggerModule(t marvin.Team) marvin.Module {
	mod := &LoggerModule{team: t}
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
	)
}

func (mod *LoggerModule) Enable(t marvin.Team) {
	t.OnEvent(Identifier, "message", mod.OnMessage)
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
	VALUES ($1, $2, $3, $4, $5::jsonb)`

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
