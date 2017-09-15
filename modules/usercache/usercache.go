package usercache

import (
	"fmt"
	"strconv"
	"time"

	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
	"github.com/riking/marvin/slack/rtm"
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
	t.ModuleConfig(Identifier).Add("last-timestamp", "0")
	t.ModuleConfig(Identifier).Add("delay", (72 * time.Hour).String())
}

func (mod *UserCacheModule) Enable(team marvin.Team) {
	go func() {
		fmt.Printf("Loading user cache entries....\n")
		err := mod.LoadEntries()
		if err != nil {
			fmt.Printf("Error whilst updating entries: %s\n", err.Error())
			return
		}

		fmt.Printf("Loaded all entries from the user cache.\n")
		go mod.UpdateTask()
	}()
}

func (mod *UserCacheModule) Disable(t marvin.Team) {
}

func (mod *UserCacheModule) UpdateTask() {
	rtmClient := mod.team.GetRTMClient().(*rtm.Client)

	for {
		timestr, _, _ := mod.team.ModuleConfig(Identifier).GetIsDefault("last-timestamp")
		delaystr, _, _ := mod.team.ModuleConfig(Identifier).GetIsDefault("delay")
		timeint, _ := strconv.ParseInt(timestr, 10, 64)
		var timeres = time.Unix(timeint, 0)
		delayres, err := time.ParseDuration(delaystr)

		if err != nil || timeres.Before(time.Now().Add(-delayres)) {
			fmt.Printf("Repolling user list....\n")
			go rtmClient.FillUsersList()
			err = mod.team.ModuleConfig(Identifier).Set("last-timestamp", strconv.FormatInt(time.Now().Unix(), 10))
		}
		time.Sleep(1 * time.Hour)
	}
}
