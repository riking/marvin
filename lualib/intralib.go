package lualib

import (
	"net/url"

	"github.com/pkg/errors"
	"github.com/yuin/gopher-lua"

	"github.com/riking/marvin/intra"
	"github.com/riking/marvin/modules/weblogin"
	"github.com/riking/marvin/util"
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
		webModule := g.Team().GetModule(weblogin.Identifier).(weblogin.API)
		user, err := webModule.GetUserBySlack(g.ActionSource().UserID())
		if err != nil {
			return nil
		}
		if user == nil {
			return nil
		}
		if user.IntraToken == nil {
			return nil
		}
		return intra.Client(L.Ctx, intra.OAuthConfig(g.Team()), user.IntraToken)
	}

	tab.RawSetString("valid_token", L.NewFunction(func(L *lua.LState) int {
		client := getClient(L)
		if client == nil {
			L.Push(lua.LFalse)
			L.Push(lua.LString("no_token"))
			return 2
		}
		var t intra.Campus
		resp, err := client.DoGetFormJSON(L.Ctx, "/v2/campus/" + fremontCampusString,
			nil, &t)
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
	tab.RawSetString("get", L.NewFunction(func(L *lua.LState) int {
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
		var form url.Values
		args.(*lua.LTable).ForEach(func(k, v lua.LValue) {
			if v.Type() == lua.LTTable {
				vtable := v.(*lua.LTable)
				var ary []string
				for i := 0; i < vtable.Len(); i++ {
					ary = append(ary, lua.LVAsString(L.ToStringMeta(vtable.RawGetInt(i + 1))))
				}
				form[lua.LVAsString(L.ToStringMeta(k))] = ary
			} else {
				form.Set(lua.LVAsString(L.ToStringMeta(k)), lua.LVAsString(L.ToStringMeta(v)))
			}
		})
		var json interface{}
		resp, err := client.DoGetFormJSON(L.Ctx, endpoint, form, &json)
		if err != nil {
			util.LogError(errors.Wrap(err, "intra http error"))
			L.Push(lua.LNil)
			L.Push(LNewResponse(L, resp))
			L.Push(lua.LString(err.Error()))
			return 3
		}
		parsed := jsonToLua(L, json)
		L.Push(parsed)
		L.Push(LNewResponse(L, resp))
		L.Push(lua.LNil)
		return 3
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
		client := getClient(L)
		if client == nil {
			L.RaiseError("intra: no client")
		}
		table := L.CheckTable(1)
		id := L.CheckNumber(2)
		intraUser, err := client.UserByID(L.Ctx, int(id))
		if err != nil {
			L.RaiseError(err.Error())
			return 0
		}
		// TODO GoToLua()
		_ = intraUser
		_ = table
		L.RaiseError("TODO not implemented")
		L.Push(lua.LNil)
		return 1
	}))
	tab.RawSetString("users", usersT)

	L.SetGlobal("intra", tab)
	L.Push(tab)
	return 1
}
