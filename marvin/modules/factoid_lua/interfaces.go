package factoid_lua

import (
	"github.com/riking/homeapi/marvin"
	"github.com/yuin/gopher-lua"
)

type LuaProvider interface {
	Setup(L *lua.LState)
}

var providers map[marvin.ModuleID]LuaProvider

type FactoidInterpreter interface {
}
