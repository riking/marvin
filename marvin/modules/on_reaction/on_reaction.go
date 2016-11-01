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
	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
)

// ---
// Public Interface

// API is the cross-module interface that on_reaction presents.
type API interface {
	marvin.Module

	RegisterHandler(h ReactionHandler, moduleIdentifier string)
	RegisterFunc(f ReactionCallbackFunc, moduleIdentifier string)
	Unregister(moduleIdentifier string)

	ListenMessage(which slack.MessageID, moduleIdentifier string, data []byte)
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
	sqlCreateTable = `CREATE TABLE module_on_reaction_data (
		id		SERIAL PRIMARY KEY,
		channel		char(9) NOT NULL,	-- "C01234ABC"
		ts		char(17) NOT NULL,	-- "1477979163.000007"
		module		varchar(255) NOT NULL,
		data		BLOB,

		-- [{UserID, emoji}, {UserID, emoji}, ...]
		last_seen	jsonb DEFAULT '[]'::jsonb,

		CONSTRAINT channel_ts UNIQUE(channel, ts)
	)`
	sqlListenMessage = `INSERT INTO module_on_reaction_data
	(channel, ts, module, data)
	VALUES ($1, $2, $3, $4)
	`
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
	team    marvin.Team
	botUser slack.UserID
}

func NewOnReactionModule(t marvin.Team) marvin.Module {
	mod := &OnReactionModule{team: t}
	return mod
}

func (mod *OnReactionModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *OnReactionModule) Load(t marvin.Team) {
}

func (mod *OnReactionModule) Disable(t marvin.Team) {
}

func (mod *OnReactionModule) Enable(t marvin.Team) {
}

// ---
