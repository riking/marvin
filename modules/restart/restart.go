package restart

import (
	"github.com/riking/marvin"
	"fmt"
	"syscall"
	"os/exec"
	"os"
)

func init() {
	marvin.RegisterModule(NewRestartModule)
}

const Identifier = "restart"

var RecompileInProgress = false
var RestartInProgress = false

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
	cmd := team.RegisterCommandFunc("restart", mod.RestartCommand,
		"`@marvin restart`"+
			"This restarts the active Marvin instance.\n")
	cmd2 := team.RegisterCommandFunc("recompile", mod.RecompileCommand,
		"`@marvin recompile`"+
			"This recompiles Marvin from git.\n")
	team.RegisterCommand("restart", cmd)
	team.RegisterCommand("recompile", cmd2)
}

func (mod *RestartModule) Disable(t marvin.Team) {
}


func (mod *RestartModule) RecompileCommand(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if args.Source.AccessLevel() < marvin.AccessLevelAdmin {
		return marvin.CmdFailuref(args, "This command is restricted to admins only.")
	}

	if RecompileInProgress {
		return marvin.CmdFailuref(args, "There is an already existing recompile in progress.")
	}

	go mod.RecompileMarvin(t)
	return marvin.CmdSuccess(args, "Sent command to recompile.")
}

func (mod *RestartModule) RestartCommand(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if args.Source.AccessLevel() < marvin.AccessLevelAdmin {
		return marvin.CmdFailuref(args, "This command is restricted to admins only.")
	}

	if RecompileInProgress {
		return marvin.CmdFailuref(args, "There is an recompile in progress.")
	}

	go mod.RestartMarvin(t)
	return marvin.CmdSuccess(args, "I am restarting now, be back soon.")
}

func (mod *RestartModule) RecompileMarvin(t marvin.Team) {
	RecompileInProgress = true
	t.SendMessage(t.TeamConfig().LogChannel, "Recompiling Marvin....")
	cmd := exec.Command("/bin/sh", os.Getenv("PWD") + "/recompile.sh")
	stdout, err := cmd.Output()
	if err != nil {
		fmt.Printf("Failed to recompile Marvin: \n%s", stdout)
		t.SendMessage(t.TeamConfig().LogChannel, fmt.Sprintf("Failed to recompile Marvin: \n%s", stdout))
		RecompileInProgress = false
		return
	}

	t.SendMessage(t.TeamConfig().LogChannel, "Successfully recompiled Marvin!")
	fmt.Printf("Compile Logs: \n%s", fmt.Sprintf("%s", stdout))
	RecompileInProgress = false
}

func (mod *RestartModule) RestartMarvin(t marvin.Team) {
	fmt.Printf("Restarting Marvin...")
	RestartInProgress = true
	t.SendMessage(t.TeamConfig().LogChannel, "Restarting Marvin...be back soon.")
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)
}