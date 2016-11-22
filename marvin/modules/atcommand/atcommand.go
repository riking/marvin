package atcommand

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
)

func init() {
	marvin.RegisterModule(NewAtCommandModule)
}

const Identifier = "atcommand"

type AtCommandModule struct {
	team        marvin.Team
	enabled     int
	botUser     slack.UserID
	mentionRgx2 *regexp.Regexp
	mentionRgx1 *regexp.Regexp

	recentCommandsLock sync.Mutex
	recentCommands     map[slack.MessageID]*FinishedCommandInfo
}

func NewAtCommandModule(t marvin.Team) marvin.Module {
	mod := &AtCommandModule{
		team:           t,
		recentCommands: make(map[slack.MessageID]*FinishedCommandInfo),
	}
	return mod
}

func (mod *AtCommandModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *AtCommandModule) Load(t marvin.Team) {
	mod.botUser = mod.team.BotUser()
	mod.mentionRgx1 = regexp.MustCompile(fmt.Sprintf(`<@%s>`, mod.botUser))
	mod.mentionRgx2 = regexp.MustCompile(fmt.Sprintf(`(?m:(?:\n|^)\s*(<@%s>)\s+())`, mod.botUser))

	c := mod.team.ModuleConfig(Identifier)
	c.Add(confKeyEmojiHi, "wave")
	c.Add(confKeyEmojiOk, "white_check_mark")
	c.Add(confKeyEmojiFail, "negative_squared_cross_mark")
	c.Add(confKeyEmojiError, "warning")
	c.Add(confKeyEmojiUnkCmd, "question")
	c.Add(confKeyEmojiUsage, "confused")
	c.Add(confKeyEmojiHelp, "memo")
}

func (mod *AtCommandModule) Enable(t marvin.Team) {
	t.OnNormalMessage(Identifier, mod.HandleMessage)
	t.OnSpecialMessage(Identifier, []string{"message_changed", "message_deleted"}, mod.HandleEdit)
	mod.enabled += 1
	go mod.janitorRecentMessages(mod.enabled)
}

func (mod *AtCommandModule) Disable(t marvin.Team) {
	mod.enabled += 1
	t.OffAllEvents(Identifier)
}

// -----

const (
	confKeyEmojiHi     = "emoji-hi"
	confKeyEmojiOk     = "emoji-ok"
	confKeyEmojiFail   = "emoji-fail"
	confKeyEmojiError  = "emoji-error"
	confKeyEmojiUnkCmd = "emoji-unknown"
	confKeyEmojiUsage  = "emoji-usage"
	confKeyEmojiHelp   = "emoji-help"
)

type ReplyActionEmoji struct {
	MessageID slack.MessageID
	Emoji     string
}

func (rae ReplyActionEmoji) Undo(mod *AtCommandModule) {
	form := url.Values{
		"name":      []string{rae.Emoji},
		"channel":   []string{string(rae.MessageID.ChannelID)},
		"timestamp": []string{string(rae.MessageID.MessageTS)},
	}
	util.LogIfError(mod.team.SlackAPIPostJSON("reactions.remove", form, nil))
}

type ReplyActionSentMessage struct {
	MessageID slack.MessageID
	Text      string
}

func (rsm ReplyActionSentMessage) Update(mod *AtCommandModule, newText string) error {
	if rsm.Text == "" {
		return nil
	}
	form := url.Values{
		"channel": []string{string(rsm.MessageID.ChannelID)},
		"ts":      []string{string(rsm.MessageID.MessageTS)},
		"text":    []string{newText},
		"parse":   []string{"client"},
	}
	return util.LogIfError(mod.team.SlackAPIPostJSON("chat.update", form, nil))
}

type FinishedCommandInfo struct {
	MyTimestamp time.Time

	Lock        sync.Mutex
	OriginalMsg slack.RTMRawMessage
	LatestEdit  slack.RTMRawMessage
	parseResult parseMessageReturn

	FoundCommand  bool
	CommandArgs   *marvin.CommandArguments
	CommandResult marvin.CommandResult
	FailedUndo    bool

	ActionEmoji    []ReplyActionEmoji
	ActionChanMsg  ReplyActionSentMessage
	ActionPMMsg    ReplyActionSentMessage
	ActionPMLogMsg ReplyActionSentMessage
	ActionLogMsg   ReplyActionSentMessage
}

func (fci *FinishedCommandInfo) AddEmojiReaction(messageID slack.MessageID, emoji string) {
	fci.ActionEmoji = append(fci.ActionEmoji, ReplyActionEmoji{MessageID: messageID, Emoji: emoji})
}

func (fci *FinishedCommandInfo) ChangeEmoji(mod *AtCommandModule, new []ReplyActionEmoji) {
	for _, v := range fci.ActionEmoji {
		match := false
		for _, v2 := range new {
			if v.MessageID == v2.MessageID && v.Emoji == v2.Emoji {
				match = true
				break
			}
		}
		if !match {
			go v.Undo(mod)
		}
	}
	for _, v := range new {
		match := false
		for _, v2 := range fci.ActionEmoji {
			if v.MessageID == v2.MessageID && v.Emoji == v2.Emoji {
				match = true
				break
			}
		}
		if !match {
			go mod.team.ReactMessage(v.MessageID, v.Emoji)
		}
	}
	fci.ActionEmoji = new
}

func (mod *AtCommandModule) janitorRecentMessages(epoch int) {
	for mod.enabled == epoch {
		time.Sleep(30 * time.Minute)
		mod._cleanRecentMessages()
	}
}

func (mod *AtCommandModule) _cleanRecentMessages() {
	mod.recentCommandsLock.Lock()
	defer mod.recentCommandsLock.Unlock()
	threshold := time.Now().Add(-2 * time.Hour)
	for k, v := range mod.recentCommands {
		if v.MyTimestamp.Before(threshold) {
			delete(mod.recentCommands, k)
		}
	}
}

type parseMessageReturn struct {
	wave                 bool
	argSplit             []string
	splitErr             error
	lenientNoSuchCommand bool
}

func (mod *AtCommandModule) ParseMessage(rtm slack.SlackTextMessage) parseMessageReturn {
	result := parseMessageReturn{}

	msgText := rtm.Text()
	matches := mod.mentionRgx2.FindStringSubmatchIndex(msgText)
	fullMsg := false
	if len(matches) == 0 {
		if rtm.ChannelID()[0] == 'D' {
			util.LogDebug("Got full-message command", msgText)
			fullMsg = true
			result.lenientNoSuchCommand = true
		} else {
			m := mod.mentionRgx1.FindString(msgText)
			if m != "" {
				result.wave = true
				return result
			}
			return result
		}
	}

	if fullMsg {
		result.argSplit, result.splitErr = ParseArgs(msgText, 0)
	} else {
		result.argSplit, result.splitErr = ParseArgs(msgText, matches[2*2+1])
	}
	if len(result.argSplit) == 0 {
		result.wave = true
	}
	return result
}

func (mod *AtCommandModule) HandleMessage(_rtm slack.RTMRawMessage) {
	fciResult := &FinishedCommandInfo{MyTimestamp: time.Now()}
	fciResult.Lock.Lock()
	defer fciResult.Lock.Unlock()

	fciResult.OriginalMsg = _rtm
	rtm := slack.SlackTextMessage(_rtm)
	if !rtm.AssertText() {
		return
	}
	factoidChars, _ := mod.team.ModuleConfig("factoid").Get("factoid-char")
	if strings.ContainsAny(rtm.Text()[:1], factoidChars) {
		return
	}

	parseResult := mod.ParseMessage(rtm)
	fciResult.parseResult = parseResult

	// ###########
	// IMPORTANT: KEEP SYNCHRONIZED WITH HandleEdit()
	// Do not modify fciResult.parseResult, used for equality check (reflect.DeepEqual)

	if parseResult.wave {
		mod.recentCommandsLock.Lock()
		mod.recentCommands[_rtm.MessageID()] = fciResult
		mod.recentCommandsLock.Unlock()

		reactEmoji, _ := mod.team.ModuleConfig(Identifier).Get(confKeyEmojiHi)
		fciResult.AddEmojiReaction(rtm.MessageID(), reactEmoji)
		mod.team.ReactMessage(rtm.MessageID(), reactEmoji)
		return
	} else if parseResult.argSplit != nil || parseResult.splitErr != nil {
		mod.recentCommandsLock.Lock()
		mod.recentCommands[_rtm.MessageID()] = fciResult
		mod.recentCommandsLock.Unlock()

		fciResult.FoundCommand = true
		// continue
	} else {
		return
	}

	mod.ProcessInitialCommandMessage(fciResult, rtm)
}

func (mod *AtCommandModule) HandleEdit(_rtm slack.RTMRawMessage) {
	if _rtm.Subtype() != "message_changed" {
		return
	}
	rtm := slack.EditMessage{_rtm}
	if !rtm.AssertText() {
		return
	}
	msgID := rtm.MessageID()
	time.Sleep(50 * time.Millisecond) // slack is out-of-order sometimes

	mod.recentCommandsLock.Lock()
	fciMeta, ok := mod.recentCommands[msgID]
	mod.recentCommandsLock.Unlock()

	if !ok {
		parseResult := mod.ParseMessage(rtm)
		if parseResult.argSplit != nil {
			imChannel, _ := mod.team.GetIM(rtm.EditingUserID())
			mod.team.SendMessage(imChannel, fmt.Sprintf(
				"Oops, I seem to have forgotten about that one. I can only cope with edits of recent messages. %s",
				mod.team.ArchiveURL(rtm.MessageID())))
		}
		return
	}

	util.LogDebug("Got edit to command message", mod.team.ArchiveURL(msgID))
	fciMeta.Lock.Lock()
	defer fciMeta.Lock.Unlock()

	if fciMeta.LatestEdit != nil {
		// TODO something??
	}
	fciMeta.LatestEdit = _rtm

	parseResult := mod.ParseMessage(rtm)
	if reflect.DeepEqual(parseResult, fciMeta.parseResult) {
		util.LogGood("Ignoring edit as it didn't affect the command at all", mod.team.ArchiveURL(msgID))
		return
	}

	myFoundCommand := false
	if parseResult.argSplit != nil || parseResult.splitErr != nil {
		myFoundCommand = true
	}

	fciMeta.parseResult = parseResult
	if fciMeta.FoundCommand == false {
		if myFoundCommand == true {
			fciMeta.ChangeEmoji(mod, nil)
			fciMeta.parseResult = parseResult
			fciMeta.FoundCommand = true
			fciMeta.ActionEmoji = nil
			mod.ProcessInitialCommandMessage(fciMeta, rtm)
		}
		return
	}

	if myFoundCommand == false {
		mod.UndoCommand(fciMeta, marvin.ActionSourceUserMessage{Team: mod.team, Msg: rtm})
	} else {
		canEdit := mod.canEdit(fciMeta)
		if !canEdit {
			canUndo, _ := mod.canUndo(fciMeta)
			if canUndo {
				mod.UndoCommand(fciMeta, marvin.ActionSourceUserMessage{Team: mod.team, Msg: rtm})
				fciMeta.FoundCommand = true
				mod.ProcessInitialCommandMessage(fciMeta, rtm)
			}
		}
		mod.EditCommand(fciMeta, marvin.ActionSourceUserMessage{Team: mod.team, Msg: rtm})
	}
}

func (mod *AtCommandModule) canEdit(fciMeta *FinishedCommandInfo) (canEdit bool) {

	if fciMeta.CommandResult.CanEdit == marvin.TriYes {
		// Custom edit
		return true
	} else if fciMeta.CommandResult.CanEdit == marvin.TriNo {
		return false
	} else { // TriDefault
		switch fciMeta.CommandResult.Code {
		case marvin.CmdResultFailure:
			return true
		case marvin.CmdResultOK:
			return false
		case marvin.CmdResultError:
			return false // error
		case marvin.CmdResultNoSuchCommand, marvin.CmdResultPrintUsage, marvin.CmdResultPrintHelp:
			return true
		}
	}
	panic("unrecognized command result code")
}

func (mod *AtCommandModule) canUndo(fciMeta *FinishedCommandInfo) (canUndo, custom bool) {
	if fciMeta.FailedUndo {
		return false, false
	}

	if fciMeta.CommandResult.CanEdit == marvin.TriNo {
		// Undo not supported
		return false, false
	} else if fciMeta.CommandResult.CanEdit == marvin.UndoCustom {
		// Custom undo
		return true, true
	} else if fciMeta.CommandResult.CanEdit == marvin.UndoSimple {
		return true, false
	} else { // TriDefault
		switch fciMeta.CommandResult.Code {
		case marvin.CmdResultOK:
			// Undo not supported
			return false, false
		case marvin.CmdResultFailure:
			return true, false
		case marvin.CmdResultError:
			return false, false // error
		case marvin.CmdResultNoSuchCommand, marvin.CmdResultPrintUsage, marvin.CmdResultPrintHelp:
			return true, false
		}
	}
	panic("unrecognized command result code")
}

func (mod *AtCommandModule) EditCommand(fciMeta *FinishedCommandInfo, source marvin.ActionSource) {
	imChannel, _ := mod.team.GetIM(source.UserID())
	logChannel := mod.team.TeamConfig().LogChannel

	canEdit := mod.canEdit(fciMeta)

	if fciMeta.CommandResult.Code == marvin.CmdResultError {
		if fciMeta.ActionChanMsg.Text != "" {
			mod.team.SendMessage(fciMeta.ActionChanMsg.MessageID.ChannelID, fmt.Sprintf(
				"%v: You may not edit a command that resulted in an error. Repeat the corrected command in a new message.", source.UserID()))
		} else {
			mod.team.SendMessage(imChannel, fmt.Sprintf(
				"For safety, you cannot edit a command that resulted in an error. %s", source.ArchiveLink()))
		}
		mod.team.ReactMessage(fciMeta.OriginalMsg.MessageID(), "x")
		fciMeta.AddEmojiReaction(fciMeta.OriginalMsg.MessageID(), "x")
		return
	}

	if !canEdit {
		mod.team.SendMessage(imChannel, fmt.Sprintf(
			"That command does not support editing. %s", source.ArchiveLink()))
		mod.team.ReactMessage(fciMeta.OriginalMsg.MessageID(), "x")
		fciMeta.AddEmojiReaction(fciMeta.OriginalMsg.MessageID(), "x")
		return
	}

	args := &marvin.CommandArguments{
		Arguments:         fciMeta.parseResult.argSplit,
		OriginalArguments: fciMeta.parseResult.argSplit,
		Source:            source,
		Command:           "",

		IsEdit:         true,
		IsUndo:         false,
		PreviousResult: &fciMeta.CommandResult,
		ModuleData:     fciMeta.CommandResult.Args.ModuleData,
	}
	result := mod.team.DispatchCommand(args)
	fciMeta.CommandResult = result
	fciMeta.CommandArgs = args

	var newEmojiAry []ReplyActionEmoji
	newEmoji := mod.GetEmojiForResponse(result)
	newEmojiAry = append(newEmojiAry, ReplyActionEmoji{MessageID: fciMeta.OriginalMsg.MessageID(), Emoji: newEmoji})
	newEmojiAry = append(newEmojiAry, ReplyActionEmoji{MessageID: fciMeta.OriginalMsg.MessageID(), Emoji: "fast_forward"})
	fciMeta.ChangeEmoji(mod, newEmojiAry)

	didSendMessageChannel := false
	didSendMessageIM := false
	sendMessageChannel := func(msg string) {
		didSendMessageChannel = true
		if fciMeta.ActionChanMsg.Text != "" {
			fciMeta.ActionChanMsg.Update(mod, msg)
		} else {
			ts, _, err := mod.team.SendMessage(source.ChannelID(), SanitizeForChannel(msg))
			if err != nil {
				util.LogError(err)
			}
			fciMeta.ActionChanMsg = ReplyActionSentMessage{MessageID: slack.MsgID(imChannel, ts), Text: msg}
		}
	}
	sendMessageIM := func(msg string) {
		didSendMessageIM = true
		if fciMeta.ActionPMMsg.Text != "" {
			fciMeta.ActionPMMsg.Update(mod, msg)
		} else {
			ts, _, err := mod.team.SendMessage(imChannel, SanitizeLoose(msg))
			if err != nil {
				util.LogError(err)
			}
			fciMeta.ActionPMMsg = ReplyActionSentMessage{MessageID: slack.MsgID(imChannel, ts), Text: msg}
		}
	}
	sendMessageIMLog := func(msg string) {
		_, _, err := mod.team.SendMessage(imChannel, SanitizeLoose(msg))
		if err != nil {
			util.LogError(err)
		}
	}
	sendMessageLog := func(msg string) {
		_, _, err := mod.team.SendMessage(logChannel, SanitizeForChannel(msg))
		if err != nil {
			util.LogError(err)
		}
	}

	mod.SendReplyMessages(result, source, source.ChannelID() == imChannel, sendMessageChannel, sendMessageIM, sendMessageIMLog, sendMessageLog)

	if fciMeta.ActionChanMsg.Text != "" && !didSendMessageChannel {
		fciMeta.ActionChanMsg.Update(mod, "(removed)")
		fciMeta.ActionChanMsg = ReplyActionSentMessage{}
	}
	if fciMeta.ActionPMMsg.Text != "" && !didSendMessageIM {
		fciMeta.ActionPMMsg.Update(mod, "(removed)")
		fciMeta.ActionPMMsg = ReplyActionSentMessage{}
	}
}

func (mod *AtCommandModule) UndoCommand(fciMeta *FinishedCommandInfo, source marvin.ActionSource) {
	var newEmoji []ReplyActionEmoji

	if fciMeta.FailedUndo {
		// already notified
		return
	}

	canUndo, customUndo := mod.canUndo(fciMeta)

	if fciMeta.CommandResult.Code == marvin.CmdResultError {
		imChannel, _ := mod.team.GetIM(source.UserID())
		mod.team.SendMessage(imChannel, fmt.Sprintf(
			"For safety, you cannot undo a command that resulted in an error. %s", source.ArchiveLink()))
		fciMeta.FailedUndo = true
		mod.team.ReactMessage(fciMeta.OriginalMsg.MessageID(), "x")
		fciMeta.AddEmojiReaction(fciMeta.OriginalMsg.MessageID(), "x")
		return
	}
	if fciMeta.parseResult.wave {
		reactEmoji, _ := mod.team.ModuleConfig(Identifier).Get(confKeyEmojiHi)
		newEmoji = append(newEmoji, ReplyActionEmoji{MessageID: fciMeta.OriginalMsg.MessageID(), Emoji: reactEmoji})
	}
	if canUndo && !customUndo {
		fciMeta.ActionChanMsg.Update(mod, "(removed)")
		fciMeta.ActionPMMsg.Update(mod, "(removed)")
		newEmoji = append(newEmoji, ReplyActionEmoji{MessageID: fciMeta.OriginalMsg.MessageID(), Emoji: "leftwards_arrow_with_hook"})
		fciMeta.ChangeEmoji(mod, newEmoji)
		return
	}
	if !customUndo {
		imChannel, _ := mod.team.GetIM(source.UserID())
		mod.team.SendMessage(imChannel, fmt.Sprintf(
			"That command does not support undo. %s", source.ArchiveLink()))
		fciMeta.FailedUndo = true
		mod.team.ReactMessage(fciMeta.OriginalMsg.MessageID(), "x")
		fciMeta.AddEmojiReaction(fciMeta.OriginalMsg.MessageID(), "x")
		return
	}

	imChannel, _ := mod.team.GetIM(source.UserID())
	mod.team.SendMessage(imChannel, fmt.Sprintf(
		"TODO - custom undo support. %s", source.ArchiveLink()))
	mod.team.ReactMessage(fciMeta.OriginalMsg.MessageID(), "x")
	fciMeta.AddEmojiReaction(fciMeta.OriginalMsg.MessageID(), "x")
	return // TODO

	args := &marvin.CommandArguments{
		OriginalArguments: fciMeta.CommandArgs.OriginalArguments,
		Arguments:         fciMeta.CommandArgs.OriginalArguments,
		Command:           "",
		Source:            source,

		PreviousResult: &fciMeta.CommandResult,
		IsUndo:         true,
		IsEdit:         false,
	}
	result := mod.team.DispatchCommand(args)

	resultEmoji := mod.GetEmojiForResponse(result)
	newEmoji = append(newEmoji, ReplyActionEmoji{MessageID: fciMeta.OriginalMsg.MessageID(), Emoji: resultEmoji})
	newEmoji = append(newEmoji, ReplyActionEmoji{MessageID: fciMeta.OriginalMsg.MessageID(), Emoji: "leftwards_arrow_with_hook"})
	fciMeta.ChangeEmoji(mod, newEmoji)

	logChannel := mod.team.TeamConfig().LogChannel
	didSendMessageChannel := false
	sendMessageChannel := func(msg string) {
		if fciMeta.ActionChanMsg.Text != "" {
			didSendMessageChannel = true
			fciMeta.ActionChanMsg.Update(mod, msg)
		} else {
			ts, _, err := mod.team.SendMessage(source.ChannelID(), SanitizeForChannel(msg))
			if err != nil {
				util.LogError(err)
			}
			fciMeta.ActionChanMsg = ReplyActionSentMessage{MessageID: slack.MsgID(imChannel, ts), Text: msg}
		}
	}
	sendMessageIM := func(msg string) {
		if fciMeta.ActionPMMsg.Text != "" {
			fciMeta.ActionPMMsg.Update(mod, msg)
		} else {
			ts, _, err := mod.team.SendMessage(imChannel, SanitizeLoose(msg))
			if err != nil {
				util.LogError(err)
			}
			fciMeta.ActionPMMsg = ReplyActionSentMessage{MessageID: slack.MsgID(imChannel, ts), Text: msg}
		}
	}
	sendMessageIMLog := func(msg string) {
		_, _, err := mod.team.SendMessage(imChannel, SanitizeLoose(msg))
		if err != nil {
			util.LogError(err)
		}
	}
	sendMessageLog := func(msg string) {
		_, _, err := mod.team.SendMessage(logChannel, SanitizeForChannel(msg))
		if err != nil {
			util.LogError(err)
		}
	}

	mod.SendReplyMessages(result, source, source.ChannelID() == imChannel, sendMessageChannel, sendMessageIM, sendMessageIMLog, sendMessageLog)

	if fciMeta.ActionChanMsg.Text != "" && !didSendMessageChannel {
		fciMeta.ActionChanMsg.Update(mod, fmt.Sprintf("(command removed) %s", fciMeta.ActionChanMsg.Text))
	}
}

func (mod *AtCommandModule) ProcessInitialCommandMessage(fciResult *FinishedCommandInfo, rtm slack.SlackTextMessage) {
	parseResult := fciResult.parseResult
	source := marvin.ActionSourceUserMessage{Msg: fciResult.OriginalMsg, Team: mod.team}
	args := &marvin.CommandArguments{
		OriginalArguments: parseResult.argSplit,
		Arguments:         parseResult.argSplit,
		Command:           "",
		Source:            source,
		IsEdit:            false,
		ModuleData:        nil,
	}
	fciResult.CommandArgs = args

	var result marvin.CommandResult
	if parseResult.splitErr != nil {
		result = marvin.CmdFailuref(args, parseResult.splitErr.Error())
	} else {
		util.LogDebug("args: [", strings.Join(args.OriginalArguments, "] ["), "]")
		result = mod.team.DispatchCommand(args)
	}
	fciResult.CommandResult = result

	reactEmoji := mod.GetEmojiForResponse(result)
	var wg sync.WaitGroup
	if reactEmoji != "" {
		wg.Add(1)
		go mod.team.ReactMessage(rtm.MessageID(), reactEmoji)
		fciResult.AddEmojiReaction(rtm.MessageID(), reactEmoji)
	}

	logChannel := mod.team.TeamConfig().LogChannel
	imChannel, _ := mod.team.GetIM(rtm.UserID())
	sendMessageChannel := func(msg string) {
		ts, _, err := mod.team.SendMessage(rtm.ChannelID(), SanitizeForChannel(msg))
		if err != nil {
			util.LogError(err)
		} else {
			fciResult.ActionChanMsg = ReplyActionSentMessage{Text: msg, MessageID: slack.MessageID{ChannelID: rtm.ChannelID(), MessageTS: ts}}
		}
	}
	sendMessageIM := func(msg string) {
		ts, _, err := mod.team.SendMessage(imChannel, SanitizeLoose(msg))
		if err != nil {
			util.LogError(err)
		} else {
			fciResult.ActionPMMsg = ReplyActionSentMessage{MessageID: slack.MsgID(imChannel, ts), Text: msg}
		}
	}
	sendMessageIMLog := func(msg string) {
		ts, _, err := mod.team.SendMessage(imChannel, SanitizeLoose(msg))
		if err != nil {
			util.LogError(err)
		} else {
			fciResult.ActionPMLogMsg = ReplyActionSentMessage{MessageID: slack.MsgID(imChannel, ts), Text: msg}
		}
	}
	sendMessageLog := func(msg string) {
		ts, _, err := mod.team.SendMessage(logChannel, SanitizeForChannel(msg))
		if err != nil {
			util.LogError(err)
		} else {
			fciResult.ActionLogMsg = ReplyActionSentMessage{MessageID: slack.MsgID(logChannel, ts), Text: msg}
		}
	}

	mod.SendReplyMessages(result, source, rtm.ChannelID() == imChannel, sendMessageChannel, sendMessageIM, sendMessageIMLog, sendMessageLog)
}

func (mod *AtCommandModule) SendReplyMessages(
	result marvin.CommandResult,
	source marvin.ActionSource,
	isChannelIMChannel bool,
	sendMessageChannel, sendMessageIM, sendMessageIMLog, sendMessageLog func(string),
) {
	replyType := marvin.ReplyTypeInvalid
	switch result.Code {
	case marvin.CmdResultOK:
		replyType = marvin.ReplyTypeInChannel
	case marvin.CmdResultFailure:
		replyType = marvin.ReplyTypeShortProblem
	case marvin.CmdResultError:
		replyType = marvin.ReplyTypeShortProblem
	case marvin.CmdResultNoSuchCommand:
		replyType = marvin.ReplyTypePM
	case marvin.CmdResultPrintUsage:
		replyType = marvin.ReplyTypePM
	case marvin.CmdResultPrintHelp:
		replyType = marvin.ReplyTypeInChannel
	default:
		replyType = marvin.ReplyTypeShortProblem
	}

	if result.ReplyType&marvin.ReplyTypeDestinations == marvin.ReplyTypeInvalid {
		result.ReplyType = result.ReplyType | replyType
	}

	// Reply in the public / group channel message was sent from
	replyChannel := result.ReplyType&marvin.ReplyTypeInChannel != 0
	// Reply in a DM
	replyIM := result.ReplyType&marvin.ReplyTypePM != 0
	// Post in the logging channel
	replyLog := result.ReplyType&marvin.ReplyTypeLog != 0

	// Message was sent from a DM; do not include archive link
	replyIMPrimary := false

	if (replyChannel || replyIM) && isChannelIMChannel {
		replyIMPrimary = true
		replyIM = false
		replyChannel = false
	}

	switch result.Code {
	case marvin.CmdResultOK, marvin.CmdResultFailure:
		fallthrough
	default:
		if result.Message == "" {
			break
		}
		// Prefer Channel > PM > Log
		if replyChannel {
			channelMsg := result.Message
			if len(result.Message) > marvin.LongReplyThreshold {
				channelMsg = "[Reply truncated]\n" + util.PreviewString(result.Message, marvin.LongReplyCut) + "…\n"
				replyIM = true
			}
			if result.ReplyType&marvin.ReplyTypeFlagOmitUsername == 0 {
				channelMsg = fmt.Sprintf("%v: %s", source.UserID(), channelMsg)
			}
			sendMessageChannel(channelMsg)
		}
		if replyIMPrimary {
			sendMessageIM(result.Message)
		}
		if replyIM {
			sendMessageIM(result.Message)
		}
		if replyLog {
			sendMessageLog(fmt.Sprintf("%s\n%s", result.Message, source.ArchiveLink()))
			util.LogDebug("Command", fmt.Sprintf("[%s]", strings.Join(result.Args.OriginalArguments, "] [")), "result", result.Message)
		}
	case marvin.CmdResultError:
		// Print terse in channel, detail in PM, full in log
		if result.Message == "" {
			result.Message = "Error"
		}
		if replyChannel {
			if len(result.Err.Error()) > marvin.ShortReplyThreshold {
				replyIM = true
			}
			sendMessageChannel(fmt.Sprintf("%s: %s", result.Message, util.PreviewString(errors.Cause(result.Err).Error(), marvin.ShortReplyThreshold)))
		}
		if replyIMPrimary {
			sendMessageIMLog(fmt.Sprintf("%s: %v", result.Message, result.Err))
		} else if replyIM {
			sendMessageIMLog(fmt.Sprintf("%s: %v\n%s", result.Message, result.Err, source.ArchiveLink()))
		}
		if replyLog {
			sendMessageLog(fmt.Sprintf("%s\n```\n%+v\n```", source.ArchiveLink(), result.Err))
			util.LogError(result.Err)
		}
	case marvin.CmdResultNoSuchCommand:
		if replyChannel {
			// Nothing
		}
		if replyIM || replyIMPrimary {
			sendMessageIMLog(fmt.Sprintf("I didn't quite understand that, sorry.\nYou said: [%s]",
				strings.Join(result.Args.OriginalArguments, "] [")))
		}
		if replyLog {
			sendMessageLog(fmt.Sprintf("No such command from %v\nArgs: [%s]\nLink: %s",
				source.UserID(),
				strings.Join(result.Args.OriginalArguments, "] ["),
				source.ArchiveLink()))
		}
	case marvin.CmdResultPrintHelp:
		if replyChannel {
			msg := result.Message
			if len(result.Message) > marvin.LongReplyThreshold {
				replyIM = true
				msg = util.PreviewString(result.Message, marvin.LongReplyCut)
			}
			sendMessageChannel(msg)
		}
		if replyIM || replyIMPrimary {
			sendMessageIM(result.Message)
		}
		if replyLog {
			// err, no?
		}
	case marvin.CmdResultPrintUsage:
		if replyChannel {
			msg := result.Message
			if len(result.Message) > marvin.LongReplyThreshold {
				replyIM = true
				msg = util.PreviewString(result.Message, marvin.LongReplyCut)
			}
			sendMessageChannel(msg)
		}
		if replyIM || replyIMPrimary {
			sendMessageIM(result.Message)
		}
		if replyLog {
			// err, no?
		}
	}
}

var rgxTakeCodeBlock = regexp.MustCompile(`^&amp;(\d+)$`)
var rgxCodeBlock = regexp.MustCompile("(?m:^)```\n?(?s:(.*?))\n?```()(?m:$|\\s)")

func ParseArgs(raw string, startIdx int) ([]string, error) {
	endOfLine := strings.IndexByte(raw[startIdx:], '\n')
	if endOfLine == -1 {
		endOfLine = len(raw[startIdx:])
	}
	cmdLine := raw[startIdx : startIdx+endOfLine]

	var argSplit []string
	argSplit = strings.Split(cmdLine, " ")
	var codeBlocks [][]string
	var retErr error

	for i, v := range argSplit {
		str := strings.TrimSpace(v)
		m := rgxTakeCodeBlock.FindStringSubmatch(str)
		if m != nil {
			if codeBlocks == nil {
				codeBlocks = rgxCodeBlock.FindAllStringSubmatch(raw, -1)
			}
			which, err := strconv.Atoi(m[1])
			if err != nil {
				retErr = errors.Errorf("Expected a code block number after &, got [%s]", m[1])
				continue
			}
			if which == 0 {
				retErr = errors.Errorf("Code block indices start at &1")
				continue
			}
			if which > len(codeBlocks) {
				retErr = errors.Errorf("Found code block ref '%s' but only %d code blocks in message", m[1], len(codeBlocks))
				continue
			}
			argSplit[i] = codeBlocks[which-1][1]
		} else {
			argSplit[i] = str
		}
	}

	return argSplit, retErr
}

func SanitizeLoose(msg string) string {
	if strings.HasPrefix(msg, "/") {
		msg = "." + msg
	}
	return msg
}

var regexSanitizeChannel = regexp.MustCompile(`<!(channel|group).*?>`)
var regexSanitizeEveryone = regexp.MustCompile(`<!(everyone).*?>`)
var regexSanitizeHere = regexp.MustCompile(`<!(here).*?>`)

func SanitizeAt(msg string) string {
	for _, v := range []regexp.Regexp{regexSanitizeChannel, regexSanitizeEveryone, regexSanitizeHere} {
		m := v.FindAllStringSubmatch(msg, -1)
		for _, match := range m {
			msg = strings.Replace(msg, match[0], fmt.Sprintf("@\\%s", match[1]), 1)
		}
	}
	return msg
}

func SanitizeForChannel(msg string) string {
	return SanitizeLoose(SanitizeAt(msg))
}

func (mod *AtCommandModule) GetEmojiForResponse(result marvin.CommandResult) string {
	var reactEmoji string
	switch result.Code {
	case marvin.CmdResultOK:
		reactEmoji, _ = mod.team.ModuleConfig(Identifier).Get(confKeyEmojiOk)
	case marvin.CmdResultFailure:
		reactEmoji, _ = mod.team.ModuleConfig(Identifier).Get(confKeyEmojiFail)
	case marvin.CmdResultError:
		reactEmoji, _ = mod.team.ModuleConfig(Identifier).Get(confKeyEmojiError)
	case marvin.CmdResultNoSuchCommand:
		reactEmoji, _ = mod.team.ModuleConfig(Identifier).Get(confKeyEmojiUnkCmd)
	case marvin.CmdResultPrintUsage:
		reactEmoji, _ = mod.team.ModuleConfig(Identifier).Get(confKeyEmojiUsage)
	case marvin.CmdResultPrintHelp:
		reactEmoji, _ = mod.team.ModuleConfig(Identifier).Get(confKeyEmojiHelp)
	default:
		reactEmoji, _ = mod.team.ModuleConfig(Identifier).Get(confKeyEmojiError)
	}
	return reactEmoji
}
