package usercache

import (
	"sync"

	"fmt"

	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
)

type API interface {
	marvin.Module

	GetEntry(userid slack.UserID) (slack.User, error)
	UpdateEntry(userobject slack.User) error
	UpdateEntries(userobjects []*slack.User) error
}

var _ API = &UserCacheModule{}

// ---
func init() {
	marvin.RegisterModule(NewUserCacheModule)
}

const Identifier = "usercache"

type UserCacheModule struct {
	team marvin.Team

	cacheLock sync.Mutex
	cacheMap  map[slack.UserID]slack.User
}

func NewUserCacheModule(t marvin.Team) marvin.Module {
	mod := &UserCacheModule{
		team:     t,
		cacheMap: make(map[slack.UserID]slack.User),
	}
	return mod
}

func (mod *UserCacheModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *UserCacheModule) Load(t marvin.Team) {
	t.DB().MustMigrate(Identifier, 1505192548, sqlMigrate1)
	t.DB().SyntaxCheck(sqlGetAllEntries, sqlGetEntry, sqlAddEntry, sqlUpdateEntry)
}

func (mod *UserCacheModule) Enable(team marvin.Team) {
	go func() {
		err := mod.LoadEntries()
		if err != nil {
			fmt.Errorf("Error whilst updating entries: %s", err.Error())
			return
		}
	}()
}

func (mod *UserCacheModule) Disable(t marvin.Team) {
}
