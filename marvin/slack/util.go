package slack

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/pkg/errors"

	"github.com/riking/homeapi/marvin/util"
)

func SlackAPILog(resp *http.Response, err error) {
	if err != nil {
		util.LogError(err)
	}
	var response struct {
		*APIResponse
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		util.LogError(errors.Wrap(err, "decode json"))
	}
	if !response.OK {
		util.LogError(errors.Wrap(response, "Slack error"))
	}
}

var (
	mentionRegexp = regexp.MustCompile(`<@(U[A-Z0-9]+)>`)
	channelMentionRgx = regexp.MustCompile(`<#(C[A-Z0-9]+)\|([a-z0-9_-]+)>`)
	groupIDRgx = regexp.MustCompile(`G[A-Z0-9]+`)
	dmIDRgx = regexp.MustCompile(`D[A-Z0-9]+`)
)

func UserMentionRegexp() *regexp.Regexp {
	return mentionRegexp
}

func ParseUserMention(arg string) UserID {
	match := mentionRegexp.FindStringSubmatch(arg)
	if match == nil {
		return ""
	}
	return UserID(match[1])
}

func ParseChannelID(arg string) ChannelID {
	match := channelMentionRgx.FindStringSubmatch(arg)
	if match != nil {
		return ChannelID(match[1])
	}
	strMatch := groupIDRgx.FindString(arg)
	if strMatch != "" {
		return ChannelID(strMatch)
	}
	strMatch = dmIDRgx.FindString(arg)
	if strMatch != "" {
		return ChannelID(strMatch)
	}
	return ""
}