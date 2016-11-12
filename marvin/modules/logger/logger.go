package logger

import "github.com/riking/homeapi/marvin"

func init() {
	marvin.RegisterModule(NewDebugModule)
}

const Identifier = "debug"

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
}

func (mod *DebugModule) Disable(t marvin.Team) {
}
