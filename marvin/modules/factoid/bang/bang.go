package bang

import (
	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/modules/factoid"
	"github.com/riking/homeapi/marvin/slack"
)


// ---

func init() {
	marvin.RegisterModule(NewBangFactoidModule)
}

const Identifier = "factoid-bang"

type BangFactoidModule struct {
	team marvin.Team

	factoidModule *marvin.Module
	recentMessages map[slack.ChannelID]resultInfo
}

func NewBangFactoidModule(t marvin.Team) marvin.Module {
	mod := &BangFactoidModule{
		team:      t,
		recentMessages: make(map[slack.ChannelID]resultInfo),
	}
	return mod
}

func (mod *BangFactoidModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *BangFactoidModule) Load(t marvin.Team) {
	t.DependModule(Identifier, factoid.Identifier, &mod.factoidModule)
}

func (mod *BangFactoidModule) Enable(team marvin.Team) {
	team.OnNormalMessage(Identifier, mod.OnMessage)
	team.OnSpecialMessage(Identifier, []string{"message_changed"}, mod.OnEdit)
}

func (mod *BangFactoidModule) Disable(team marvin.Team) {
	team.OffAllEvents(Identifier)
}

// --

type resultInfo struct {
	Response slack.MessageID
	SideEffects bool
}

func (mod *BangFactoidModule) OnMessage(_rtm slack.RTMRawMessage) {
	rtm := slack.SlackTextMessage(_rtm)
}

func (mod *BangFactoidModule) OnEdit(_rtm slack.RTMRawMessage) {
	rtm := slack.EditMessage{_rtm}
}