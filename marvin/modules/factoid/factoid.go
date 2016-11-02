package factoid

import (
	"flag"
	"sync"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
)

type API interface {
	marvin.Module

	RunFactoidSplit(args []string, source marvin.ActionSource)
	RunFactoidLine(rawLine string, source marvin.ActionSource)
	RunFactoidMessage(msg slack.RTMRawMessage)
}

// ---

func init() {
	marvin.RegisterModule(NewFactoidModule)
}

const Identifier = "factoid"

type FactoidModule struct {
	team marvin.Team

	onReact marvin.Module
}

func NewFactoidModule(t marvin.Team) marvin.Module {
	mod := &FactoidModule{team: t}
	return mod
}

func (mod *FactoidModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *FactoidModule) Load(t marvin.Team) {
	mod.team.DB().MustMigrate(Identifier, 1478065581, sqlMigrate1)
	mod.team.DB().SyntaxCheck(
		sqlMakeFactoid,
	)
}

func (mod *FactoidModule) Enable(t marvin.Team) {
}

func (mod *FactoidModule) Disable(t marvin.Team) {
}

// ---

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
		last_set          datetime NOT NULL,    -- time.Time

		locked            boolean NOT NULL DEFAULT FALSE,
		forgotten         boolean NOT NULL DEFAULT FALSE,

		INDEX factoid_get (name, channel_only)
	)`

	// $1 = name $2 = scopeChannel $3 = source $4 = userid $5 = msg_chan $6 = msg_ts
	sqlMakeFactoid = `
	INSERT INTO module_factoid_factoids
	(name, channel_only, rawtext, last_set_user, last_set_channel, last_set_ts, last_set)
	VALUES
	($1,   $2,           $3,      $4,            $5,               $6, CURRENT_TIMESTAMP)
	`

	// $1 = channel $2 = scopeChannel
	sqlGetFactoid = `
	SELECT rawtext, channel_only IS NULL as was_channel_only
	FROM module_factoid_factoids
	WHERE name = $1 AND (channel_only IS NULL OR channel_only = $2)
	AND forgotten = FALSE
	ORDER BY last_set
	LIMIT 1
	`
)

type rememberArgs struct {
	flagSet   *flag.FlagSet
	wantHelp  bool
	makeLocal bool
}

func makeRememberArgs() interface{} {
	var obj = new(rememberArgs)
	obj.flagSet = flag.NewFlagSet("remember", flag.ContinueOnError)
	obj.flagSet.BoolVar(&obj.makeLocal, "local", false, "make a local (one channel only) factoid")
	return obj
}

var rememberArgsPool = sync.Pool{New: makeRememberArgs}

const rememberHelp = "Usage: `@marvin [--local] [name] [contents...]"

func (mod *FactoidModule) CmdRemember(t marvin.Team, args *marvin.CommandArguments) {
	flags := rememberArgsPool.Get().(*rememberArgs)
	flagErr := flags.flagSet.Parse(args.Arguments)
	if flagErr == flag.ErrHelp {
		flags.wantHelp = true
	}
	if flags.flagSet.NArg() < 2 {
		flags.wantHelp = true
	}

	//if flags.wantHelp
}
