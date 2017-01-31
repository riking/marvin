package lualib

import (
	"fmt"

	"github.com/yuin/gopher-lua"

	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
)

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
//   ch.purpose: string
//   ch.purpose_user: LUser
//   ch.purpose_changed: unixMillis
type LChannel struct {
	g        *G
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
const metatableLChannelIdx = "_metatable_LChannelIndex"

func (*LChannel) SetupMetatable(L *lua.LState) int {
	tab := L.NewTypeMetatable(metatableLChannel)
	tab.RawSetString("__tostring", L.NewFunction(luaChannel__ToString))
	tab.RawSetString("__eq", L.NewFunction(luaChannel__Eq))
	tab.RawSetString("__index", L.NewFunction(luaChannel__Index))

	tab = L.NewTypeMetatable(metatableLChannelIdx)
	tab.RawSetString("__index", L.NewFunction(luaChannelIndex__Index))
	return 0
}

func LNewChannel(g *G, ch slack.ChannelID) lua.LValue {
	v := &LChannel{g: g, ID: ch, Info: nil}
	if ch[0] == 'C' {
		v.IsPublic = true
		info, err := g.Team().PublicChannelInfo(ch)
		if err != nil {
			g.L.RaiseError("could not get public channel info: %s", err)
		}
		v.Info = info
	} else if ch[0] == 'G' {
		v.IsGroup = true
		info, err := g.Team().PrivateChannelInfo(ch)
		if err != nil {
			g.L.RaiseError("could not get private channel info: %s", err)
		}
		v.Info = info
	} else if ch[0] == 'D' {
		v.IsIM = true
		otherUID, _ := g.Team().GetIMOtherUser(ch)
		u, _ := LNewUser(g, otherUID, false)
		v.IMOther = u
	}

	u := g.L.NewUserData()
	u.Value = v
	u.Metatable = g.L.GetTypeMetatable(metatableLChannel)
	return u
}

type LChannelIndex struct {
	g *G
}

func luaChannelIndex__Index(L *lua.LState) int {
	ud := L.CheckUserData(1)
	nameV := L.Get(2)
	if nameV.Type() != lua.LTString {
		return 0
	}
	name := L.CheckString(2)
	lci := ud.Value.(*LChannelIndex)
	g := lci.g

	if name == "" {
		return 0
	}
	chID := ""
	if name[0] == 'C' || name[0] == 'G' {
		chID = name
	} else if name[0] == 'D' {
		chID, _ := g.Team().GetIM(g.ActionSource().UserID())
		if slack.ChannelID(name) != chID {
			// vischeck: only the IM of marvin<=>the active user may be used
			if g.ActionSource().AccessLevel() < marvin.AccessLevelAdmin {
				return 0
			}
		}
		tab := L.NewTable()
		tab.RawSetString("id", lua.LString(chID))
		tab.RawSetString("type", lua.LString("im"))
		u, _ := LNewUser(g, g.ActionSource().UserID(), false)
		tab.RawSetString("im_user", u)
		L.Push(tab)
		return 1
	} else {
		chID = string(g.Team().ChannelIDByName(name))
	}

	if chID == "" {
		return 0
	}
	top := L.GetTop()
	err := L.GPCall(func(L *lua.LState) int {
		L.Push(LNewChannel(g, slack.ChannelID(chID)))
		return 1
	}, lua.LNil)
	if err != nil {
		// remove the gpcall() return, if any
		L.SetTop(L.GetTop() - top - 1)
		L.Push(lua.LNil)
		L.Push(err.(*lua.ApiError).Object)
		return 2
	}

	// vischeck the channel
	// this has an timing attack showing the channel exists but not what's in it
	if chID[0] == 'G' {
		memberMap := g.Team().UserInChannels(g.ActionSource().UserID(), slack.ChannelID(chID))
		if !memberMap[slack.ChannelID(chID)] {
			// vischeck failed
			if g.ActionSource().AccessLevel() < marvin.AccessLevelAdmin {
				return 0
			}
		}
	}
	return 1
}

func luaChannel__ToString(L *lua.LState) int {
	ud := L.CheckUserData(1)
	lc := ud.Value.(*LChannel)
	L.Push(lua.LString(lc.g.Team().FormatChannel(lc.ID)))
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
			otherUser, _ := lc.g.Team().GetIMOtherUser(lc.ID)
			L.Push(lua.LString(fmt.Sprintf("[IM with %v]", otherUser)))
			return 1
		}
		L.Push(lua.LString(lc.Info.Name))
		return 1
	case "creator":
		if lc.Creator == nil {
			u, _ := LNewUser(lc.g, lc.Info.Creator, false)
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
			members := lc.g.Team().ChannelMemberList(lc.ID)
			for i, v := range members {
				u, _ := LNewUser(lc.g, v, false)
				tab.RawSetInt(i+1, u)
			}
			lc.Users = tab
		}
		L.Push(lc.Users)
		return 1
	case "mention":
		L.Push(lua.LString(lc.g.Team().FormatChannel(lc.ID)))
		return 1
	case "topic", "purpose":
		var s string
		if lc.Info == nil {
			return 0
		}
		if key == "topic" {
			s = lc.Info.Topic.Value
		} else {
			s = lc.Info.Purpose.Value
		}
		L.Push(lua.LString(s))
		return 1
	case "topic_changed", "purpose_changed":
		var s float64
		if lc.Info == nil {
			return 0
		}
		if key == "topic" {
			s = lc.Info.Topic.LastSet
		} else {
			s = lc.Info.Purpose.LastSet
		}
		L.Push(lua.LNumber(s))
		return 1
	case "topic_user", "purpose_user":
		var s slack.UserID
		if lc.Info == nil {
			return 0
		}
		if key == "topic" {
			s = lc.Info.Topic.Creator
		} else {
			s = lc.Info.Purpose.Creator
		}
		u, _ := LNewUser(lc.g, s, false)
		L.Push(u)
		return 1
	default:
		L.RaiseError("no such member %s in Channel (have: id type name mention topic purpose)", key)
		return 0
	}
}
