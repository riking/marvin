package factoid

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/riking/marvin"
	"github.com/riking/marvin/modules/antiflood"
	"github.com/riking/marvin/modules/atcommand"
	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/util"
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
	//if -2 == t.DependModule(mod, Identifier, &mod.factoidModule) {
	//	panic("Failure in dependency")
	//}
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

	if rtm.UserID() == "USLACKBOT" || rtm.UserID() == mod.team.BotUser() || rtm.ChannelID() == "D00" {
		return
	}
	if _, isThread := _rtm["thread_ts"]; isThread {
		// TODO thread operation
		return
	}
	result, of := mod.Process(rtm, false)
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
	rtm := slack.EditMessage{RTMRawMessage: _rtm}
	if rtm.EditingUserID() == "" {
		return // unfurl edit
	}
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
	result, of := mod.Process(rtm, true)
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
		"text":    []string{atcommand.SanitizeForChannel(result)},
		"parse":   []string{"client"},
	}
	util.LogIfError(mod.team.SlackAPIPostJSON("chat.update", form, nil))
}

func (mod *BangFactoidModule) Process(rtm slack.SlackTextMessage, isEditing bool) (string, OutputFlags) {
	var of OutputFlags

	if len(rtm.Text()) == 0 {
		return "", of
	}
	fchars, _ := mod.team.ModuleConfig(Identifier).Get("factoid-char")
	if !strings.ContainsAny(rtm.Text()[:1], fchars) {
		return "", of
	}
	userLvl := mod.team.UserLevel(rtm.UserID())
	if userLvl < marvin.AccessLevelNormal {
		mod.team.ReactMessage(rtm.MessageID(), "x")
		return "", of
	}
	// Check anti flood module.
	if !isEditing && userLvl < marvin.AccessLevelAdmin &&
		!mod.team.GetModule(antiflood.Identifier).(antiflood.API).CheckChannel(rtm.ChannelID()) {
		return "", of
	}
	text := slack.UnescapeTextAll(rtm.Text()[1:])
	line := strings.Split(text, " ")

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	source := &marvin.ActionSourceUserMessage{Team: mod.team, Msg: rtm}

	result, err := mod.team.GetModule(Identifier).(API).RunFactoid(ctx, line, &of, source)
	if err == ErrNoSuchFactoid {
		return "", of
	} else if err != nil {
		result = fmt.Sprintf("Error: %s", err)
	} else if of.NoReply {
		return "", of
	} else if of.Pre {
		result = fmt.Sprintf("```\n%s\n```", result)
	}
	util.LogGood(fmt.Sprintf("Factoid result:\n%s\n%s", line, result))
	return result, of
}
