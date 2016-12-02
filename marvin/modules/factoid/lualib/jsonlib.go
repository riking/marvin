package lualib

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/yuin/gopher-lua"
)

func OpenJson(L *lua.LState) int {
	module := L.NewTable()
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

}

var jsonFuncs = map[string]lua.LGFunction{
	"load": jsonLoad,
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
		return t
	case map[string]interface{}:
		t := L.NewTable()
		for k, val := range v {
			t.RawSetString(k, jsonToLua(L, val))
		}
		return t
	default:
		panic(errors.Errorf("Unrecognized json decode type %T", obj))
	}
}

func jsonFromLua(L *lua.LState, obj lua.LValue) interface{} {
	specialNull := L.GetGlobal("jsonNull")

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
		v := obj.(*lua.LTable)
		isArray := false
		isOOBNumbers := false
		isHash := false
		var maxNum int64 = 0

		v.ForEach(func(key lua.LValue, _ lua.LValue) {
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
				L.RaiseError("Can't encode key of type", L.Type().String())
			}
		})

		if !isHash && !isOOBNumbers && isArray {
			// Numeric-only
			r := make([]interface{})

		}
	}
}

func jsonLoad(L *lua.LState) int {
	str := L.CheckString(1)
	var obj interface{}
	err := json.Unmarshal([]byte(str), &obj)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("json load error: %s", err)))
		return 2
	}
	var ret lua.LValue
}
