package factoid

import (
	"flag"
	"fmt"
	"strings"
	"sync"

	"context"

	"github.com/pkg/errors"
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
	prevFactoidInfo, err := mod.GetFactoidInfo(factoidName, scopeChannel, false)
	if err == ErrNoSuchFactoid {
		// make a pseudo value that passes all the checks
		prevFactoidInfo = &Factoid{IsLocked: false, ScopeChannel: ""}
	} else if err != nil {
		return marvin.CmdError(args, err, "Could not check existing factoid")
	}

	if prevFactoidInfo.IsLocked {
		if flags.makeLocal {
			// Overriding a locked global with a local is OK
			if prevFactoidInfo.ScopeChannel != "" {
				flags.wasLockFailure = true
				if args.Source.AccessLevel() < marvin.AccessLevelChannelAdmin {
					return marvin.CmdFailuref(args, "Factoid is locked (last edited by %v)", prevFactoidInfo.LastUser)
				} else {
					return marvin.CmdFailuref(args, "Factoid is locked; use `@marvin factoid unlock %s` to edit.", factoidName).WithEdit()
				}
			}
		} else {
			flags.wasLockFailure = true
			if args.Source.AccessLevel() < marvin.AccessLevelAdmin {
				return marvin.CmdFailuref(args, "Factoid is locked (last edited by %v)", prevFactoidInfo.LastUser)
			} else {
				return marvin.CmdFailuref(args, "Factoid is locked; use `@marvin factoid unlock %s` to edit.", factoidName).WithEdit()
			}
		}
	}

	fi := Factoid{
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

	var of OutputFlags
	result, err := mod.RunFactoid(context.Background(), args.Arguments, &of, args.Source)
	if err == ErrNoSuchFactoid {
		return marvin.CmdFailuref(args, "No such factoid %s", result).WithEdit().WithReplyType(marvin.ReplyTypeInChannel)
	} else if err != nil {
		cErr := errors.Cause(err)
		if _, ok := cErr.(ErrUser); ok {
			return marvin.CmdFailuref(args, "Failed: %s", cErr).WithEdit().WithReplyType(marvin.ReplyTypeInChannel)
		}
		return marvin.CmdError(args, err, "Factoid run error")
	}

	cmdResult := marvin.CmdSuccess(args, result).WithEdit()
	if of.SideEffects {
		cmdResult = cmdResult.WithNoEdit().WithNoUndo()
	}
	if of.Pre {
		cmdResult.Message = fmt.Sprintf("```\n%s\n```", cmdResult.Message)
	}
	if of.NoReply {
		cmdResult.Message = ""
	} else if of.Say {
		cmdResult = cmdResult.WithReplyType(marvin.ReplyTypeFlagOmitUsername)
	}
	return cmdResult
}

func (mod *FactoidModule) CmdSend(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if len(args.Arguments) < 2 {
		return marvin.CmdUsage(args, "`@marvin factoid send <channel> <name> [args...]` (args optional)")
	}

	channelName := args.Pop()
	channelID := slack.ParseChannelID(channelName)
	if channelID == "" {
		userID := slack.ParseUserMention(channelName)
		if userID != "" {
			channelID, _ = mod.team.GetIM(userID)
		}
	}
	if channelID == "" {
		return marvin.CmdFailuref(args, "You must specify a target channel")
	}

	var of OutputFlags
	result, err := mod.RunFactoid(context.Background(), args.Arguments, &of, args.Source)
	if err == ErrNoSuchFactoid {
		return marvin.CmdFailuref(args, "No such factoid %s", result).WithEdit().WithReplyType(marvin.ReplyTypeInChannel)
	} else if err != nil {
		cErr := errors.Cause(err)
		if _, ok := cErr.(ErrUser); ok {
			return marvin.CmdFailuref(args, "Failed: %s", cErr).WithReplyType(marvin.ReplyTypeInChannel)
		}
		return marvin.CmdError(args, err, "Factoid run error")
	}

	panic("NotImplemented")
	// TODO sending messages to other channels
	_ = result

	cmdResult := marvin.CmdSuccess(args, "").WithEdit()
	if of.SideEffects {
		cmdResult = cmdResult.WithNoEdit().WithNoUndo()
	}
	if of.NoReply {
		return cmdResult
	}
	return cmdResult
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

func (mod *FactoidModule) CmdInfo(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if len(args.Arguments) < 1 {
		return marvin.CmdUsage(args, "`@marvin factoid info [-f] <name>`")
	}

	factoidName := args.Pop()
	withForgotten := false
	if factoidName == "-f" {
		withForgotten = true
		factoidName = args.Pop()
	}

	factoidInfo, err := mod.GetFactoidInfo(factoidName, args.Source.ChannelID(), withForgotten)
	if err == ErrNoSuchFactoid {
		return marvin.CmdFailuref(args, "No such factoid").WithEdit()
	} else if err != nil {
		return marvin.CmdError(args, err, "Error retrieving factoid")
	}

	isLocal := ""
	if factoidInfo.ScopeChannel != "" {
		isLocal = "(local to this channel) "
	}
	isLocked := ""
	if factoidInfo.IsLocked {
		isLocked = "(locked) "
	}
	msg := fmt.Sprintf("`%s` %s%swas last edited by %v in %s\n[Archive link: %s]\n```%s\n```",
		factoidName,
		isLocal, isLocked,
		factoidInfo.LastUser,
		mod.team.FormatChannel(factoidInfo.LastChannel),
		mod.team.ArchiveURL(slack.MsgID(factoidInfo.LastChannel, factoidInfo.LastMessage)),
		factoidInfo.RawSource,
	)
	return marvin.CmdSuccess(args, msg).WithEdit().WithSimpleUndo()
}
