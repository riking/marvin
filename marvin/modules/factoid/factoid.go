package factoid

import (
	"flag"
	"strings"
	"sync"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
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
	mod.doMigrate(t)
	mod.doSyntaxCheck(t)
}

func (mod *FactoidModule) Enable(team marvin.Team) {
	parent := marvin.NewParentCommand()
	remember := parent.RegisterCommandFunc("remember", mod.CmdRemember, "`@marvin remember [--local] [name] [value]` (alias `r`) saves a factoid.")
	parent.RegisterCommand("rem", remember)
	parent.RegisterCommand("r", remember)
	parent.RegisterCommandFunc("get", mod.CmdGet, "`get` runs a factoid with the standard argument parsing instead of the factoid argument parsing.")

	team.RegisterCommand("factoid", parent)
	team.RegisterCommand("f", parent) // TODO RegisterAlias
	team.RegisterCommand("remember", remember)
	team.RegisterCommand("rem", remember)
	team.RegisterCommand("r", remember)
}

func (mod *FactoidModule) Disable(t marvin.Team) {
	t.UnregisterCommand("factoid")
	t.UnregisterCommand("f")
	t.UnregisterCommand("remember")
	t.UnregisterCommand("rem")
	t.UnregisterCommand("r")
}

// ---

type rememberArgs struct {
	flagSet   *flag.FlagSet
	wantHelp  bool
	makeLocal bool
}

func makeRememberArgs() interface{} {
	var obj = new(rememberArgs)
	obj.flagSet = flag.NewFlagSet("remember", flag.ContinueOnError)
	obj.flagSet.BoolVar(&obj.makeLocal, "--local", false, "make a local (one channel only) factoid")
	return obj
}

var rememberArgsPool = sync.Pool{New: makeRememberArgs}

const rememberHelp = "Usage: `@marvin [--local] [name] [contents...]`"

func (mod *FactoidModule) CmdRemember(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	flags := rememberArgsPool.Get().(*rememberArgs)
	flagErr := flags.flagSet.Parse(args.Arguments)
	if flagErr == flag.ErrHelp {
		flags.wantHelp = true
	} else if flags.flagSet.NArg() < 2 {
		flags.wantHelp = true
	}

	if flags.wantHelp {
		return marvin.CmdUsage(args, rememberHelp)
	}

	factoidName := flags.flagSet.Arg(0)
	scopeChannel := args.Source.ChannelID()
	if !flags.makeLocal {
		scopeChannel = ""
	}
	factoidSource := slack.UnescapeText(strings.Join(flags.flagSet.Args()[1:], " "))

	if len(factoidName) > FactoidNameMaxLen {
		return marvin.CmdFailuref(args, "Factoid name is too long: %s", factoidName)
	}

	factoidInfo, err := mod.GetFactoidInfo(factoidName, scopeChannel, false)
	if err == ErrNoSuchFactoid {
		// make a pseudo value that passes all the checks
		factoidInfo = FactoidInfo{IsLocked: false, ScopeChannel: ""}
	} else if err != nil {
		return marvin.CmdError(args, err, "Could not check existing factoid")
	}

	if factoidInfo.IsLocked {
		if flags.makeLocal {
			if factoidInfo.ScopeChannel == "" {
				// Overriding a locked global with a local is OK
			} else if args.Source.AccessLevel() < marvin.AccessLevelChannelAdmin {
				return marvin.CmdFailuref(args, "Factoid is locked (last edited by %v)", factoidInfo.LastUser)
			} else {
				return marvin.CmdFailuref(args, "Factoid is locked; use `@marvin factoid unlock %s` to edit.", factoidName)
			}
		} else {
			if args.Source.AccessLevel() < marvin.AccessLevelAdmin {
				return marvin.CmdFailuref(args, "Factoid is locked (last edited by %v)", factoidInfo.LastUser)
			} else {
				return marvin.CmdFailuref(args, "Factoid is locked; use `@marvin factoid unlock %s` to edit.", factoidName)
			}
		}
	}

	util.LogGood("Saving factoid", factoidName, "-", factoidSource)
	err = mod.SaveFactoid(factoidName, scopeChannel, factoidSource, args.Source)
	if err != nil {
		return marvin.CmdError(args, err, "Could not save factoid")
	}
	return marvin.CmdSuccess(args, "")
}

func (mod *FactoidModule) ProcessMsg(msg string, source marvin.ActionSource) {

}

func (mod *FactoidModule) CmdGet(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if len(args.Arguments) < 1 {
		return marvin.CmdUsage(args, "`@marvin factoid get <name> [args...]` (args optional)")
	}

	factoidName := args.Pop()
	factoidArgs := args.Arguments

	factoidInfo, err := mod.GetFactoidBare(factoidName, args.Source.ChannelID())
	if err == ErrNoSuchFactoid {
		return marvin.CmdFailuref(args, "No such factoid")
	} else if err != nil {
		return marvin.CmdError(args, err, "Error retrieving factoid")
	}

	result, err := RunFactoid(factoidInfo, args.Source, factoidArgs)
	if err != nil {
		return marvin.CmdError(args, err, "Factoid run error")
	}
	return marvin.CmdSuccess(args, result)
}
