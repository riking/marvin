package lualib

import (
	"net/url"
	"time"

	"github.com/yuin/gopher-lua"

	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
)

type pasteAPI interface {
	marvin.Module

	CreatePaste(content string) (int64, error)
	GetPaste(id int64) (string, error)
	URLForPaste(id int64) string

	CreateLink(content string) (int64, error)
	GetLink(id int64) (string, error)
	URLForLink(id int64) string
}

func OpenBot(team marvin.Team) func(L *lua.LState) int {
	pasteModule := team.GetModule("paste").(pasteAPI)

	return func(L *lua.LState) int {
		tab := L.NewTable()
		tab.RawSetString("now", L.NewFunction(func(L *lua.LState) int {
			L.Push(lua.LNumber(time.Now().Unix()))
			return 1
		}))
		tab.RawSetString("uriencode", L.NewFunction(func(L *lua.LState) int {
			str := L.CheckString(1)
			L.Push(lua.LString(url.QueryEscape(str)))
			return 1
		}))
		tab.RawSetString("uridecode", L.NewFunction(func(L *lua.LState) int {
			str := L.CheckString(1)
			result, err := url.QueryUnescape(str)
			if err != nil {
				L.RaiseError("non-encoded string passed to uridecode: %s", err)
			}
			L.Push(lua.LString(result))
			return 1
		}))
		tab.RawSetString("unescape", L.NewFunction(func(L *lua.LState) int {
			str := L.CheckString(1)
			L.Push(lua.LString(slack.UnescapeTextAll(str)))
			return 1
		}))
		tab.RawSetString("unichr", L.NewFunction(func(L *lua.LState) int {
			n := L.CheckNumber(1)
			L.Push(lua.LString(string(rune(n))))
			return 1
		}))
		tab.RawSetString("paste", L.NewFunction(func(L *lua.LState) int {
			if pasteModule == nil {
				L.RaiseError("paste module not available")
			}
			str := L.CheckString(1)
			id, err := pasteModule.CreatePaste(str)
			if err != nil {
				L.RaiseError("paste() failed: %s", err)
			}
			pasteURL := pasteModule.URLForPaste(id)
			L.Push(lua.LString(pasteURL))
			return 1
		}))
		tab.RawSetString("shortlink", L.NewFunction(func(L *lua.LState) int {
			if pasteModule == nil {
				L.RaiseError("paste module not available")
			}
			str := L.CheckString(1)
			id, err := pasteModule.CreateLink(str)
			if err != nil {
				L.RaiseError("paste() failed: %s", err)
			}
			pasteURL := pasteModule.URLForLink(id)
			L.Push(lua.LString(pasteURL))
			return 1
		}))

		L.SetGlobal("bot", tab)
		return 0
	}
}
