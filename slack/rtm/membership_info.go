package rtm

import (
	"github.com/riking/marvin/slack"
)

type membershipMap map[slack.ChannelID]map[slack.UserID]bool

type membershipRequest struct {
	F func(membershipMap) interface{}
	C chan interface{}
}

func (c *Client) membershipWorker() {
	for req := range c.membershipCh {
		req.C <- req.F(c.channelMembers)
	}
}

func (c *Client) rebuildMembershipMapFunc() func(m membershipMap) interface{} {
	return func(m membershipMap) interface{} {
		for _, v := range c.Channels {
			m := make(map[slack.UserID]bool)
			for _, userID := range v.Members {
				m[userID] = true
			}
			c.channelMembers[v.ID] = m
		}
		for _, v := range c.Groups {
			m := make(map[slack.UserID]bool)
			for _, userID := range v.Members {
				m[userID] = true
			}
			c.channelMembers[v.ID] = m
		}
		for _, v := range c.Mpims {
			m := make(map[slack.UserID]bool)
			for _, userID := range v.Members {
				m[userID] = true
			}
			c.channelMembers[v.ID] = m
		}
		return nil
	}
}

func userInChannelList(user slack.UserID, channels ...slack.ChannelID) func(m membershipMap) interface{} {
	r := make(map[slack.ChannelID]bool)
	return func(m membershipMap) interface{} {
		for _, v := range channels {
			chMap := m[v]
			if chMap != nil {
				r[v] = chMap[user]
			}
		}
		return r
	}
}

func userJoinChannel(user slack.UserID, channel slack.ChannelID, join bool) func(m membershipMap) interface{} {
	return func(m membershipMap) interface{} {
		chMap, ok := m[channel]
		if !ok {
			chMap = make(map[slack.UserID]bool)
			m[channel] = chMap
		}
		chMap[user] = join
		return nil
	}
}

func (c *Client) UserInChannels(user slack.UserID, channels ...slack.ChannelID) map[slack.ChannelID]bool {
	ch := make(chan interface{})
	c.membershipCh <- membershipRequest{C: ch,
		F: userInChannelList(user, channels...),
	}
	return (<-ch).(map[slack.ChannelID]bool)
}

func (c *Client) MemberCount(channel slack.ChannelID) int {
	ch := make(chan interface{})
	c.membershipCh <- membershipRequest{C: ch,
		F: func(m membershipMap) interface{} {
			chMap, ok := m[channel]
			if !ok {
				return int(0)
			}
			return int(len(chMap))
		},
	}
	return (<-ch).(int)
}

func (c *Client) MemberList(channel slack.ChannelID) []slack.UserID {
	ch := make(chan interface{})
	c.membershipCh <- membershipRequest{C: ch,
		F: func(m membershipMap) interface{} {
			chMap, ok := m[channel]
			if !ok {
				return int(0)
			}
			users := make([]slack.UserID, 0, len(chMap))
			for k := range chMap {
				users = append(users, k)
			}
			return users
		},
	}
	return (<-ch).([]slack.UserID)
}

func (c *Client) onUserJoinChannel(msg slack.RTMRawMessage) {
	ch := make(chan interface{}, 1)
	c.membershipCh <- membershipRequest{C: ch,
		F: userJoinChannel(msg.UserID(), msg.ChannelID(), true),
	}
}

func (c *Client) onUserLeaveChannel(msg slack.RTMRawMessage) {
	ch := make(chan interface{}, 1)
	c.membershipCh <- membershipRequest{C: ch,
		F: userJoinChannel(msg.UserID(), msg.ChannelID(), false),
	}
}
