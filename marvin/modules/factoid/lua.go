package factoid

import (
	"bytes"
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/util"
	"github.com/yuin/gopher-lua"
)

type FactoidLua struct {
	mod *FactoidModule
	L   *lua.LState

	printBuf bytes.Buffer
}

func RunLua(ctx context.Context, mod *FactoidModule, factoidSource string, factoidArgs []string, source marvin.ActionSource) (string, error) {
	var result string
	err := util.PCall(func() error {
		var err error
		result, err = runLua(ctx, mod, factoidSource, factoidArgs, source)
		return err
	})
	return result, err
}

func runLua(ctx context.Context, mod *FactoidModule, factoidSource string, factoidArgs []string, source marvin.ActionSource) (string, error) {
	L := lua.NewState(lua.Options{
		SkipOpenLibs: true,
	})
	L.Ctx = ctx
	fl := &FactoidLua{
		mod: mod,
		L:   L,
	}
	fl.Setup()

	fn, err := L.Load(strings.NewReader(factoidSource), "<factoid>")
	if err != nil {
		return errors.Wrap(err, "lua.compile")
	}
	L.Push(lua.LFalse)
	err = L.PCall(0, 1, fn)
	if err != nil {
		return errors.Wrap(err, "lua.run")
	}
	lv := L.Get(1)
	if ls, ok := lv.(*lua.LString); ok {
		fl.printBuf.WriteString(string(ls))
	}
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
			f.printBuf.WriteByte(" ")
		}
	}
}

func (f *FactoidLua) Setup() {
	lua.OpenBase(f.L)
	basemod := f.L.CheckTable(1)
	basemod.RawSetString("print", f.luaPrint)
	for k, v := range forbiddenBase {
		basemod.RawSetString(k, v)
	}
	lua.OpenTable(f.L)
	lua.OpenString(f.L)
	lua.OpenMath(f.L)
	lua.OpenDebug(f.L)

	// TODO OpenFactoid etc
}
