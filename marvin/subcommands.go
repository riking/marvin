package marvin

import (
	"fmt"
	"strings"
	"sync"
)

// CommandArguments contains the post-split arguments arrays. The pre-split
// string is not available, as the arguments can sometimes come in as an array.
type CommandArguments struct {
	Source            ActionSource
	Command           string
	Arguments         []string
	OriginalArguments []string
}

// Pop moves the first element of Arguments to Command and returns the new
// value of Command.
func (args *CommandArguments) Pop() string {
	str := args.Arguments[0]
	args.Arguments = args.Arguments[1:]
	return str
}

// PreArgs returns the slice of all arguments that have been Pop()ped.
func (args *CommandArguments) PreArgs() []string {
	return args.OriginalArguments[:len(args.OriginalArguments)-len(args.Arguments)]
}

type CommandResultCode int

const (
	CmdResultOK CommandResultCode = iota
	CmdResultFailure
	CmdResultError
	CmdResultNoSuchCommand
	CmdResultPrintUsage
	CmdResultPrintHelp
)

// A CommandResult is the return of a command. Use the Cmd* constructors to make them.
// If you want to specify where the reply will go, call WithReplyType().
//
// CommandResult objects are to be passed and modified by-value.
//
// A panicking command is converted into an Err-containing CommandResult.
type CommandResult struct {
	Args      *CommandArguments
	Message   string
	Err       error
	Code      CommandResultCode
	ReplyType ReplyType
	Sent      bool
}

// CmdError includes the Err field for the CmdResultError code.
func CmdError(args *CommandArguments, err error, msg string) CommandResult {
	return CommandResult{Args: args, Message: msg, Err: err, Code: CmdResultError}
}

// CmdFailuref formats a string to create a CmdResultFailure result.
func CmdFailuref(args *CommandArguments, format string, v ...interface{}) CommandResult {
	return CommandResult{Args: args, Message: fmt.Sprintf(format, v...), Code: CmdResultFailure}
}

// CmdHelpf formats a string to create a CmdResultPrintHelp result.
func CmdHelpf(args *CommandArguments, format string, v ...interface{}) CommandResult {
	return CommandResult{Args: args, Message: fmt.Sprintf(format, v...), Code: CmdResultPrintHelp}
}

// CmdSuccess simply takes a message to give to the user for an OK result.
func CmdSuccess(args *CommandArguments, msg string) CommandResult {
	return CommandResult{Args: args, Message: msg, Code: CmdResultOK}
}

// CmdUsage takes the usage string for a CmdResultPrintUsage result.
func CmdUsage(args *CommandArguments, usage string) CommandResult {
	return CommandResult{Args: args, Message: usage, Code: CmdResultPrintUsage}
}

// WithReplyType explicitly sets where the response should be directed.
//
// The caller can override this if desired, but it will be respected for all
// commands initiated directly by users.
func (r CommandResult) WithReplyType(rt ReplyType) CommandResult {
	r.ReplyType = rt
	return r
}

type subCommandWithHelp struct {
	f    SubCommandFunc
	help string
}

func (sc subCommandWithHelp) Handle(t Team, args *CommandArguments) CommandResult {
	return sc.f(t, args)
}

func (sc subCommandWithHelp) Help(t Team, args *CommandArguments) CommandResult {
	return CmdHelpf(args, sc.help)
}

type ParentCommand struct {
	lock    sync.Mutex
	nameMap map[string]SubCommand
}

func NewParentCommand() *ParentCommand {
	return &ParentCommand{
		nameMap: make(map[string]SubCommand),
	}
}

func (pc *ParentCommand) RegisterCommand(name string, c SubCommand) {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	pc.nameMap[name] = c
}

func (pc *ParentCommand) RegisterCommandFunc(name string, f SubCommandFunc, help string) SubCommand {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	sc := subCommandWithHelp{f: f, help: help}
	pc.nameMap[name] = sc
	return sc
}

func (pc *ParentCommand) UnregisterCommand(name string) {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	delete(pc.nameMap, name)
}

func (pc *ParentCommand) Help(t Team, args *CommandArguments) CommandResult {
	if len(args.Arguments) == 0 {
		return pc.helpListCommands(t, args)
	}
	args.Command = args.Pop()

	pc.lock.Lock()
	subC, ok := pc.nameMap[args.Command]
	pc.lock.Unlock()

	if !ok {
		cmdErr := CmdFailuref(args, "help: No such command `%s`", strings.Join(args.PreArgs(), " "))
		cmdErr.Code = CmdResultNoSuchCommand
		return cmdErr
	}
	return subC.Help(t, args)
}

func (pc *ParentCommand) helpListCommands(t Team, args *CommandArguments) CommandResult {
	var subNames []string

	for k := range pc.nameMap {
		subNames = append(subNames, k)
	}
	preArgs := args.PreArgs()
	if len(preArgs) > 1 {
		return CmdHelpf(args, "Subcommands of `%s`:\n`%s`", strings.Join(preArgs[1:], " "), strings.Join(subNames, "` `"))
	}
	return CmdHelpf(args, "Available commands:\n`%s`", strings.Join(subNames, "` `"))
}

func (pc *ParentCommand) Handle(t Team, args *CommandArguments) CommandResult {
	if len(args.Arguments) == 0 {
		return pc.helpListCommands(t, args)
	}
	args.Command = args.Pop()

	if args.Command == "help" {
		if len(args.Arguments) == 0 {
			return pc.Help(t, args)
		} else {
			args.Command = args.Pop()

			pc.lock.Lock()
			subC, ok := pc.nameMap[args.Command]
			pc.lock.Unlock()

			if !ok {
				cmdErr := CmdFailuref(args, "help: No such command '%s'", args.Command)
				cmdErr.Code = CmdResultNoSuchCommand
				return cmdErr
			}
			return subC.Help(t, args)
		}
	}

	pc.lock.Lock()
	subC, ok := pc.nameMap[args.Command]
	pc.lock.Unlock()

	if !ok {
		cmdErr := CmdFailuref(args, "No such subcommand '%s'", args.Command)
		cmdErr.Code = CmdResultNoSuchCommand
		return cmdErr
	}

	return subC.Handle(t, args)
}
