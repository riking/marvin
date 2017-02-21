package lualib

import "github.com/yuin/gopher-lua"

// https://www.googleapis.com/customsearch/v1?key=...&cx=...&q=test

func OpenGoogle(g *G, L *lua.LState) int {
	apikey, isDefault, err := g.Team().ModuleConfig("apikeys").GetIsDefault("googlesearch")
	if isDefault || err != nil {
		return 0
	}
	_ = apikey
	return 0
}
