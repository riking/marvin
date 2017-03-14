package lualib

import (
	"time"

	"github.com/yuin/gopher-lua"
)

// The time module provides access to time formatting functions from Lua.
//
// Module:
// time.rfc3339 -> string: formatting constant
// time.now() -> Time: returns the current time
// time.fromunix(sec, nsec=0) -> Time: creates a new Time from a unix timestamp
//
// Type: Time
//  t.year -> number: year number
//  t.month -> number
//  t.day -> number
//  t.hour -> number
//  t.minute -> number
//  t.second -> number
//  t.ns -> number: nanoseconds into the second
//  t.tz -> string: tzinfo-based timezone string (e.g. Europe/Paris)
//  t.__is_a_time -> bool: type indicator
//  t:format(str) -> string: formats a time according to https://godoc.org/time#pkg-constants
//  t:unix() -> number, number: returns the unix time (seconds, nanoseconds)
func OpenTime(L *lua.LState) int {
	mod := L.NewTable()
	mod.RawSetString("now", L.NewFunction(lua_time_now))
	mod.RawSetString("fromunix", L.NewFunction(lua_time_fromunix))
	mod.RawSetString("rfc3339", lua.LString(time.RFC3339))

	mt := L.NewTypeMetatable("time")
	timeMethods := L.NewTable()
	timeMethods.RawSetString("format", L.NewFunction(lua_time_format))
	timeMethods.RawSetString("unix", L.NewFunction(lua_time_unix))
	mt.RawSetString("__index", timeMethods)

	L.SetGlobal("time", mod)
	return 0
}

func timeToLua(t time.Time, L *lua.LState) lua.LValue {
	tab := L.NewTable()
	y, mo, d := t.Date()
	h, mn, s := t.Clock()
	ns := t.Nanosecond()
	tz := t.Location().String()
	tab.RawSetString("year", lua.LNumber(y))
	tab.RawSetString("month", lua.LNumber(mo))
	tab.RawSetString("day", lua.LNumber(d))
	tab.RawSetString("hour", lua.LNumber(h))
	tab.RawSetString("minute", lua.LNumber(mn))
	tab.RawSetString("second", lua.LNumber(s))
	tab.RawSetString("ns", lua.LNumber(ns))
	tab.RawSetString("tz", lua.LString(tz))
	tab.RawSetString("__is_a_time", lua.LTrue)
	tab.Metatable = L.GetTypeMetatable("time")
	return tab
}

func luaToTime(v lua.LValue, L *lua.LState) time.Time {
	if v.Type() != lua.LTTable {
		L.RaiseError("provided object is not a time (got %s)", v.Type().String())
		return time.Time{}
	}
	tab := v.(*lua.LTable)
	if !L.Equal(tab.RawGetString("__is_a_time"), lua.LTrue) {
		L.RaiseError("provided table is not a time (missing __is_a_time=true)")
		return time.Time{}
	}
	y := int(lua.LVAsNumber(tab.RawGetString("year")))
	mo := time.Month(lua.LVAsNumber(tab.RawGetString("month")))
	d := int(lua.LVAsNumber(tab.RawGetString("day")))
	h := int(lua.LVAsNumber(tab.RawGetString("hour")))
	mn := int(lua.LVAsNumber(tab.RawGetString("minute")))
	s := int(lua.LVAsNumber(tab.RawGetString("second")))
	ns := int(lua.LVAsNumber(tab.RawGetString("ns")))
	tzStr := string(lua.LVAsString(tab.RawGetString("tz")))
	tz, err := time.LoadLocation(tzStr)
	if err != nil {
		tz = time.Local
	}
	return time.Date(y, mo, d, h, mn, s, ns, tz)
}

func lua_time_now(L *lua.LState) int {
	now := time.Now()
	L.Push(timeToLua(now, L))
	return 1
}

func lua_time_fromunix(L *lua.LState) int {
	sec := L.CheckNumber(1)
	ns := L.OptNumber(2, 0)
	t := time.Unix(int64(sec), int64(ns))
	L.Push(timeToLua(t, L))
	return 1
}

func lua_time_format(L *lua.LState) int {
	luaTime := L.Get(1)
	fmt := L.CheckString(2)
	t := luaToTime(luaTime, L)

	L.Push(lua.LString(t.Format(fmt)))
	return 1
}

func lua_time_unix(L *lua.LState) int {
	luaTime := L.Get(1)
	t := luaToTime(luaTime, L)

	L.Push(lua.LNumber(t.Unix()))
	L.Push(lua.LNumber(t.UnixNano()))
	return 2
}
