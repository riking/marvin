package core

import (
	"fmt"
	"strings"

	"github.com/riking/marvin"
	"github.com/riking/marvin/slack"
)

func init() {
	marvin.RegisterModule(NewDebugModule)
}

const Identifier = "core"

type DebugModule struct {
	team marvin.Team
}

func NewDebugModule(t marvin.Team) marvin.Module {
	mod := &DebugModule{team: t}
	return mod
}

func (mod *DebugModule) Identifier() marvin.ModuleID {
	return Identifier
}

func (mod *DebugModule) Load(t marvin.Team) {
}

const (
	helpSet = "`set [module] [key] [value]` sets a module configuration value."
	helpGet = "`get [module] [key]` shows module configuration values.\n" +
		"\tProtected configuration values may only be viewed by admins over DMs."
	helpList = "`list [module]` lists available module configuration values.\n" +
		"\tProtected configuration values are marked by a (*)."
)

func (mod *DebugModule) Enable(t marvin.Team) {
	parent := marvin.NewParentCommand().WithHelp(
		"The `config` command manipulates team-wide configuration. Most subcommands are restricted to admins.\n" +
			helpSet + "\n" + helpGet + "\n" + helpList,
	)
	parent.RegisterCommandFunc("set", mod.CommandConfigSet, helpSet)
	parent.RegisterCommandFunc("get", mod.CommandConfigGet, helpGet)
	parent.RegisterCommandFunc("list", mod.CommandConfigList, helpList)
	t.RegisterCommand("config", parent)
}

func (mod *DebugModule) Disable(t marvin.Team) {
	t.UnregisterCommand("config")
}

// ---

func (mod *DebugModule) CommandConfigList(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	switch len(args.Arguments) {
	case 0:
		var modList []string
		for _, v := range mod.team.ModuleConfigList() {
			ms := mod.team.GetModuleStatus(v)
			if ms == nil || (ms != nil && ms.IsEnabled()) {
				modList = append(modList, fmt.Sprintf("`%s`", v))
			} else {
				modList = append(modList, fmt.Sprintf("~`%s`~", v))
			}
		}
		return marvin.CmdUsage(args, fmt.Sprintf("Usage: `@marvin config list [module]`\nModules: %s", strings.Join(modList, " "))).WithSimpleUndo()
	}

	module := marvin.ModuleID(args.Arguments[0])
	conf := mod.team.ModuleConfig(module)
	if conf == nil {
		return marvin.CmdFailuref(args, "No such module `%s`", module).WithSimpleUndo()
	}

	var keyList []string

	prot := conf.ListProtected()
	for key := range conf.ListDefaults() {
		isProt := ""
		if prot[key] {
			isProt = " (\\*)"
		}
		keyList = append(keyList, fmt.Sprintf(
			"`%s`%s", key, isProt))
	}

	return marvin.CmdSuccess(args, fmt.Sprintf("Configuration values for %s:\n%s", module, strings.Join(keyList, ", "))).WithSimpleUndo()
}

func (mod *DebugModule) CommandConfigGet(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	switch len(args.Arguments) {
	default:
		fallthrough
	case 0:
		return marvin.CmdUsage(args, "Usage: `@marvin config get [module] [key]`")
	case 1:
		return mod.CommandConfigList(t, args)
	case 2:
		break
	}

	module := marvin.ModuleID(args.Arguments[0])
	key := args.Arguments[1]

	var val string
	var isDefault bool
	var err error
	if args.Source.AccessLevel() >= marvin.AccessLevelAdmin && slack.IsDMChannel(args.Source.ChannelID()) {
		val, isDefault, err = mod.team.ModuleConfig(module).GetIsDefault(key)
	} else {
		val, isDefault, err = mod.team.ModuleConfig(module).GetIsDefaultNotProtected(key)
	}
	if _, ok := err.(marvin.ErrConfProtected); ok {
		return marvin.CmdFailuref(args, "`%s.%s` is a protected configuration value. Viewing is restricted to admin DMs.", module, key).WithSimpleUndo()
	} else if _, ok := err.(marvin.ErrConfNoDefault); ok {
		return marvin.CmdFailuref(args, "`%s.%s` is not a configuration value.", module, key).WithSimpleUndo()
	} else if err != nil {
		return marvin.CmdError(args, err, "Database error").WithNoUndo()
	} else if isDefault {
		return marvin.CmdSuccess(args, fmt.Sprintf("%s _(default)_", val)).WithSimpleUndo()
	}
	return marvin.CmdSuccess(args, val).WithSimpleUndo()
}

func (mod *DebugModule) CommandConfigSet(t marvin.Team, args *marvin.CommandArguments) marvin.CommandResult {
	switch len(args.Arguments) {
	default:
		fallthrough
	case 0, 1:
		return marvin.CmdUsage(args, "Usage: `@marvin config set {module} {key} [value]`\nIf a value is not specified, the key will be reset to default.").WithSimpleUndo()
	case 2, 3:
		break
	}
	if args.Source.AccessLevel() < marvin.AccessLevelAdmin {
		return marvin.CmdFailuref(args, "Sorry, %v, I can't let you do that. `config set` is restricted to admins.", args.Source.UserID()).WithSimpleUndo()
	}

	module := marvin.ModuleID(args.Arguments[0])
	key := args.Arguments[1]

	conf := mod.team.ModuleConfig(module)
	if len(args.Arguments) == 3 {
		value := args.Arguments[2]
		err := conf.Set(key, value)
		if err != nil {
			return marvin.CmdError(args, err, "Database error")
		}
		return marvin.CmdSuccess(args, "Configuration value set").WithNoUndo()
	} else if conf == nil {
		return marvin.CmdFailuref(args, "'%s' is not a valid module name", module).WithSimpleUndo()
	} else {
		err := conf.SetDefault(key)
		if err != nil {
			return marvin.CmdError(args, err, "Database error")
		}
		return marvin.CmdSuccess(args, "Configuration value reset to default").WithNoUndo()
	}
}
