package factoid

import (
	"fmt"

	"github.com/yuin/gopher-lua"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
)

//region LFactoid
var _ = 1

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
	tab := L.NewTypeMetatable(metatableLFactoid)
	tab.RawSetString("__index", L.NewFunction(luaFactoid__Index))
	tab.RawSetString("__newindex", L.NewFunction(luaFactoid__Set))
	tab.RawSetString("__call", L.NewFunction(luaFactoid__Call))
}

func LNewFactoid(flua *FactoidLua, name string) lua.LValue {
	v := &LFactoid{flua: flua, Name: name}
	u := flua.L.NewUserData()
	u.Value = v
	u.Metatable = flua.L.GetTypeMetatable(metatableLFactoid)
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
	case "src", "raw", "author":
		finfo, err := lfv.flua.mod.GetFactoidBare(lfv.Name, lfv.flua.ActSource.ChannelID())
		if err == ErrNoSuchFactoid {
			L.RaiseError("No such factoid %s", lfv.Name)
		} else if err != nil {
			L.RaiseError("err %s.src: %s")
		}

		if method == "src" || method == "raw" {
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
		L.Push(LNewFDataMap(lfv.flua, lfv.Name))
		return 1
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
	tab := L.NewTypeMetatable(metatableLUser)
	tab.RawSetString("__tostring", L.NewFunction(luaUser__ToString))
	tab.RawSetString("__eq", L.NewFunction(luaUser__Eq))
	tab.RawSetString("__index", L.NewFunction(luaUser__Index))
}

func LNewUser(flua *FactoidLua, user slack.UserID, preload bool) (lua.LValue, error) {
	v := &LUser{flua: flua, ID: user, Info: nil}
	if flua.mod.team != nil {
		v.Acc = flua.mod.team.UserLevel(user)
	}
	if preload {
		err := v.LoadInfo()
		if err != nil {
			return nil, err
		}
	}

	u := flua.L.NewUserData()
	u.Value = v
	u.Metatable = flua.L.GetTypeMetatable(metatableLUser)
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
//region LChannel
var _ = 1

// LChannel ...
//
//   ch.id: slack.ChannelID
//   ch.type: "public", "group", "mpim", or "im"
//   ch.name: string
//   ch.creator: LUser
//   ch.im_other: slack.UserID
//   ch.users: []slack.UserID
//   ch.topic: string
//   ch.topic_user: LUser
//   ch.topic_changed: unixMillis
//   ch.topic = string TODO determine if this is a good idea
//   ch.purpose: string
//   ch.purpose_user: LUser
//   ch.purpose_changed: unixMillis
//   ch.purpose = string
type LChannel struct {
	flua     *FactoidLua
	ID       slack.ChannelID
	IsPublic bool
	IsGroup  bool
	IsIM     bool
	IMOther  lua.LValue
	Info     *slack.Channel
	Creator  lua.LValue
	Users    *lua.LTable
}

const metatableLChannel = "_metatable_LChannel"

func (LChannel) SetupMetatable(L *lua.LState) {
	tab := L.NewTypeMetatable(metatableLChannel)
	tab.RawSetString("__tostring", L.NewFunction(luaChannel__ToString))
	tab.RawSetString("__eq", L.NewFunction(luaChannel__Eq))
	tab.RawSetString("__index", L.NewFunction(luaChannel__Index))
}

func LNewChannel(flua *FactoidLua, ch slack.ChannelID) lua.LValue {
	v := &LChannel{flua: flua, ID: ch, Info: nil}
	if ch[0] == 'C' {
		v.IsPublic = true
		info, err := flua.mod.team.PublicChannelInfo(ch)
		if err != nil {
			flua.L.RaiseError("could not get public channel info: %s", err)
		}
		v.Info = info
	} else if ch[0] == 'G' {
		v.IsGroup = true
		info, err := flua.mod.team.PrivateChannelInfo(ch)
		if err != nil {
			flua.L.RaiseError("could not get private channel info: %s", err)
		}
		v.Info = info
	} else if ch[0] == 'D' {
		v.IsIM = true
		otherUID, _ := flua.mod.team.GetIMOtherUser(flua.ActSource.ChannelID())
		u, _ := LNewUser(flua, otherUID, false)
		v.IMOther = u
	}

	u := flua.L.NewUserData()
	u.Value = v
	u.Metatable = flua.L.GetTypeMetatable(metatableLChannel)
	return u
}

func luaChannel__ToString(L *lua.LState) int {
	ud := L.CheckUserData(1)
	lc := ud.Value.(*LChannel)
	L.Push(lua.LString(lc.flua.mod.team.FormatChannel(lc.ID)))
	return 1
}

func luaChannel__Eq(L *lua.LState) int {
	if L.GetTop() != 2 {
		L.RaiseError("__eq() takes two arguments")
	}
	ud1 := L.CheckUserData(1)
	ud2 := L.CheckUserData(2)
	lc1 := ud1.Value.(*LChannel)
	lc2 := ud2.Value.(*LChannel)
	if lc1.ID == lc2.ID {
		L.Push(lua.LTrue)
	} else {
		L.Push(lua.LFalse)
	}
	return 1
}

func luaChannel__Index(L *lua.LState) int {
	if L.GetTop() != 2 {
		L.RaiseError("__index() requires 2 arguments")
	}
	ud := L.CheckUserData(1)
	lc, ok := ud.Value.(*LChannel)
	if !ok {
		L.RaiseError("user__tostring() with wrong type for self")
	}
	key := L.CheckString(2)
	switch key {
	case "id":
		L.Push(lua.LString(lc.ID))
		return 1
	case "type":
		if lc.IsPublic {
			L.Push(lua.LString("public"))
		} else if lc.IsGroup && lc.Info.IsMultiIM() {
			L.Push(lua.LString("mpim"))
		} else if lc.IsGroup {
			L.Push(lua.LString("group"))
		} else {
			L.Push(lua.LString("im"))
		}
		return 1
	case "name":
		if lc.IsIM {
			otherUser, _ := lc.flua.mod.team.GetIMOtherUser(lc.ID)
			L.Push(lua.LString(fmt.Sprintf("[IM with %v]", otherUser)))
			return 1
		}
		L.Push(lua.LString(lc.Info.Name))
		return 1
	case "im_other":
		if lc.IsIM {
			otherUser, _ := lc.flua.mod.team.GetIMOtherUser(lc.ID)
			L.Push(lua.LString(otherUser))
			return 1
		}
		return 0
	case "creator":
		if lc.Creator == nil {
			u, _ := LNewUser(lc.flua, lc.Info.Creator, false)
			if u == nil {
				return 0
			}
			lc.Creator = u
		}
		L.Push(lc.Creator)
		return 1
	case "users":
		if lc.Users == nil {
			tab := L.NewTable()
			members := lc.flua.mod.team.ChannelMemberList(lc.ID)
			for i, v := range members {
				u, _ := LNewUser(lc.flua, v, false)
				tab.RawSetInt(i+1, u)
			}
			lc.Users = tab
		}
		L.Push(lc.Users)
		return 1
	case "mention":
		L.Push(lua.LString(lc.flua.mod.team.FormatChannel(lc.ID)))
		return 1
	case "topic", "purpose":
		var s string
		if key == "topic" {
			s = lc.Info.Topic.Value
		} else {
			s = lc.Info.Purpose.Value
		}
		L.Push(lua.LString(s))
		return 1
	case "topic_changed", "purpose_changed":
		var s float64
		if key == "topic" {
			s = lc.Info.Topic.LastSet
		} else {
			s = lc.Info.Purpose.LastSet
		}
		L.Push(lua.LNumber(s))
		return 1
	case "topic_user", "purpose_user":
		var s slack.UserID
		if key == "topic" {
			s = lc.Info.Topic.Creator
		} else {
			s = lc.Info.Purpose.Creator
		}
		u, _ := LNewUser(lc.flua, s, false)
		L.Push(u)
		return 1
	default:
		L.RaiseError("no such member %s in Channel (have: id type name mention topic purpose)", key)
		return 0
	}
}
