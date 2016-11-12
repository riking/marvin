package factoid

import (
	"flag"
	"fmt"
	"strings"
	"sync"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
)

type rememberArgs struct {
	flagSet   *flag.FlagSet
	wantHelp  bool
	makeLocal bool

	wasLockFailure bool
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
		return marvin.CmdUsage(args, rememberHelp).WithNoEdit().WithSimpleUndo()
	}

	factoidName := flags.flagSet.Arg(0)
	scopeChannel := args.Source.ChannelID()
	if !flags.makeLocal {
		scopeChannel = ""
	}
	factoidSource := slack.UnescapeText(strings.Join(flags.flagSet.Args()[1:], " "))

	if len(factoidName) > FactoidNameMaxLen {
		return marvin.CmdFailuref(args, "Factoid name is too long: %s", factoidName).WithEdit().WithSimpleUndo()
	}

	args.SetModuleData(flags)
	factoidInfo, err := mod.GetFactoidInfo(factoidName, scopeChannel, false)
	if err == ErrNoSuchFactoid {
		// make a pseudo value that passes all the checks
		factoidInfo = FactoidInfo{IsLocked: false, ScopeChannel: ""}
	} else if err != nil {
		return marvin.CmdError(args, err, "Could not check existing factoid")
	}

	if factoidInfo.IsLocked {
		if flags.makeLocal {
			// Overriding a locked global with a local is OK
			if factoidInfo.ScopeChannel != "" {
				flags.wasLockFailure = true
				if args.Source.AccessLevel() < marvin.AccessLevelChannelAdmin {
					return marvin.CmdFailuref(args, "Factoid is locked (last edited by %v)", factoidInfo.LastUser)
				} else {
					return marvin.CmdFailuref(args, "Factoid is locked; use `@marvin factoid unlock %s` to edit.", factoidName).WithEdit()
				}
			}
		} else {
			flags.wasLockFailure = true
			if args.Source.AccessLevel() < marvin.AccessLevelAdmin {
				return marvin.CmdFailuref(args, "Factoid is locked (last edited by %v)", factoidInfo.LastUser)
			} else {
				return marvin.CmdFailuref(args, "Factoid is locked; use `@marvin factoid unlock %s` to edit.", factoidName).WithEdit()
			}
		}
	}

	fi := FactoidInfo{
		mod:       mod,
		RawSource: factoidSource,
	}
	err = util.PCall(func() error {
		fi.Tokens()
		return nil
	})
	if err != nil {
		return marvin.CmdFailuref(args, "Bad syntax: %v", err).WithEdit()
	}

	util.LogGood("Saving factoid", factoidName, "-", factoidSource)
	err = mod.SaveFactoid(factoidName, scopeChannel, factoidSource, args.Source)
	if err != nil {
		return marvin.CmdError(args, err, "Could not save factoid")
	}
	return marvin.CmdSuccess(args, "").WithCustomUndo().WithEdit()
}

func (mod *FactoidModule) CmdGet(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if len(args.Arguments) < 1 {
		return marvin.CmdUsage(args, "`@marvin factoid get <name> [args...]` (args optional)")
	}

	factoidName := args.Pop()
	factoidArgs := args.Arguments

	factoidInfo, err := mod.GetFactoidBare(factoidName, args.Source.ChannelID())
	if err == ErrNoSuchFactoid {
		return marvin.CmdFailuref(args, "No such factoid").WithEdit()
	} else if err != nil {
		return marvin.CmdError(args, err, "Error retrieving factoid")
	}

	// TODO side-effects...
	result, err := mod.RunFactoid(factoidInfo, args.Source, factoidArgs)
	if err != nil {
		return marvin.CmdError(args, err, "Factoid run error")
	}
	return marvin.CmdSuccess(args, result).WithNoEdit()
}

func (mod *FactoidModule) CmdSend(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if len(args.Arguments) < 2 {
		return marvin.CmdUsage(args, "`@marvin factoid send <channel> <name> [args...]` (args optional)")
	}

	factoidName := args.Pop()
	factoidArgs := args.Arguments

	factoidInfo, err := mod.GetFactoidBare(factoidName, args.Source.ChannelID())
	if err == ErrNoSuchFactoid {
		return marvin.CmdFailuref(args, "No such factoid").WithEdit()
	} else if err != nil {
		return marvin.CmdError(args, err, "Error retrieving factoid")
	}

	// TODO side-effects...
	result, err := mod.RunFactoid(factoidInfo, args.Source, factoidArgs)
	if err != nil {
		return marvin.CmdError(args, err, "Factoid run error")
	}

	panic("NotImplemented")
	// TODO sending messages to other channels
	_ = result
	return marvin.CmdFailuref(args, "NotImplemented") // TODO
}

func (mod *FactoidModule) CmdSource(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if len(args.Arguments) < 1 {
		return marvin.CmdUsage(args, "`@marvin factoid source <name>`")
	}

	factoidName := args.Pop()

	factoidInfo, err := mod.GetFactoidBare(factoidName, args.Source.ChannelID())
	if err == ErrNoSuchFactoid {
		return marvin.CmdFailuref(args, "No such factoid").WithEdit()
	} else if err != nil {
		return marvin.CmdError(args, err, "Error retrieving factoid")
	}

	return marvin.CmdSuccess(args, fmt.Sprintf("```\n%s\n```", factoidInfo.RawSource)).WithNoEdit().WithSimpleUndo()
}
