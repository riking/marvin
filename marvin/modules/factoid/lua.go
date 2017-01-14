package factoid

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/yuin/gopher-lua"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/modules/factoid/lualib"
	"github.com/riking/homeapi/marvin/modules/paste"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
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
	L := lua.NewState(lua.Options{
		IncludeGoStackTrace: true,
		SkipOpenLibs:        true,
	})
	L.Ctx = ctx
	fl := &FactoidLua{
		mod:         mod,
		L:           L,
		Ctx:         ctx,
		FactoidName: factoidName,
		Args:        factoidArgs,
		OFlags:      of,
		ActSource:   actionSource,
	}
	fl.Setup()
	fl.SetFactoidEnv()

	fn, err := L.Load(strings.NewReader(factoidSource), "<factoid>")
	if err != nil {
		return "", ErrUser{errors.Wrap(err, "lua.compile")}
	}
	L.Push(fn)
	err = L.PCall(0, 1, nil)
	if err != nil {
		return "", ErrUser{errors.Wrap(err, "lua error")}
	}
	str := lua.LVAsString(L.ToStringMeta(L.Get(-1)))
	isStr := lua.LVCanConvToString(L.Get(-1))
	L.Pop(1)
	if str != "" && (isStr || fl.printBuf.Len() == 0) {
		fl.printBuf.WriteString(str)
	}
	//util.LogDebug("Lua result:", "source:", ("[" + factoidSource + "]"), "result:", ("[" + fl.printBuf.String() + "]"))
	return fl.printBuf.String(), nil
}

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

func (f *FactoidLua) luaPrint(L *lua.LState) int {
	top := L.GetTop()
	for i := 1; i <= top; i++ {
		f.printBuf.WriteString(L.ToStringMeta(L.Get(i)).String())
		if i != top {
			f.printBuf.WriteByte(' ')
		}
	}
	return 0
}

func (f *FactoidLua) OpenFactoid(L *lua.LState) int {
	tab := L.NewTypeMetatable("_metatable_FactoidModule")
	tab.RawSetString("__index", L.NewFunction(luaFactoidModule__index))
	u := L.NewUserData()
	u.Value = f
	u.Metatable = tab
	L.SetGlobal("factoid", u)

	L.SetGlobal("factoidname", lua.LString(f.FactoidName))

	return 0
}

func luaFactoidModule__index(L *lua.LState) int {
	if L.GetTop() != 2 {
		L.RaiseError("factoidmodule.__index needs two arguments, got %d", L.GetTop())
	}
	fl, ok := L.CheckUserData(1).Value.(*FactoidLua)
	if !ok {
		L.RaiseError("bad self for factoidmodule.__index")
	}
	key := L.CheckString(2)
	L.Push(LNewFactoid(fl, key))
	return 1
}

func (f *FactoidLua) Setup() {
	L := f.L
	lua.OpenBase(L)
	basemod := L.CheckTable(1)
	basemod.RawSetString("print", L.NewFunction(f.luaPrint))
	for k, v := range forbiddenBase {
		basemod.RawSetString(k, L.NewFunction(v))
	}
	lua.OpenTable(L)
	lua.OpenString(L)
	lua.OpenMath(L)
	lua.OpenDebug(L)
	lualib.OpenBit(L)
	lualib.OpenJson(L)
	lualib.OpenRequests(L)
	lualib.OpenCorpus(L)

	LFactoid{}.SetupMetatable(L)
	LUser{}.SetupMetatable(L)
	LChannel{}.SetupMetatable(L)
	LFDataMap{}.SetupMetatable(L)

	f.OpenFactoid(L)
	f.OpenFData(L)
	f.OpenBot(L)
	L.SetGlobal("ptable", f.L.NewFunction(f.lua_printTable))
}

func (f *FactoidLua) SetFactoidEnv() {
	argv := f.L.NewTable()
	for _, v := range f.Args {
		argv.Append(lua.LString(v))
	}
	f.L.SetGlobal("argv", argv)
	f.L.SetGlobal("args", lua.LString(strings.Join(f.Args, " ")))

	u, err := LNewUser(f, f.ActSource.UserID(), true)
	if err != nil {
		panic(err)
	}
	f.L.SetGlobal("user", u)
	c := LNewChannel(f, f.ActSource.ChannelID())
	f.L.SetGlobal("channel", c)
	f.L.SetGlobal("_G", f.L.Get(lua.GlobalsIndex))
}

func (f *FactoidLua) OpenBot(L *lua.LState) int {
	tab := L.NewTable()
	tab.RawSetString("paste", L.NewFunction(f.mod.LuaPaste))
	tab.RawSetString("shortlink", L.NewFunction(f.mod.LuaShortLink))
	tab.RawSetString("now", L.NewFunction(lua_Now))
	tab.RawSetString("uriencode", L.NewFunction(lua_URIEncode))
	tab.RawSetString("uridecode", L.NewFunction(lua_URIDecode))
	tab.RawSetString("unescape", L.NewFunction(lua_SlackUnescape))
	L.SetGlobal("bot", tab)
	return 0
}

func (mod *FactoidModule) LuaPaste(L *lua.LState) int {
	if mod.pasteMod == nil {
		L.RaiseError("paste module not available")
	}
	if L.GetTop() != 1 {
		L.RaiseError("paste() takes one argument: content")
	}
	str := L.CheckString(1)
	id, err := mod.pasteMod.(paste.API).CreatePaste(str)
	if err != nil {
		L.RaiseError("paste() failed: %s", err)
	}
	pasteURL := mod.pasteMod.(paste.API).URLForPaste(id)
	L.Push(lua.LString(pasteURL))
	return 1
}

func (mod *FactoidModule) LuaShortLink(L *lua.LState) int {
	if mod.pasteMod == nil {
		L.RaiseError("paste module not available")
	}
	if L.GetTop() != 1 {
		L.RaiseError("shortlink() takes one argument: content")
	}
	str := L.CheckString(1)
	id, err := mod.pasteMod.(paste.API).CreateLink(str)
	if err != nil {
		L.RaiseError("shortlink() failed: %s", err)
	}
	linkURL := mod.pasteMod.(paste.API).URLForLink(id)
	L.Push(lua.LString(linkURL))
	return 1
}

func lua_Now(L *lua.LState) int {
	L.Push(lua.LNumber(time.Now().Unix()))
	return 1
}

func lua_URIEncode(L *lua.LState) int {
	str := L.CheckString(1)
	L.Push(lua.LString(url.QueryEscape(str)))
	return 1
}

func lua_URIDecode(L *lua.LState) int {
	str := L.CheckString(1)
	result, err := url.QueryUnescape(str)
	if err != nil {
		L.RaiseError("non-encoded string passed to uridecode: %s", err)
	}
	L.Push(lua.LString(result))
	return 1
}

func lua_SlackUnescape(L *lua.LState) int {
	str := L.CheckString(1)
	L.Push(lua.LString(slack.UnescapeTextAll(str)))
	return 1
}

func (f *FactoidLua) lua_printTable(L *lua.LState) int {
	t := L.CheckTable(1)
	first := true
	t.ForEach(func(k, v lua.LValue) {
		if !first {
			fmt.Fprint(&f.printBuf, " | ")
		}
		fmt.Fprintf(&f.printBuf, "%s: %s", lua.LVAsString(L.ToStringMeta(k)), lua.LVAsString(L.ToStringMeta(v)))
		first = false
	})
	return 0
}
