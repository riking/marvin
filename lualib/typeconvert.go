package lualib

import (
	"encoding/json"
	"reflect"

	"fmt"
	"os"
	"time"

	"github.com/yuin/gopher-lua"
)

var lvType = reflect.TypeOf(lua.LValue(lua.LNumber(1)))
var errType = reflect.TypeOf(error(os.ErrInvalid))
var timeTimeType = reflect.TypeOf(time.Now())
var jsonRawMsgType = reflect.TypeOf(json.RawMessage([]byte("")))

func GoToLua(L *lua.LState, v interface{}) lua.LValue {
	return goToLua(L, v)
}

func goToLua(L *lua.LState, v interface{}) lua.LValue {
	var val reflect.Value
	var ok bool
	// check if it's already a reflect.Value
	if val, ok = v.(reflect.Value); !ok {
		val = reflect.ValueOf(v)
	}
	typ := val.Type()

	defer func() {
		if rErr := recover(); rErr != nil {
			fmt.Println("[GoToLua] crashed on type", typ.String())
			panic(rErr)
		}
	}()

	if typ.Kind() == reflect.Ptr {
		if val.IsNil() {
			return lua.LNil
		}
		return goToLua(L, val.Elem())
	}

	// Special-case types
	if typ.ConvertibleTo(lvType) {
		convVal := val.Convert(lvType)
		return convVal.Interface().(lua.LValue)
	}
	if typ.ConvertibleTo(errType) {
		return lua.LString(val.Interface().(error).Error())
	}
	if typ.AssignableTo(jsonRawMsgType) {
		var i interface{}
		json.Unmarshal(val.Interface().(json.RawMessage), &i)
		return jsonToLua(L, i)
	}
	if typ.AssignableTo(timeTimeType) {
		tim := val.Interface().(time.Time)
		return lua.LNumber(tim.Unix()) + lua.LNumber(tim.UnixNano())/1000000000
	}

	switch typ.Kind() {
	case reflect.Ptr:
		panic("Impossible control flow")
	case reflect.Interface:
		if val.IsNil() {
			return lua.LNil
		}
		return goToLua(L, val.Elem())
	case reflect.Float32, reflect.Float64:
		return lua.LNumber(val.Float())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return lua.LNumber(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return lua.LNumber(val.Uint())
	case reflect.String:
		return lua.LString(val.String())
	case reflect.Bool:
		if val.Bool() {
			return lua.LTrue
		} else {
			return lua.LFalse
		}
	case reflect.Array, reflect.Slice:
		tab := L.NewTable()
		length := val.Len()
		for i := 0; i < length; i++ {
			tab.RawSetInt(i+1, goToLua(L, val.Index(i)))
		}
		return tab
	case reflect.Map:
		tab := L.NewTable()
		keys := val.MapKeys()
		for _, k := range keys {
			tab.RawSet(goToLua(L, k), goToLua(L, val.MapIndex(k)))
		}
		return tab
	case reflect.Struct:
		tab := L.NewTable()
		n := val.NumField()
		for i := 0; i < n; i++ {
			field := typ.Field(i)
			key := field.Name
			tag := field.Tag.Get("lua")
			if tag != "" {
				key = tag
			}
			tab.RawSetString(key, goToLua(L, val.Field(i)))
		}
		return tab
	case reflect.Invalid, reflect.Func, reflect.Chan, reflect.Complex64, reflect.Complex128:
		L.RaiseError("Cannot convert value of type %s", typ.String())
		return lua.LNil
	default:
		L.RaiseError("Cannot convert value of type %s", typ.String())
		return lua.LNil
	}
}
