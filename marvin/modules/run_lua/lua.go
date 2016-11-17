package lua

import "github.com/riking/homeapi/marvin"

type API interface {

}

// ---

func init() {
	marvin.RegisterModule(NewLuaScriptModule)
}

const Identifier = "lua"

type LuaScriptModule struct {
	team marvin.Team
}

func NewLuaScriptModule(t marvin.Team) marvin.Module {
	mod := &LuaScriptModule{team: t}
	return mod
}

func (mod *LuaScriptModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *LuaScriptModule) Load(t marvin.Team) {
}

func (mod *LuaScriptModule) Enable(t marvin.Team) {
}

func (mod *LuaScriptModule) Disable(t marvin.Team) {
}

