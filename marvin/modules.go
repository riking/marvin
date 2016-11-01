package marvin

var allModules []ModuleConstructor

//var allSetupFuncs []ModuleSetupGlobal

type Module interface {
	Identifier() ModuleID

	// Load should declare dependencies
	Load(t Team)

	// Enable has dependencies available
	Enable(t Team)

	// Disable should shut down and unregister
	Disable(t Team)
}

//type ModuleSetupGlobal func(ShockyInstance) error
type ModuleConstructor func(team Team) Module

func RegisterModule(c ModuleConstructor) {
	allModules = append(allModules, c)
}

//func RegisterGlobalModuleSetup(s ModuleSetupGlobal) {
//	allSetupFuncs = append(allSetupFuncs, s)
//}

func AllModules() []ModuleConstructor {
	return allModules
}
