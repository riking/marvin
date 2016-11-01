package at_command

import (
	"bytes"
	"fmt"
	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
	"regexp"
	"strings"
)

func init() {
	marvin.RegisterModule(NewAtCommandModule)
}

const Identifier = "autoinvite"

type AtCommandModule struct {
	team       marvin.Team
	BotUser    slack.UserID
	mentionRgx *regexp.Regexp
}

func NewAtCommandModule(t marvin.Team) marvin.Module {
	mod := &AtCommandModule{team: t}
	return mod
}

func (mod *AtCommandModule) Identifier() string {
	return Identifier
}

func (mod *AtCommandModule) Unregister(t marvin.Team) {
	t.OffAllEvents(Identifier)
}

func (mod *AtCommandModule) RegisterRTMEvents(t marvin.Team) {
	t.OnEvent(Identifier, "hello", mod.HandleHello)
	t.OnNormalMessage(Identifier, mod.HandleMessage)
}

// -----

func (mod *AtCommandModule) HandleHello(rtm slack.RTMRawMessage) {
	var err error

	mod.BotUser = mod.team.BotUser()
	mod.mentionRgx, err = regexp.Compile(fmt.Sprintf(`<@%s>`, mod.BotUser))
	if err != nil {
		panic(err)
	}
}

func (mod *AtCommandModule) HandleMessage(rtm slack.RTMRawMessage) {
	if mod.mentionRgx == nil {
		return
	}

	matches := mod.mentionRgx.FindStringIndex(rtm.Text())
	if len(matches) == 0 {
		return
	}

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
