package lualib

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/yuin/gopher-lua"

	"github.com/riking/marvin/util"
)

type corpusReq struct {
	key   string
	reply chan CorpusData
}

type CorpusData struct {
	Description string
	Field       string
	Source      string
	Content     []string
}

type corpusCache struct {
	dataMap map[string]CorpusData
	reqCh   chan corpusReq
	index   []string
}

var corpusGlobal corpusCache

func (c *corpusCache) worker() {
	for req := range c.reqCh {
		data, ok := c.dataMap[req.key]
		if ok {
			req.reply <- data
		} else {
			data = c.loadData(req.key)
			if data.Content == nil {
				// Error
				req.reply <- CorpusData{}
			} else {
				// Success, save
				c.dataMap[req.key] = data
				req.reply <- data
			}
		}
	}
}

func (c *corpusCache) loadData(key string) (result CorpusData) {
	b, err := corporaAsset(fmt.Sprintf("data/%s.json", key))
	if err != nil {
		return
	}
	var genericJson map[string]interface{}
	err = json.Unmarshal(b, &genericJson)
	if err != nil {
		util.LogError(errors.Wrapf(err, "error unmarshaling corpora data %s", key))
		return
	}
	var genericArray []interface{}
	for k, v := range genericJson {
		switch v := v.(type) {
		case []interface{}:
			result.Field = k
			genericArray = v
		case string:
			if k == "description" {
				result.Description = v
			} else if k == "source" {
				result.Source = v
			}
		}
	}
	if genericArray == nil {
		util.LogError(errors.Errorf("Could not find array for corpora data %s", key))
		return
	}
	var stringArray []string
	stringArray = make([]string, 0, len(genericArray))
	for i, v := range genericArray {
		stringArray[i] = v.(string)
	}
	result.Content = stringArray
	return result
}

func corporaListChildren(path string) []string {
	var children []string
	first, err := corporaAssetDir(path)
	if err != nil {
		return nil // either not exist or not a directory
	}
	for _, v := range first {
		childPath := fmt.Sprintf("%s/%s", path, v)
		second := corporaListChildren(childPath)
		if second == nil {
			children = append(children, childPath)
		} else if second != nil {
			children = append(children, second...)
		}
	}
	return children
}

func init() {
	listing := corporaListChildren("data")
	for i, v := range listing {
		listing[i] = listing[i][len("data/") : len(v)-len(".json")]
	}
	corpusGlobal = corpusCache{
		dataMap: make(map[string]CorpusData),
		reqCh:   make(chan corpusReq),
		index:   listing,
	}

	go corpusGlobal.worker()
}

func OpenCorpus(L *lua.LState) int {
	module := L.RegisterModule("corpus", map[string]lua.LGFunction{}).(*lua.LTable)

	mt := L.NewTable()
	mt.RawSetString("__index", L.NewFunction(corpusGlobal.lua_Get))
	module.Metatable = mt

	infoMT := L.NewTable()
	infoMT.RawSetString("__index", L.NewFunction(corpusGlobal.lua_GetInfo))
	infoT := L.NewTable()
	infoT.Metatable = infoMT
	module.RawSetString("info", infoT)

	listing := L.NewTable()
	for i, v := range corpusGlobal.index {
		listing.RawSetInt(i+1, lua.LString(v))
	}
	module.RawSetString("index", listing)

	L.Push(module)
	return 1
}

func (c *corpusCache) lua_Get(L *lua.LState) int {
	corpusTable := L.CheckTable(1)
	key := L.CheckString(2)
	req := corpusReq{
		key:   key,
		reply: make(chan CorpusData),
	}
	c.reqCh <- req
	data := <-req.reply
	if data.Content == nil {
		return 0
	}
	tab := L.NewTable()
	for i, v := range data.Content {
		tab.RawSetInt(i+1, lua.LString(v))
	}
	L.Push(tab)
	corpusTable.RawSetString(key, tab)
	return 1
}

func (c *corpusCache) lua_GetInfo(L *lua.LState) int {
	_ = L.CheckTable(1)
	req := corpusReq{
		key:   L.CheckString(2),
		reply: make(chan CorpusData),
	}
	c.reqCh <- req
	data := <-req.reply
	info := L.NewTable()
	info.RawSetString("field", lua.LString(data.Field))
	info.RawSetString("description", lua.LString(data.Description))
	info.RawSetString("source", lua.LString(data.Source))
	info.RawSetString("length", lua.LNumber(len(data.Content)))
	L.Push(info)
	return 1
}
