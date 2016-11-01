package controller

import (
	"encoding/json"
	"net/url"

	"github.com/pkg/errors"

	"github.com/riking/homeapi/marvin/slack"
)

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
