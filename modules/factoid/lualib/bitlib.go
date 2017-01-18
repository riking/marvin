package lualib

import (
	"strconv"

	"github.com/yuin/gopher-lua"
)

func OpenBit(L *lua.LState) int {
	var mod lua.LValue

	mod = L.RegisterModule("bit", bitFuncs)
	L.SetGlobal("bit", mod)
	L.Push(mod)
	return 1
}

var bitFuncs = map[string]lua.LGFunction{
	"tobit":   bitToBit,
	"tohex":   bitToHex,
	"bnot":    bitBNot,
	"band":    bitBAnd,
	"bor":     bitBOr,
	"bxor":    bitBXor,
	"lshift":  bitLShift,
	"rshift":  bitRShift,
	"arshift": bitARShift,
}

func bitToBit(L *lua.LState) int {
	num := L.CheckInt64(1)
	L.Push(lua.LNumber(int32(num)))
	return 1
}

func bitToHex(L *lua.LState) int {
	L.Push(lua.LString(strconv.FormatInt(int64(int32(L.CheckInt64(1))), 16)))
	return 1
}

func bitBNot(L *lua.LState) int {
	L.Push(lua.LNumber(^int32(L.CheckInt64(1))))
	return 1
}

func bitBAnd(L *lua.LState) int {
	L.Push(lua.LNumber(int32(L.CheckInt64(1)) & int32(L.CheckInt64(2))))
	return 1
}

func bitBOr(L *lua.LState) int {
	L.Push(lua.LNumber(int32(L.CheckInt64(1)) | int32(L.CheckInt64(2))))
	return 1
}

func bitBXor(L *lua.LState) int {
	L.Push(lua.LNumber(int32(L.CheckInt64(1)) ^ int32(L.CheckInt64(2))))
	return 1
}

func bitLShift(L *lua.LState) int {
	L.Push(lua.LNumber(int32(L.CheckInt64(1)) << uint(L.CheckInt(2))))
	return 1
}

func bitRShift(L *lua.LState) int {
	L.Push(lua.LNumber(int32(
		uint32(L.CheckInt64(1)) >> uint(L.CheckInt(2)),
	)))
	return 1
}

func bitARShift(L *lua.LState) int {
	L.Push(lua.LNumber(int32(
		int32(L.CheckInt64(1)) >> uint(L.CheckInt(2)),
	)))
	return 1
}
