package core

import "github.com/riking/homeapi/marvin"

func init() {
	marvin.RegisterModule(NewDebugModule)
}

const Identifier = "core"

type DebugModule struct {
	team marvin.Team
}

func NewDebugModule(t marvin.Team) marvin.Module {
	mod := &DebugModule{team: t}
	return mod
}

func (mod *DebugModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *DebugModule) Load(t marvin.Team) {
}

func (mod *DebugModule) Enable(t marvin.Team) {
	parent := marvin.NewParentCommand()
	_ := t.RegisterCommandFunc("whoami", mod.CommandConfig, "`config` prints out your Slack user ID.")
}

func (mod *DebugModule) Disable(t marvin.Team) {
	t.UnregisterCommand("debug")
	t.UnregisterCommand("echo")
	t.UnregisterCommand("whoami")
	t.UnregisterCommand("whereami")
}

// ---

func (mod *DebugModule) CommandConfig(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {

}