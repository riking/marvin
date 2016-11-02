package debug

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/riking/homeapi/marvin"
)

func init() {
	marvin.RegisterModule(NewDebugModule)
}

const Identifier = "debug"

type DebugModule struct {
	team marvin.Team
}

func NewDebugModule(t marvin.Team) marvin.Module {
	mod := &DebugModule{team: t}
	return mod
}

func (mod *DebugModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *DebugModule) Load(t marvin.Team) {
}

func (mod *DebugModule) Enable(t marvin.Team) {
	parent := marvin.NewParentCommand()
	parent.RegisterCommandFunc("panic", mod.DebugCommandPanic, "`debug panic` tests the behavior of panicking commands.")
	parent.RegisterCommandFunc("fail", mod.DebugCommandFail, "`debug fail` tests the behavior of commands returning a failure result.")
	parent.RegisterCommandFunc("error", mod.DebugCommandError, "`debug error` tests the behavior of commands returning an error.")
	parent.RegisterCommandFunc("usage", mod.DebugCommandUsage, "`debug usage` test the behavior of commands returning a usage string.")
	parent.RegisterCommandFunc("do_help", mod.DebugCommandHelp, "`debug do_help` tests the behavior of commands returning help text.")
	parent.RegisterCommandFunc("success", mod.DebugCommandSuccess, "`debug success` tests the behavior of successful commands.")

	t.RegisterCommand("debug", parent)
	t.RegisterCommandFunc("echo", mod.CommandEcho, "`echo` echos back the command arguments to the channel.")
}

func (mod *DebugModule) Disable(t marvin.Team) {
	t.UnregisterCommand("debug")
}

func (mod *DebugModule) DebugCommandPanic(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if len(args.Arguments) >= 1 {
		panic(args.Arguments[0])
	} else {
		panic(errors.Errorf("Sample panic"))
	}
}

func (mod *DebugModule) DebugCommandFail(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	return marvin.CmdFailuref(args, "Sample failure message")
}

func (mod *DebugModule) DebugCommandUsage(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	return marvin.CmdUsage(args, "Sample usage message")
}

func (mod *DebugModule) DebugCommandError(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	return marvin.CmdError(args, errors.Errorf("Sample error"), "Sample short message")
}

func (mod *DebugModule) DebugCommandHelp(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	return marvin.CmdHelpf(args, "Sample help text")
}

func (mod *DebugModule) DebugCommandSuccess(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	return marvin.CmdSuccess(args, "Sample success")
}

func (mod *DebugModule) CommandEcho(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	return marvin.CmdSuccess(args, strings.Join(args.Arguments, " ")).WithReplyType(marvin.ReplyTypeFlagOmitUsername)
}
