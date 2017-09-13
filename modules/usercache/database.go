package usercache

import (
	"encoding/json"
	"fmt"

	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/slack/rtm"
)

const (
	sqlMigrate1 = `CREATE TABLE module_user_cache (
		user_id           varchar(15) PRIMARY KEY NOT NULL,
		data              text

		UNIQUE(user_id)
	)`

	sqlGetAllEntries = `SELECT * FROM module_user_cache`

	// $1 = slack.UserID
	sqlGetEntry = `SELECT data FROM module_user_cache WHERE user_id = $1`

	// $1 = slack.UserID
	// $2 = data (json encoded)
	sqlUpsertEntry = `INSERT INTO module_user_cache (user_id,data) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET data = EXCLUDED.data`
)

func (mod *UserCacheModule) GetEntry(userid slack.UserID) (slack.User, error) {
	var entry slack.User

	var data string
	stmt, err := mod.team.DB().Prepare(sqlGetEntry)
	if err != nil {
		return entry, nil
	}
	defer stmt.Close()
	row := stmt.QueryRow(userid)
	err = row.Scan(&data)
	if err != nil {
		return entry, nil
	}
	err = json.Unmarshal([]byte(userid), &entry)
	if err != nil {
		return entry, nil
	}
	return entry, nil
}

func (mod *UserCacheModule) LoadEntries() error {
	stmt, err := mod.team.DB().Query(sqlGetAllEntries)
	if err != nil {
		return err
	}

	rtmClient := mod.team.GetRTMClient().(*rtm.Client)

	defer stmt.Close()
	var arr = make([]*slack.User, 200)
	for stmt.Next() {
		var id string
		var data string
		var user *slack.User = &slack.User{}

		err = stmt.Scan(&id, &data)
		if err != nil {
			return err
		}
		err = json.Unmarshal([]byte(data), user)
		if err != nil {
			return err
		}
		arr = append(arr, user)
		if len(arr) >= 199 {
			go rtmClient.ReplaceManyUserObjects(arr, false)
			arr = make([]*slack.User, 200)
		}
	}
	if len(arr) >= 0 {
		go rtmClient.ReplaceManyUserObjects(arr, false)
		arr = nil
	}

	return stmt.Err()
}

func (mod *UserCacheModule) UpdateEntry(userobject *slack.User) error {
	var objarray = make([]*slack.User, 1)
	objarray[0] = userobject
	return mod.UpdateEntries(objarray)
}

func (mod *UserCacheModule) UpdateEntries(userobjects []*slack.User) error {
	stmt, err := mod.team.DB().Prepare(sqlUpsertEntry)
	if err != nil {
		return err
	}

	defer stmt.Close()

	for _, obj := range userobjects {
		if obj != nil {
			entrydata, err := json.Marshal(obj)
			if err == nil {
				_, err := stmt.Exec(obj.ID, entrydata)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
