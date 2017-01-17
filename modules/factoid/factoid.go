package factoid

import (
	"context"

	"github.com/riking/marvin"
	"github.com/riking/marvin/modules/paste"
	"github.com/riking/marvin/util"
)

type API interface {
	marvin.Module

	RunFactoid(ctx context.Context, line []string, of *OutputFlags, source marvin.ActionSource) (result string, err error)
}

var _ API = &FactoidModule{}

// ---

func init() {
	marvin.RegisterModule(NewFactoidModule)
}

const Identifier = "factoid"

type FactoidModule struct {
	team marvin.Team

	pasteMod  marvin.Module

	fdataReqChan chan fdataReq
	fdataMap     map[string]map[string]fdataVal
	// send false for normal, true for urgent
	fdataSyncSignal chan bool
}

func NewFactoidModule(t marvin.Team) marvin.Module {
	mod := &FactoidModule{
		team:      t,

		fdataMap:        make(map[string]map[string]fdataVal),
		fdataReqChan:    make(chan fdataReq),
		fdataSyncSignal: make(chan bool),
	}
	return mod
}

func (mod *FactoidModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *FactoidModule) Load(t marvin.Team) {
	mod.doMigrate(t)
	mod.doSyntaxCheck(t)

	t.DependModule(mod, paste.Identifier, &mod.pasteMod) // TODO - softdepend?
	mod.registerHTTP()
}

func (mod *FactoidModule) Enable(team marvin.Team) {
	parent := marvin.NewParentCommand()
	remember := parent.RegisterCommandFunc("remember", mod.CmdRemember, helpRemember)
	forget := parent.RegisterCommandFunc("forget", mod.CmdForget, helpList)
	parent.RegisterCommand("rem", remember)
	parent.RegisterCommand("r", remember)
	parent.RegisterCommand("fg", forget)
	parent.RegisterCommandFunc("get", mod.CmdGet, helpGet)
	parent.RegisterCommandFunc("send", mod.CmdSend, helpSend)
	parent.RegisterCommandFunc("source", mod.CmdSource, helpSource)
	parent.RegisterCommandFunc("info", mod.CmdInfo, helpInfo)
	parent.RegisterCommandFunc("list", mod.CmdList, helpList)

	team.RegisterCommand("factoid", parent)
	team.RegisterCommand("f", parent) // TODO RegisterAlias
	team.RegisterCommand("remember", remember)
	team.RegisterCommand("rem", remember)
	team.RegisterCommand("r", remember)
	team.RegisterCommand("forget", forget)

	go mod.workerFDataChan()
	go mod.workerFDataSync()
}

func (mod *FactoidModule) Disable(t marvin.Team) {
	util.LogGood("Saving persistent factoid data...")
	mod.fdataSyncSignal <- true  // trigger immediate save
	mod.fdataSyncSignal <- false // ensure that save completed
	util.LogGood("... done saving factoid data.")
	t.UnregisterCommand("factoid")
	t.UnregisterCommand("f")
	t.UnregisterCommand("remember")
	t.UnregisterCommand("rem")
	t.UnregisterCommand("r")
}

// ---

func (mod *FactoidModule) Team() marvin.Team {
	return mod.team
}
