package factoid

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/modules/atcommand"
	"github.com/riking/homeapi/marvin/slack"
	"github.com/riking/homeapi/marvin/util"
)

// ---

func init() {
	marvin.RegisterModule(NewBangFactoidModule)
}

const BangIdentifier = "factoid-bang"

type BangFactoidModule struct {
	team marvin.Team

	factoidModule  marvin.Module
	messagesLock   sync.Mutex
	recentMessages map[slack.MessageID]resultInfo
}

func NewBangFactoidModule(t marvin.Team) marvin.Module {
	mod := &BangFactoidModule{
		team:           t,
		recentMessages: make(map[slack.MessageID]resultInfo),
	}
	return mod
}

func (mod *BangFactoidModule) Identifier() marvin.ModuleID {
	return BangIdentifier
}

func (mod *BangFactoidModule) Load(t marvin.Team) {
	if -2 == t.DependModule(mod, Identifier, &mod.factoidModule) {
		panic("Failure in dependency")
	}
	t.ModuleConfig(Identifier).Add("factoid-char", "!")
}

func (mod *BangFactoidModule) Enable(team marvin.Team) {
	team.OnNormalMessage(BangIdentifier, mod.OnMessage)
	team.OnSpecialMessage(BangIdentifier, []string{"message_changed"}, mod.OnEdit)
}

func (mod *BangFactoidModule) Disable(team marvin.Team) {
	team.OffAllEvents(BangIdentifier)
}

// --

type resultInfo struct {
	Response    slack.MessageID
	SideEffects bool
}

func (mod *BangFactoidModule) OnMessage(_rtm slack.RTMRawMessage) {
	rtm := slack.SlackTextMessage(_rtm)

	result, of := mod.Process(rtm)
	if result == "" {
		return
	}
	sentMsgID, _, err := mod.team.SendMessage(rtm.ChannelID(), " "+atcommand.SanitizeForChannel(result))
	if err != nil {
		util.LogError(err)
		return
	}
	record := resultInfo{Response: slack.MsgID(rtm.ChannelID(), sentMsgID), SideEffects: of.SideEffects}
	mod.messagesLock.Lock()
	mod.recentMessages[rtm.MessageID()] = record
	mod.messagesLock.Unlock()
}

func (mod *BangFactoidModule) OnEdit(_rtm slack.RTMRawMessage) {
	rtm := slack.EditMessage{_rtm}
	_ = rtm
	time.Sleep(350 * time.Millisecond)

	mod.messagesLock.Lock()
	record, ok := mod.recentMessages[rtm.MessageID()]
	mod.messagesLock.Unlock()

	if !ok {
		return
	}
	if record.SideEffects {
		util.LogIfError(mod.team.ReactMessage(record.Response, "eject"))
		imChannel, _ := mod.team.GetIM(rtm.EditingUserID())
		_, _, err := mod.team.SendMessage(imChannel, fmt.Sprintf(
			"Factoids with side effects cannot be edited.\n%s", mod.team.ArchiveURL(rtm.MessageID())))
		util.LogIfError(err)
		return
	}
	result, of := mod.Process(rtm)
	if result == "" {
		result = "(removed)"
	}
	if of.SideEffects {
		record.SideEffects = true
		mod.messagesLock.Lock()
		mod.recentMessages[rtm.MessageID()] = record
		mod.messagesLock.Unlock()
	}
	form := url.Values{
		"channel": []string{string(record.Response.ChannelID)},
		"ts":      []string{string(record.Response.MessageTS)},
		"text":    []string{result},
		"parse":   []string{"client"},
	}
	util.LogIfError(mod.team.SlackAPIPostJSON("chat.update", form, nil))
}

func (mod *BangFactoidModule) Process(rtm slack.SlackTextMessage) (string, OutputFlags) {
	var of OutputFlags

	if len(rtm.Text()) == 0 {
		return "", of
	}
	fchars, _ := mod.team.ModuleConfig(Identifier).Get("factoid-char")
	if !strings.ContainsAny(rtm.Text()[:1], fchars) {
		return "", of
	}
	text := rtm.Text()[1:]
	line := strings.Split(text, " ")

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	source := &marvin.ActionSourceUserMessage{Team: mod.team, Msg: rtm}

	result, err := mod.factoidModule.(API).RunFactoid(ctx, line, &of, source)
	if err == ErrNoSuchFactoid {
		return "", of
	} else if err != nil {
		result = fmt.Sprintf("Error: %s", err)
	} else if of.NoReply {
		return "", of
	} else if of.Pre {
		result = fmt.Sprintf("```\n%s\n```", result)
	}
	return result, of
}
