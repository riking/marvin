package controller

import (
	"sort"

	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/util"
)

const ConfTurnOffModule = "off"

// DependModule places the instance of the requested module in the given pointer.
//
// If the requested module is already enabled, the pointer is filled immediately and the function returns 1.
// If the requested module has errored, the pointer is left along and the function returns -2.
// During loading, when the requested module has not been enabled yet, the function returns 0 and remembers the pointer.
// If the requested module is not known, the function returns -1.
func (t *Team) DependModule(self marvin.Module, dependID marvin.ModuleID, ptr *marvin.Module) int {
	t.modulesLock.Lock()
	defer t.modulesLock.Unlock()

	selfMS := t.getModuleStatus(self.Identifier())
	if selfMS == nil {
		panic(errors.Errorf("DependModule() self parameter is not loaded?"))
	}
	dependMS := t.getModuleStatus(dependID)
	if dependMS == nil {
		return -1
	}

	selfMS.Dependencies = append(selfMS.Dependencies, oneModuleDependency{
		Identifier: dependMS.identifier,
		Pointer:    ptr,
	})
	if dependMS.Degraded() {
		return -2
	}
	if dependMS.state == marvin.ModuleStateEnabled {
		*ptr = dependMS.instance
		return 1
	}
	return 0
}

func (t *Team) getModuleStatus(ident marvin.ModuleID) *moduleStatus {
	for _, ms := range t.modules {
		if ms.identifier == ident {
			return ms
		}
	}
	return nil
}

func (t *Team) GetModule(ident marvin.ModuleID) marvin.Module {
	ms := t.getModuleStatus(ident)
	if ms != nil {
		return ms.instance
	}
	return nil
}

func (t *Team) GetModuleStatus(ident marvin.ModuleID) marvin.ModuleStatus {
	ms := t.getModuleStatus(ident)
	if ms != nil {
		return ms
	}
	return nil
}

func (t *Team) GetAllModules() []marvin.ModuleStatus {
	var all []marvin.ModuleStatus

	t.modulesLock.Lock()
	defer t.modulesLock.Unlock()
	for i := range t.modules {
		all = append(all, t.modules[i])
	}
	return all
}

func (t *Team) GetAllEnabledModules() []marvin.ModuleStatus {
	var all []marvin.ModuleStatus

	t.modulesLock.Lock()
	defer t.modulesLock.Unlock()
	for _, v := range t.modules {
		if v.state == marvin.ModuleStateEnabled {
			all = append(all, v)
		}
	}
	return all
}

func (t *Team) EnableModule(ident marvin.ModuleID) error {
	var idx int = -1

	for i, ms := range t.modules {
		if ms.identifier == ident {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errors.Errorf("No such module '%s'", ident)
	}

	switch t.modules[idx].state {
	case marvin.ModuleStateEnabled:
		// Do nothing
		return nil
	case marvin.ModuleStateConstructed:
	case marvin.ModuleStateErrorLoading:
	default:
		return errors.Errorf("module must complete loading first")
	case marvin.ModuleStateLoaded:
	case marvin.ModuleStateErrorEnabling:
	case marvin.ModuleStateDisabled:
		// OK
		break
	}

	err := t.enableModule2(t.modules[idx])
	if err != nil {
		return errors.Wrapf(err, "Could not enable '%s'", ident)
	}

	for _, v := range t.modules[idx].Dependencies {
		*v.Pointer = t.modules[idx].instance
	}
	return nil
}

func (t *Team) DisableModule(ident marvin.ModuleID) error {
	var ms *moduleStatus

	for _, v := range t.modules {
		if v.identifier == ident {
			ms = v
			break
		}
	}
	if ms == nil {
		return errors.Errorf("No such module '%s'", ident)
	}

	switch ms.state {
	case marvin.ModuleStateDisabled:
	case marvin.ModuleStateLoaded:
	case marvin.ModuleStateConstructed:
		// Do nothing
		return nil
	case marvin.ModuleStateErrorLoading:
	case marvin.ModuleStateErrorEnabling:
	default:
		return errors.Errorf("module must complete loading first")
	case marvin.ModuleStateEnabled:
		// OK
		break
	}

	err := protectedCallT(t, ms.instance.Disable)

	for _, v := range ms.Dependencies {
		*v.Pointer = nil
	}

	if err != nil {
		return errors.Wrapf(err, "Failure disabling '%s'", ident)
	}
	return nil
}

// Loading

type oneModuleDependency struct {
	Identifier marvin.ModuleID
	Pointer    *marvin.Module
}

type moduleStatus struct {
	identifier    marvin.ModuleID
	instance      marvin.Module
	state         marvin.ModuleState
	degradeReason error
	Dependencies  []oneModuleDependency
}

func (ms *moduleStatus) Identifier() marvin.ModuleID {
	return ms.identifier
}

func (ms *moduleStatus) Instance() marvin.Module {
	return ms.instance
}

func (ms *moduleStatus) State() marvin.ModuleState {
	return ms.state
}

func (ms *moduleStatus) IsLoaded() bool {
	return ms.state == marvin.ModuleStateLoaded || ms.state == marvin.ModuleStateErrorEnabling
}

func (ms *moduleStatus) IsEnabled() bool {
	return ms.state == marvin.ModuleStateEnabled
}

func (ms *moduleStatus) Degraded() bool {
	return ms.degradeReason != nil
}

func (ms *moduleStatus) Err() error {
	return ms.degradeReason
}

func (t *Team) constructModules() {
	var modList []*moduleStatus
	var err error

	for _, constructor := range marvin.AllModules() {
		var mod marvin.Module
		err = protectedCallT(t, func(team marvin.Team) {
			mod = constructor(team)
		})
		if err != nil {
			util.LogWarnf("Could not construct module: %s",
				strings.Replace(fmt.Sprintf("%+v", err), "\n", "\t\n", -1))
			continue
		}
		id := mod.Identifier()
		for _, v := range modList {
			if v.identifier == id {
				panic(errors.Errorf("Duplicate identifier %s", id))
			}
		}
		modList = append(modList, &moduleStatus{
			instance:     mod,
			identifier:   id,
			state:        marvin.ModuleStateConstructed,
			Dependencies: nil,
		})
	}
	t.modules = modList
}

type sortModules []*moduleStatus

func (sm sortModules) Len() int      { return len(sm) }
func (sm sortModules) Swap(i, j int) { var tmp *moduleStatus; tmp = sm[i]; sm[i] = sm[j]; sm[j] = tmp }
func (sm sortModules) Less(i, j int) bool {
	for _, v := range sm[j].Dependencies {
		if v.Identifier == sm[i].identifier {
			return true
		}
	}
	return false
}

func (t *Team) loadModules() {
	for _, v := range t.modules {
		err := protectedCallT(t, v.instance.Load)
		if err != nil {
			v.state = marvin.ModuleStateErrorLoading
			v.degradeReason = err
			util.LogBadf("Module %s failed to load: %v\n", v.identifier, err)
			continue
		}
		util.LogGood("Loaded module", v.identifier)
		v.state = marvin.ModuleStateLoaded

		// Lock configuration
		_ = t.ModuleConfig(v.identifier)
		t.confLock.Lock()
		mc := t.confMap[v.identifier]
		t.confLock.Unlock()
		mc.DefaultsLocked = true
	}

	sort.Sort(sortModules(t.modules))
}

func (t *Team) enableModules() {
	conf := t.ModuleConfig("modules")
	for _, ms := range t.modules {
		desired, _, _ := conf.GetIsDefault(string(ms.identifier))
		if desired == ConfTurnOffModule {
			ms.state = marvin.ModuleStateDisabled
			util.LogWarn("Left disabled module", ms.identifier)
			continue
		}

		// Set dependency pointers
		ok := true
		for _, v := range ms.Dependencies {
			dependMS := t.getModuleStatus(v.Identifier)
			if !dependMS.IsEnabled() {
				util.LogWarnf("Enabling module %s failed: dependency %s is not enabled\n%v", ms.identifier, dependMS.identifier, dependMS)
				ok = false
				break
			}
			*v.Pointer = dependMS.instance
		}
		if !ok {
			continue
		}

		t.enableModule2(ms)
	}
}

func (t *Team) enableModule2(ms *moduleStatus) error {
	err := protectedCallT(t, ms.instance.Enable)
	if err != nil {
		ms.state = marvin.ModuleStateErrorEnabling
		ms.degradeReason = err
		protectedCallT(t, ms.instance.Disable)
		util.LogBadf("Enabling module %s failed: %+v\n", ms.identifier, err)
		return err
	}
	util.LogGood("Enabled module", ms.identifier)
	ms.state = marvin.ModuleStateEnabled
	ms.degradeReason = nil
	return nil
}

func protectedCallT(t marvin.Team, f func(t marvin.Team)) (err error) {
	defer func() {
		rec := recover()
		if rec != nil {
			if recErr, ok := rec.(error); ok {
				err = errors.Wrap(recErr, "panic")
			} else if recStr, ok := rec.(string); ok {
				err = errors.Errorf(recStr)
			} else {
				err = errors.Errorf("Unrecognized panic object type=[%T] val=[%#v]", rec, rec)
			}
		}
	}()

	f(t)
	return nil
}
