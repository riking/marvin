package awake

import (
	"net/url"
	"time"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/util"
)

func init() {
	marvin.RegisterModule(NewAwakeModule)
}

const Identifier = "awake"

type AwakeModule struct {
	team   marvin.Team
	quit   chan struct{}
	ticker *time.Ticker
}

func NewAwakeModule(t marvin.Team) marvin.Module {
	mod := &AwakeModule{team: t}
	return mod
}

func (mod *AwakeModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *AwakeModule) Load(t marvin.Team) {
}

func (mod *AwakeModule) Enable(t marvin.Team) {
	mod.ticker = time.NewTicker(20 * time.Minute)
	mod.quit = make(chan struct{})
	go mod.onTick()
}

func (mod *AwakeModule) Disable(t marvin.Team) {
	mod.ticker.Stop()
	close(mod.quit)

}

func (mod *AwakeModule) onTick() {
	var retryChan <-chan time.Time

	run := func() {
		err := mod.team.SlackAPIPostJSON("users.setActive", url.Values{}, nil)
		if err != nil {
			util.LogError(err)
			retryChan = time.After(5 * time.Minute)
		} else {
			retryChan = nil
		}
	}

	for {
		select {
		case <-mod.ticker.C:
			run()
		case <-retryChan:
			run()
		case <-mod.quit:
			return
		}
	}
}
