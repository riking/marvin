package marvin

import (
	"github.com/riking/marvin/slack"
)

type ReplyType int

const (
	ReplyTypePM ReplyType = 1 << iota
	ReplyTypeInChannel
	ReplyTypeLog
	ReplyTypeFlagOmitUsername
)

const (
	ReplyTypeInvalid      ReplyType = 0
	ReplyTypeShortProblem           = ReplyTypeInChannel | ReplyTypeLog
	ReplyTypeLongProblem            = ReplyTypePM | ReplyTypeLog
	ReplyTypeDestinations           = ReplyTypeInChannel | ReplyTypePM | ReplyTypeLog
)

const LongReplyThreshold = 400
const LongReplyCut = 100
const ShortReplyThreshold = 35

type ActionSourceUserMessage struct {
	Team Team
	Msg  slack.SlackTextMessage
}

func (um ActionSourceUserMessage) UserID() slack.UserID          { return um.Msg.UserID() }
func (um ActionSourceUserMessage) ChannelID() slack.ChannelID    { return um.Msg.ChannelID() }
func (um ActionSourceUserMessage) ArchiveLink() string           { return um.Team.ArchiveURL(um.Msg.MessageID()) }
func (um ActionSourceUserMessage) MsgTimestamp() slack.MessageTS { return um.Msg.MessageTS() }
func (um ActionSourceUserMessage) AccessLevel() AccessLevel      { return um.Team.UserLevel(um.Msg.UserID()) }
