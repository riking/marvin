package factoid

import (
	"bytes"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/riking/marvin/lualib"
	"github.com/riking/marvin/util"
	"github.com/yuin/gopher-lua"
)

const (
	sqlMigrate3 = `CREATE TABLE module_factoid_data (
		id    SERIAL PRIMARY KEY,
		map   text NOT NULL,
		key   text NOT NULL,
		data  json,

		UNIQUE(map, key)
	)`

	// $1 = map $2 = key
	sqlFDataGetOne = `SELECT data FROM module_factoid_data WHERE map = $1 AND key = $2`

	// $1 = map
	sqlFDataGetAll = `SELECT key, data FROM module_factoid_data WHERE map = $1`

	_sqlFDataTempTableName = `module_factoid_data_load`
	sqlFDataMakeTempTable  = `CREATE TEMPORARY TABLE module_factoid_data_load
	(map text NOT NULL, key text NOT NULL, data json)
	ON COMMIT DROP`

	// (!!) No syntax checks
	sqlFDataSetFromTemp = `INSERT INTO module_factoid_data (map, key, data)
	SELECT map, key, data FROM module_factoid_data_load
	ON CONFLICT (map, key) DO UPDATE SET data = EXCLUDED.data`

	// TODO use this when volume low
	sqlFDataSet = `INSERT INTO module_factoid_data (map, key, data)
	VALUES ($1, $2, $3)
	ON CONFLICT (map, key) DO UPDATE SET data = EXCLUDED.data`
)

const (
	fdataKeyMaxLen = 500
	fdataValMaxLen = 1024 * 40
)

type fdataKey struct {
	Map string
	Key string
}

type fdataVal struct {
	JSON []byte

	// DBSync value meanings:
	//  - TriYes when the value is fresh with the database.
	//  - TriNo when the value is dirty and needs to be saved.
	//  - TriDefault when the entry is missing
	DBSync util.TriValue
}

type fdataReqFunc func(*FactoidModule) interface{}

type fdataReq struct {
	C chan interface{}
	F fdataReqFunc
}

func (mod *FactoidModule) workerFDataChan() {
	for req := range mod.fdataReqChan {
		req.C <- req.F(mod)
	}
}

func (mod *FactoidModule) workerFDataSync() {
	var (
		responseChan = make(chan interface{})
		timerChan    <-chan time.Time
		updateData   map[fdataKey][]byte
		doneWaiting  bool
	)

	for {
		// Wait for dirty value write
		q := <-mod.fdataSyncSignal

		// Grace period (skipped if q was true)
		timerChan = time.After(30 * time.Second)
		doneWaiting = q
		for !doneWaiting {
			select {
			case q = <-mod.fdataSyncSignal:
				if q == true {
					doneWaiting = true
				}
			case <-timerChan:
				doneWaiting = true
			}
		}
		timerChan = nil

		fmt.Println("[fdata] Saving factoid data...")

		// Gather all dirty data
		updateData = make(map[fdataKey][]byte)
		mod.fdataReqChan <- fdataReq{C: responseChan,
			F: func(mod *FactoidModule) interface{} {
				var key fdataKey
				for mapName, mapContent := range mod.fdataMap {
					key.Map = mapName
					for keyName, keyContent := range mapContent {
						if keyContent.DBSync == util.TriYes {
							continue
						}
						key.Key = keyName
						updateData[key] = keyContent.JSON
					}
				}
				return nil
			},
		}
		<-responseChan

		// Write to database
		err := mod.fdataSaveToDBBulk(updateData)
		if err != nil {
			util.LogError(errors.Wrap(err, "Failed saving factoid persistent data"))
		}

		fmt.Println("[fdata] Marking factoid data as saved")
		// Flag data in the cache as saved
		mod.fdataReqChan <- fdataReq{C: responseChan,
			F: func(mod *FactoidModule) interface{} {
				if err != nil {
					return nil
				}
				for key, val := range updateData {
					m := mod.fdataMap[key.Map]
					if m == nil {
						continue
					}
					q := m[key.Key]
					if bytes.Equal(m[key.Key].JSON, val) {
						q.DBSync = util.TriYes
						m[key.Key] = q
					} else {
						q.DBSync = util.TriNo
						m[key.Key] = q
					}
				}
				return nil
			},
		}
		<-responseChan
		fmt.Println("[fdata] mark done")
	}
}

func (mod *FactoidModule) fdataSaveToDBBulk(data map[fdataKey][]byte) (err error) {
	tx, err := mod.team.DB().Begin()
	if err != nil {
		return errors.Wrap(err, "begin transaction")
	}
	txOK := false
	defer func() {
		if !txOK {
			tx.Rollback()
		}
	}()

	_, err = tx.Exec(sqlFDataMakeTempTable) // CREATE TEMPORARY TABLE ... ON COMMIT DROP
	if err != nil {
		return errors.Wrap(err, "create temporary table")
	}

	stmt, err := tx.Prepare(pq.CopyIn(_sqlFDataTempTableName, "map", "key", "data"))
	for key, val := range data {
		if val == nil {
			_, err = stmt.Exec(string(key.Map), string(key.Key), nil)
		} else {
			_, err = stmt.Exec(string(key.Map), string(key.Key), string(val))
		}

		if err != nil {
			return errors.Wrap(err, "loading COPY data")
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return errors.Wrap(err, "flush COPY data")
	}
	err = stmt.Close()
	if err != nil {
		return errors.Wrap(err, "close COPY stmt")
	}

	_, err = tx.Exec(sqlFDataSetFromTemp) // INSERT INTO ... SELECT ... ON CONFLICT DO UPDATE ...
	if err != nil {
		return errors.Wrap(err, "move from temporary to real table")
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "commit transaction")
	}
	txOK = true
	return nil
}

func (mod *FactoidModule) fdataTriggerSync() {
	select {
	case mod.fdataSyncSignal <- false:
	default:
		// Already working on it ¯\_(ツ)_/¯
	}
}

func (mod *FactoidModule) GetFDataAll(mapName string) (map[string][]byte, error) {
	stmt, err := mod.team.DB().Prepare(sqlFDataGetAll)
	if err != nil {
		return nil, errors.Wrap(err, "prepare stmt")
	}
	rows, err := stmt.Query(mapName)
	if err != nil {
		return nil, errors.Wrapf(err, "fdata query map=%s", mapName)
	}

	result := make(map[string][]byte)
	var b []byte
	var key string

	for rows.Next() {
		err = rows.Scan(&key, &b)
		if err != nil {
			return nil, errors.Wrapf(err, "fdata scan map=%s", mapName)
		}
		if len(b) == 0 {
			result[key] = nil
		} else {
			result[key] = b
		}
	}

	if rows.Err() != nil {
		return nil, errors.Wrapf(err, "fdata query map=%s", mapName)
	}

	// Fill in dirty values that haven't hit the DB
	mod.fdataFreshenMap(mapName, result)
	return result, nil
}

func (mod *FactoidModule) fdataDBGetOne(mapName, keyName string) ([]byte, error) {
	stmt, err := mod.team.DB().Prepare(sqlFDataGetOne)
	if err != nil {
		return nil, errors.Wrap(err, "prepare stmt")
	}
	var data []byte
	row := stmt.QueryRow(mapName, keyName)
	err = row.Scan(&data)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	return data, nil
}

// fdataVal states:
//  - DBSync is 0, JSON is nil - unknown status, check the DB
//  - DBSync is yes/no, JSON is nil - no value in DB
//  - DBSync is yes/no, JSON not nil - value
func (mod *FactoidModule) fdataGetCachedEntry(mapName, keyName string) fdataVal {
	ch := make(chan interface{})
	mod.fdataReqChan <- fdataReq{C: ch,
		F: func(mod *FactoidModule) interface{} {
			m := mod.fdataMap[mapName]
			if m == nil {
				return fdataVal{JSON: nil, DBSync: util.TriDefault}
			}
			return m[keyName]
		},
	}
	return (<-ch).(fdataVal)
}

func (mod *FactoidModule) fdataFreshenMap(mapName string, content map[string][]byte) {
	ch := make(chan interface{})
	mod.fdataReqChan <- fdataReq{C: ch,
		F: func(mod *FactoidModule) interface{} {
			m := mod.fdataMap[mapName]
			if m == nil {
				return nil
			}
			for k, v := range m {
				content[k] = v.JSON
			}
			return nil
		},
	}
	<-ch
	return
}

func fdataFuncSetEntry(mapName, keyName string, data []byte) fdataReqFunc {
	return func(mod *FactoidModule) interface{} {
		m := mod.fdataMap[mapName]
		if m == nil {
			m = make(map[string]fdataVal)
			mod.fdataMap[mapName] = m
		}
		m[keyName] = fdataVal{
			JSON:   data,
			DBSync: util.TriNo,
		}

		mod.fdataTriggerSync()
		return nil
	}
}

func fdataFuncStoreCache(mapName, keyName string, data []byte) fdataReqFunc {
	return func(mod *FactoidModule) interface{} {
		m := mod.fdataMap[mapName]
		if m == nil {
			m = make(map[string]fdataVal)
			mod.fdataMap[mapName] = m
		}
		m[keyName] = fdataVal{
			JSON:   data,
			DBSync: util.TriYes,
		}
		return nil
	}
}

func (mod *FactoidModule) GetFDataValue(mapName, keyName string) ([]byte, error) {
	fval := mod.fdataGetCachedEntry(mapName, keyName)
	if fval.DBSync != util.TriDefault {
		return fval.JSON, nil
	}
	b, err := mod.fdataDBGetOne(mapName, keyName)
	if err != nil {
		return nil, err
	}
	ch := make(chan interface{}, 1)
	mod.fdataReqChan <- fdataReq{C: ch, F: fdataFuncStoreCache(mapName, keyName, b)}
	return b, nil
}

func (mod *FactoidModule) SetFDataValue(mapName, keyName string, val []byte) {
	if mod.team.TeamConfig().IsReadOnly {
		return
	}
	ch := make(chan interface{}, 1)
	mod.fdataReqChan <- fdataReq{C: ch, F: fdataFuncSetEntry(mapName, keyName, val)}
	return
}

type LFDataMap struct {
	mod     *FactoidModule
	MapName string

	lcache      *lua.LTable
	fullContent map[string][]byte
}

const metatableFDataMap = "_metatable_FDataMap"

func (*LFDataMap) SetupMetatable(L *lua.LState) {
	tab := L.NewTypeMetatable(metatableFDataMap)
	tab.RawSetString("__index", L.NewFunction(luaFData_get))
	tab.RawSetString("__newindex", L.NewFunction(luaFData_set))
	tab.RawSetString("__len", L.NewFunction(luaFData_count))
	tab.RawSetString("__preload", L.NewFunction(luaFData_preload))

	L.SetGlobal("iterfdata", L.NewFunction(luaFData_iter))
	L.SetGlobal("iterfmap", L.NewFunction(luaFData_iter))
}

func LNewFDataMap(g *lualib.G, mod *FactoidModule, mapName string) lua.LValue {
	v := &LFDataMap{mod: mod, MapName: mapName}
	v.lcache = g.L.NewTable()
	u := g.L.NewUserData()
	u.Value = v
	u.Metatable = g.L.GetTypeMetatable(metatableFDataMap)
	return u
}

func luaFData_get(L *lua.LState) int {
	u := L.CheckUserData(1)
	kv := L.Get(2)
	if kv.Type() != lua.LTString {
		return 0
	}
	key := L.CheckString(2)
	fdm, ok := u.Value.(*LFDataMap)
	if !ok {
		L.RaiseError("Wrong self for FDataMap.__index, got %T", u.Value)
	}

	if len(key) > fdataKeyMaxLen {
		L.RaiseError("fdata keys cannot be longer than %d", fdataKeyMaxLen)
	}

	lv := fdm.lcache.RawGetString(key)
	if lv != lua.LNil {
		L.Push(lv)
		return 1
	}

	var jsonBytes []byte
	if fdm.fullContent != nil {
		b, ok := fdm.fullContent[key]
		if !ok {
			L.Push(lua.LNil)
			return 1
		}
		jsonBytes = b
	} else {
		b, err := fdm.mod.GetFDataValue(fdm.MapName, key)
		if err != nil {
			L.RaiseError("Factoid map error: %s", err)
		}
		jsonBytes = b
	}

	top := L.GetTop()
	L.Push(L.GetTable(L.GetGlobal("json"), lua.LString("load")))
	L.Push(lua.LString(jsonBytes))
	L.Call(1, 1)
	fdm.lcache.RawSetString(key, L.Get(top+1))
	return 1
}

func luaFData_set(L *lua.LState) int {

	u := L.CheckUserData(1)
	kv := L.Get(2)
	if kv.Type() != lua.LTString {
		L.RaiseError("FactoidData only allows string-typed keys. To store an array, choose a name for it.")
		return 0
	}
	key := L.CheckString(2)
	val := L.CheckAny(3)
	fdm, ok := u.Value.(*LFDataMap)
	if !ok {
		L.RaiseError("Wrong self for FDataMap.__newindex, got %T", u.Value)
	}

	if len(key) > fdataKeyMaxLen {
		L.RaiseError("fdata keys cannot be longer than %d", fdataKeyMaxLen)
	}

	fdm.lcache.RawSetString(key, val)

	if val == lua.LNil {
		if fdm.fullContent != nil {
			delete(fdm.fullContent, key)
		}
		fdm.mod.SetFDataValue(fdm.MapName, key, nil)
		return 0
	}

	top := L.GetTop()
	L.Push(L.GetTable(L.GetGlobal("json"), lua.LString("dump")))
	L.Push(val)
	L.Call(1, 1)
	jsonLV := L.Get(top + 1)
	if jsonLV.Type() != lua.LTString {
		L.RaiseError("bad return from json.dump (expected lua string, got %s)", jsonLV.Type())
	}
	jsonData := []byte(jsonLV.(lua.LString))
	if len(jsonData) > fdataValMaxLen {
		// Need additional checks to have an actual quota
		L.RaiseError("fdata values cannot be larger than %d bytes (JSON)", fdataValMaxLen)
	}
	if fdm.fullContent != nil {
		fdm.fullContent[key] = jsonData
	}
	fdm.mod.SetFDataValue(fdm.MapName, key, jsonData)
	return 0
}

func luaFData_preload(L *lua.LState) int {
	u := L.CheckUserData(1)
	fdm, ok := u.Value.(*LFDataMap)
	if !ok {
		L.RaiseError("Wrong self for FDataMap.__index, got %T", u.Value)
	}

	if fdm.fullContent != nil {
		return 0
	}

	m, err := fdm.mod.GetFDataAll(fdm.MapName)
	if err != nil {
		L.RaiseError("preload(): database error: %s", err)
	}
	fdm.lcache = L.NewTable()
	fdm.fullContent = m
	return 0
}

func luaFData_count(L *lua.LState) int {
	u := L.CheckUserData(1)
	fdm, ok := u.Value.(*LFDataMap)
	if !ok {
		L.RaiseError("fmap.__len requires a fmap as argument #1, got %T", u.Value)
	}

	// preload full table
	if fdm.fullContent == nil {
		L.Push(L.GetTable(u.Metatable, lua.LString("__preload")))
		L.Push(u)
		L.Call(1, 0)
	}

	L.Push(lua.LNumber(len(fdm.fullContent)))
	return 1
}

func luaFData_iter(L *lua.LState) int {
	u := L.CheckUserData(1)
	fdm, ok := u.Value.(*LFDataMap)
	if !ok {
		L.RaiseError("iterfdata() requires a fmap as argument #1, got %T", u.Value)
	}

	// preload full table
	if fdm.fullContent == nil {
		L.Push(L.GetTable(u.Metatable, lua.LString("__preload")))
		L.Push(u)
		L.Call(1, 0)
	}

	keys := make([]string, 0, len(fdm.fullContent))
	for k := range fdm.fullContent {
		keys = append(keys, k)
	}
	var idx int = 0
	cl := L.NewFunction(func(L *lua.LState) int {
		if idx >= len(keys) {
			return 0
		}
		k := keys[idx]
		idx++
		val := L.GetField(u, k)
		L.Push(lua.LString(k))
		L.Push(val)
		return 2
	})
	L.Push(cl)
	L.Push(u)
	L.Push(lua.LNumber(0))
	return 3
}
