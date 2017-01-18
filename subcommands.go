package marvin

import (
	"strings"
	"sync"
)

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
	extraHelp string
	lock      sync.Mutex
	nameMap   map[string]SubCommand
}

func NewParentCommand() *ParentCommand {
	return &ParentCommand{
		nameMap: make(map[string]SubCommand),
	}
}

func (pc *ParentCommand) WithHelp(extraHelp string) *ParentCommand {
	pc.extraHelp = extraHelp
	return pc
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
		if pc.extraHelp != "" {
			return CmdHelpf(args, "%s\nSubcommands: `%s`", pc.extraHelp, strings.Join(subNames, "` `"))
		}
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
