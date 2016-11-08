package awake

import (
	"github.com/riking/homeapi/marvin"
	"time"
	"sync"
	"net/url"
)

func init() {
	marvin.RegisterModule(NewAwakeModule)
}

const Identifier = "core"

type AwakeModule struct {
	team marvin.Team
	lock sync.Mutex
	ticker time.Ticker
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
	mod.lock.Lock()
	mod.ticker = time.NewTicker(20*time.Minute)
	mod.lock.Unlock()
}

func (mod *AwakeModule) Disable(t marvin.Team) {
	mod.lock.Lock()
	mod.ticker.Stop()
	close(mod.ticker.C)
	mod.lock.Unlock()
}

func (mod *AwakeModule) onTick() {
	for range mod.ticker.C {
		mod.team.SlackAPIPost("users.setActive", url.Values{})
	}
}