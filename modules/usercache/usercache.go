package usercache

import (
	"fmt"

	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
)

// interface duplicated in rtm package
type API interface {
	marvin.Module

	GetEntry(userid slack.UserID) (slack.User, error)
	LoadEntries() error
	UpdateEntry(userobject *slack.User) error
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
}

func NewUserCacheModule(t marvin.Team) marvin.Module {
	mod := &UserCacheModule{
		team: t,
	}
	return mod
}

func (mod *UserCacheModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *UserCacheModule) Load(t marvin.Team) {
	t.DB().MustMigrate(Identifier, 1505192548, sqlMigrate1)
	t.DB().SyntaxCheck(sqlGetAllEntries, sqlGetEntry, sqlUpsertEntry)
}

func (mod *UserCacheModule) Enable(team marvin.Team) {
	go func() {
		fmt.Printf("Loading cache entries....\n")
		err := mod.LoadEntries()
		if err != nil {
			fmt.Printf("Error whilst updating entries: %s\n", err.Error())
			return
		}
		fmt.Printf("Loaded all entries from the cache.\n")
	}()
}

func (mod *UserCacheModule) Disable(t marvin.Team) {
}
