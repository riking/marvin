package lualib

import (
	"strings"

	"github.com/yuin/gopher-lua"
)

const funclibMapName = "funclib"

func OpenFuncs(L *lua.LState) int {
	mt := L.NewTable()
	mt.RawSetString("__index", L.NewFunction(luaFuncs__index))
	mt.RawSetString("__newindex", L.NewFunction(luaFuncs__newindex))
	ud := L.NewTable()
	ud.Metatable = mt
	L.SetGlobal("funcs", ud)
	return 0
}

func luaFuncs__index(L *lua.LState) int {
	fmapg := L.GetGlobal("fmap")
	fmap := L.GetTable(fmapg, lua.LString(funclibMapName))
	name := L.CheckString(2)
	value := L.GetTable(fmap, lua.LString(name))
	if value == lua.LNil {
		return 0
	}
	if value.Type() != lua.LTString {
		return 0
	}
	f, err := L.Load(strings.NewReader(string(value.(lua.LString))), name)
	if err != nil {
		L.Push(nil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	L.Push(f)
	L.Call(0, 1)
	return 1
}

func luaFuncs__newindex(L *lua.LState) int {
	fmapg := L.GetGlobal("fmap")
	fmap := L.GetTable(fmapg, lua.LString(funclibMapName))
	name := L.CheckString(2)
	code := L.CheckString(3)
	L.SetTable(fmap, lua.LString(name), lua.LString(code))
	return 0
}
