package factoid

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/util"
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

const (
	helpRemember = "`@marvin remember [--local] [name] [value]` (alias `r`) saves a factoid."
	helpGet      = "`factoid get <name> [args...]` runs a factoid with the standard argument parsing instead of the factoid argument parsing."
	helpSource   = "`factoid source <name>` views the source of a factoid."
	helpInfo     = "`factoid info [-f] <name>` views detailed information about a factoid."
	helpList     = "`factoid list [pattern]` lists all factoids with `pattern` in their name."
	helpForget   = "`factoid forget <name>` forgets the most recent version of a factoid."
	helpUnforget = "`factoid unforget <name>` un-forgets a previously forgotten factoid."
)

func (mod *FactoidModule) CmdRemember(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	flags := rememberArgsPool.Get().(*rememberArgs)
	flagErr := flags.flagSet.Parse(args.Arguments)
	if flagErr == flag.ErrHelp {
		flags.wantHelp = true
	} else if flags.flagSet.NArg() < 2 {
		flags.wantHelp = true
	}

	if flags.wantHelp {
		return marvin.CmdUsage(args, helpRemember).WithNoEdit().WithSimpleUndo()
	}

	factoidName := flags.flagSet.Arg(0)
	scopeChannel := args.Source.ChannelID()
	if !flags.makeLocal {
		scopeChannel = ""
	}
	factoidSource := slack.UnescapeTextAll(strings.Join(flags.flagSet.Args()[1:], " "))

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
		Mod:       mod,
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
		return marvin.CmdUsage(args, helpGet)
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

func (mod *FactoidModule) CmdSource(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if len(args.Arguments) < 1 {
		return marvin.CmdUsage(args, helpSource)
	}

	factoidName := args.Pop()

	if len(factoidName) > FactoidNameMaxLen {
		return marvin.CmdFailuref(args, "Factoid name too long")
	}

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
		return marvin.CmdUsage(args, helpInfo)
	}

	factoidName := args.Pop()
	withForgotten := false
	if factoidName == "-f" {
		withForgotten = true
		factoidName = args.Pop()
	}

	if len(factoidName) > FactoidNameMaxLen {
		return marvin.CmdFailuref(args, "Factoid name too long")
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
	isForgotten := ""
	if factoidInfo.IsForgotten {
		isForgotten = "(forgotten) "
	}
	msg := fmt.Sprintf("`%s` %s%s%swas last edited by %v in %s. DB ID: %d.\n[Archive link: %s]\n```%s\n```",
		factoidName,
		isLocal, isLocked, isForgotten,
		factoidInfo.LastUser,
		mod.team.FormatChannel(factoidInfo.LastChannel),
		factoidInfo.DbID,
		mod.team.ArchiveURL(slack.MsgID(factoidInfo.LastChannel, factoidInfo.LastMessage)),
		factoidInfo.RawSource,
	)
	return marvin.CmdSuccess(args, msg).WithEdit().WithSimpleUndo()
}

func (mod *FactoidModule) CmdList(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	var match = ""

	if len(args.Arguments) > 1 {
		return marvin.CmdUsage(args, helpList)
	} else if len(args.Arguments) == 1 {
		match = args.Pop()
	}

	if len(match) > FactoidNameMaxLen {
		return marvin.CmdFailuref(args, "Factoid name too long")
	}

	channelScoped, global, err := mod.ListFactoids(match, args.Source.ChannelID())
	if err != nil {
		return marvin.CmdError(args, err, "Error listing factoids")
	}

	sort.Strings(channelScoped)
	sort.Strings(global)

	var buf bytes.Buffer
	if match == "" {
		fmt.Fprint(&buf, "List of factoids:\n")
	} else {
		fmt.Fprintf(&buf, "List of factoids matching `*%s*`:\n", match)
	}

	for _, v := range channelScoped {
		fmt.Fprintf(&buf, "`%s`\\* ", v)
	}
	if len(channelScoped) != 0 {
		fmt.Fprint(&buf, "\n")
	}
	for _, v := range global {
		fmt.Fprintf(&buf, "`%s` ", v)
	}
	return marvin.CmdSuccess(args, buf.String())
}

func (mod *FactoidModule) CmdForget(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if len(args.Arguments) != 1 {
		return marvin.CmdUsage(args, helpForget)
	}

	factoidName := args.Pop()
	if len(factoidName) > FactoidNameMaxLen {
		return marvin.CmdFailuref(args, "Factoid name too long").WithEdit().WithSimpleUndo()
	}

	factoidInfo, err := mod.GetFactoidInfo(factoidName, args.Source.ChannelID(), false)
	if err == ErrNoSuchFactoid {
		return marvin.CmdFailuref(args, "No such factoid").WithEdit().WithSimpleUndo()
	} else if err != nil {
		return marvin.CmdError(args, err, "Error retrieving factoid")
	}

	if factoidInfo.IsLocked {
		return marvin.CmdFailuref(args, "A locked factoid cannot be forgotten.").WithEdit().WithSimpleUndo()
	}

	err = mod.ForgetFactoid(factoidInfo.DbID, true)
	if err != nil {
		return marvin.CmdError(args, err, "Error forgetting factoid")
	}
	return marvin.CmdSuccess(args, fmt.Sprintf("Forgot `%s` with database ID %d", factoidName, factoidInfo.DbID)).WithNoEdit().WithNoUndo()
}
