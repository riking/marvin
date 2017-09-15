package rtm

import (
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/util"
)

type membershipMap map[slack.ChannelID]map[slack.UserID]bool

type membershipRequest struct {
	F func(membershipMap) interface{}
	C chan interface{}
}

type userCacheAPI interface {
	marvin.Module

	UpdateEntry(userobject *slack.User) error
	UpdateEntries(userobjects []*slack.User) error
}

func (c *Client) membershipWorker() {
	for req := range c.membershipCh {
		req.C <- req.F(c.channelMembers)
	}
}

func (c *Client) rebuildMembershipMapFunc(groups []*slack.Channel) func(m membershipMap) interface{} {
	return func(m membershipMap) interface{} {
		// Only groups are relevant
		for _, v := range groups {
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

func (c *Client) ListPublicChannels() []*slack.Channel {
	c.MetadataLock.RLock()
	defer c.MetadataLock.RUnlock()
	return c.Channels // TODO channels.list
}

func (c *Client) ListPrivateChannels() []*slack.Channel {
	c.MetadataLock.RLock()
	defer c.MetadataLock.RUnlock()
	return c.Groups
}

func (c *Client) ListMPIMs() []*slack.Channel {
	c.MetadataLock.RLock()
	defer c.MetadataLock.RUnlock()
	return c.Groups
}

func (c *Client) ListIMs() []*slack.ChannelIM {
	c.MetadataLock.RLock()
	defer c.MetadataLock.RUnlock()
	return c.Ims
}

func (c *Client) fetchTeamInfo() {
	go c.fillGroupList()

	// TODO(kyork): list normal channels too
	// TODO(kyork): use the listChannels() from logger module
}

func (c *Client) FillUsersList() {
	var response struct {
		slack.APIResponse
		Members  []*slack.User
		PageInfo struct {
			NextCursor string `json:"next_cursor"`
		} `json:"response_metadata"`
	}
	var form = url.Values{
		"presence": []string{"false"},
		"limit":    []string{"200"},
	}

	err := c.team.SlackAPIPostJSON("users.list", form, &response)
	if err != nil {
		util.LogError(errors.Wrapf(err, "[%s] Could not retrieve users list", c.Team.Domain))
	}

	c.ReplaceManyUserObjects(response.Members, true)

	for response.PageInfo.NextCursor != "" {
		time.Sleep(2 * time.Second)
		form.Set("cursor", response.PageInfo.NextCursor)
		err := c.team.SlackAPIPostJSON("users.list", form, &response)
		if err != nil {
			util.LogError(errors.Wrapf(err, "[%s] Could not retrieve users list", c.Team.Domain))
			continue
		}
		c.ReplaceManyUserObjects(response.Members, true)
	}
}

func (c *Client) fillGroupList() {
	var response struct {
		slack.APIResponse
		Groups []*slack.Channel
	}
	err := c.team.SlackAPIPostJSON("groups.list", url.Values{}, &response)
	if err != nil {
		util.LogError(errors.Wrapf(err, "[%s] Could not retrieve groups list", c.Team.Domain))
		return
	}

	c.MetadataLock.Lock()
	c.Groups = response.Groups
	c.MetadataLock.Unlock()

	c.membershipCh <- membershipRequest{
		C: make(chan interface{}, 1),
		F: c.rebuildMembershipMapFunc(response.Groups),
	}
}
