package lualib

import (
	"fmt"
	"net/url"

	"github.com/yuin/gopher-lua"

	"github.com/riking/marvin/intra"
)

const fremontCampusString = "7"
const fremontCampusID = 7

// API:
// intra.get("/v2/endpoint/:id", {id=3}) -> requests.response
// intra.valid_token() -> bool
// intra.projects.get_next_line.id
// intra.user_id[user.username]
// intra.users[intra.user_id[user.username]].id

func OpenIntra(g *G, L *lua.LState) int {
	tab := L.NewTable()

	getClient := func(L *lua.LState) *intra.Helper {
		return intra.Client(L.Ctx, intra.ClientCredentialsTokenSource(L.Ctx, g.Team()))
	}

	tab.RawSetString("valid_token", L.NewFunction(func(L *lua.LState) int {
		client := getClient(L)
		if client == nil {
			L.Push(lua.LFalse)
			L.Push(lua.LString("no_token"))
			return 2
		}
		var t intra.Campus
		resp, err := client.DoGetFormJSON(L.Ctx, "/v2/campus/7", nil, &t)
		if err != nil {
			L.Push(lua.LFalse)
			L.Push(lua.LString("verify_err_connect"))
			return 2
		}
		if resp.StatusCode != 200 {
			L.Push(lua.LFalse)
			L.Push(lua.LString("verify_err_non2xx"))
			return 2
		}
		if t.ID != fremontCampusID {
			L.Push(lua.LFalse)
			L.Push(lua.LString("verify_err_baddata"))
			return 2
		}
		L.Push(lua.LTrue)
		return 1
	}))
	tab.RawSetString("getall", L.NewFunction(func(L *lua.LState) int {
		client := getClient(L)
		if client == nil {
			L.RaiseError("intra: no client")
		}
		endpoint := L.CheckString(1)
		args := L.Get(2)
		if args.Type() == lua.LTNil {
			args = L.NewTable()
		}
		if args.Type() != lua.LTTable {
			L.TypeError(2, lua.LTTable)
		}
		var argsTab *lua.LTable = args.(*lua.LTable)

		var form url.Values
		argsTab.ForEach(func(k, v lua.LValue) {
			if v.Type() == lua.LTTable {
				vtable := v.(*lua.LTable)
				var ary []string
				for i := 0; i < vtable.Len(); i++ {
					ary = append(ary, lua.LVAsString(L.ToStringMeta(vtable.RawGetInt(i+1))))
				}
				form[lua.LVAsString(L.ToStringMeta(k))] = ary
			} else {
				form.Set(lua.LVAsString(L.ToStringMeta(k)), lua.LVAsString(L.ToStringMeta(v)))
			}
		})
		var j interface{}
		resultCh := client.PaginatedGet(L.Ctx, endpoint, form, &j)
		var err error
		tab := L.NewTable()
		for q := range resultCh {
			if !q.OK {
				err = q.Error
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				return 2
			}
			tab.Append(jsonToLua(L, q.Value))
		}
		L.Push(tab)
		return 1
	}))
	tab.RawSetString("getch", L.NewFunction(func(L *lua.LState) int {
		client := getClient(L)
		if client == nil {
			L.RaiseError("intra: no client")
		}
		endpoint := L.CheckString(1)
		args := L.CheckTable(2)
		ch := L.CheckChannel(3)
		var form url.Values
		args.ForEach(func(k, v lua.LValue) {
			if v.Type() == lua.LTTable {
				vtable := v.(*lua.LTable)
				var ary []string
				for i := 0; i < vtable.Len(); i++ {
					ary = append(ary, lua.LVAsString(L.ToStringMeta(vtable.RawGetInt(i+1))))
				}
				form[lua.LVAsString(L.ToStringMeta(k))] = ary
			} else {
				form.Set(lua.LVAsString(L.ToStringMeta(k)), lua.LVAsString(L.ToStringMeta(v)))
			}
		})

		var typ interface{}
		resultCh := client.PaginatedGet(L.Ctx, endpoint, form, &typ)
		go func() {
			for v := range resultCh {
				ch <- GoToLua(L, v)
			}
		}()
		return 0
	}))
	// .user_id
	mt := L.NewTable()
	userIDT := L.NewTable()
	userIDT.Metatable = mt
	mt.RawSetString("__index", L.NewFunction(func(L *lua.LState) int {
		client := getClient(L)
		if client == nil {
			L.RaiseError("intra: no client")
		}
		table := L.CheckTable(1)
		login := L.CheckString(2)
		id, err := client.UserIDByLogin(L.Ctx, login)
		if err != nil {
			L.RaiseError("intra api error: %v", err)
		}
		if id == -1 {
			L.RaiseError("no such user %s", login)
		}
		table.RawSetString(login, lua.LNumber(id))
		L.Push(lua.LNumber(id))
		return 1
	}))
	tab.RawSetString("user_id", userIDT)
	// .users
	mt = L.NewTable()
	usersT := L.NewTable()
	usersT.Metatable = mt
	mt.RawSetString("__index", L.NewFunction(func(L *lua.LState) int {
		tab := L.CheckTable(1)
		id := L.CheckNumber(2)
		// normal indexing uses the array, but we want to use the hashtable to save memory
		// so we must explicitly re-check the lua cache
		val := tab.RawGetH(lua.LNumber(id))
		if val != lua.LNil {
			L.Push(val)
			return 1
		}

		client := getClient(L)
		if client == nil {
			L.RaiseError("intra: no client")
		}
		intraUser, err := client.UserByID(L.Ctx, int(id))
		if err != nil {
			L.RaiseError(err.Error())
			return 0
		}
		val = GoToLua(L, intraUser)
		tab.RawSetH(lua.LNumber(id), val)
		L.Push(val)
		return 1
	}))
	tab.RawSetString("users", usersT)
	// .campus
	mt = L.NewTable()
	campusT := L.NewTable()
	campusT.Metatable = mt
	mt.RawSetString("__index", L.NewFunction(func(L *lua.LState) int {
		client := getClient(L)
		if client == nil {
			L.RaiseError("intra: no client")
		}

		tab := L.CheckTable(1)
		key := L.Get(2)
		switch key.Type() {
		case lua.LTNumber:
			intKey := int(key.(lua.LNumber))
			campus, err := client.CampusByID(L.Ctx, intKey)
			if err != nil {
				L.RaiseError(err.Error())
			}
			val := GoToLua(L, campus)
			tab.RawSetInt(intKey, val)
			L.Push(val)
			return 1
		case lua.LTString:
			strKey := string(key.(lua.LString))
			campus, err := client.CampusByName(L.Ctx, strKey)
			if err != nil {
				L.RaiseError(err.Error())
			}
			fmt.Println("CampusByName return", campus, err)
			val := GoToLua(L, campus)
			tab.RawSetString(strKey, val)
			L.Push(val)
			return 1
		default:
			return 0
		}
	}))
	tab.RawSetString("campus", campusT)

	L.SetGlobal("intra", tab)
	L.Push(tab)
	return 1
}
