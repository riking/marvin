package marvin

import (
	"fmt"
	"strings"
	"sync"

	"github.com/riking/homeapi/marvin/slack"
)

type ReplyType int

const (
	ReplyTypePM ReplyType = 1 << iota
	ReplyTypeInChannel
	ReplyTypeLog
)
const (
	ReplyTypePreferChannel = ReplyTypeInChannel | ReplyTypePM
	ReplyTypeShortProblem  = ReplyTypeInChannel | ReplyTypeLog
	ReplyTypeLongProblem   = ReplyTypePM | ReplyTypeLog
	ReplyTypeAny           = ReplyTypeInChannel | ReplyTypePM | ReplyTypeLog
)

const LongReplyThreshold = 400

type ActionSource interface {
	UserID() slack.UserID
	ChannelID() slack.ChannelID
	ArchiveLink(t Team) string

	SendCmdReply(t Team, typ ReplyType, result CommandResult)
}

type ActionSourceUserMessage struct {
	Msg slack.RTMRawMessage
}

func (um ActionSourceUserMessage) UserID() slack.UserID       { return um.Msg.UserID() }
func (um ActionSourceUserMessage) ChannelID() slack.ChannelID { return um.Msg.ChannelID() }
func (um ActionSourceUserMessage) ArchiveLink(t Team) string  { return t.ArchiveURL(um.Msg.MessageID()) }

func (um ActionSourceUserMessage) SendCmdReply(t Team, typ ReplyType, result CommandResult) {
	logChannel := t.TeamConfig().LogChannel
	imChannel, _ := t.GetIM(um.UserID())

	replyChannel := typ&ReplyTypeInChannel != 0
	replyIM := typ&ReplyTypePM != 0
	replyLog := typ&ReplyTypeLog != 0

	switch result.Code {
	case CmdResultOK:
	case CmdResultFailure:
		if result.Message == "" {
			return
		}
		// Prefer Channel > PM > Log
		if replyChannel {
			channelMsg := result.Message
			if len(result.Message) > LongReplyThreshold {
				channelMsg = "This reply has been truncated. The full message is in your PMs.\n" + result.Message[:100] + "...\n"
				replyIM = true
			} else {
				replyIM = false
			}
			t.SendMessage(um.Msg.ChannelID(), channelMsg)
		}
		if replyIM {
			t.SendMessage(imChannel, fmt.Sprintf("%s\n%s", result.Message, um.ArchiveLink(t)))
		}
		if replyLog {
			t.SendMessage(logChannel, fmt.Sprintf("%s\n%s", result.Message, um.ArchiveLink(t)))
		}
	case CmdResultError:
		// Print terse in channel, detail in PM, full in log
		if replyChannel {
			if result.Message == "" {
				result.Message = "An error occurred."
			}
			t.SendMessage(um.Msg.ChannelID(), result.Message)
		}
		if replyLog {
			t.SendMessage(logChannel, fmt.Sprintf("%s\n```\n%+v\n```", um.ArchiveLink(t), result.Error))
		}
	case CmdResultNoSuchCommand:
		if replyChannel {
			t.ReactMessage(um.Msg.MessageID(), "question")
		}
		if replyIM {
			t.SendMessage(imChannel, fmt.Sprintf("I didn't quite understand that, sorry.\nYou said: [%s]",
				strings.Join(result.Args.OriginalArguments, "] [")))
		}
		if replyLog {
			t.SendMessage(logChannel, fmt.Sprintf("No such command by %v\nArgs: [%s]\nLink: %s",
				um.UserID(),
				strings.Join(result.Args.OriginalArguments, "] ["),
				um.ArchiveLink(t)))
		}
	}
}

type CommandArguments struct {
	//Msg       slack.RTMRawMessage
	Source            ActionSource
	Command           string
	Arguments         []string
	OriginalArguments []string
}

func (ca *CommandArguments) Pop() string {
	str := ca.Arguments[0]
	ca.Arguments = ca.Arguments[1:]
	return str
}

const (
	CmdResultOK = iota
	CmdResultFailure
	CmdResultError
	CmdResultNoSuchCommand
)

type CommandResult struct {
	Args    *CommandArguments
	Message string
	Err     error
	Code    int
}

func CmdError(args *CommandArguments, err error, msg string) CommandResult {
	return CommandResult{Args: args, Message: msg, Err: err, Code: CmdResultError}
}

func CmdFailuref(args *CommandArguments, format string, v ...interface{}) CommandResult {
	return CommandResult{Args: args, Message: fmt.Sprintf(format, v...), Code: CmdResultFailure}
}

func CmdSuccess(args *CommandArguments, msg string) CommandResult {
	return CommandResult{Args: args, Message: msg, Code: CmdResultOK}
}

func (r CommandResult) Error() string {
	return r.Message
}

type ModuleID string

type ParentCommand struct {
	lock    sync.Mutex
	nameMap map[string]SubCommand
}

func NewParentCommand() ParentCommand {
	return ParentCommand{
		nameMap: make(map[string]SubCommand),
	}
}

func (pc *ParentCommand) RegisterCommand(name string, c SubCommand) {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	pc.nameMap[name] = c
}

func (pc *ParentCommand) UnregisterCommand(name string, c SubCommand) {
	pc.lock.Lock()
	defer pc.lock.Unlock()

	delete(pc.nameMap, name)
}

func (pc *ParentCommand) Help(t Team, args *CommandArguments) error {
	// TODO
	return nil
}

func (pc *ParentCommand) Handle(t Team, args *CommandArguments) error {
	if len(args.Arguments) == 0 {
		return pc.Help(t, args)
	}
	args.Command = args.Arguments[0]
	args.Arguments = args.Arguments[1:]

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
