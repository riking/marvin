package marvin

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
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

type ActionSource interface {
	UserID() slack.UserID
	ChannelID() slack.ChannelID
	ArchiveLink(t Team) string

	SendCmdReply(t Team, result CommandResult) CommandResult
}

type ActionSourceUserMessage struct {
	Msg slack.RTMRawMessage
}

func (um ActionSourceUserMessage) UserID() slack.UserID       { return um.Msg.UserID() }
func (um ActionSourceUserMessage) ChannelID() slack.ChannelID { return um.Msg.ChannelID() }
func (um ActionSourceUserMessage) ArchiveLink(t Team) string  { return t.ArchiveURL(um.Msg.MessageID()) }

func (um ActionSourceUserMessage) SendCmdReply(t Team, result CommandResult) CommandResult {
	logChannel := t.TeamConfig().LogChannel
	imChannel, _ := t.GetIM(um.UserID())

	replyChannel := result.ReplyType&ReplyTypeInChannel != 0
	replyIM := result.ReplyType&ReplyTypePM != 0
	replyLog := result.ReplyType&ReplyTypeLog != 0

	if um.Msg.ChannelID() == imChannel {
		replyIM = true
		replyChannel = false
	}

	switch result.Code {
	case CmdResultOK, CmdResultFailure:
		fallthrough
	default:
		if result.Message == "" {
			break
		}
		// Prefer Channel > PM > Log
		if replyChannel {
			channelMsg := result.Message
			if len(result.Message) > LongReplyThreshold {
				channelMsg = "[Reply truncated]\n" + util.PreviewString(result.Message, LongReplyCut) + "…\n"
				replyIM = true
			}
			if result.ReplyType&ReplyTypeFlagOmitUsername == 0 {
				channelMsg = fmt.Sprintf("%v: %s", um.UserID(), channelMsg)
			}
			t.SendMessage(um.Msg.ChannelID(), channelMsg)
		}
		if replyIM {
			t.SendMessage(imChannel, fmt.Sprintf("%s\n%s", result.Message, um.ArchiveLink(t)))
		}
		if replyLog {
			_, _, err := t.SendMessage(logChannel, fmt.Sprintf("%s\n%s", result.Message, um.ArchiveLink(t)))
			if err != nil {
				util.LogError(errors.Wrapf(err, "send to log channel"))
			}
			util.LogDebug("Command", fmt.Sprintf("[%s]", strings.Join(result.Args.OriginalArguments, "] [")), "result", result.Message)
		}
	case CmdResultError:
		// Print terse in channel, detail in PM, full in log
		if result.Message == "" {
			result.Message = "Error"
		}
		if replyChannel {
			if len(result.Err.Error()) > ShortReplyThreshold {
				replyIM = true
			}
			t.SendMessage(um.Msg.ChannelID(), fmt.Sprintf("%s: %s", result.Message, util.PreviewString(errors.Cause(result.Err).Error(), ShortReplyThreshold)))
		}
		if replyIM {
			t.SendMessage(imChannel, fmt.Sprintf("%s: %v\n%s", result.Message, result.Err, um.ArchiveLink(t)))
		}
		if replyLog {
			_, _, err := t.SendMessage(logChannel, fmt.Sprintf("%s\n```\n%+v\n```", um.ArchiveLink(t), result.Err))
			if err != nil {
				util.LogError(errors.Wrapf(err, "send to log channel %s", logChannel))
			}
			util.LogError(result.Err)
		}
	case CmdResultNoSuchCommand:
		if replyChannel {
			// Nothing
		}
		if replyIM {
			t.SendMessage(imChannel, fmt.Sprintf("I didn't quite understand that, sorry.\nYou said: [%s]",
				strings.Join(result.Args.OriginalArguments, "] [")))
		}
		if replyLog {
			t.SendMessage(logChannel, fmt.Sprintf("No such command from %v\nArgs: [%s]\nLink: %s",
				um.UserID(),
				strings.Join(result.Args.OriginalArguments, "] ["),
				um.ArchiveLink(t)))
		}
	case CmdResultPrintHelp:
		if replyChannel {
			msg := result.Message
			if len(result.Message) > LongReplyThreshold {
				replyIM = true
				msg = util.PreviewString(result.Message, LongReplyCut)
			}
			t.SendMessage(um.Msg.ChannelID(), msg)
		}
		if replyIM {
			t.SendMessage(imChannel, result.Message)
		}
		if replyLog {
			// err, no?
		}
	case CmdResultPrintUsage:
		if replyChannel {
			msg := result.Message
			if len(result.Message) > LongReplyThreshold {
				replyIM = true
				msg = util.PreviewString(result.Message, LongReplyCut)
			}
			t.SendMessage(um.Msg.ChannelID(), msg)
		}
		if replyIM {
			t.SendMessage(imChannel, result.Message)
		}
		if replyLog {
			// err, no?
		}
	}
	result.Sent = true
	return result
}
