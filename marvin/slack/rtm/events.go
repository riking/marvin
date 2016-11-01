package rtm

import (
	"github.com/riking/homeapi/marvin/slack"
)

func (c *Client) onChannelCreate(msg slack.RTMRawMessage) {
	var parseMsg struct {
		// only ID, Name, Created, Creator
		Channel slack.Channel
	}
	msg.ReMarshal(&parseMsg)

}
