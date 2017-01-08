package debug

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/modules/paste"
	"github.com/riking/homeapi/marvin/slack"
)

func init() {
	marvin.RegisterModule(NewDebugModule)
}

const Identifier = "debug"

type DebugModule struct {
	team marvin.Team

	pasteModule marvin.Module
}

func NewDebugModule(t marvin.Team) marvin.Module {
	mod := &DebugModule{team: t}
	return mod
}

func (mod *DebugModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *DebugModule) Load(t marvin.Team) {
	t.DependModule(mod, paste.Identifier, &mod.pasteModule)
}

func (mod *DebugModule) Enable(t marvin.Team) {
	if mod.pasteModule != nil {
		mod.pasteModule.(paste.API).Identifier()
	}

	parent := marvin.NewParentCommand()
	parent.RegisterCommandFunc("panic", mod.DebugCommandPanic, "`debug panic` tests the behavior of panicking commands.")
	parent.RegisterCommandFunc("fail", mod.DebugCommandFail, "`debug fail` tests the behavior of commands returning a failure result.")
	parent.RegisterCommandFunc("error", mod.DebugCommandError, "`debug error` tests the behavior of commands returning an error.")
	parent.RegisterCommandFunc("usage", mod.DebugCommandUsage, "`debug usage` test the behavior of commands returning a usage string.")
	parent.RegisterCommandFunc("do_help", mod.DebugCommandHelp, "`debug do_help` tests the behavior of commands returning help text.")
	parent.RegisterCommandFunc("success", mod.DebugCommandSuccess, "`debug success` tests the behavior of successful commands.")
	parent.RegisterCommandFunc("paste", mod.DebugCommandPaste, "`debug paste` tests the paste module.")

	whoami := parent.RegisterCommandFunc("whoami", mod.CommandWhoAmI, "`debug whoami [@user]` prints out your Slack user ID.")
	whereami := parent.RegisterCommandFunc("whereami", mod.CommandWhereAmI, "`debug whereami` prints out the current channel ID.")

	t.RegisterCommand("debug", parent)
	t.RegisterCommandFunc("echo", mod.CommandEcho, "`echo` echos back the command arguments to the channel.")
	t.RegisterCommand("whoami", whoami)
	t.RegisterCommand("whereami", whereami)
}

func (mod *DebugModule) Disable(t marvin.Team) {
	t.UnregisterCommand("debug")
	t.UnregisterCommand("echo")
	t.UnregisterCommand("whoami")
	t.UnregisterCommand("whereami")
}

func (mod *DebugModule) DebugCommandPanic(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if len(args.Arguments) >= 1 {
		panic(args.Arguments[0])
	} else {
		panic(errors.Errorf("Sample panic"))
	}
}

func (mod *DebugModule) DebugCommandFail(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	return marvin.CmdFailuref(args, "Sample failure message").WithEdit()
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
	return marvin.CmdSuccess(args, "Sample success").WithEdit()
}

func (mod *DebugModule) DebugCommandPaste(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	content := strings.Join(args.Arguments, " ")
	id, err := mod.pasteModule.(paste.API).CreatePaste(content)
	if err != nil {
		return marvin.CmdError(args, err, "creating paste")
	}
	return marvin.CmdSuccess(args, mod.pasteModule.(paste.API).URLForPaste(id))
}

func (mod *DebugModule) CommandEcho(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if args.Source.AccessLevel() < marvin.AccessLevelAdmin {
		return marvin.CmdFailuref(args, "This command is restricted to admins.")
	}
	return marvin.CmdSuccess(args, strings.Join(args.Arguments, " ")).WithReplyType(marvin.ReplyTypeFlagOmitUsername).WithEdit()
}

func (mod *DebugModule) CommandWhoAmI(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	var uid slack.UserID

	uid = args.Source.UserID()
	if len(args.Arguments) > 0 {
		uid = slack.ParseUserMention(args.Arguments[0])
		if uid == "" {
			return marvin.CmdFailuref(args, "'%s' is not a valid @mention", args.Arguments[0])
		}
		return marvin.CmdSuccess(args, fmt.Sprintf("%v's user ID is %s", uid, string(uid)))
	}
	return marvin.CmdSuccess(args, fmt.Sprintf("%v, your user ID is %s", uid, string(uid))).WithReplyType(marvin.ReplyTypeFlagOmitUsername)
}

func (mod *DebugModule) CommandWhereAmI(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	var cid slack.ChannelID

	cid = args.Source.ChannelID()
	//if len(args.Arguments) > 0 {
	//	uid = slack.ParseUserMention(args.Arguments[0])
	//	if uid == "" {
	//		uid = args.Source.UserID()
	//	}
	//}
	return marvin.CmdSuccess(args,
		fmt.Sprintf("You sent that command in channel ID %s.", string(cid)),
	)
}
