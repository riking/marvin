package rss

import (
	"database/sql"
	"database/sql/driver"

	"github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/riking/marvin/database"
	"github.com/riking/marvin/slack"
)

type db struct {
	*database.Conn
}

const (
	sqlMigrate1 = `
	CREATE TABLE module_rss_subs (
		id         SERIAL PRIMARY KEY,
		feed_type  int,
		sl_channel varchar(10),
		feed_id    text,

		UNIQUE(feed_type, sl_channel, feed_id)
	)`
	sqlMigrate2 = `
	CREATE TABLE module_rss_seenitems (
		id        SERIAL PRIMARY KEY,
		feed_type int,
		feed_id   text,
		item_id   text
	)`
	sqlMigrate3 = `
	ALTER TABLE module_rss_seenitems
	ADD COLUMN seen_at timestamptz DEFAULT ( now() )`

	sqlGetAllSubscriptions = `
	SELECT feed_type, feed_id, '' sl_channel
	  FROM module_rss_subs
	 GROUP BY feed_type, feed_id`

	sqlGetChannelSubscriptions = `
	SELECT feed_type, feed_id, sl_channel
	  FROM module_rss_subs
	 WHERE sl_channel = $1`

	sqlGetFeedChannels = `
	SELECT feed_type, feed_id, sl_channel
	  FROM module_rss_subs
	 WHERE feed_type = $1
	   AND feed_id = $2`

	// $1: feed type
	// $2: feed id
	// $3: item id array
	sqlCheckSeen = `
	SELECT now_items.item_id FROM
	unnest($3::text[]) as now_items (item_id)
	 LEFT JOIN module_rss_seenitems seen
	   ON now_items.item_id = seen.item_id
	  AND seen.feed_type = $1
	  AND seen.feed_id = $2
	WHERE seen.item_id IS NULL`

	sqlMarkSeen = `
	INSERT INTO module_rss_seenitems
	(feed_type, feed_id, item_id)
	VALUES ($1, $2, $3)`

	sqlLastSeen = `
	SELECT item_id FROM module_rss_seenitems
	 WHERE feed_type = $1
	   AND feed_id = $2
	ORDER BY seen_at DESC
	 LIMIT 1`

	sqlSubscribe = `
	INSERT INTO module_rss_subs
	(feed_type, feed_id, sl_channel)
	VALUES ($1, $2, $3)`
)

type TypeID byte

const (
	feedTypeFacebook TypeID = 'F'
	feedTypeTwitter         = 'T'
	feedTypeRSS             = 'R'
)

func (t *TypeID) Scan(value interface{}) error {
	if i, ok := value.(int64); ok {
		*t = TypeID(i)
		return nil
	}
	return errors.Errorf("cannot convert %T to TypeID", value)
}

func (t TypeID) Value() (driver.Value, error) { return int64(t), nil }

type subscription struct {
	FeedType TypeID
	FeedID   string
	Channel  slack.ChannelID
}

func (d *db) readSubscriptions(rows *sql.Rows) ([]subscription, error) {
	var r []subscription
	for rows.Next() {
		var fType int
		var fID, ch string
		err := rows.Scan(&fType, &fID, &ch)
		if err != nil {
			return nil, err
		}
		r = append(r, subscription{
			FeedType: TypeID(fType),
			FeedID:   fID,
			Channel:  slack.ChannelID(ch),
		})
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}
	return r, nil
}

// Get all the unique feed IDs of a certain type.
//
// The returned subscriptions have empty channel IDs.
func (d *db) GetAllSubscriptions() ([]subscription, error) {
	stmt, err := d.Conn.Prepare(sqlGetAllSubscriptions)
	if err != nil {
		return nil, errors.Wrap(err, "db prepare")
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	return d.readSubscriptions(rows)
}

// Get all the feeds a channel is subscribed to.
func (d *db) GetChannelSubscriptions(channel slack.ChannelID) ([]subscription, error) {
	stmt, err := d.Conn.Prepare(sqlGetChannelSubscriptions)
	if err != nil {
		return nil, errors.Wrap(err, "db prepare")
	}
	defer stmt.Close()

	rows, err := stmt.Query(channel)
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	return d.readSubscriptions(rows)
}

// Get all the channels subscribed to a feed.
func (d *db) GetFeedChannels(feedType TypeID, feedID string) ([]subscription, error) {
	stmt, err := d.Conn.Prepare(sqlGetFeedChannels)
	if err != nil {
		return nil, errors.Wrap(err, "db prepare")
	}
	defer stmt.Close()

	rows, err := stmt.Query(feedType, feedID)
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	return d.readSubscriptions(rows)
}

// Check if any of the given item IDs haven't been seen yet.
func (d *db) GetUnseen(feedType TypeID, feedID string, itemIDs []string) ([]string, error) {
	stmt, err := d.Conn.Prepare(sqlCheckSeen)
	if err != nil {
		return nil, errors.Wrap(err, "db prepare")
	}
	defer stmt.Close()

	rows, err := stmt.Query(feedType, feedID, pq.StringArray(itemIDs))
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	var result []string
	for rows.Next() {
		var one string
		err = rows.Scan(&one)
		if err != nil {
			return nil, errors.Wrap(err, "query")
		}
		result = append(result, one)
	}
	if rows.Err() != nil {
		return nil, errors.Wrap(err, "query")
	}
	return result, nil
}

// Mark one item of a feed as seen.
func (d *db) MarkSeen(feedType TypeID, feedID string, itemID string) error {
	stmt, err := d.Conn.Prepare(sqlMarkSeen)
	if err != nil {
		return errors.Wrap(err, "db prepare")
	}
	defer stmt.Close()

	_, err = stmt.Exec(feedType, feedID, itemID)
	return err
}

func (d *db) LastSeen(feedType TypeID, feedID string) (itemID string, e error) {
	stmt, err := d.Conn.Prepare(sqlLastSeen)
	if err != nil {
		return "", errors.Wrap(err, "db prepare")
	}
	defer stmt.Close()

	row := stmt.QueryRow(feedType, feedID)
	var itemid sql.NullString
	err = row.Scan(&itemid)
	if err == sql.ErrNoRows {
		return "", nil
	} else if err != nil {
		return "", err
	} else if !itemid.Valid {
		return "", nil
	}
	return itemid.String, nil

}

func (d *db) Subscribe(feedType TypeID, feedID string, channel slack.ChannelID, items []Item) (retErr error) {
	tx, err := d.Conn.Begin()
	if err != nil {
		return errors.Wrap(err, "db prepare")
	}
	txOk := false
	defer func() {
		if !txOk {
			tx.Rollback()
			if retErr == nil {
				retErr = errors.Errorf("db: Transaction rolled back")
			}
		}
	}()

	stmt, err := tx.Prepare(sqlMarkSeen)
	if err != nil {
		return errors.Wrap(err, "db prepare")
	}

	for _, v := range items {
		_, err = stmt.Exec(feedType, feedID, v.ItemID())
		if err != nil {
			stmt.Close()
			return errors.Wrap(err, "db mark seen")
		}
	}
	stmt.Close()

	stmt, err = tx.Prepare(sqlSubscribe)
	if err != nil {
		return errors.Wrap(err, "db prepare")
	}
	_, err = stmt.Exec(feedType, feedID, string(channel))
	if err != nil {
		stmt.Close()
		return errors.Wrap(err, "db subscribe")
	}
	stmt.Close()

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "tx commit")
	}
	txOk = true
	return nil
}
