/*
on_reaction provides a public cross-module interface.

Call Register* during the Register phase of setup.
A module may only register one handler. If RegisterHandler is called multiple times, only the last is used.

	struct Module {
		onReact on_reaction.API
	}

	// NewModule()
	err = team.DependModule(on_reaction.Identifier, &mod.onReact)

	// Enable()
	mod.onReact.RegisterHandler(mod, Identifier)

	// Disable()
	if mod.onReact != nil {
		mod.onReact.Unregister(Identifier)
	}

To start listening for reactions, call ListenMessage with the channel/timestamp.

	mod.onReact.ListenMessage(slack.MsgID(channel, ts), Identifier, jsonBytes)

*/
package on_reaction

import (
	"fmt"

	"sync"

	"github.com/pkg/errors"
	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
)

// ---
// Public Interface

// API is the cross-module interface that on_reaction presents.
type API interface {
	marvin.Module

	RegisterHandler(h ReactionHandler, moduleIdentifier string)
	RegisterFunc(f ReactionCallbackFunc, moduleIdentifier string)
	Unregister(moduleIdentifier string)

	ListenMessage(which slack.MessageID, moduleIdentifier string, data []byte) error
}

// ReactionEvent contains all the available data about a reaction event.
type ReactionEvent struct {
	slack.MessageID
	EmojiName string
	IsAdded   bool
	UserID    slack.UserID
	// EventTS may be empty in the event of a backfill call
	EventTS slack.MessageTS
}

// ReactionHandler is the interface for the callbacks that OnReactionModule exports.
type ReactionHandler interface {
	OnReaction(event *ReactionEvent, customData []byte) error
}

type ReactionCallbackFunc func(event *ReactionEvent, customData []byte) error

// OnReaction implements ReactionHandler by calling the function.
func (f ReactionCallbackFunc) OnReaction(event *ReactionEvent, customData []byte) error {
	return f(event, customData)
}

// ---
// Database

const (
	sqlMigrate1 = `CREATE TABLE module_on_reaction_data (
		id		SERIAL PRIMARY KEY,
		channel		char(9) NOT NULL,	-- "C01234ABC"
		ts		char(17) NOT NULL,	-- "1477979163.000007"
		module		varchar(255) NOT NULL,
		data		BLOB,

		-- [{UserID, emoji}, {UserID, emoji}, ...]
		last_seen	jsonb DEFAULT '[]'::jsonb,

		CONSTRAINT channel_ts UNIQUE(channel, ts)
	)`
	// $1 = channel $2 = ts $3 = module $4 = data
	sqlListenMessage = `INSERT INTO module_on_reaction_data
	(channel, ts, module, data)
	VALUES ($1, $2, $3, $4)
	`
	// $1 = channel $2 = ts
	sqlCheckMessage = `SELECT module, data
	FROM module_on_reaction_data
	WHERE channel = $1 AND ts = $2
	`
	// $1 = channel $2 = ts $3 = user $4 = emoji
	sqlRecordReaction = `UPDATE module_on_reaction_data
	WHERE channel = $1 AND ts = $2
	SET last_seen = last_seen || json_build_object('user', $3, 'emoji', $4)
	`
)

// ---
// Setup

func init() {
	marvin.RegisterModule(NewOnReactionModule)
}

const Identifier = "on_reaction"

type OnReactionModule struct {
	team marvin.Team

	listenLock sync.Mutex
	listenMap  map[marvin.ModuleID]ReactionHandler
}

func NewOnReactionModule(t marvin.Team) marvin.Module {
	mod := &OnReactionModule{team: t}
	return mod
}

func (mod *OnReactionModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *OnReactionModule) Load(t marvin.Team) {
	t.DB().MustMigrate(Identifier, 1478042524, sqlMigrate1)
	t.DB().SyntaxCheck(
		sqlListenMessage,
		sqlCheckMessage,
		sqlRecordReaction,
	)
}

func (mod *OnReactionModule) Disable(t marvin.Team) {
}

func (mod *OnReactionModule) Enable(t marvin.Team) {
	t.OnEvent(Identifier, "reaction_added", mod.ReactionEvent)
	t.OnEvent(Identifier, "reaction_removed", mod.ReactionEvent)
}

// ---

// https://api.slack.com/events/reaction_added
func (mod *OnReactionModule) ReactionEvent(rtm slack.RTMRawMessage) {
	var msg struct {
		TargetUser slack.UserID `json:"item_user"`
		Item       struct {
			Type    string          `json:"type"`
			Channel slack.ChannelID `json:"channel"`
			TS      slack.MessageTS `json:"ts"`
		}
		EventTS slack.MessageTS `json:"event_ts"`
	}

	rtm.ReMarshal(&msg)
	if msg.TargetUser != mod.team.BotUser() {
		return
	}
	if msg.Item.Type != "message" {
		return
	}
	reactionEvent := ReactionEvent{
		MessageID: slack.MessageID{
			ChannelID: msg.Item.Channel,
			MessageTS: msg.Item.TS,
		},
		UserID:    rtm.UserID(),
		EmojiName: rtm.StringField("reaction"),
		EventTS:   msg.EventTS,
		IsAdded:   rtm.Type() == "reaction_added",
	}
	cbs, err := mod.getListens(reactionEvent.MessageID)
	if err != nil {
		// TODO not quite right
		mod.team.ReportError(err, marvin.ActionSourceUserMessage{Msg: rtm})
		return
	}
	for _, v := range cbs {
		err = util.PCall(func() error {
			return v.Cb.OnReaction(&reactionEvent, v.Data)
		})
		if err != nil {
			fmt.Println("[ERR]", err) // TODO
		}
	}
	fmt.Println("[DEBUG]", "dispatched reaction to", len(cbs), "callbacks")
}

type cbData struct {
	Cb   ReactionHandler
	Data []byte
}

func (mod *OnReactionModule) getListens(msgID slack.MessageID) ([]cbData, error) {
	stmt, err := mod.team.DB().Prepare(sqlCheckMessage)
	if err != nil {
		return nil, errors.Wrap(err, "on_reaction preparing SQL statement")
	}
	defer stmt.Close()
	rows, err := stmt.Query(string(msgID.ChannelID), string(msgID.MessageTS))
	if err != nil {
		return nil, errors.Wrap(err, "on_reaction checking DB")
	}
	var moduleID string
	var target cbData
	var ret []cbData
	for rows.Next() {
		err := rows.Scan(&moduleID, &target.Data)
		if err != nil {
			return nil, errors.Wrap(err, "on_reaction reading SQL results")
		}
		handler, ok := mod.listenMap[marvin.ModuleID(moduleID)]
		if !ok {
			continue
		}
		target.Cb = handler
		ret = append(ret, target)
	}
	return ret, nil
}
