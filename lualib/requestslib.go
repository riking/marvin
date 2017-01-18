package lualib

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/yuin/gopher-lua"
)

func OpenRequests(L *lua.LState) int {
	mod := L.RegisterModule("requests", requestsFuncs).(*lua.LTable)

	L.Push(mod)
	return 1
}

var requestsFuncs = map[string]lua.LGFunction{
	"get":    requestsHelperNoBody(http.MethodGet),
	"head":   requestsHelperNoBody(http.MethodHead),
	"delete": requestsHelperNoBody(http.MethodDelete),
	"trace":  requestsHelperNoBody(http.MethodTrace),

	"post":    requestsHelperBody(http.MethodPost),
	"put":     requestsHelperBody(http.MethodPut),
	"options": requestsHelperBody(http.MethodOptions),
	"patch":   requestsHelperBody(http.MethodPatch),

	"request": luaRequest,
}

func requestsHelperNoBody(method string) func(L *lua.LState) int {
	return func(L *lua.LState) int {
		url, headers, options := L.Get(1), L.Get(2), L.Get(3)
		r := L.GetGlobal("requests").(*lua.LTable).RawGetString("request").(*lua.LFunction)
		L.Push(r)
		L.Push(url)
		L.Push(headers)
		L.Push(lua.LNil)
		L.Push(lua.LString(method))
		L.Push(options)
		L.Call(5, 2)
		return 2
	}
}
func requestsHelperBody(method string) func(L *lua.LState) int {
	return func(L *lua.LState) int {
		url, headers, data, options := L.Get(1), L.Get(2), L.Get(3), L.Get(4)
		r := L.GetGlobal("requests").(*lua.LTable).RawGetString("request").(*lua.LFunction)
		L.Push(r)
		L.Push(url)
		L.Push(headers)
		L.Push(data)
		L.Push(lua.LString(method))
		L.Push(options)
		L.Call(5, 2)
		return 2
	}
}

func luaToStringArray(L *lua.LState, lv lua.LValue) []string {
	switch v := lv.(type) {
	case lua.LString:
		return []string{string(v)}
	case *lua.LTable:
		r := make([]string, v.Len())
		for i := 1; i <= v.Len(); i++ {
			r[i-1] = lua.LVAsString(L.ToStringMeta(v.RawGetInt(i)))
		}
		return r
	default:
		return []string{lua.LVAsString(L.ToStringMeta(lv))}
	}
}

func luaRequest(L *lua.LState) int {
	pUrl, pHeaders, pData, pMethod, pOptions := L.Get(1), L.Get(2), L.Get(3), L.Get(4), L.Get(5)
	var urlStr string
	var headers *lua.LTable
	var data string
	var form url.Values
	var method string
	var options *lua.LTable
	if pUrl.Type() == lua.LTString {
		urlStr = string(pUrl.(lua.LString))
	} else {
		L.TypeError(1, lua.LTString)
	}
	if pHeaders == lua.LNil {
		headers = L.NewTable()
	} else if pHeaders.Type() == lua.LTTable {
		headers = pHeaders.(*lua.LTable)
	} else {
		L.TypeError(2, lua.LTTable)
	}
	if pData == lua.LNil {
		data = ""
	} else if pData.Type() == lua.LTString {
		data = string(pData.(lua.LString))
	} else if pData.Type() == lua.LTTable {
		form = make(url.Values)
		pData.(*lua.LTable).ForEach(func(key, value lua.LValue) {
			k := lua.LVAsString(L.ToStringMeta(key))
			v := luaToStringArray(L, value)
			form[k] = v
		})
		data = form.Encode()
	} else {
		L.TypeError(3, lua.LTString)
	}
	if pMethod == lua.LNil {
		method = http.MethodGet
	} else if pMethod.Type() == lua.LTString {
		method = string(pMethod.(lua.LString))
	} else {
		L.TypeError(4, lua.LTString)
	}
	if pOptions == lua.LNil {
		options = L.NewTable()
	} else if pOptions.Type() == lua.LTTable {
		options = pOptions.(*lua.LTable)
	} else {
		L.TypeError(5, lua.LTTable)
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		L.RaiseError("bad URL: %s", err)
	}

	if form != nil && (method == http.MethodGet || method == http.MethodHead) {
		q := parsedURL.Query()
		for k, v := range form {
			q[k] = v
		}
		parsedURL.RawQuery = q.Encode()
		urlStr = parsedURL.String()
	}

	req, err := http.NewRequest(method, urlStr, strings.NewReader(data))
	if err != nil {
		L.RaiseError(err.Error())
	}
	req = req.WithContext(L.Ctx)
	req.Header.Set("User-Agent", "Marvin, bot for 42schoolusa.slack.com")
	headers.ForEach(func(key, value lua.LValue) {
		k := lua.LVAsString(L.ToStringMeta(key))
		v := luaToStringArray(L, value)
		req.Header[k] = v
	})
	_ = options

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		L.RaiseError("%s: %s", urlStr, err.Error())
	}
	L.Push(LNewResponse(L, resp))
	return 1
}

func LNewResponse(L *lua.LState, resp *http.Response) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = resp
	mt := L.NewTable()
	mt.RawSetString("__index", mt)

	headers := L.NewTable()
	for k, v := range resp.Header {
		headers.RawSetString(k, lua.LString(strings.Join(v, ",")))
	}
	mt.RawSetString("headers", headers)
	mt.RawSetString("text", L.NewFunction(luaResponseText))
	mt.RawSetString("json", L.NewFunction(luaResponseJson))
	mt.RawSetString("statuscode", lua.LNumber(resp.StatusCode))
	mt.RawSetString("status", lua.LString(resp.Status))
	mt.RawSetString("proto", lua.LString(resp.Proto))
	ud.Metatable = mt
	return ud
}

func luaResponseText(L *lua.LState) int {
	ud := L.CheckUserData(1)
	resp, ok := ud.Value.(*http.Response)
	if !ok {
		L.RaiseError("bad argument #0 to response:text() - expected instance of response")
	}
	lErr := L.GetField(ud.Metatable, "_bodyErr")
	if lErr != lua.LNil {
		L.Error(lErr, 1)
	}
	text := L.GetField(ud.Metatable, "_bodyText")
	if text == lua.LNil {
		bytes, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			ud.Metatable.(*lua.LTable).RawSetString("_bodyErr", lua.LString(err.Error()))
			L.Error(L.GetField(ud.Metatable, "_bodyErr"), 1)
		}
		text = lua.LString(string(bytes))
		ud.Metatable.(*lua.LTable).RawSetString("_bodyText", text)
	}
	L.Push(text)
	return 1
}

func luaResponseJson(L *lua.LState) int {
	ud := L.CheckUserData(1)
	_, ok := ud.Value.(*http.Response)
	if !ok {
		L.RaiseError("bad argument #0 to response:json() - expected instance of response")
	}
	json := L.GetField(ud.Metatable, "_bodyJson")
	if json == lua.LNil {
		idx := L.GetTop()
		L.Push(L.GetField(ud.Metatable, "text"))
		L.Push(ud)
		L.Call(1, 1)
		txtV := L.Get(idx + 1)
		L.Pop(1)
		jsonLoad := L.GetTable(L.GetGlobal("json"), lua.LString("load"))
		L.Push(jsonLoad)
		L.Push(txtV)
		L.Call(1, 1)
		json = L.Get(idx + 1)
		L.SetField(ud.Metatable, "_bodyJson", json)
		return 1
	}
	L.Push(json)
	return 1
}
