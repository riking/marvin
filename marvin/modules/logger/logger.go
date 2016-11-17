package logger

import "github.com/riking/homeapi/marvin"

func init() {
	marvin.RegisterModule(NewLoggerModule)
}

const Identifier = "logger"

type LoggerModule struct {
	team marvin.Team
}

func NewLoggerModule(t marvin.Team) marvin.Module {
	mod := &LoggerModule{team: t}
	return mod
}

func (mod *LoggerModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *LoggerModule) Load(t marvin.Team) {
}

func (mod *LoggerModule) Enable(t marvin.Team) {
}

func (mod *LoggerModule) Disable(t marvin.Team) {
}
