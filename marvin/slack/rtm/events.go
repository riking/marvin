package rtm

import (
	"time"
	"math"

	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
)

func (c *Client) onChannelCreate(msg slack.RTMRawMessage) {
	var parseMsg struct {
		// only ID, Name, Created, Creator
		Channel slack.Channel
	}
	msg.ReMarshal(&parseMsg)

}

func (c *Client) setTopicPurpose(channel slack.ChannelID, isTopic bool, new slack.ChannelTopicPurpose) {
	var ary *[]*slack.Channel

	c.MetadataLock.Lock()
	defer c.MetadataLock.Unlock()

	if channel[0] == 'C' {
		ary = &c.Channels
	} else {
		ary = &c.Groups
	}
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

func (c *Client) onUserChange(msg slack.RTMRawMessage) {
	cacheTS := msg["cache_ts"].(float64)
	cacheInt, cacheFrac := math.Modf(cacheTS)
	cacheTime := time.Unix(int64(cacheInt), int64(cacheFrac*1000000000))

	var resp struct {
		User *slack.User `json:"user"`
	}
	err := msg.ReMarshal(&resp)
	if err != nil {
		util.LogError(err)
		return
	}

	c.ReplaceUserObject(cacheTime, resp.User)
}

func (c *Client) ReplaceUserObject(cacheTS time.Time, obj *slack.User) {
	c.MetadataLock.Lock()
	defer c.MetadataLock.Unlock()

	for i, v := range c.Users {
		if v.ID == obj.ID {
			c.Users[i] = obj
			return
		}
	}
	c.Users = append(c.Users, obj)
}

func (c *Client) ReplaceChannelObject(cacheTS time.Time, obj *slack.Channel) {
	c.MetadataLock.Lock()
	defer c.MetadataLock.Unlock()

	for i, v := range c.Channels {
		if v.ID == obj.ID {
			c.Channels[i] = obj
			return
		}
	}
	c.Channels = append(c.Channels, obj)
}

func (c *Client) ReplaceGroupObject(cacheTS time.Time, obj *slack.Channel) {
	c.MetadataLock.Lock()
	defer c.MetadataLock.Unlock()

	for i, v := range c.Groups {
		if v.ID == obj.ID {
			c.Groups[i] = obj
			return
		}
	}
	c.Groups = append(c.Groups, obj)
}
