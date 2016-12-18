package lualib

import (
	"encoding/json"
	"fmt"

	"github.com/dariusk/corpora"
	"github.com/pkg/errors"
	"github.com/yuin/gopher-lua"

	"github.com/riking/homeapi/marvin/util"
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
	b, err := corpora.Asset(fmt.Sprintf("data/%s.json", key))
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

func init() {
	corpusGlobal = corpusCache{
		dataMap: make(map[string]CorpusData),
		reqCh:   make(chan corpusReq),
	}
	go corpusGlobal.worker()
}

func OpenCorpus(L *lua.LState) int {
	module := L.RegisterModule("corpus", map[string]lua.LGFunction{}).(*lua.LTable)
	mt := L.NewTable()
	module.Metatable = mt
	mt.RawSetString("__index", L.NewFunction(corpusGlobal.lua_Get))
	L.Push(module)
	return 1
}

func (c *corpusCache) lua_Get(L *lua.LState) int {
	req := corpusReq{
		key:   L.CheckString(1),
		reply: make(chan CorpusData),
	}
	c.reqCh <- req
	data := <-req.reply
	ary := L.NewTable()
	for i, v := range data.Content {
		ary.RawSetInt(i+1, lua.LString(v))
	}
	L.Push(lua.LValue(ary))
	return 1
}
