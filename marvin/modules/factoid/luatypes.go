package factoid

import (
	"github.com/yuin/gopher-lua"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
)

//region LFactoid

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
	flua *FactoidLua
	Name string
}

const metatableLFactoid = "_metatable_LFactoid"

func (LFactoid) SetupMetatable(L *lua.LState) {
	tab := L.NewTable()
	tab.RawSetString("__index", L.NewFunction(luaFactoid__Index))
	tab.RawSetString("__newindex", L.NewFunction(luaFactoid__Set))
	tab.RawSetString("__call", L.NewFunction(luaFactoid__Call))
	L.SetGlobal(metatableLFactoid, tab)
}

func LNewFactoid(flua *FactoidLua, name string) lua.LValue {
	v := &LFactoid{flua: flua, Name: name}
	u := flua.L.NewUserData()
	u.Value = v
	u.Metatable = flua.L.GetGlobal(metatableLFactoid)
	return u
}

// ---

func luaFactoid__Call(L *lua.LState) int {
	if L.GetTop() == 0 {
		L.RaiseError("__call() requires >1 argument")
	}
	lfv, ok := L.CheckUserData(1).Value.(*LFactoid)
	if !ok {
		L.RaiseError("factoid__call() with wrong type for self")
	}

	var args = []string{lfv.Name}
	for i := 1; i <= L.GetTop(); i++ {
		args = append(args, lua.LVAsString(L.ToStringMeta(L.Get(i))))
	}

	var of OutputFlags
	result, err := lfv.flua.mod.RunFactoid(lfv.flua.Ctx, args, &of, lfv.flua.ActSource)
	if err != nil {
		L.RaiseError("factoid error: %s", err)
	}

	if of.SideEffects {
		lfv.flua.OFlags.SideEffects = true
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

	// TODO - forget, unforget

	switch method {
	case "exists":
		_, err := lfv.flua.mod.GetFactoidBare(lfv.Name, lfv.flua.ActSource.ChannelID())
		if err == ErrNoSuchFactoid {
			L.Push(lua.LFalse)
		} else {
			L.Push(lua.LTrue)
		}
		return 1
	case "src", "author":
		finfo, err := lfv.flua.mod.GetFactoidBare(lfv.Name, lfv.flua.ActSource.ChannelID())
		if err == ErrNoSuchFactoid {
			L.RaiseError("No such factoid %s", lfv.Name)
		} else if err != nil {
			L.RaiseError("err %s.src: %s")
		}

		if method == "src" {
			L.Push(lua.LString(finfo.RawSource))
		} else if method == "author" {
			u, err := LNewUser(lfv.flua, finfo.LastUser, true)
			if err != nil {
				L.RaiseError("could not load user data for %v", finfo.LastUser)
			}
			L.Push(u)
		}
		return 1
	case "locked", "islocal", "created", "time":
		finfo, err := lfv.flua.mod.GetFactoidInfo(lfv.Name, lfv.flua.ActSource.ChannelID(), false)
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
			archiveUrl := lfv.flua.mod.team.ArchiveURL(slack.MsgID(finfo.LastChannel, finfo.LastMessage))
			L.Push(lua.LString(archiveUrl))
		} else if method == "time" {
			L.Push(lua.LNumber(finfo.LastTimestamp.Unix()))
		}
		return 1
	case "history":
		L.RaiseError("NotImplemented")
		return 0
	case "data":
		L.RaiseError("NotImplemented")
		return 0
	default:
		L.RaiseError("No such method factoid.%s", method)
		return 0
	}
}

func luaFactoid__Set(L *lua.LState) int {
	if L.GetTop() != 2 {
		L.RaiseError("__newindex() requires 2 arguments")
		return 0
	}
	lfv, ok := L.CheckUserData(1).Value.(*LFactoid)
	if !ok {
		L.RaiseError("factoid__newindex() with wrong type for self")
		return 0
	}
	method := L.CheckString(2)
	switch method {
	case "data":
		_ = lfv.flua
		L.RaiseError("NotImplemented")
		return 0
	default:
		L.RaiseError("No such method factoid.%s=", method)
		return 0
	}
}

//endregion
//region LUser

// LUser has the following API:
//
//   tostring(user) -> "<@USLACKBOT>"
//   user.id -> "USLACKBOT"
//   user.is_blacklisted
//   user.is_admin
//   user.is_controller
//   user.username -> "slackbot"
//   user.tz -> "America/Los_Angeles"
//   user.tz_offset -> -28800 (seconds)
//   user.profile.real
//   user.profile.first
//   user.profile.last
//   user.profile.phone
type LUser struct {
	flua    *FactoidLua
	ID      slack.UserID
	Acc     marvin.AccessLevel
	Info    *slack.User
	profile *lua.LTable
}

const metatableLUser = "_metatable_LUser"

func (LUser) SetupMetatable(L *lua.LState) {
	tab := L.NewTable()
	tab.RawSetString("__tostring", L.NewFunction(luaUser__ToString))
	tab.RawSetString("__eq", L.NewFunction(luaUser__Eq))
	tab.RawSetString("__index", L.NewFunction(luaUser__Index))
	L.SetGlobal(metatableLUser, tab)
}

func LNewUser(flua *FactoidLua, user slack.UserID, preload bool) (lua.LValue, error) {
	v := &LUser{flua: flua, ID: user, Info: nil}
	if flua.mod.team != nil {
		v.Acc = flua.mod.team.UserLevel(user)
		if preload {
			err := v.LoadInfo()
			if err != nil {
				return nil, err
			}
		}
	}

	u := flua.L.NewUserData()
	u.Value = v
	u.Metatable = flua.L.GetGlobal(metatableLUser)
	return u, nil
}

func (u *LUser) LoadInfo() error {
	if u.Info != nil {
		return nil
	}
	info, err := u.flua.mod.team.UserInfo(u.ID)
	if err != nil {
		return err
	}
	u.Info = info
	return nil
}

func (u *LUser) Profile(L *lua.LState) *lua.LTable {
	if u.profile != nil {
		return u.profile
	}

	prof := L.NewTable()
	if u.Info.Profile.RealName == "" {
		prof.RawSetString("real", lua.LString(u.Info.Name))
		prof.RawSetString("first", lua.LString(u.Info.Name))
		prof.RawSetString("last", lua.LString(u.Info.Name))
	} else {
		prof.RawSetString("real", lua.LString(u.Info.Profile.RealName))
		prof.RawSetString("first", lua.LString(u.Info.Profile.FirstName))
		prof.RawSetString("last", lua.LString(u.Info.Profile.LastName))
	}
	prof.RawSetString("phone", lua.LString(u.Info.Profile.Phone))
	prof.RawSetString("title", lua.LString(u.Info.Profile.Title))
	u.profile = prof
	return u.profile
}

func luaUser__Eq(L *lua.LState) int {
	if L.GetTop() != 2 {
		L.RaiseError("__eq() requires 2 arguments")
	}
	u1 := L.CheckUserData(1).Value.(*LUser)
	u2 := L.CheckUserData(2).Value.(*LUser)
	if u1.ID == u2.ID {
		L.Push(lua.LTrue)
		return 1
	}
	L.Push(lua.LFalse)
	return 1
}

func luaUser__ToString(L *lua.LState) int {
	if L.GetTop() != 1 {
		L.RaiseError("__tostring() requires 1 arguments")
	}
	u, ok := L.CheckUserData(1).Value.(*LUser)
	if !ok {
		L.RaiseError("user__tostring() with wrong type for self")
	}
	L.Push(lua.LString(u.ID.ToAtForm()))
	return 1
}

func luaUser__Index(L *lua.LState) int {
	if L.GetTop() != 2 {
		L.RaiseError("__index() requires 2 arguments")
	}
	ud := L.CheckUserData(1)
	u, ok := ud.Value.(*LUser)
	if !ok {
		L.RaiseError("user__tostring() with wrong type for self")
	}
	key := L.CheckString(2)
	switch key {
	case "id":
		L.Push(lua.LString(u.ID))
		return 1
	case "is_blacklisted":
		if u.Acc <= marvin.AccessLevelBlacklisted {
			L.Push(lua.LTrue)
			return 1
		}
		L.Push(lua.LFalse)
		return 1
	case "is_admin":
		if u.Acc >= marvin.AccessLevelAdmin {
			L.Push(lua.LTrue)
			return 1
		}
		L.Push(lua.LFalse)
		return 1
	case "is_controller":
		if u.Acc >= marvin.AccessLevelController {
			L.Push(lua.LTrue)
			return 1
		}
		L.Push(lua.LFalse)
		return 1
	case "profile":
		err := u.LoadInfo()
		if err != nil {
			L.RaiseError("Error getting information for user %v", u.ID)
			return 0
		}
		L.Push(u.Profile(L))
		return 1
	case "username", "tz", "tz_offset", "fname", "lname", "name":
		err := u.LoadInfo()
		if err != nil {
			L.RaiseError("Error getting information for user %v", u.ID)
			return 0
		}
		switch key {
		case "username":
			L.Push(lua.LString(u.Info.Name))
		case "fname":
			if u.Info.Profile.FirstName == "" {
				L.Push(lua.LString(u.Info.Name))
			} else {
				L.Push(lua.LString(u.Info.Profile.FirstName))
			}
		case "lname":
			if u.Info.Profile.FirstName == "" {
				L.Push(lua.LString(u.Info.Name))
			} else {
				L.Push(lua.LString(u.Info.Profile.LastName))
			}
		case "name":
			if u.Info.Profile.FirstName == "" {
				L.Push(lua.LString(u.Info.Name))
			} else {
				L.Push(lua.LString(u.Info.Profile.RealName))
			}
		case "tz":
			L.Push(lua.LString(u.Info.Tz))
		case "tz_offset":
			L.Push(lua.LNumber(u.Info.TzOffset))
		}
		return 1
	default:
		L.RaiseError("no such field %s in User", key)
		return 0
	}
}

//endregion
