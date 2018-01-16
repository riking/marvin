package restart

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/riking/marvin"
)

var recompileSemaphore = make(chan struct{}, 1)

const Identifier = "restart"

func init() {
	marvin.RegisterModule(NewRestartModule)
	recompileSemaphore <- struct{}{}
}

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
		"`@marvin recompile [restart]`"+
			"This recompiles Marvin, pulling the latest changes.\n"+
			"With optional parameter, restarts the server after a successful compile.\n")
}

func (mod *RestartModule) Disable(t marvin.Team) {
}

func (mod *RestartModule) RecompileCommand(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if args.Source.AccessLevel() < marvin.AccessLevelController {
		return marvin.CmdFailuref(args, "This command is restricted to controllers only.")
	}

	// This will check if it can take a buffer slot, if not, it means there's a recompile in progress.
	// Otherwise it will recompile.
	select {
	case <-recompileSemaphore:
		break
	default:
		return marvin.CmdFailuref(args, "There is a recompile in progress.")
	}

	// defer reinserting the token until the recompile command is finished.
	defer func() { recompileSemaphore <- struct{}{} }()
	stdout, err := mod.Recompile()
	if err != nil {
		return marvin.CmdError(args, err, fmt.Sprintf("Failed to recompile: \n%s", stdout))
	}

	mod.team.SendMessage(mod.team.TeamConfig().LogChannel, fmt.Sprintf("Successfully recompiled: \n%s", stdout))

	if len(args.Arguments) == 1 && args.Arguments[0] == "restart" {
		go mod.Restart()
		return marvin.CmdSuccess(args, "Successfully recompiled, restarting.")
	}
	return marvin.CmdSuccess(args, "Successfully recompiled; not restarting.")
}

func (mod *RestartModule) RestartCommand(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	if args.Source.AccessLevel() < marvin.AccessLevelController {
		return marvin.CmdFailuref(args, "This command is restricted to controllers only.")
	}

	select {
	case <-recompileSemaphore:
		break
	default:
		return marvin.CmdFailuref(args, "There is a recompile in progress.")
	}

	defer func() { recompileSemaphore <- struct{}{} }()

	go mod.Restart()
	return marvin.CmdSuccess(args, "Restarting, be back soon.")
}

// Execute the shell script located in $HOME/marvin/build (with +x perms).
func (mod *RestartModule) Recompile() (string, error) {
	cmd := exec.Command(os.Getenv("HOME") + "/marvin/build.sh")
	stdout, err := cmd.CombinedOutput()
	fmt.Printf("Recompile output: \n%s", stdout)
	return string(stdout), err
}

// This sends the SIGINT signal to itself so the proper shutdown procedures can be run from the main.
func (mod *RestartModule) Restart() {
	fmt.Printf("Restarting...")
	mod.team.SendMessage(mod.team.TeamConfig().LogChannel, "Restarting...be back soon.")
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)
}
