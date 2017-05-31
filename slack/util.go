package slack

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var (
	mentionRegexp     = regexp.MustCompile(`<@(U[A-Z0-9]+)>`)
	channelMentionRgx = regexp.MustCompile(`<#(C[A-Z0-9]+)\|?([a-z0-9_-]+)?>`)
	userIDRgx         = regexp.MustCompile(`U[A-Z0-9]+`)
	channelIDRgx      = regexp.MustCompile(`C[A-Z0-9]+`)
	groupIDRgx        = regexp.MustCompile(`G[A-Z0-9]+`)
	dmIDRgx           = regexp.MustCompile(`D[A-Z0-9]+`)
)

func ParseUserMention(arg string) UserID {
	match := mentionRegexp.FindStringSubmatch(arg)
	if match != nil {
		return UserID(match[1])
	}
	strMatch := userIDRgx.FindString(arg)
	if strMatch != "" {
		return UserID(strMatch)
	}
	return ""
}

func ParseChannelID(arg string) ChannelID {
	match := channelMentionRgx.FindStringSubmatch(arg)
	if match != nil {
		return ChannelID(match[1])
	}
	strMatch := channelIDRgx.FindString(arg)
	if strMatch != "" {
		return ChannelID(strMatch)
	}
	strMatch = groupIDRgx.FindString(arg)
	if strMatch != "" {
		return ChannelID(strMatch)
	}
	strMatch = dmIDRgx.FindString(arg)
	if strMatch != "" {
		return ChannelID(strMatch)
	}
	return ""
}

func IsDMChannel(channel ChannelID) bool {
	if len(channel) == 0 {
		return false
	}
	return channel[0] == 'D'
}

func ArchiveURL(teamDomain string, channelName string, msgID MessageID) string {
	splitTS := strings.Split(string(msgID.MessageTS), ".")
	stripTS := "p" + splitTS[0] + splitTS[1]

	channel := msgID.ChannelID
	if channel[0] == 'D' {
		return fmt.Sprintf("https://%s.slack.com/archives/%s/%s",
			teamDomain, channel, stripTS)
	}
	if channel[0] == 'G' {
		if channelName != "" {
			return fmt.Sprintf("https://%s.slack.com/archives/%s/%s",
				teamDomain, channelName, stripTS)
		} else {
			return fmt.Sprintf("https://%s.slack.com/archives/%s/%s",
				teamDomain, msgID.ChannelID, stripTS)
		}
	}
	if channel[0] == 'C' {
		return fmt.Sprintf("https://%s.slack.com/archives/%s/%s",
			teamDomain, channelName, stripTS)
	}
	panic(errors.Errorf("Invalid channel id '%s' passed to ArchiveURL", channel))
}
