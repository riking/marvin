package restart

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/riking/marvin"
)

func init() {
	marvin.RegisterModule(NewRestartModule)
	recompileChannel <- struct{}{}
}

const Identifier = "restart"

var recompileChannel = make(chan struct{}, 1)

type RestartModule struct {
	team marvin.Team
}

func NewRestartModule(t marvin.Team) marvin.Module {
	mod := &RestartModule{
		team: t,
	}
	return mod
}

func (mod *RestartModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *RestartModule) Load(t marvin.Team) {
}

func (mod *RestartModule) Enable(team marvin.Team) {
	team.RegisterCommandFunc("restart", mod.RestartCommand,
		"`@marvin restart`"+
			"This restarts the active Marvin instance.\n")
	team.RegisterCommandFunc("recompile", mod.RecompileCommand,
		"`@marvin recompile`"+
			"This recompiles Marvin, pulling the latest changes.\n")
}

func (mod *RestartModule) Disable(t marvin.Team) {
}

func (mod *RestartModule) RecompileCommand(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if args.Source.AccessLevel() < marvin.AccessLevelController {
		return marvin.CmdFailuref(args, "This command is restricted to controllers only.")
	}

	select {
	case <-recompileChannel:
		return mod.RecompileMarvin(args)
	default:
		return marvin.CmdFailuref(args, "There is a recompile in progress.")
	}
}

func (mod *RestartModule) RestartCommand(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if args.Source.AccessLevel() < marvin.AccessLevelController {
		return marvin.CmdFailuref(args, "This command is restricted to controllers only.")
	}

	select {
	case <-recompileChannel:
		defer func() { recompileChannel <- struct{}{} }()
	default:
		return marvin.CmdFailuref(args, "There is a recompile in progress.")
	}

	go mod.RestartMarvin()
	return marvin.CmdSuccess(args, "Restarting, be back soon.")
}

func (mod *RestartModule) RecompileMarvin(args *marvin.CommandArguments) marvin.CommandResult {
	mod.team.SendMessage(args.Source.ChannelID(), "Recompiling....")
	cmd := exec.Command(os.Getenv("HOME") + "/marvin/build.sh")
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to recompile: \n%s", stdout)
		defer func() { recompileChannel <- struct{}{} }()
		return marvin.CmdFailuref(args, fmt.Sprintf("Failed to recompile: \n%s", stdout))
	}

	mod.team.SendMessage(mod.team.TeamConfig().LogChannel, "Successfully recompiled!")
	fmt.Printf("Compile Logs: \n%s", fmt.Sprintf("%s", stdout))
	defer func() { recompileChannel <- struct{}{} }()
	if len(args.Arguments) == 1 && args.Arguments[0] == "restart" {
		go mod.RestartMarvin()
	}
	return marvin.CmdSuccess(args, "Successfully recompiled!")
}

func (mod *RestartModule) RestartMarvin() {
	fmt.Printf("Restarting...")
	mod.team.SendMessage(mod.team.TeamConfig().LogChannel, "Restarting...be back soon.")
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)
}
