package rss

import (
	"github.com/riking/marvin/database"
	"github.com/riking/marvin/slack"
	"database/sql"
)

type db struct {
	*database.Conn
}

const (
	sqlMigrate2 = `
	CREATE TABLE module_rss_subs (
		id         SERIAL PRIMARY KEY
		item_type  char(1),
		sl_channel varchar(10),
		feed_ident text
	)`
	sqlMigrate4 = `
	CREATE TABLE module_rss_seenitems (
		id        SERIAL PRIMARY KEY,
		item_type char(1),
		item_id   text
	)`

	sqlGetAllSubscriptions = `
	SELECT item_type, feed_ident, sl_channel
	FROM module_rss_subs
	WHERE item_type = $1
	ORDER BY feed_ident`

	sqlGetChannelSubscriptions = `
	SELECT item_type, feed_ident, sl_channel
	FROM module_rss_subs
	WHERE sl_channel = $1`

	sqlCheckSeen = `
	SELECT item_id
	FROM module_rss_seenitems
	WHERE item_type = $1
	AND item_id IN ($2)`
)

const (
	itemTypeFacebook = 'F'
	itemTypeTwitter  = 'T'
	itemTypeRSS      = 'R'
)

type subscription struct {
	ItemType   byte
	Identifier string
	Channel    slack.ChannelID
}

func (d *db) readSubscriptions(rows *sql.Rows) ([]subscription, error) {
	var r []subscription
	for rows.Next() {
		var it, id, ch string
		err := rows.Scan(&it, &id, &ch)
		if err != nil {
			return nil, err
		}
		r = append(r, subscription{
			ItemType: it[0],
			Identifier: id,
			Channel: slack.ChannelID(ch),
		})
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return r, nil
}

func (d *db) GetAllSubscriptions(itemType byte) ([]subscription, error) {
	stmt, err := d.Conn.Prepare(sqlGetAllSubscriptions)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(string(itemType))
	if err != nil {
		return nil, err
	}
	return d.readSubscriptions(rows)
}

func (d *db) GetChannelSubscriptions(channel slack.ChannelID) ([]subscription, error) {
	stmt, err := d.Conn.Prepare(sqlGetChannelSubscriptions)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(string(channel))
	if err != nil {
		return nil, err
	}
	return d.readSubscriptions(rows)
}

func (d *db) GetUnseen(itemType byte, identifiers []string) (map[string]bool, error) {
	m := make(map[string]bool)
	for _, v := range identifiers {
		m[v] = false
	}

	return nil, nil
}