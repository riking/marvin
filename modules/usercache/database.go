package usercache

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/slack/rtm"
)

const (
	sqlMigrate1 = `CREATE TABLE module_user_cache (
						user_id varchar(15) PRIMARY KEY NOT NULL,
						data text
					)`

	sqlGetAllEntries = `SELECT * FROM module_user_cache`

	// $1 = slack.UserID
	sqlGetEntry = `SELECT data FROM module_user_cache WHERE user_id = $1`

	// $1 = slack.UserID
	// $2 = data (json encoded)
	sqlAddEntry = `INSERT INTO module_user_cache (user_id,data) VALUES ($1, $2)`

	// $1 = data (json encoded)
	// $2 = slack.UserID
	sqlUpdateEntry = `UPDATE module_user_cache SET data = $1 WHERE user_id = $2`
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

	defer stmt.Close()
	for stmt.Next() {
		var id string
		var data string
		var user slack.User

		err = stmt.Scan(&id, &data)
		if err != nil {
			return errors.Wrap(err, "error in user cache: obtaining row info")
			continue
		}
		err = json.Unmarshal([]byte(data), &user)
		if err != nil {
			return errors.Wrap(err, "error in user cache: unmarshal user object")
		}
		rtmClient := mod.team.GetRTMClient().(*rtm.Client)
		rtmClient.ReplaceUserObject(&user)
	}
	return stmt.Err()
}

func (mod *UserCacheModule) UpdateEntry(userobject slack.User) error {
	_, exists := mod.GetEntry(userobject.ID)

	var entrydata []byte
	entrydata, err := json.Marshal(&userobject)
	if err != nil {
		return err
	}

	var query = sqlAddEntry
	if exists != nil {
		query = sqlUpdateEntry
	}

	stmt, err := mod.team.DB().Prepare(query)
	if err != nil {
		return err
	}

	defer stmt.Close()
	row := stmt.QueryRow(userobject.ID, entrydata)
	var id slack.UserID
	err = row.Scan(&id)
	return err
}

func (mod *UserCacheModule) UpdateEntries(userobjects []*slack.User) error {
	for _, obj := range userobjects {
		if obj != nil {
			err := mod.UpdateEntry(*obj)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
