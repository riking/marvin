package atcommand

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"sync"

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
	botUser     slack.UserID
	mentionRgx2 *regexp.Regexp
	mentionRgx1 *regexp.Regexp
}

func NewAtCommandModule(t marvin.Team) marvin.Module {
	mod := &AtCommandModule{team: t}
	return mod
}

func (mod *AtCommandModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *AtCommandModule) Load(t marvin.Team) {
	mod.botUser = mod.team.BotUser()
	mod.mentionRgx1 = regexp.MustCompile(fmt.Sprintf(`<@%s>`, mod.botUser))
	mod.mentionRgx2 = regexp.MustCompile(fmt.Sprintf(`(?m:(\n|^)\s*<@%s>\s+)`, mod.botUser))

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
}

func (mod *AtCommandModule) Disable(t marvin.Team) {
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

func (mod *AtCommandModule) HandleMessage(rtm slack.RTMRawMessage) {
	msgRawTxt := rtm.Text()
	matches := mod.mentionRgx2.FindStringIndex(msgRawTxt)
	if len(matches) == 0 {
		m := mod.mentionRgx1.FindString(msgRawTxt)
		if m != "" {
			reactEmoji, _ := mod.team.ModuleConfig(Identifier).Get(confKeyEmojiHi)
			mod.team.ReactMessage(rtm.MessageID(), reactEmoji)
		}
		return
	}

	util.LogDebug("Found mention in message", rtm.Text())
	util.LogDebug("Mention starts at", rtm.Text()[matches[0]:])
	matchIdx := matches // removed loop, limited to one command per message
	{
		args := ParseArgs(msgRawTxt, matchIdx[1])
		if len(args.OriginalArguments) == 0 {
			util.LogDebug("Mention has no arguments, stopping")
			return
		}
		util.LogDebug("args: [", strings.Join(args.OriginalArguments, "] ["), "]")

		source := marvin.ActionSourceUserMessage{Msg: rtm, Team: mod.team}
		args.Source = source
		result := mod.team.DispatchCommand(&args)

		mod.DispatchResponse(rtm, result, source)
		util.LogGood("command result:", result)
	}
}

func (mod *AtCommandModule) DispatchResponse(rtm slack.RTMRawMessage, result marvin.CommandResult, source marvin.ActionSourceUserMessage) {
	reactEmoji := ""
	replyType := marvin.ReplyTypeInvalid

	util.LogGood(fmt.Sprintf("command reply type: %x", result.ReplyType))

	if strings.Contains(result.Message, "<!channel>") {
		// && !rtm.User().IsAdmin()
		result.Message = strings.Replace(result.Message, "<!channel>", "@\\channel", -1)
	}
	if strings.Contains(result.Message, "<!everyone>") {
		// && !rtm.User().IsAdmin()
		result.Message = strings.Replace(result.Message, "<!everyone>", "@\\everyone", -1)
	}
	if strings.Contains(result.Message, "<!here|@here>") {
		// && !rtm.User().IsAdmin()
		result.Message = strings.Replace(result.Message, "<!here|@here>", "@\\here", -1)
	}
	if strings.HasPrefix(result.Message, "/") {
		result.Message = "." + result.Message
	}

	switch result.Code {
	case marvin.CmdResultOK:
		reactEmoji, _ = mod.team.ModuleConfig(Identifier).Get(confKeyEmojiOk)
		replyType = marvin.ReplyTypeInChannel
	case marvin.CmdResultFailure:
		reactEmoji, _ = mod.team.ModuleConfig(Identifier).Get(confKeyEmojiFail)
		replyType = marvin.ReplyTypeShortProblem
	case marvin.CmdResultError:
		reactEmoji, _ = mod.team.ModuleConfig(Identifier).Get(confKeyEmojiError)
		replyType = marvin.ReplyTypeShortProblem
	case marvin.CmdResultNoSuchCommand:
		reactEmoji, _ = mod.team.ModuleConfig(Identifier).Get(confKeyEmojiUnkCmd)
		replyType = marvin.ReplyTypePM
	case marvin.CmdResultPrintUsage:
		reactEmoji, _ = mod.team.ModuleConfig(Identifier).Get(confKeyEmojiUsage)
		replyType = marvin.ReplyTypePM
	case marvin.CmdResultPrintHelp:
		reactEmoji, _ = mod.team.ModuleConfig(Identifier).Get(confKeyEmojiHelp)
		replyType = marvin.ReplyTypeInChannel
	default:
		reactEmoji, _ = mod.team.ModuleConfig(Identifier).Get(confKeyEmojiError)
		replyType = marvin.ReplyTypeShortProblem
	}

	if result.ReplyType&marvin.ReplyTypeDestinations == marvin.ReplyTypeInvalid {
		result.ReplyType = result.ReplyType | replyType
	}

	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()
	go func() {
		source.SendCmdReply(result)
		wg.Done()
	}()

	if reactEmoji == "" {
		return
	}
	err := mod.team.ReactMessage(rtm.MessageID(), reactEmoji)
	if err != nil {
		util.LogError(errors.Wrap(err, "reacting to command"))
	}
}

func ParseArgs(raw string, match int) marvin.CommandArguments {
	endOfLine := strings.IndexByte(raw[match:], '\n')
	if endOfLine == -1 {
		endOfLine = len(raw)
	}
	cmdLine := raw[match:endOfLine]

	var argSplit []string
	argSplit = shellSplit(strings.TrimLeft(cmdLine, " "))

	for i, v := range argSplit {
		str := strings.TrimSpace(v)

		argSplit[i] = str
	}
	// TODO code block support

	var args marvin.CommandArguments
	args.Arguments = argSplit
	args.OriginalArguments = argSplit
	return args
}

// TODO(kyork) this code sucks, need to find / write replacement
func shellSplit(s string) []string {
	split := strings.Split(s, " ")

	var result []string
	var inquote string
	var block bytes.Buffer

	for _, i := range split {
		if inquote == "" {
			if strings.HasPrefix(i, "'") || strings.HasPrefix(i, "\"") {
				inquote = string(i[0])
				block.Reset()
				block.WriteString(strings.TrimPrefix(i, inquote))
				block.WriteByte(' ')
			} else {
				result = append(result, i)
			}
		} else {
			if !strings.HasSuffix(i, inquote) {
				block.WriteString(i)
				block.WriteByte(' ')
			} else {
				block.WriteString(strings.TrimSuffix(i, inquote))
				inquote = ""
				result = append(result, block.String())
				block.Reset()
			}
		}
	}

	return result
}
