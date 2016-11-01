package marvin

var allModules []ModuleConstructor
var allSetupFuncs []ModuleSetupGlobal

type Module interface {
	Identifier() ModuleID

	Unregister(t Team)
	RegisterRTMEvents(t Team)
}

type ModuleSetupGlobal func(ShockyInstance) error
type ModuleConstructor func(team Team) Module

func RegisterModule(c ModuleConstructor) {
	allModules = append(allModules, c)
}

func RegisterGlobalModuleSetup(s ModuleSetupGlobal) {
	allSetupFuncs = append(allSetupFuncs, s)
}
