package factoid

import (
	"bytes"
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/yuin/gopher-lua"

	"github.com/riking/marvin"
	"github.com/riking/marvin/lualib"
	"github.com/riking/marvin/util"
)

type FactoidLua struct {
	mod *FactoidModule
	L   *lua.LState
	Ctx context.Context

	FactoidName string
	Args        []string
	OFlags      *OutputFlags
	ActSource   marvin.ActionSource

	printBuf bytes.Buffer
}

type ctxKeyOutputFlags struct{}

func RunFactoidLua(ctx context.Context, mod *FactoidModule, factoidName, factoidSource string, factoidArgs []string, of *OutputFlags, actSource marvin.ActionSource) (string, error) {
	var result string
	err := util.PCall(func() error {
		var err error
		result, err = runLua(ctx, mod, factoidName, factoidSource, factoidArgs, of, actSource)
		return err
	})
	return result, err
}

func runLua(ctx context.Context, mod *FactoidModule, factoidName, factoidSource string, factoidArgs []string, of *OutputFlags, actionSource marvin.ActionSource) (string, error) {
	ctx = context.WithValue(ctx, ctxKeyOutputFlags{}, of)
	g := lualib.NewLua(ctx, mod.team, actionSource)

	// Set globals
	argv := g.L.NewTable()
	for _, v := range factoidArgs {
		argv.Append(lua.LString(v))
	}
	g.L.SetGlobal("argv", argv)
	g.L.SetGlobal("args", lua.LString(strings.Join(factoidArgs, " ")))
	u, err := lualib.LNewUser(g, actionSource.UserID(), true)
	if err != nil {
		panic(err)
	}
	g.L.SetGlobal("user", u)
	g.L.SetGlobal("channel", lualib.LNewChannel(g, actionSource.ChannelID()))
	g.L.SetGlobal("_G", g.L.Get(lua.GlobalsIndex))

	// factoid global
	(*LFactoid)(nil).SetupMetatable(g.L)
	g.L.SetGlobal("factoidname", lua.LString(factoidName))
	mt := g.L.NewTable()
	mt.RawSetString("__index", g.L.NewFunction(func(L *lua.LState) int {
		L.CheckUserData(1)
		name := L.CheckString(2)
		L.Push(LNewFactoid(g, mod, name))
		return 1
	}))
	fmodUD := g.L.NewUserData()
	fmodUD.Value = nil
	fmodUD.Metatable = mt
	g.L.SetGlobal("factoid", fmodUD)

	// factoidmap data
	mt = g.L.NewTable()
	mt.RawSetString("__index", g.L.NewFunction(func(L *lua.LState) int {
		L.CheckUserData(1)
		mapName := L.CheckString(2)
		L.Push(LNewFDataMap(g, mod, "M-"+mapName))
		return 1
	}))
	fmapUD := g.L.NewUserData()
	fmapUD.Value = nil
	fmapUD.Metatable = mt
	g.L.SetGlobal("fmap", fmapUD)
	g.L.SetGlobal("fdata", LNewFDataMap(g, mod, "F-"+factoidName))

	// ------- setup done

	// Load and run
	fn, err := g.L.Load(strings.NewReader(factoidSource), "<factoid>")
	if err != nil {
		return "", ErrUser{errors.Wrap(err, "lua.compile")}
	}
	g.L.Push(fn)
	err = g.L.PCall(0, 1, nil)
	if err != nil {
		return "", ErrUser{errors.Wrap(err, "lua error")}
	}
	str := lua.LVAsString(g.L.ToStringMeta(g.L.Get(-1)))
	isStr := lua.LVCanConvToString(g.L.Get(-1))
	g.L.Pop(1)
	if str != "" && (isStr || g.PrintBuf.Len() == 0) {
		g.PrintBuf.WriteString(str)
	}
	return g.PrintBuf.String(), nil
}
