package rtm

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/util"
)

type membershipMap map[slack.ChannelID]map[slack.UserID]bool

type membershipRequest struct {
	F func(membershipMap) interface{}
	C chan interface{}
}

const (
	csvSlackMember = "Member"
	csvSlackAdmin  = "Admin"
	csvSlackOwner  = "Owner"
	csvSlackBot    = "Bot"
)

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
	if c.team.TeamConfig().IsSlackAdmin {
		go c.fillUsersCsv()
	} else {
		go c.fillUsersList()
	}

	// TODO(kyork): list normal channels too
	// TODO(kyork): use the listChannels() from logger module
}

func (c *Client) fillUsersList() {
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

	for response.PageInfo.NextCursor != "" {
		c.ReplaceManyUserObjects(response.Members)
		time.Sleep(10 * time.Second)

		form.Set("cursor", response.PageInfo.NextCursor)
		err := c.team.SlackAPIPostJSON("users.list", form, &response)
		if err != nil {
			util.LogError(errors.Wrapf(err, "[%s] Could not retrieve users list", c.Team.Domain))
			break
		}
	}
}

// This method currently requires having admin privileges on the workspace.
// Add IsSlackAdmin=true to config to enable this feature.
// insufficient_permissions
// Fields in CSV:
// username,email,status,billing-active,has-2fa,has-sso,userid,fullname,displayname
// Example: marvin,exampleemail@example.com,Admin,1,0,0,UXXXXXXXX,Marvin,Marvin
func (c *Client) fillUsersCsv() {
	resp, err := c.team.SlackAPIPostRaw("users.admin.fetchTeamUsersCsv", url.Values{})
	if err != nil {
		util.LogError(errors.Wrapf(err, "[%s] Could not retrieve users csv file", c.Team.Domain))
		return
	}
	if resp.StatusCode != 200 {
		util.LogError(errors.New(fmt.Sprintf("[%s] Could not retrieve users csv file [Status code %d]", c.Team.Domain, resp.StatusCode)))
		return
	}
	r := csv.NewReader(resp.Body)
	for true {
		record, err := r.Read()
		if err != nil && err != io.EOF {
			util.LogError(errors.Wrapf(err, "[%s] Could not parse users csv file", c.Team.Domain))
			return
		} else if err == io.EOF {
			break
		}
		// Use preliminary checks to avoid adding too many users into database.
		if strings.Compare(record[3], "0") == 0 {
			// Inactive user, do not count.
			continue
		}
		isAdmin := strings.Compare(record[2], csvSlackAdmin) == 0
		isOwner := strings.Compare(record[2], csvSlackOwner) == 0
		isMember := strings.Compare(record[2], csvSlackMember) == 0
		isBot := strings.Compare(record[2], csvSlackBot) == 0
		if !isAdmin && !isOwner && !isMember && !isBot {
			continue
		}
		realnameSplit := strings.Split(record[7], " ")
		firstname := record[7]
		lastname := record[7]
		if len(realnameSplit) > 1 {
			firstname = realnameSplit[0]
			lastname = realnameSplit[1]
		}
		//fmt.Printf("%s\n", slack.UserID(record[6]))
		user := &slack.User{
			ID:      slack.UserID(record[6]),
			TeamID:  c.team.TeamID(),
			Name:    record[8],
			IsAdmin: isAdmin,
			IsOwner: isOwner,
			IsBot:   isBot,
			Profile: slack.Profile{
				DisplayName: record[8],
				RealName:    record[7],
				Email:       record[1],
				FirstName:   firstname,
				LastName:    lastname,
			},
		}
		c.ReplaceUserObject(user)
		// Force an update on the object but still return cached data
		// this calls a goroutine.
		user.CacheTS = time.Unix(0, 0)
	}
	fmt.Printf("[%s] Populated database with fetched CSV successfully!\n", c.team.Domain())
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
