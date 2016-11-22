package rtm

import (
	"time"

	"github.com/riking/homeapi/marvin/slack"
)

func (c *Client) onChannelCreate(msg slack.RTMRawMessage) {
	var parseMsg struct {
		// only ID, Name, Created, Creator
		Channel slack.Channel
	}
	msg.ReMarshal(&parseMsg)

}

func (c *Client) setTopicPurpose(channel slack.ChannelID, isTopic bool, new slack.ChannelTopicPurpose) {
	var ary *[]slack.Channel

	if channel[0] == 'C' {
		ary = &c.Channels
	} else {
		ary = &c.Groups
	}

	c.MetadataLock.Lock()
	defer c.MetadataLock.Unlock()
	for i, v := range *ary {
		if v.ID == channel {
			if isTopic {
				(*ary)[i].Topic.Value = new.Value
				(*ary)[i].Topic.Creator = new.Creator
				(*ary)[i].Topic.LastSet = float64(time.Now().Unix())
			} else {
				(*ary)[i].Purpose.Value = new.Value
				(*ary)[i].Purpose.Creator = new.Creator
				(*ary)[i].Purpose.LastSet = float64(time.Now().Unix())
			}
		}
	}
}

func (c *Client) onTopicChange(msg slack.RTMRawMessage) {
	ch := msg.ChannelID()
	topic := msg.StringField("topic")
	user := msg.UserID()

	c.setTopicPurpose(ch, true, slack.ChannelTopicPurpose{Value: topic, Creator: user})
}

func (c *Client) onPurposeChange(msg slack.RTMRawMessage) {
	ch := msg.ChannelID()
	purpose := msg.StringField("purpose")
	user := msg.UserID()

	c.setTopicPurpose(ch, false, slack.ChannelTopicPurpose{Value: purpose, Creator: user})
}
