package lualib

import (
	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
	"github.com/yuin/gopher-lua"
)

const metatableLTeam = "_metatable_LTeam"

func LNewTeam(g *G) lua.LValue {
	var u *lua.LUserData

	tab := g.L.NewTable()

	u = g.L.NewUserData()
	u.Value = &LChannelIndex{g: g}
	u.Metatable = g.L.GetTypeMetatable(metatableLChannelIdx)
	tab.RawSetString("channel", u)
	tab.RawSetString("channels", u)

	tab.RawSetString("archive", g.L.NewFunction(func(L *lua.LState) int {
		channel := L.CheckString(1)
		ts := L.CheckString(2)

		L.Push(lua.LString(g.Team().ArchiveURL(slack.MessageID{
			ChannelID: slack.ChannelID(channel),
			MessageTS: slack.MessageTS(ts),
		})))
		return 1
	}))
	tab.RawSetString("rawsend", g.L.NewFunction(func(L *lua.LState) int {
		if g.ActionSource().AccessLevel() < marvin.AccessLevelAdmin {
			L.RaiseError("rawsend() is restricted to administrators")
			return 0
		}
		//g.Team().SendComplexMessage()
		return 0
	}))
	return tab
}
