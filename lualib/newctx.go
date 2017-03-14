package lualib

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

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
	OpenIntra(g, L)
	OpenRequests(L)
	OpenTime(L)

	(*LUser)(nil).SetupMetatable(L)
	(*LChannel)(nil).SetupMetatable(L)
	L.SetGlobal("slack", LNewTeam(g))
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

type tableKeySort []lua.LValue

func (t tableKeySort) Len() int      { return len(t) }
func (t tableKeySort) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t tableKeySort) Less(i, j int) bool {
	left := t[i]
	right := t[j]
	if left == right {
		return false // equal
	}
	if left.Type() < right.Type() {
		return true
	} else if left.Type() > right.Type() {
		return false
	}
	switch left.Type() {
	case lua.LTNil:
		return false // all nils are equal
	case lua.LTBool:
		if left == lua.LFalse && right == lua.LTrue {
			return true
		}
		return false // equal or greater
	case lua.LTNumber:
		l := left.(lua.LNumber)
		r := right.(lua.LNumber)
		if l < r {
			return true
		}
		return false
	case lua.LTString:
		l := left.(lua.LString)
		r := right.(lua.LString)
		if strings.Compare(string(l), string(r)) < 0 {
			return true
		}
		return false
	default:
		// use default string representation
		l := left.String()
		r := right.String()
		if strings.Compare(l, r) < 0 {
			return true
		}
		return false
	}
}

func (g *G) lua_printTable(L *lua.LState) int {
	t := L.CheckTable(1)
	var keys []lua.LValue
	t.ForEach(func(k, v lua.LValue) {
		keys = append(keys, k)
	})
	sort.Sort(tableKeySort(keys))
	first := true
	for _, k := range keys {
		v := t.RawGet(k)
		if !first {
			fmt.Fprint(&g.PrintBuf, " | ")
		}
		valStr := lua.LVAsString(L.ToStringMeta(v))
		fmt.Fprintf(&g.PrintBuf, "%s: %s", lua.LVAsString(L.ToStringMeta(k)), valStr)
		first = false
	}
	return 0
}
