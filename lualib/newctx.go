package lualib

import (
	"bytes"
	"context"
	"fmt"

	"github.com/riking/marvin"
	"github.com/yuin/gopher-lua"
)

type G struct {
	team     marvin.Team
	L        *lua.LState
	Ctx      context.Context
	PrintBuf bytes.Buffer
	actS     marvin.ActionSource
}

func NewLua(ctx context.Context, team marvin.Team, actionSource marvin.ActionSource) *G {
	L := lua.NewState(lua.Options{
		IncludeGoStackTrace: true,
		SkipOpenLibs:        true,
	})
	L.Ctx = ctx
	g := &G{
		L:    L,
		Ctx:  ctx,
		team: team,
		actS: actionSource,
	}

	return g
}

func (g *G) Team() marvin.Team                 { return g.team }
func (g *G) ActionSource() marvin.ActionSource { return g.actS }

func luaForbidden(name string) func(L *lua.LState) int {
	return func(L *lua.LState) int {
		L.RaiseError("%s is a forbidden function", name)
		return 0
	}
}

var forbiddenBase = map[string]lua.LGFunction{
	"collectgarbage": luaForbidden("collectgarbage"),
	"dofile":         luaForbidden("dofile"),
	"loadfile":       luaForbidden("loadfile"),
	"_printregs":     luaForbidden("_printregs"),
}

func (g *G) OpenLibraries() {
	L := g.L
	lua.OpenBase(L)
	basemod := L.CheckTable(1)
	basemod.RawSetString("print", L.NewFunction(g.lua_print))
	basemod.RawSetString("ptable", L.NewFunction(g.lua_printTable))
	for k, v := range forbiddenBase {
		basemod.RawSetString(k, L.NewFunction(v))
	}
	lua.OpenTable(L)
	lua.OpenString(L)
	lua.OpenMath(L)
	lua.OpenDebug(L)

	OpenBit(L)
	OpenBot(g.team)(L)
	OpenCorpus(L)
	OpenJson(L)
	OpenRequests(L)

	(*LUser)(nil).SetupMetatable(L)
	(*LChannel)(nil).SetupMetatable(L)
}

func (g *G) lua_print(L *lua.LState) int {
	top := L.GetTop()
	for i := 1; i <= top; i++ {
		g.PrintBuf.WriteString(L.ToStringMeta(L.Get(i)).String())
		if i != top {
			g.PrintBuf.WriteByte(' ')
		}
	}
	return 0
}

func (g *G) lua_printTable(L *lua.LState) int {
	t := L.CheckTable(1)
	first := true
	t.ForEach(func(k, v lua.LValue) {
		if !first {
			fmt.Fprint(&g.PrintBuf, " | ")
		}
		valStr := lua.LVAsString(L.ToStringMeta(v))
		fmt.Fprintf(&g.PrintBuf, "%s: %s", lua.LVAsString(L.ToStringMeta(k)), valStr)
		first = false
	})
	return 0
}
