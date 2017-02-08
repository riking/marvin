package factoid

import (
	"github.com/yuin/gopher-lua"

	"github.com/riking/marvin/lualib"
	"github.com/riking/marvin/slack"
)

// LFactoid has the following API, where `factoid` is the global of the same name:
//
//   f = factoid.name
//   factoid.test(...args...) -> String, result of running !test ...args...
//   factoid.test.src -> String
//   factoid.test.exists -> Bool
//   factoid.test.author -> LUser
//   factoid.test.islocal -> Bool
//   factoid.test.time -> UnixSecs
//   factoid.test.created -> Slack archive link
//   factoid.test.history -> List<
//   factoid.test.data -> String
//   factoid.test.data = v
//
// If f.exists is false, most other methods will error.
type LFactoid struct {
	g    *lualib.G
	mod  *FactoidModule
	Name string
}

const metatableLFactoid = "_metatable_LFactoid"

func (*LFactoid) SetupMetatable(L *lua.LState) {
	tab := L.NewTypeMetatable(metatableLFactoid)
	tab.RawSetString("__index", L.NewFunction(luaFactoid__Index))
	tab.RawSetString("__call", L.NewFunction(luaFactoid__Call))
}

func LNewFactoid(g *lualib.G, mod *FactoidModule, name string) lua.LValue {
	v := &LFactoid{g: g, mod: mod, Name: name}
	u := g.L.NewUserData()
	u.Value = v
	u.Metatable = g.L.GetTypeMetatable(metatableLFactoid)
	return u
}

// ---

func luaFactoid__Call(L *lua.LState) int {
	lfv, ok := L.CheckUserData(1).Value.(*LFactoid)
	if !ok {
		L.RaiseError("factoid__call() with wrong type for self")
	}

	var args = []string{lfv.Name}
	for i := 1 + 1; i <= L.GetTop(); i++ {
		args = append(args, lua.LVAsString(L.ToStringMeta(L.Get(i))))
	}

	var parentOF *OutputFlags
	parentOF = L.Ctx.Value(ctxKeyOutputFlags{}).(*OutputFlags)
	var of OutputFlags

	result, err := lfv.mod.RunFactoid(L.Ctx, args, &of, lfv.g.ActionSource())
	if err != nil {
		L.RaiseError("factoid error: %s", err)
	}

	if of.SideEffects {
		parentOF.SideEffects = true
	}
	if of.NoReply {
		L.Push(lua.LString(""))
		return 1
	}
	L.Push(lua.LString(result))
	return 1
}

func luaFactoid__Index(L *lua.LState) int {
	if L.GetTop() != 2 {
		L.RaiseError("__index() requires 2 arguments")
		return 0
	}
	lfv, ok := L.CheckUserData(1).Value.(*LFactoid)
	if !ok {
		L.RaiseError("factoid__index() with wrong type for self")
		return 0
	}
	method := L.CheckString(2)

	switch method {
	case "exists":
		_, err := lfv.mod.GetFactoidBare(lfv.Name, lfv.g.ActionSource().ChannelID())
		if err == ErrNoSuchFactoid {
			L.Push(lua.LFalse)
		} else {
			L.Push(lua.LTrue)
		}
		return 1
	case "src", "raw", "author":
		finfo, err := lfv.mod.GetFactoidBare(lfv.Name, lfv.g.ActionSource().ChannelID())
		if err == ErrNoSuchFactoid {
			L.RaiseError("No such factoid %s", lfv.Name)
		} else if err != nil {
			L.RaiseError("err %s.src: %s")
		}

		if method == "src" || method == "raw" {
			L.Push(lua.LString(finfo.RawSource))
		} else if method == "author" {
			u, err := lualib.LNewUser(lfv.g, finfo.LastUser, true)
			if err != nil {
				L.RaiseError("could not load user data for %v", finfo.LastUser)
			}
			L.Push(u)
		}
		return 1
	case "locked", "islocal", "created", "time":
		finfo, err := lfv.mod.GetFactoidInfo(lfv.Name, lfv.g.ActionSource().ChannelID(), false)
		if err == ErrNoSuchFactoid {
			L.RaiseError("No such factoid %s", lfv.Name)
		} else if err != nil {
			L.RaiseError("err %s.src: %s")
		}

		if method == "islocal" {
			if finfo.ScopeChannel == "" {
				L.Push(lua.LFalse)
			} else {
				L.Push(lua.LTrue)
			}
		} else if method == "locked" {
			if finfo.IsLocked {
				L.Push(lua.LTrue)
			} else {
				L.Push(lua.LFalse)
			}
		} else if method == "created" {
			archiveUrl := lfv.g.Team().ArchiveURL(slack.MsgID(finfo.LastChannel, finfo.LastMessage))
			L.Push(lua.LString(archiveUrl))
		} else if method == "time" {
			L.Push(lua.LNumber(finfo.LastTimestamp.Unix()))
		}
		return 1
	case "history":
		L.RaiseError("NotImplemented")
		return 0
	case "data":
		L.Push(LNewFDataMap(lfv.g, lfv.mod, "F-"+lfv.Name))
		return 1
	default:
		L.RaiseError("No such method factoid.%s", method)
		return 0
	}
}
