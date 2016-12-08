package factoid

import (
	"context"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/modules/paste"
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

	functions map[string]FactoidFunction
	pasteMod  marvin.Module
}

func NewFactoidModule(t marvin.Team) marvin.Module {
	mod := &FactoidModule{
		team:      t,
		functions: make(map[string]FactoidFunction),
	}
	return mod
}

func (mod *FactoidModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *FactoidModule) Load(t marvin.Team) {
	mod.doMigrate(t)
	mod.doSyntaxCheck(t)

	setupFunctions(mod)
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
}

func (mod *FactoidModule) Disable(t marvin.Team) {
	t.UnregisterCommand("factoid")
	t.UnregisterCommand("f")
	t.UnregisterCommand("remember")
	t.UnregisterCommand("rem")
	t.UnregisterCommand("r")
}

// ---
