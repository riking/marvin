package controller

import (
	"encoding/json"
	"net/url"

	"github.com/pkg/errors"

	"fmt"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
)

func (t *Team) ChannelName(channel slack.ChannelID) string {
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
			return fmt.Sprintf("#[MultiIM %v]", ch.Members)
		}
		return "#" + ch.Name
	case 'D':
		otherUser := t.otherIMParty(channel)
		if otherUser == "" {
			return fmt.Sprintf("<!error getting other user for %s>", string(channel))
		}
		return fmt.Sprintf("#[IM %v]", otherUser)
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
			return fmt.Sprintf("#[MultiIM %v]", ch.Members)
		}
		return fmt.Sprintf("<#%s|%s>", channel, ch.Name)
	case 'D':
		otherUser := t.otherIMParty(channel)
		if otherUser == "" {
			return fmt.Sprintf("<!error getting other user for %s>", string(channel))
		}
		return fmt.Sprintf("#[IM %v]", otherUser)
	}
	return string(channel)

}

func (t *Team) UserName(user slack.UserID) string {
	u := t.cachedUserInfo(user)
	if u == nil {
		return fmt.Sprintf("<!error getting channel name for %s>", string(user))
	}
	return u.Name
}

func (t *Team) UserLevel(user slack.UserID) marvin.AccessLevel {
	if user == "U2223J70R" { // TODO store in teamconfig? database?
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

func (t *Team) cachedUserInfo(user slack.UserID) *slack.User {
	t.client.MetadataLock.RLock()
	defer t.client.MetadataLock.RUnlock()

	for i, v := range t.client.Users {
		if v.ID == user {
			return &t.client.Users[i]
		}
	}
	return nil
}

func (t *Team) cachedPublicChannelInfo(channel slack.ChannelID) *slack.Channel {
	t.client.MetadataLock.RLock()
	defer t.client.MetadataLock.RUnlock()

	for i, v := range t.client.Channels {
		if v.ID == channel {
			return &t.client.Channels[i]
		}
	}
	return nil
}

func (t *Team) PublicChannelInfo(channel slack.ChannelID) (*slack.Channel, error) {
	result := t.cachedPublicChannelInfo(channel)
	if result != nil {
		return result, nil
	}

	form := url.Values{"channel": []string{string(channel)}}
	resp, err := t.SlackAPIPost("channels.info", form)
	if err != nil {
		return nil, err
	}
	var response struct {
		slack.APIResponse
		Channel slack.Channel `json:"channel"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, errors.Wrap(err, "decode json")
	}
	resp.Body.Close()
	if !response.OK {
		return nil, response.APIResponse
	}

	// TODO save result

	return &response.Channel, nil
}

func (t *Team) cachedPrivateChannelInfo(channel slack.ChannelID) *slack.Channel {
	t.client.MetadataLock.RLock()
	defer t.client.MetadataLock.RUnlock()

	for i, v := range t.client.Groups {
		if v.ID == channel {
			return &t.client.Groups[i]
		}
	}
	return nil
}

func (t *Team) PrivateChannelInfo(channel slack.ChannelID) (*slack.Channel, error) {
	result := t.cachedPrivateChannelInfo(channel)
	if result != nil {
		return result, nil
	}

	form := url.Values{"channel": []string{string(channel)}}
	resp, err := t.SlackAPIPost("groups.info", form)
	if err != nil {
		return nil, err
	}
	var response struct {
		slack.APIResponse
		Group slack.Channel `json:"group"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, errors.Wrap(err, "decode json")
	}
	resp.Body.Close()
	if !response.OK {
		return nil, response.APIResponse
	}

	// TODO save result

	return &response.Group, nil
}

func (t *Team) otherIMParty(im slack.ChannelID) slack.UserID {
	t.client.MetadataLock.RLock()
	defer t.client.MetadataLock.RUnlock()

	for _, v := range t.client.Ims {
		if v.ID == im {
			return v.User
		}
	}
	return ""
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

	// TODO caching
	form := url.Values{"user": []string{string(user)}}
	resp, err := t.SlackAPIPost("im.open", form)
	if err != nil {
		return "", err
	}
	var response struct {
		slack.APIResponse
		Channel struct {
			ID slack.ChannelID `json:"id"`
		} `json:"channel"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", errors.Wrap(err, "decode json")
	}
	resp.Body.Close()
	if !response.OK {
		return "", response.APIResponse
	}

	// TODO save result

	return response.Channel.ID, nil
}
