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
}

const Identifier = "restart"

var RecompileInProgress = false

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
		return marvin.CmdFailuref(args, "This command is restricted to admins only.")
	}

	if RecompileInProgress {
		return marvin.CmdFailuref(args, "There is an already existing recompile in progress.")
	}

	return mod.RecompileMarvin(args)
}

func (mod *RestartModule) RestartCommand(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if args.Source.AccessLevel() < marvin.AccessLevelController {
		return marvin.CmdFailuref(args, "This command is restricted to admins only.")
	}

	if RecompileInProgress {
		return marvin.CmdFailuref(args, "There is an recompile in progress.")
	}

	go mod.RestartMarvin()
	return marvin.CmdSuccess(args, "I am restarting now, be back soon.")
}

func (mod *RestartModule) RecompileMarvin(args *marvin.CommandArguments) marvin.CommandResult {
	RecompileInProgress = true
	mod.team.SendMessage(args.Source.ChannelID(), "Recompiling Marvin....")
	cmd := exec.Command("/bin/sh", os.Getenv("HOME")+"/marvin/build.sh")
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to recompile Marvin: \n%s", stdout)
		RecompileInProgress = false
		return marvin.CmdFailuref(args, fmt.Sprintf("Failed to recompile Marvin: \n%s", stdout))
	}

	mod.team.SendMessage(mod.team.TeamConfig().LogChannel, "Successfully recompiled Marvin!")
	fmt.Printf("Compile Logs: \n%s", fmt.Sprintf("%s", stdout))
	RecompileInProgress = false
	return marvin.CmdSuccess(args, "Successfully recompiled marvin.")
}

func (mod *RestartModule) RestartMarvin() {
	fmt.Printf("Restarting Marvin...")
	mod.team.SendMessage(mod.team.TeamConfig().LogChannel, "Restarting Marvin...be back soon.")
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)
}
