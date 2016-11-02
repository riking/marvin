package atcommand

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

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
	team       marvin.Team
	botUser    slack.UserID
	mentionRgx *regexp.Regexp
}

func NewAtCommandModule(t marvin.Team) marvin.Module {
	mod := &AtCommandModule{team: t}
	return mod
}

func (mod *AtCommandModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *AtCommandModule) Load(t marvin.Team) {
}

func (mod *AtCommandModule) Enable(t marvin.Team) {
	t.OnEvent(Identifier, "hello", mod.HandleHello)
	t.OnNormalMessage(Identifier, mod.HandleMessage)
}

func (mod *AtCommandModule) Disable(t marvin.Team) {
	t.OffAllEvents(Identifier)
}

// -----

func (mod *AtCommandModule) HandleHello(rtm slack.RTMRawMessage) {
	var err error

	mod.botUser = mod.team.BotUser()
	mod.mentionRgx, err = regexp.Compile(fmt.Sprintf(`<@%s>`, mod.botUser))
	if err != nil {
		panic(err)
	}
}

func (mod *AtCommandModule) HandleMessage(rtm slack.RTMRawMessage) {
	if mod.mentionRgx == nil {
		util.LogBad("AtCommand regex not set up!")
		return
	}

	msgRawTxt := rtm.Text()
	matches := mod.mentionRgx.FindStringIndex(msgRawTxt)
	if len(matches) == 0 {
		return
	}

	util.LogDebug("Found mention in message", rtm.Text())
	util.LogDebug("Mention starts at", rtm.Text()[matches[0]:])
	matchIdx := matches // removed loop, limited to one command per message
	{
		args := ParseArgs(msgRawTxt, matchIdx[1])
		args.Source = marvin.ActionSourceUserMessage{Msg: rtm}
		util.LogDebug("args: [", strings.Join(args.OriginalArguments, "] ["), "]")
		result := mod.team.DispatchCommand(&args)
		mod.DispatchResponse(rtm, result)
		util.LogGood("command result:", result)
	}
}

func (mod *AtCommandModule) DispatchResponse(rtm slack.RTMRawMessage, result marvin.CommandResult) {
	reactEmoji := ""
	replyType := result.ReplyType

	if replyType == 0 {
		replyType = marvin.ReplyTypePreferChannel
	}
	result = marvin.ActionSourceUserMessage{Msg: rtm}.SendCmdReply(mod.team, result)

	switch result.Code {
	case marvin.CmdResultOK:
		reactEmoji, _ = mod.team.ModuleConfig("main").Get("emoji-ok", "white_check_mark")
	case marvin.CmdResultFailure:
		reactEmoji, _ = mod.team.ModuleConfig("main").Get("emoji-fail", "negative_squared_cross_mark")
	case marvin.CmdResultError:
		reactEmoji, _ = mod.team.ModuleConfig("main").Get("emoji-error", "warning")
	case marvin.CmdResultNoSuchCommand:
		reactEmoji, _ = mod.team.ModuleConfig("main").Get("emoji-unknown", "question")
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
	semiIdx := strings.IndexByte(cmdLine, ';')
	if semiIdx != -1 {
		argSplit = strings.Split(cmdLine, ";")
	} else {
		argSplit = shellSplit(strings.TrimLeft(cmdLine, " "))
	}

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
