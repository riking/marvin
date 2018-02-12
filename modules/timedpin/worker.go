package timedpin

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/util"
)

func (mod *TimedPinModule) unpinLoop() {
	for {
		until, err := nextUnpinTime(mod.team)
		if err != nil {
			util.LogError(err)
			mod.team.SendMessage(mod.team.TeamConfig().LogChannel, "<!channel> timed unpin has encountered a DB error and will quit")
			return
		}
		if until > 0 {
			if until < 5*time.Second {
				until = 5 * time.Second
			} else {
				until += 5 * time.Second
			}

			// interruptible sleep
			fmt.Println("timedpin worker: sleeping for", until)
			select {
			case <-time.After(until):
			case <-mod.notifyCh:
				fmt.Println("timedpin worker: wakeup")
			}
			continue
		}
		err = doUnpins(mod.team)
		if err != nil {
			util.LogError(err)
		}
		fmt.Println("timedpin worker: sleeping for 1 minute")
		time.Sleep(1 * time.Minute)
	}
}

func nextUnpinTime(t marvin.Team) (time.Duration, error) {
	stmt, err := t.DB().Prepare(sqlGetNextUnpin)
	if err != nil {
		return 0, errors.Wrap(err, "unpin: prepare")
	}
	defer stmt.Close()

	res := stmt.QueryRow()
	var nextUnpin *time.Time
	err = res.Scan(&nextUnpin)
	if err == sql.ErrNoRows || nextUnpin == nil {
		return 24*time.Hour - 5*time.Second, nil
	} else if err != nil {
		return 0, errors.Wrap(err, "unpin: read from db")
	}
	return (*nextUnpin).Sub(time.Now()), nil
}

func doUnpins(t marvin.Team) error {
	list, err := currentUnpins(t)
	if err != nil {
		return err
	}

	deleteStmt, err := t.DB().Prepare(sqlDeleteUnpin)
	if err != nil {
		return err
	}
	defer deleteStmt.Close()

	for _, v := range list {
		form := pinForm(slack.ChannelID(v.Channel), v.ThingID)
		err = t.SlackAPIPostJSON("pins.remove", form, nil)
		if slErr, ok := errors.Cause(err).(slack.APIResponse); ok &&
			slErr.SlackError == "not_pinned" {
			// OK, delete from database
		} else if err != nil {
			util.LogError(errors.Wrap(err, "Failed to unpin"))
			t.SendMessage(t.TeamConfig().LogChannel, fmt.Sprintf(
				"<!channel> failed to unpin %s %s", v.Channel, v.ThingID))

			// Delete from database so it doesn't spam the log channel
		}
		_, err = deleteStmt.Exec(v.Id)
		if err != nil {
			util.LogError(errors.Wrap(err, "Failed to record unpin in DB"))
			// it's sorta okay -  we'll get not_pinned next time
		}
		thingMention := v.ThingID
		if strings.Contains(v.ThingID, ".") {
			thingMention = fmt.Sprintf(
				"%s", t.ArchiveURL(slack.MessageID{ChannelID: slack.ChannelID(v.Channel), MessageTS: slack.MessageTS(v.ThingID)}))
		}
		t.SendMessage(slack.ChannelID(v.Channel), fmt.Sprintf(
			"%v: Unpinned %s after %s.", slack.UserID(v.SourceUser), thingMention, v.OrigDuration))
	}
	return nil
}

type todoUnpin struct {
	Id      int64
	Channel string
	ThingID string

	SourceUser   slack.UserID
	OrigDuration string
}

func currentUnpins(t marvin.Team) ([]todoUnpin, error) {
	stmt, err := t.DB().Prepare(sqlGetCurrentUnpins)
	if err != nil {
		return nil, errors.Wrap(err, "unpin: prepare")
	}
	defer stmt.Close()

	var result []todoUnpin
	rows, err := stmt.Query()
	if err != nil {
		return nil, errors.Wrap(rows.Err(), "unpin: query")
	}
	for rows.Next() {
		var one todoUnpin
		err = rows.Scan(&one.Id, &one.Channel, &one.ThingID, &one.OrigDuration, (*string)(&one.SourceUser))
		if err != nil {
			return nil, errors.Wrap(err, "unpin: scan")
		}
		result = append(result, one)
	}
	if rows.Err() != nil {
		return nil, errors.Wrap(rows.Err(), "unpin: query")
	}
	return result, nil
}
