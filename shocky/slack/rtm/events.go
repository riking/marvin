package rtm

import (
	"github.com/riking/homeapi/shocky/slack"
)

func (c *Client) onChannelCreate(msg slack.RTMRawMessage) {
	var parseMsg struct {
		// only ID, Name, Created, Creator
		Channel slack.Channel
	}
	msg.ReMarshal(&parseMsg)

}
