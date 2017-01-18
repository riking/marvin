package marvin

var allModules []ModuleConstructor

// ModuleConstructor is the type of the function that init() must pass to RegisterModule.
type ModuleConstructor func(team Team) Module

// RegisterModule should be called during package init() and stores the constructor for a module.
func RegisterModule(c ModuleConstructor) {
	allModules = append(allModules, c)
}

// AllModules returns all constructors given to RegisterModule().
func AllModules() []ModuleConstructor {
	return allModules
}
