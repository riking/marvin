package marvin

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/riking/homeapi/marvin/slack"
)

type ActionSource interface {
	UserID() slack.UserID
	ChannelID() slack.ChannelID
	MessageTS() slack.MessageTS
}

type ActionSourceUserMessage struct {
	Msg slack.RTMRawMessage
}

func (um ActionSourceUserMessage) UserID() slack.UserID       { return um.Msg.UserID() }
func (um ActionSourceUserMessage) ChannelID() slack.ChannelID { return um.Msg.ChannelID() }
func (um ActionSourceUserMessage) MessageTS() slack.MessageTS { return um.Msg.MessageTS() }

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
	CmdErrGeneric = iota
	CmdErrNoSuchCommand
)

type CommandError struct {
	Args    *CommandArguments
	Message string
	Code    int
	Success bool
}

func CmdErrorf(args *CommandArguments, format string, v ...interface{}) CommandError {
	return CommandError{Args: args, Message: fmt.Sprintf(format, v...), Success: false}
}

func CmdSuccess(args *CommandArguments, format string, v ...interface{}) CommandError {
	return CommandError{Args: args, Message: fmt.Sprintf(format, v...), Success: true}
}

func (e CommandError) Error() string {
	return e.Message
}

func (e CommandError) SendReply(t Team) error {
	if e.Success && e.Message == "" {
		return nil
	}
	if e.Code == CmdErrNoSuchCommand {
		_, _, err := t.SendMessage(e.Args.Source.ChannelID(), "I'm not quite sure what you meant by that.")
		return err
	}

	imChannel, err := t.GetIM(e.Args.Source.UserID())
	if err != nil {
		return err
	}
	if !e.Success {
		_, _, err = t.SendMessage(imChannel,
			fmt.Sprintf("Your command failed. %s\n%s",
				t.ArchiveURL(e.Args.Source.ChannelID(), e.Args.Source.MessageTS()),
				e.Message,
			))
	} else if e.Message != "" {
		_, _, err = t.SendMessage(imChannel, e.Message)
	}
	return err
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
		cmdErr := CmdErrorf(args, "No such subcommand '%s'", args.Command)
		cmdErr.Code = CmdErrNoSuchCommand
		return cmdErr
	}

	err := subC.Handle(t, args)
	if err, ok := err.(CommandError); ok {
		if err.Success {
			return err
		} else if err.Code == CmdErrNoSuchCommand {
			err.Code = CmdErrGeneric
		}
	}
	return errors.Wrap(err, args.Command)
}
