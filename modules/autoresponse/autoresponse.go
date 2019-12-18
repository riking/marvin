package autoresponse

import (
	"regexp"

	"github.com/riking/marvin"
	"github.com/riking/marvin/modules/antiflood"
	"github.com/riking/marvin/slack"
)

// ---

func init() {
	marvin.RegisterModule(NewAutoResponseModule)
}

const Identifier = "autoresponse"

type AutoResponseModule struct {
	team marvin.Team
}

func NewAutoResponseModule(t marvin.Team) marvin.Module {
	mod := &AutoResponseModule{
		team: t,
	}
	return mod
}

func (mod *AutoResponseModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *AutoResponseModule) Load(t marvin.Team) {
}

func (mod *AutoResponseModule) Enable(team marvin.Team) {
	team.OnNormalMessage(Identifier, mod.OnMessage)
	//team.OnSpecialMessage(Identifier, []string{"message_changed"}, mod.OnEdit)
}

func (mod *AutoResponseModule) Disable(team marvin.Team) {
	team.OffAllEvents(Identifier)
}

// ---

type AutoEmojiResponse struct {
	Regexp *regexp.Regexp
	Emoji  string
}

var responses = []AutoEmojiResponse{
	{regexp.MustCompile("(?i:beyonc[eé])"), "hankey"},
	{regexp.MustCompile("(?i:thank).*<@U2E00L22Y>"), "pray"},
}

// ---

func (mod *AutoResponseModule) OnMessage(_rtm slack.RTMRawMessage) {
	rtm := slack.SlackTextMessage(_rtm)
	text := rtm.Text()
	// TODO: This module needs to check user's level with mod.team.UserLevel(_rtm.UserID())
	//       However since it is not currently in use, the following code will stay here.
	if !mod.team.GetModule(antiflood.Identifier).(antiflood.API).CheckChannel(rtm.ChannelID()) {
		return
	}

	for _, v := range responses {
		if v.Regexp.FindString(text) != "" {
			mod.team.ReactMessage(rtm.MessageID(), v.Emoji)
		}
	}
}
