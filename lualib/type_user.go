package lualib

import (
	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
	"github.com/yuin/gopher-lua"
)

// LUser has the following API:
//
//   tostring(user) -> "<@USLACKBOT>"
//   user.id -> "USLACKBOT"
//   user.is_blacklisted
//   user.is_admin
//   user.is_controller
//   user.username -> "slackbot"
//    fname, lname, name
//   user.tz -> "America/Los_Angeles"
//   user.tz_offset -> -28800 (seconds)
//   user.profile.real
//   user.profile.first
//   user.profile.last
//   user.profile.phone
type LUser struct {
	g       *G
	ID      slack.UserID
	Acc     marvin.AccessLevel
	Info    *slack.User
	profile *lua.LTable
}

const metatableLUser = "_metatable_LUser"

func (*LUser) SetupMetatable(L *lua.LState) {
	tab := L.NewTypeMetatable(metatableLUser)
	tab.RawSetString("__tostring", L.NewFunction(luaUser__ToString))
	tab.RawSetString("__eq", L.NewFunction(luaUser__Eq))
	tab.RawSetString("__index", L.NewFunction(luaUser__Index))
}

func LNewUser(g *G, user slack.UserID, preload bool) (lua.LValue, error) {
	v := &LUser{g: g, ID: user, Info: nil}
	if g.Team() != nil {
		v.Acc = g.Team().UserLevel(user)
	}
	if preload {
		err := v.loadInfo()
		if err != nil {
			return nil, err
		}
	}

	u := g.L.NewUserData()
	u.Value = v
	u.Metatable = g.L.GetTypeMetatable(metatableLUser)
	return u, nil
}

func (u *LUser) loadInfo() error {
	if u.Info != nil {
		return nil
	}
	info, err := u.g.Team().UserInfo(u.ID)
	if err != nil {
		return err
	}
	u.Info = info
	return nil
}

func (u *LUser) getProfile(L *lua.LState) *lua.LTable {
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
		L.RaiseError("user__tostring() with wrong type for self, got %T", L.CheckUserData(1).Value)
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
		L.RaiseError("user__index() with wrong type for self, got %T", ud.Value)
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
		err := u.loadInfo()
		if err != nil {
			L.RaiseError("Error getting information for user %v", u.ID)
			return 0
		}
		L.Push(u.getProfile(L))
		return 1
	case "username", "tz", "tz_offset", "fname", "lname", "name", "deleted":
		err := u.loadInfo()
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
			if u.Info.Profile.LastName == "" {
				L.Push(lua.LString(u.Info.Name))
			} else {
				L.Push(lua.LString(u.Info.Profile.LastName))
			}
		case "name":
			if u.Info.Profile.RealName == "" {
				L.Push(lua.LString(u.Info.Name))
			} else {
				L.Push(lua.LString(u.Info.Profile.RealName))
			}
		case "tz":
			L.Push(lua.LString(u.Info.Tz))
		case "tz_offset":
			L.Push(lua.LNumber(u.Info.TzOffset))
		case "deleted":
			L.Push(lua.LBool(u.Info.Deleted))
		}
		return 1
	default:
		L.RaiseError("no such field %s in User", key)
		return 0
	}
}
