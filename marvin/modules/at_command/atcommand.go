package at_command

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
)

func init() {
	marvin.RegisterModule(NewAtCommandModule)
}

const Identifier = "autoinvite"

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
		fmt.Println("[ERR] regex not set up!")
		return
	}

	msgRawTxt := rtm.Text()
	matches := mod.mentionRgx.FindStringIndex(msgRawTxt)
	if len(matches) == 0 {
		return
	}

	fmt.Println("[DEBUG]", "Found mention in message", rtm.Text())
	fmt.Println("[DEBUG]", "Mention starts at", rtm.Text()[matches[0]:])
	matchIdx := matches // removed loop, limited to one command per message
	{
		args := ParseArgs(msgRawTxt, matchIdx[1])
		args.Source = marvin.ActionSourceUserMessage{Msg: rtm}
		fmt.Println("[DEBUG]", "args:")
		for i, v := range args.OriginalArguments {
			fmt.Println(i, v)
		}
		fmt.Println("[DEBUG]", "args=", args.OriginalArguments)
		result := mod.team.DispatchCommand(&args)
		mod.DispatchResponse(rtm, &args, result)
		fmt.Println("[DEBUG]", "command result:", result)
	}
}

func (mod *AtCommandModule) DispatchResponse(rtm slack.RTMRawMessage, args *marvin.CommandArguments, result error) {
	reactEmoji := ""
	if cmdErr, ok := errors.Cause(result).(marvin.CommandError); ok {
		if cmdErr.Success {
			// TODO configurable
			reactEmoji = "white_check_mark"
		} else if ok && cmdErr.Code == marvin.CmdErrNoSuchCommand {
			reactEmoji = "question"
		} else {
			reactEmoji = "negative_squared_cross_mark"
		}
		err := cmdErr.SendReply(mod.team)
		if err != nil {
			fmt.Printf("[ERR] %+v\n", err) // TODO
		}
	} else if result == nil {
		reactEmoji = "white_check_mark"
	} else {
		reactEmoji = "warning"

		imChannel, err := mod.team.GetIM(rtm.UserID())
		if err != nil {
			fmt.Printf("[ERR] %+v\n", err) // TODO
		} else {
			_, _, err = mod.team.SendMessage(imChannel,
				fmt.Sprintf("Your command encountered an error. %s\n%s",
					mod.team.ArchiveURL(rtm.ChannelID(), rtm.MessageTS()),
					result.Error()))
			// TODO pm the failure to controllers?
			fmt.Println("[ERR]", args.OriginalArguments)
			fmt.Printf("[ERR] %+v\n", err) // TODO
		}
	}
	err := mod.team.ReactMessage(rtm.ChannelID(), rtm.MessageTS(), reactEmoji)
	if err != nil {
		fmt.Printf("[ERR] %+v\n", err)
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
