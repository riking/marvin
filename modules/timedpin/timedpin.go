package timedpin

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
)

func init() {
	marvin.RegisterModule(NewTimedPinModule)
}

const Identifier = "timedpin"

type TimedPinModule struct {
	team     marvin.Team
	notifyCh chan struct{}
}

func NewTimedPinModule(t marvin.Team) marvin.Module {
	mod := &TimedPinModule{
		team:     t,
		notifyCh: make(chan struct{}, 1),
	}
	return mod
}

func (mod *TimedPinModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *TimedPinModule) Load(t marvin.Team) {
	t.DB().MustMigrate(Identifier, 1486001919, sqlMigrate1, sqlMigrate1b)
	t.DB().SyntaxCheck(
		sqlGetNextUnpin,
		sqlInsertTimedPin,
		sqlGetCurrentUnpins,
		sqlDeleteUnpin,
	)
}

func (mod *TimedPinModule) Enable(t marvin.Team) {
	cmd := t.RegisterCommandFunc("timedpin", mod.CommandTimedPin,
		"`@marvin timedpin &lt;duration: 10h30m&gt; &lt;slack archive link | _last_&gt;`\n"+
			"Pins the linked message to the current channel, and unpins the message after the given duration expires.\n"+
			"If `last` is given instead of an archive link, the most recently pinned item is scheduled.")
	t.RegisterCommand("timed-pin", cmd)
	go mod.unpinLoop()
}

func (mod *TimedPinModule) Disable(t marvin.Team) {
}

// ---

const (
	sqlMigrate1 = `
	CREATE TABLE module_timedpin_pins (
		id         SERIAL PRIMARY KEY,
		channel    varchar(10),
		ts_or_file varchar(30),
		unpin_time timestamptz,

		original_duration text,
		pinning_user      varchar(10)
	)`
	sqlMigrate1b = `CREATE INDEX idx_pins_by_time ON module_timedpin_pins (unpin_time)`

	sqlGetNextUnpin = `SELECT MIN(unpin_time) as next_unpin FROM module_timedpin_pins`

	// $1 = channel $2 = item ID $3 = time
	sqlInsertTimedPin = `
	INSERT INTO module_timedpin_pins
	(channel, ts_or_file, unpin_time, original_duration, pinning_user)
	VALUES ($1, $2, $3, $4, $5)`

	sqlGetCurrentUnpins = `
	SELECT id, channel, ts_or_file, original_duration, pinning_user
	FROM module_timedpin_pins
	WHERE unpin_time < CURRENT_TIMESTAMP`

	// $1 = id
	sqlDeleteUnpin = `
	DELETE FROM module_timedpin_pins
	WHERE id = $1`
)

// https://42schoolusa.slack.com/files/crenfrow/F40DWMBGW/cp2slj8.jpeg
var regexpFileArchive = regexp.MustCompile(`https://[^./]+\.slack\.com/files/[^/]+/(F[A-Z0-9]+)/.*`)

// https://42schoolusa.slack.com/archives/general/p1485982126006451
var regexpArchiveLink = regexp.MustCompile(`https://[^./]+\.slack\.com/archives/([^/]+)/p([0-9]+)`)

func (mod *TimedPinModule) CommandTimedPin(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {

	if len(args.Arguments) < 1 {
		return marvin.CmdFailuref(args, "Usage: timed-pin &lt;duration&gt; &lt;archive link or `last`&gt;").WithSimpleUndo()
	}
	durationArg := args.Pop()
	duration, err := time.ParseDuration(durationArg)
	if err != nil {
		return marvin.CmdFailuref(args, err.Error()).WithSimpleUndo()
	}

	thingArg := args.Pop()
	var thingID string
	var channelID slack.ChannelID
	if thingArg == "last" {
		channelID = args.Source.ChannelID()
		thingID, err = mostRecentPin(mod.team, args.Source.ChannelID())
		if err != nil {
			return marvin.CmdFailuref(args, err.Error()).WithSimpleUndo()
		}
	} else if m := regexpFileArchive.FindStringSubmatch(thingArg); m != nil {
		channelID = args.Source.ChannelID()
		thingID = m[1]
	} else if m := regexpArchiveLink.FindStringSubmatch(thingArg); m != nil {
		channelID = slack.ParseChannelID(m[1])
		if channelID == "" {
			channelID = t.ChannelIDByName(m[1])
		}
		if channelID == "" {
			return marvin.CmdFailuref(args, "Bad archive URL? Couldn't find channel ID/name").WithSimpleUndo()
		}
		thingID = m[2]
		thingID = thingID[:len(thingID)-slack.MessageTSCharsAfterDot] + "." + thingID[len(thingID)-slack.MessageTSCharsAfterDot:]
	} else {
		return marvin.CmdFailuref(args, "'%s' isn't a thing I know how to pin.", thingArg).WithSimpleUndo()
	}

	stmt, err := t.DB().Prepare(sqlInsertTimedPin)
	if err != nil {
		return marvin.CmdFailuref(args, "database error: %s").WithSimpleUndo()
	}
	defer stmt.Close()
	unpinTime := time.Now().Add(duration)

	// Pin the thing
	if thingArg != "last" {
		form := pinForm(channelID, thingID)
		err = t.SlackAPIPostJSON("pins.add", form, nil)
		if slErr, ok := errors.Cause(err).(slack.APIResponse); ok {
			// Check if it's already pinned
			if slErr.SlackError != "already_pinned" {
				return marvin.CmdFailuref(args, "Couldn't pin: %v", slErr.SlackError).WithSimpleUndo()
			}
		} else if err != nil {
			return marvin.CmdError(args, err, "Couldn't pin message (needs manual unpin, cannot undo)").WithNoUndo()
		}
	}

	_, err = stmt.Exec(string(channelID), thingID, unpinTime, durationArg, string(args.Source.UserID()))
	if err != nil {
		return marvin.CmdError(args, err, "Couldn't save record (needs manual unpin, cannot undo)").WithNoUndo()
	}
	select {
	case mod.notifyCh <- struct{}{}:
	default:
	}

	return marvin.CmdSuccess(args, fmt.Sprintf(
		"Okay, %s will be unpinned in %s.", thingID, duration.String(),
	))
}

func mostRecentPin(t marvin.Team, channel slack.ChannelID) (thingID string, err error) {
	var response struct {
		Items []slack.PinnedItem `json:"items"`
	}

	err = t.SlackAPIPostJSON("pins.list", url.Values{"channel": []string{string(channel)}}, &response)
	if err != nil {
		return "", err
	}
	if len(response.Items) == 0 {
		return "", errors.Errorf("No pinned items in this channel.")
	}
	item := response.Items[0]
	if item.Type == "file_comment" {
		return string(item.Comment.ID), nil
	} else if item.Type == "file" {
		return string(item.File.ID), nil
	} else {
		return string(item.Message.TS), nil
	}
}

func pinForm(channel slack.ChannelID, thingID string) url.Values {
	form := url.Values{"channel": []string{string(channel)}}
	if strings.HasPrefix(thingID, "Fc") {
		form.Set("file_comment", thingID)
	} else if strings.HasPrefix(thingID, "F") {
		form.Set("file", thingID)
	} else {
		form.Set("timestamp", thingID)
	}
	return form
}
