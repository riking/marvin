package antiflood

import (
	"fmt"
	"github.com/riking/marvin/util"
	"strings"
	"sync"
	"time"

	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
)

func init() {
	marvin.RegisterModule(NewAntifloodModule)
}

const Identifier = "antiflood"

type API interface {
	marvin.Module

	CheckChannel(channelID slack.ChannelID) bool
}

var _ API = &AntifloodModule{}

type AntifloodModule struct {
	marvin.Module

	team marvin.Team

	threshold time.Duration

	recentChannels map[slack.ChannelID]time.Time
	antifloodMutex sync.Mutex
}

func NewAntifloodModule(t marvin.Team) marvin.Module {
	mod := &AntifloodModule{
		team:           t,
		recentChannels: make(map[slack.ChannelID]time.Time),
	}
	return mod
}

func (mod *AntifloodModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *AntifloodModule) Load(t marvin.Team) {
	c := mod.team.ModuleConfig(Identifier)
	c.AddProtect(confKeyMsgThreshold, confThresholdDefault, true)
	c.OnModify(func(key string) {
		if strings.Compare(key, confKeyMsgThreshold) == 0 {
			go mod.ReloadConfig()
		}
	})
}

func (mod *AntifloodModule) Enable(t marvin.Team) {
	mod.ReloadConfig()
}

func (mod *AntifloodModule) Disable(t marvin.Team) {
}

// -----

const (
	confKeyMsgThreshold  = "threshold"
	confThresholdDefault = "10s"
)

func (mod *AntifloodModule) ReloadConfig() {
	val, _ := mod.team.ModuleConfig(Identifier).Get(confKeyMsgThreshold)
	threshold, err := time.ParseDuration(val)
	if err != nil {
		util.LogBad(fmt.Sprintf("[%s] Configuration item %s is tainted, switching to default value of %s",
			mod.team.Domain(), confKeyMsgThreshold, confThresholdDefault))
		threshold, _ = time.ParseDuration(confThresholdDefault)
	}
	mod.antifloodMutex.Lock()
	defer mod.antifloodMutex.Unlock()
	mod.threshold = threshold
}

func (mod *AntifloodModule) CheckChannel(channelID slack.ChannelID) bool {
	if channelID[0] == 'C' || channelID[0] == 'G' {
		mod.antifloodMutex.Lock()
		defer mod.antifloodMutex.Unlock()
		if val, ok := mod.recentChannels[channelID]; ok {
			if !val.Before(time.Now().Add(-mod.threshold)) || val.Unix() == 0 {
				return false
			} else {
				mod.recentChannels[channelID] = time.Now()
				return true
			}
		} else if !mod.team.TeamConfig().CheckChannelName(mod.team.ChannelName(channelID)) {
			mod.recentChannels[channelID] = time.Unix(0, 0)
			return false
		} else {
			// Should only be called only once per unrecognized channel.
			mod.recentChannels[channelID] = time.Now()
		}
	}
	return true
}
