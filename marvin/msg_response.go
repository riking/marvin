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

type ActionSourceUserMessage struct {
	Team Team
	Msg  slack.RTMRawMessage
}

func (um ActionSourceUserMessage) UserID() slack.UserID       { return um.Msg.UserID() }
func (um ActionSourceUserMessage) ChannelID() slack.ChannelID { return um.Msg.ChannelID() }
func (um ActionSourceUserMessage) ArchiveLink() string        { return um.Team.ArchiveURL(um.Msg.MessageID()) }
func (um ActionSourceUserMessage) AccessLevel() AccessLevel   { return um.Team.UserLevel(um.Msg.UserID()) }

func (um ActionSourceUserMessage) SendCmdReply(result CommandResult) CommandResult {
	t := um.Team
	logChannel := t.TeamConfig().LogChannel
	imChannel, _ := t.GetIM(um.UserID())

	// Reply in the public / group channel message was sent from
	replyChannel := result.ReplyType&ReplyTypeInChannel != 0
	// Reply in a DM
	replyIM := result.ReplyType&ReplyTypePM != 0
	// Post in the logging channel
	replyLog := result.ReplyType&ReplyTypeLog != 0

	// Message was sent from a DM; do not include archive link
	replyIMPrimary := false

	if (replyChannel || replyIM) && um.Msg.ChannelID() == imChannel {
		replyIMPrimary = true
		replyIM = false
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
		if replyIMPrimary {
			t.SendMessage(imChannel, result.Message)
		}
		if replyIM {
			t.SendMessage(imChannel, fmt.Sprintf("%s\n%s", result.Message, um.ArchiveLink()))
		}
		if replyLog {
			_, _, err := t.SendMessage(logChannel, fmt.Sprintf("%s\n%s", result.Message, um.ArchiveLink()))
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
		if replyIMPrimary {
			t.SendMessage(imChannel, fmt.Sprintf("%s: %v", result.Message, result.Err))
		}
		if replyIM {
			t.SendMessage(imChannel, fmt.Sprintf("%s: %v\n%s", result.Message, result.Err, um.ArchiveLink()))
		}
		if replyLog {
			_, _, err := t.SendMessage(logChannel, fmt.Sprintf("%s\n```\n%+v\n```", um.ArchiveLink(), result.Err))
			if err != nil {
				util.LogError(errors.Wrapf(err, "send to log channel %s", logChannel))
			}
			util.LogError(result.Err)
		}
	case CmdResultNoSuchCommand:
		if replyChannel {
			// Nothing
		}
		if replyIM || replyIMPrimary {
			t.SendMessage(imChannel, fmt.Sprintf("I didn't quite understand that, sorry.\nYou said: [%s]",
				strings.Join(result.Args.OriginalArguments, "] [")))
		}
		if replyLog {
			t.SendMessage(logChannel, fmt.Sprintf("No such command from %v\nArgs: [%s]\nLink: %s",
				um.UserID(),
				strings.Join(result.Args.OriginalArguments, "] ["),
				um.ArchiveLink()))
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
		if replyIM || replyIMPrimary {
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
		if replyIM || replyIMPrimary {
			t.SendMessage(imChannel, result.Message)
		}
		if replyLog {
			// err, no?
		}
	}
	result.Sent = true
	return result
}
