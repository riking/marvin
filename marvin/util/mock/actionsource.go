package mock

import (
	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
)

type ActionSource struct {
	MUserID      slack.UserID
	MChannelID   slack.ChannelID
	MChannelName string
	MMessageTS   slack.MessageTS
	MAccessLevel marvin.AccessLevel
}

func (m ActionSource) UserID() slack.UserID            { return m.MUserID }
func (m ActionSource) ChannelID() slack.ChannelID      { return m.MChannelID }
func (m ActionSource) MsgTimestamp() slack.MessageTS   { return m.MMessageTS }
func (m ActionSource) AccessLevel() marvin.AccessLevel { return m.MAccessLevel }

func (m ActionSource) ArchiveLink() string {
	return slack.ArchiveURL("example", m.MChannelName, slack.MessageID{ChannelID: m.MChannelID, MessageTS: m.MMessageTS})
}
