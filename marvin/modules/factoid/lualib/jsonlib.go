package lualib

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/yuin/gopher-lua"
)

func OpenJson(L *lua.LState) int {
	module := L.RegisterModule("json", jsonFuncs).(*lua.LTable)
	null := L.NewUserData()
	tab := L.NewTypeMetatable("_metatable_JsonNull")
	tab.RawSetString("__tostring", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("(null)"))
		return 1
	}))
	null.Value = nil
	null.Metatable = tab
	L.SetGlobal("jsonNull", null)
	module.RawSetString("null", null)

	isArray := L.NewTable()
	module.RawSetString("mt_isarray", isArray)
	isObject := L.NewTable()
	module.RawSetString("mt_isobject", isObject)

	L.SetGlobal("json", module)
	L.Push(module)
	return 1
}

var jsonFuncs = map[string]lua.LGFunction{
	"load": jsonLoad,
	"dump": jsonDump,
}

func jsonToLua(L *lua.LState, obj interface{}) lua.LValue {
	switch v := obj.(type) {
	case nil:
		return lua.LNil
	case bool:
		if v {
			return lua.LTrue
		}
		return lua.LFalse
	case float64:
		return lua.LNumber(v)
	case string:
		return lua.LString(v)
	case []interface{}:
		t := L.NewTable()
		for i, val := range v {
			t.RawSetInt(i+1, jsonToLua(L, val))
		}
		t.Metatable = L.GetTable(L.GetGlobal("json"), lua.LString("mt_isarray"))
		return t
	case map[string]interface{}:
		t := L.NewTable()
		for k, val := range v {
			t.RawSetString(k, jsonToLua(L, val))
		}
		t.Metatable = L.GetTable(L.GetGlobal("json"), lua.LString("mt_isobject"))
		return t
	default:
		panic(errors.Errorf("Unrecognized json decode type %T", obj))
	}
}

func jsonFromLua(L *lua.LState, obj lua.LValue) interface{} {
	switch obj.Type() {
	case lua.LTNil:
		return nil
	case lua.LTBool:
		v := obj.(lua.LBool)
		return bool(v)
	case lua.LTNumber:
		v := obj.(lua.LNumber)
		return float64(v)
	case lua.LTString:
		v := obj.(lua.LString)
		return string(v)
	case lua.LTUserData:
		v := obj.(*lua.LUserData)
		return v.Value
	case lua.LTTable:
		tab := obj.(*lua.LTable)
		isArray := false
		isOOBNumbers := false
		isHash := false
		var maxNum int64 = 0

		tab.ForEach(func(key lua.LValue, _ lua.LValue) {
			switch k := key.(type) {
			case lua.LNumber:
				isArray = true
				if k < 0 || k > 1000*1000 {
					isOOBNumbers = true
					return
				}
				maxNum = int64(k)
			case lua.LString:
				isHash = true
			default:
				L.RaiseError("Can't encode key of type %s", L.Type().String())
			}
		})
		if tab.Metatable == L.GetTable(L.GetGlobal("json"), lua.LString("mt_isobject")) {
			isHash = true
			isArray = false
		} else if tab.Metatable == L.GetTable(L.GetGlobal("json"), lua.LString("mt_isarray")) {
			isHash = false
			isArray = true
		}

		if !isHash && !isOOBNumbers && isArray {
			// Numeric-only
			r := make([]interface{}, tab.Len())
			for i := 0; i < tab.Len(); i++ {
				r[i] = jsonFromLua(L, tab.RawGetInt(i+1))
			}
			return r
		}
		r := make(map[string]interface{})
		tab.ForEach(func(key lua.LValue, value lua.LValue) {
			r[lua.LVAsString(L.ToStringMeta(key))] = jsonFromLua(L, value)
		})
		return r
	case lua.LTThread, lua.LTFunction, lua.LTChannel:
		fallthrough
	default:
		L.RaiseError("Can't encode value of type %s", L.Type().String())
		return nil
	}
}

func jsonLoad(L *lua.LState) int {
	str := L.CheckString(1)
	if len(str) == 0 {
		L.Push(lua.LNil)
		return 1
	}

	var obj interface{}
	err := json.Unmarshal([]byte(str), &obj)
	if err != nil {
		L.RaiseError("json load error: %s", err)
		return 0
	}

	var ret lua.LValue
	ret = jsonToLua(L, obj)
	L.Push(ret)
	return 1
}

func jsonDump(L *lua.LState) int {
	val := L.CheckAny(1)
	var obj interface{}
	obj = jsonFromLua(L, val)
	bytes, err := json.Marshal(obj)
	if err != nil {
		L.RaiseError("json dump error: %s", err)
		return 0
	}
	L.Push(lua.LString(bytes))
	return 1
}
