package wolfgame

import (
	"sync"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
)

func init() {
	marvin.RegisterModule(NewWolfGameModule)
}

const Identifier = "wolfgame"

type WolfGameModule struct {
	team marvin.Team

	lock    sync.Mutex
	players []Player
}

type Player struct {
	ID slack.UserID
}

func NewWolfGameModule(t marvin.Team) marvin.Module {
	mod := &WolfGameModule{
		team: t,
	}
	return mod
}

func (mod *WolfGameModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *WolfGameModule) Load(t marvin.Team) {
}

func (mod *WolfGameModule) Enable(team marvin.Team) {
}

func (mod *WolfGameModule) Disable(t marvin.Team) {
}
