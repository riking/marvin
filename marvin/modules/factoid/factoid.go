package factoid

import (
	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/slack"
)

type API interface {
	marvin.Module

	RunFactoidSplit(args []string, source marvin.ActionSource)
	RunFactoidLine(rawLine string, source marvin.ActionSource)
	RunFactoidMessage(msg slack.RTMRawMessage)
}

// ---

func init() {
	marvin.RegisterModule(NewFactoidModule)
}

const Identifier = "factoid"

type FactoidModule struct {
	team marvin.Team

	functions map[string]FactoidFunction
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
}

func (mod *FactoidModule) Enable(team marvin.Team) {
	parent := marvin.NewParentCommand()
	remember := parent.RegisterCommandFunc("remember", mod.CmdRemember, "`@marvin remember [--local] [name] [value]` (alias `r`) saves a factoid.")
	parent.RegisterCommand("rem", remember)
	parent.RegisterCommand("r", remember)
	parent.RegisterCommandFunc("get", mod.CmdGet, "`factoid get <name> [args...]` runs a factoid with the standard argument parsing instead of the factoid argument parsing.")
	parent.RegisterCommandFunc("send", mod.CmdSend, "`factoid send <channel> <name> [args...]` sends the result of a factoid to another channel.")
	parent.RegisterCommandFunc("source", mod.CmdSource, "`factoid source <name>` views the source of a factoid.")
	parent.RegisterCommandFunc("info", mod.CmdInfo, "`factoid info [-f] <name>` views detailed information about a factoid. Add -f to use the most recent forgotten entry.")

	team.RegisterCommand("factoid", parent)
	team.RegisterCommand("f", parent) // TODO RegisterAlias
	team.RegisterCommand("remember", remember)
	team.RegisterCommand("rem", remember)
	team.RegisterCommand("r", remember)
}

func (mod *FactoidModule) Disable(t marvin.Team) {
	t.UnregisterCommand("factoid")
	t.UnregisterCommand("f")
	t.UnregisterCommand("remember")
	t.UnregisterCommand("rem")
	t.UnregisterCommand("r")
}

// ---

func (mod *FactoidModule) ProcessMsg(msg string, source marvin.ActionSource) {
	// TODO !factoid
}
