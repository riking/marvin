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

	"net/url"
	"time"

	"encoding/json"

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

	RegisterHandler(h ReactionHandler, modID marvin.ModuleID)
	RegisterFunc(f ReactionCallbackFunc, modID marvin.ModuleID)
	Unregister(modID marvin.ModuleID)

	ListenMessage(which slack.MessageID, modID marvin.ModuleID, data []byte) error
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

type ErrNoHandler struct{}

func (e ErrNoHandler) Error() string {
	return "You must register a handler before calling ListenMessage()"
}

// ---
// Setup

func init() {
	marvin.RegisterModule(NewOnReactionModule)
}

const Identifier marvin.ModuleID = "on_reaction"

type OnReactionModule struct {
	team marvin.Team

	listenLock sync.Mutex
	listenMap  map[marvin.ModuleID]ReactionHandler
}

func NewOnReactionModule(t marvin.Team) marvin.Module {
	mod := &OnReactionModule{
		team:      t,
		listenMap: make(map[marvin.ModuleID]ReactionHandler),
	}
	return mod
}

func (mod *OnReactionModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *OnReactionModule) Load(t marvin.Team) {
	t.DB().MustMigrate(string(Identifier), 1478042524, sqlMigrate1)
	t.DB().SyntaxCheck(
		sqlListenMessage,
		sqlCheckMessage,
	)
}

func (mod *OnReactionModule) Enable(t marvin.Team) {
	t.OnEvent(Identifier, "reaction_added", mod.ReactionEvent)
	t.OnEvent(Identifier, "reaction_removed", mod.ReactionEvent)
}

func (mod *OnReactionModule) Disable(t marvin.Team) {
	t.OffAllEvents(Identifier)
}

// ---
// Database

const (
	sqlMigrate1 = `CREATE TABLE module_on_reaction_data (
		id		SERIAL PRIMARY KEY,
		channel		varchar(10) NOT NULL,	-- "C01234ABC"
		ts		varchar(20) NOT NULL,	-- "1477979163.000007"
		module		varchar(255) NOT NULL,
		data		bytea,

		-- [{UserID, emoji}, {UserID, emoji}, ...]
		last_seen	jsonb DEFAULT '[]'::jsonb,

		CONSTRAINT channel_ts UNIQUE(channel, ts)
	)`

	// $1 = channel $2 = ts $3 = module $4 = data
	sqlListenMessage = `
	INSERT INTO module_on_reaction_data
	(channel, ts, module, data)
	VALUES ($1, $2, $3, $4)
	`

	// $1 = channel $2 = ts
	sqlCheckMessage = `
	SELECT module, data
	FROM module_on_reaction_data
	WHERE channel = $1 AND ts = $2
	`
)

func (mod *OnReactionModule) ListenMessage(which slack.MessageID, modID marvin.ModuleID, data []byte) error {
	handler := mod.getHandler(modID)
	if handler == nil {
		return errors.Wrapf(ErrNoHandler{}, "No handler for %s", modID)
	}

	stmt, err := mod.team.DB().Prepare(sqlListenMessage)
	if err != nil {
		return errors.Wrap(err, "on_reaction preparing SQL statement")
	}
	defer stmt.Close()
	_, err = stmt.Exec(string(which.ChannelID), string(which.MessageTS), string(modID), []byte(data))
	if err != nil {
		return errors.Wrap(err, "on_reaction inserting row")
	}
	go mod.backfillReactions(which, handler, data, 0)
	return nil
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
		target.Cb = mod.getHandler(marvin.ModuleID(moduleID))
		ret = append(ret, target)
	}
	return ret, nil
}

// ---

func (mod *OnReactionModule) RegisterHandler(h ReactionHandler, modID marvin.ModuleID) {
	mod.listenLock.Lock()
	defer mod.listenLock.Unlock()

	mod.listenMap[modID] = h
}

func (mod *OnReactionModule) RegisterFunc(f ReactionCallbackFunc, modID marvin.ModuleID) {
	mod.RegisterHandler(f, modID)
}

func (mod *OnReactionModule) Unregister(modID marvin.ModuleID) {
	mod.listenLock.Lock()
	delete(mod.listenMap, modID)
	mod.listenLock.Unlock()
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
			util.LogError(err)
		}
	}
	util.LogDebug("dispatched reaction to", len(cbs), "callbacks")
}

func (mod *OnReactionModule) getHandler(modID marvin.ModuleID) ReactionHandler {
	mod.listenLock.Lock()
	defer mod.listenLock.Unlock()

	for k, v := range mod.listenMap {
		if k == modID {
			return v
		}
	}
	return nil
}

func (mod *OnReactionModule) backfillReactions(which slack.MessageID, handler ReactionHandler, data []byte, retries int) {
	form := url.Values{
		"channel": []string{string(which.ChannelID)},
		"ts":      []string{string(which.MessageTS)},
		"full":    []string{"true"},
	}
	resp, err := mod.team.SlackAPIPost("reactions.get", form)
	if err != nil {
		if retries >= 5 {
			fmt.Printf("[ERR] %+v\n", errors.Wrap(err, "cannot contact slack API"))
			return
		} else {
			time.Sleep(3 * time.Second)
			mod.backfillReactions(which, handler, data, retries+1)
		}
	}
	var response struct {
		slack.APIResponse
		Channel slack.ChannelID `json:"channel"`
		Message struct {
			Reactions []struct {
				Emoji string         `json:"name"`
				Count int            `json:"count"`
				Users []slack.UserID `json:"users"`
			} `json:"reactions"`
		} `json:"message"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	resp.Body.Close()
	if err != nil {
		if retries >= 5 {
			fmt.Printf("[ERR] %+v\n", errors.Wrap(err, "cannot contact slack API"))
			return
		} else {
			time.Sleep(3 * time.Second)
			mod.backfillReactions(which, handler, data, retries+1)
		}
	}
	if !response.OK {
		fmt.Printf("[ERR] %+v\n", errors.Wrap(response, "Slack API returned error: reactions.get"))
		return
	}

	var reactionEvent = ReactionEvent{
		MessageID: which,
		EventTS:   "",
		IsAdded:   true,
	}
	reactionEvent.MessageID.ChannelID = response.Channel
	for _, v := range response.Message.Reactions {
		reactionEvent.EmojiName = v.Emoji
		for _, uid := range v.Users {
			reactionEvent.UserID = uid
			util.PCall(func() error {
				return handler.OnReaction(&reactionEvent, data)
			})
		}
	}
	return
}
