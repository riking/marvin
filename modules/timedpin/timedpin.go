package timedpin

import (
	"github.com/riking/marvin"
)

func init() {
	marvin.RegisterModule(NewTimedPinModule)
}

const Identifier = "timedpin"

type TimedPinModule struct {
	team marvin.Team
}

func NewTimedPinModule(t marvin.Team) marvin.Module {
	mod := &TimedPinModule{team: t}
	return mod
}

func (mod *TimedPinModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *TimedPinModule) Load(t marvin.Team) {
	t.DB().MustMigrate(Identifier, 1479767598, "")
	t.DB().SyntaxCheck()
}

func (mod *TimedPinModule) Enable(t marvin.Team) {
	t.RegisterCommandFunc("timedpin", mod.CommandTimedPin, "`@marvin timedpin <duration: 10h30m> <slack archive link>`\n"+
		"Pins the linked message to the current channel, and unpins the message after the given duration expires.")

}

func (mod *TimedPinModule) Disable(t marvin.Team) {
}

// ---

const (
	sqlMigrate1 = `
	CREATE TABLE module_timedpin_pins (
		id SERIAL PRIMARY KEY,
		varchar(10)  channel,
		varchar(30)  ts_or_file,
	)`
)

func (mod *TimedPinModule) CommandTimedPin(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {

}
