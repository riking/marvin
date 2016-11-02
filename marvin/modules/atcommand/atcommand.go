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
	mod.mentionRgx2 = regexp.MustCompile(fmt.Sprintf(`(?m:\n?\s*<@%s>\s+)`, mod.botUser))
}

func (mod *AtCommandModule) Enable(t marvin.Team) {
	t.OnNormalMessage(Identifier, mod.HandleMessage)
}

func (mod *AtCommandModule) Disable(t marvin.Team) {
	t.OffAllEvents(Identifier)
}

// -----

func (mod *AtCommandModule) HandleMessage(rtm slack.RTMRawMessage) {
	msgRawTxt := rtm.Text()
	matches := mod.mentionRgx2.FindStringIndex(msgRawTxt)
	if len(matches) == 0 {
		m := mod.mentionRgx1.FindString(msgRawTxt)
		if m != "" {
			reactEmoji, _ := mod.team.ModuleConfig("main").Get("emoji-hi", "wave")
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
		args.Source = marvin.ActionSourceUserMessage{Msg: rtm}
		util.LogDebug("args: [", strings.Join(args.OriginalArguments, "] ["), "]")
		result := mod.team.DispatchCommand(&args)
		mod.DispatchResponse(rtm, result)
		util.LogGood("command result:", result)
	}
}

func (mod *AtCommandModule) DispatchResponse(rtm slack.RTMRawMessage, result marvin.CommandResult) {
	reactEmoji := ""
	replyType := marvin.ReplyTypeInvalid

	util.LogGood(fmt.Sprintf("command reply type: %x", result.ReplyType))

	if strings.Contains(result.Message, "<!channel>") {
		// && !rtm.User().IsAdmin()
		result.Message = strings.Replace(result.Message, "<!channel>", "@\\channel", -1)
	}

	switch result.Code {
	case marvin.CmdResultOK:
		reactEmoji, _ = mod.team.ModuleConfig("main").Get("emoji-ok", "white_check_mark")
		replyType = marvin.ReplyTypeInChannel
	case marvin.CmdResultFailure:
		reactEmoji, _ = mod.team.ModuleConfig("main").Get("emoji-fail", "negative_squared_cross_mark")
		replyType = marvin.ReplyTypeShortProblem
	case marvin.CmdResultError:
		reactEmoji, _ = mod.team.ModuleConfig("main").Get("emoji-error", "warning")
		replyType = marvin.ReplyTypeShortProblem
	case marvin.CmdResultNoSuchCommand:
		reactEmoji, _ = mod.team.ModuleConfig("main").Get("emoji-unknown", "question")
		replyType = marvin.ReplyTypePM
	case marvin.CmdResultPrintUsage:
		reactEmoji, _ = mod.team.ModuleConfig("main").Get("emoji-usage", "confused")
		replyType = marvin.ReplyTypePM
	case marvin.CmdResultPrintHelp:
		reactEmoji = ""
		replyType = marvin.ReplyTypeInChannel
	default:
		replyType = marvin.ReplyTypeShortProblem
	}

	if result.ReplyType&marvin.ReplyTypeDestinations == marvin.ReplyTypeInvalid {
		result.ReplyType = result.ReplyType | replyType
	}

	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Wait()
	go func() {
		marvin.ActionSourceUserMessage{Msg: rtm}.SendCmdReply(mod.team, result)
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
