package marvin

import (
	"fmt"
	"strings"

	"sync"

	"github.com/pkg/errors"
	"github.com/riking/homeapi/marvin/slack"
)

type ActionSource interface {
	UserID() slack.UserID
	ChannelID() slack.ChannelID
}

type ActionSourceUserMessage struct {
	Msg slack.RTMRawMessage
}

func (um ActionSourceUserMessage) UserID() slack.UserID       { return um.Msg.UserID() }
func (um ActionSourceUserMessage) ChannelID() slack.ChannelID { return um.Msg.ChannelID() }

type CommandArguments struct {
	//Msg       slack.RTMRawMessage
	Source            ActionSource
	Command           string
	Arguments         []string
	OriginalArguments []string
}

type CommandError struct {
	Args    *CommandArguments
	Message string
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
	imChannel, err := t.GetIM(e.Args.Source.UserID())
	if err != nil {
		return err
	}
	_, err = t.SendMessage(imChannel,
		fmt.Sprintf("Your command `%s` failed.\n%s",
			strings.Join(e.Args.Arguments, " "),
			e.Message,
		))
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

func (pc *ParentCommand) DispatchCommand(t Team, args *CommandArguments) error {
	if len(args.Arguments) == 0 {
		return pc.Help(t, args)
	}
	args.Command = args.Arguments[0]
	args.Arguments = args.Arguments[1:]

	pc.lock.Lock()
	subC, ok := pc.nameMap[args.Command]
	pc.lock.Unlock()

	if !ok {
		return CmdErrorf(args, "no such command %s", args.Command)
	}

	err := subC.Handle(t, args)
	if err, ok := err.(CommandError); ok && err.Success {
		return err
	}
	return errors.Wrap(err, args.Command)
}
