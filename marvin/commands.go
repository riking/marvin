package marvin

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/riking/homeapi/marvin/slack"
)

// CommandArguments contains the post-split arguments arrays. The pre-split
// string is not available, as the arguments can sometimes come in as an array.
type CommandArguments struct {
	Source            ActionSource
	Command           string
	Arguments         []string
	OriginalArguments []string

	IsEdit         bool
	IsUndo         bool
	PreviousResult *CommandResult
	ModuleData     interface{}
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

// SetModuleData is useful for editing.
func (args *CommandArguments) SetModuleData(v interface{}) {
	args.ModuleData = v
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

type TriValue int

const (
	TriNo      = -1
	TriDefault = 0
	TriYes     = 1
)

const UndoSimple = 2
const UndoCustom = TriYes

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

	ExtraMessages []SupplementalMessage

	CanEdit TriValue
	CanUndo TriValue
}

type SupplementalMessage struct {
	slack.ChannelID
	Message string

	slack.MessageTS
}

// CmdError includes the Err field for the CmdResultError code.
// An error is something that shouldn't normally happen - access violations go under Failure.
func CmdError(args *CommandArguments, err error, msg string) CommandResult {
	return CommandResult{Args: args, Message: msg, Err: errors.Wrap(err, msg), Code: CmdResultError}
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

func (r CommandResult) WithEdit() CommandResult {
	r.CanEdit = TriYes
	return r
}

func (r CommandResult) WithNoEdit() CommandResult {
	r.CanEdit = TriNo
	return r
}

func (r CommandResult) WithCustomUndo() CommandResult {
	r.CanUndo = UndoCustom
	return r
}

func (r CommandResult) WithSimpleUndo() CommandResult {
	r.CanUndo = UndoSimple
	return r
}

func (r CommandResult) WithNoUndo() CommandResult {
	r.CanUndo = TriNo
	return r
}

// WithReplyType explicitly sets where the response should be directed.
//
// The caller can override this if desired, but it will be respected for all
// commands initiated directly by users.
func (r CommandResult) WithReplyType(rt ReplyType) CommandResult {
	r.ReplyType = rt
	return r
}
