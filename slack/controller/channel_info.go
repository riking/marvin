package controller

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
)

func (t *Team) ChannelName(channel slack.ChannelID) string {
	if channel == "" {
		return "#(!Empty channel ID)"
	}
	switch channel[0] {
	case 'C':
		ch, err := t.PublicChannelInfo(channel)
		if err != nil {
			return fmt.Sprintf("<!error getting channel name for %s>", string(channel))
		}
		return "#" + ch.Name
	case 'G':
		ch, err := t.PrivateChannelInfo(channel)
		if err != nil {
			return fmt.Sprintf("<!error getting channel name for %s>", string(channel))
		}
		if ch.IsMultiIM() {
			var membersStr bytes.Buffer
			for i, v := range ch.Members {
				if i != 0 {
					membersStr.WriteByte(' ')
				}
				membersStr.WriteByte('@')
				membersStr.WriteString(t.UserName(v))
			}
			return fmt.Sprintf("#[MultiIM %v]", membersStr.String())
		}
		return "#" + ch.Name
	case 'D':
		otherUser, _ := t.GetIMOtherUser(channel)
		if otherUser == "" {
			return fmt.Sprintf("<!error getting other user for %s>", string(channel))
		}
		return fmt.Sprintf("#[IM @%s]", t.UserName(otherUser))
	}
	return string(channel)
}

func (t *Team) FormatChannel(channel slack.ChannelID) string {
	switch channel[0] {
	case 'C':
		ch, err := t.PublicChannelInfo(channel)
		if err != nil {
			return fmt.Sprintf("<!error getting channel name for %s>", string(channel))
		}
		return fmt.Sprintf("<#%s|%s>", channel, ch.Name)
	case 'G':
		ch, err := t.PrivateChannelInfo(channel)
		if err != nil {
			return fmt.Sprintf("<!error getting channel name for %s>", string(channel))
		}
		if ch.IsMultiIM() {
			var membersStr bytes.Buffer
			for i, v := range ch.Members {
				if i != 0 {
					membersStr.WriteByte(' ')
				}
				membersStr.WriteByte('@')
				membersStr.WriteString(t.UserName(v))
			}
			return fmt.Sprintf("#[MultiIM %v]", membersStr.String())
		}
		return fmt.Sprintf("<#%s|%s>", channel, ch.Name)
	case 'D':
		otherUser, _ := t.GetIMOtherUser(channel)
		if otherUser == "" {
			return fmt.Sprintf("<!error getting other user for %s>", string(channel))
		}
		return fmt.Sprintf("#[IM @%s]", t.UserName(otherUser))
	case '(':
		// (via web)
		return string(channel)
	}
	return string(channel)

}

func (t *Team) UserName(user slack.UserID) string {
	if user == "" {
		return "<empty>"
	}
	u := t.cachedUserInfo(user)
	if u == nil {
		return fmt.Sprintf("<!error getting user name for %s>", string(user))
	}
	return u.Name
}

func (t *Team) UserLevel(user slack.UserID) marvin.AccessLevel {
	if t.TeamConfig().IsController(string(user)) {
		return marvin.AccessLevelController
	}

	u := t.cachedUserInfo(user)
	if u == nil {
		// unknown user ID
		return marvin.AccessLevelBlacklisted
	}

	val, isDefault, err := t.ModuleConfig("blacklist").GetIsDefault(string(u.ID))
	if isDefault {
		// user not blacklisted
	} else if err != nil {
		// DB error, continue
	} else if val != "" {
		// user is blacklisted
		return marvin.AccessLevelBlacklisted
	}

	if u.IsOwner || u.IsAdmin {
		return marvin.AccessLevelAdmin
	}
	if u.IsBot {
		return marvin.AccessLevelBlacklisted
	}
	return marvin.AccessLevelNormal
}

var rgxPlainTextChannelName = regexp.MustCompile(`^#([a-z0-9_\-]+)$`)

func (t *Team) ResolveChannelName(input string) slack.ChannelID {
	chID := slack.ParseChannelID(input)
	if chID != "" {
		return chID
	}
	chID = t.ChannelIDByName(strings.TrimPrefix(input, "#"))
	if chID != "" {
		return chID
	}
	return ""
}

func (t *Team) ResolveUserName(input string) slack.UserID {
	uID := slack.ParseUserMention(input)
	if uID != "" {
		return uID
	}
	input2 := strings.TrimLeft(input, "@")
	t.client.MetadataLock.RLock()
	defer t.client.MetadataLock.RUnlock()
	for _, v := range t.client.Users {
		if v.Name == input {
			return v.ID
		}
		if v.Name == input2 {
			return v.ID
		}
	}
	for _, v := range t.client.Users {
		if v.RealName == input {
			return v.ID
		}
		if v.RealName == input2 {
			return v.ID
		}
		//if len(v.Profile.FirstName) > 0 && strings.EqualFold(string(v.Profile.FirstName[0]) + v.Profile.LastName, input) {
		//	return v.ID
		//}
	}
	return ""
}

func (t *Team) cachedUserInfo(user slack.UserID) *slack.User {
	t.client.MetadataLock.RLock()
	defer t.client.MetadataLock.RUnlock()

	for i, v := range t.client.Users {
		if v.ID == user {
			if v.CacheTS.Before(time.Now().Add(-24 * time.Hour)) {
				go t.updateUserInfo(user)
			}
			return t.client.Users[i]
		}
	}
	return nil
}

func (t *Team) UserInfo(user slack.UserID) (*slack.User, error) {
	info := t.cachedUserInfo(user)
	if info != nil {
		return info, nil
	}
	return t.updateUserInfo(user)
}

func (t *Team) updateUserInfo(user slack.UserID) (*slack.User, error) {
	form := url.Values{"user": []string{string(user)}}
	var response struct {
		User *slack.User `json:"user"`
	}
	err := t.SlackAPIPostJSON("users.info", form, &response)
	if err != nil {
		return nil, err
	}
	t.client.ReplaceUserObject(response.User)
	return response.User, nil
}

func (t *Team) UserInChannels(user slack.UserID, channels ...slack.ChannelID) map[slack.ChannelID]bool {
	return t.client.UserInChannels(user, channels...)
}

func (t *Team) cachedPublicChannelInfo(channel slack.ChannelID) *slack.Channel {
	t.client.MetadataLock.RLock()
	defer t.client.MetadataLock.RUnlock()

	for i, v := range t.client.Channels {
		if v.ID == channel {
			if v.CacheTS.Before(time.Now().Add(-24 * time.Hour)) {
				return nil
			}
			return t.client.Channels[i]
		}
	}
	return nil
}

func (t *Team) ChannelIDByName(chName string) slack.ChannelID {
	t.client.MetadataLock.RLock()
	defer t.client.MetadataLock.RUnlock()

	for _, v := range t.client.Channels {
		if v.Name == chName {
			return v.ID
		}
	}
	for _, v := range t.client.Groups {
		if v.Name == chName {
			return v.ID
		}
	}
	return ""
}

func (t *Team) PublicChannelInfo(channel slack.ChannelID) (*slack.Channel, error) {
	result := t.cachedPublicChannelInfo(channel)
	if result != nil {
		return result, nil
	}

	var response struct {
		Channel *slack.Channel `json:"channel"`
	}
	form := url.Values{"channel": []string{string(channel)}}
	err := t.SlackAPIPostJSON("channels.info", form, &response)
	if err != nil {
		return nil, err
	}

	go t.client.ReplaceChannelObject(time.Now(), response.Channel)
	return response.Channel, nil
}

func (t *Team) cachedPrivateChannelInfo(channel slack.ChannelID) *slack.Channel {
	t.client.MetadataLock.RLock()
	defer t.client.MetadataLock.RUnlock()

	for i, v := range t.client.Groups {
		if v.ID == channel {
			if v.CacheTS.Before(time.Now().Add(-24 * time.Hour)) {
				return nil
			}
			return t.client.Groups[i]
		}
	}
	return nil
}

func (t *Team) PrivateChannelInfo(channel slack.ChannelID) (*slack.Channel, error) {
	result := t.cachedPrivateChannelInfo(channel)
	if result != nil {
		return result, nil
	}

	var response struct {
		Group *slack.Channel `json:"group"`
	}
	form := url.Values{"channel": []string{string(channel)}}
	err := t.SlackAPIPostJSON("groups.info", form, &response)
	if err != nil {
		return nil, err
	}

	go t.client.ReplaceGroupObject(time.Now(), response.Group)
	return response.Group, nil
}

func (t *Team) GetIMOtherUser(im slack.ChannelID) (slack.UserID, error) {
	t.client.MetadataLock.RLock()
	defer t.client.MetadataLock.RUnlock()

	for _, v := range t.client.Ims {
		if v.ID == im {
			return v.User, nil
		}
	}
	return "", nil
}

func (t *Team) cachedIMEntry(user slack.UserID) slack.ChannelID {
	t.client.MetadataLock.RLock()
	defer t.client.MetadataLock.RUnlock()

	for _, v := range t.client.Ims {
		if v.User == user {
			return v.ID
		}
	}
	return ""
}

func (t *Team) GetIM(user slack.UserID) (slack.ChannelID, error) {
	result := t.cachedIMEntry(user)
	if result != "" {
		return result, nil
	}

	form := url.Values{"user": []string{string(user)}}
	var response struct {
		Channel struct {
			ID slack.ChannelID `json:"id"`
		} `json:"channel"`
	}
	err := t.SlackAPIPostJSON("im.open", form, &response)
	if err != nil {
		return "", err
	}
	t.client.ReplaceIMObject(time.Now(), &slack.ChannelIM{
		ID:   response.Channel.ID,
		User: user,
	})
	return response.Channel.ID, nil
}

func (t *Team) ChannelMemberCount(channel slack.ChannelID) int {
	if channel == "" {
		return 0
	}
	if channel[0] == 'D' {
		return 2
	}
	return t.client.MemberCount(channel)
}

func (t *Team) ChannelMemberList(channel slack.ChannelID) []slack.UserID {
	return t.client.MemberList(channel)
}
