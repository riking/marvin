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

	selfMS := t.GetModuleStatus(self.Identifier())
	if selfMS == nil {
		panic(errors.Errorf("DependModule() self parameter is not loaded?"))
	}
	dependMS := t.GetModuleStatus(dependID)
	if dependMS == nil {
		return -1
	}

	selfMS.Dependencies = append(selfMS.Dependencies, oneModuleDependency{
		Identifier: dependMS.Identifier,
		Pointer:    ptr,
	})
	if dependMS.Degraded() {
		return -2
	}
	if dependMS.State == ModuleStateEnabled {
		*ptr = dependMS.Instance
		return 1
	}
	return 0
}

func (t *Team) GetModuleStatus(ident marvin.ModuleID) *moduleStatus {
	for i, ms := range t.modules {
		if ms.Identifier == ident {
			return &t.modules[i]
		}
	}
	return nil
}

func (t *Team) GetModule(ident marvin.ModuleID) marvin.Module {
	ms := t.GetModuleStatus(ident)
	if ms != nil {
		return ms.Instance
	}
	return nil
}

func (t *Team) EnableModule(ident marvin.ModuleID) error {
	var idx int = -1

	for i, ms := range t.modules {
		if ms.Identifier == ident {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errors.Errorf("No such module '%s'", ident)
	}

	switch t.modules[idx].State {
	case ModuleStateEnabled:
		// Do nothing
		return nil
	case ModuleStateConstructed:
	case ModuleStateErrorLoading:
	default:
		return errors.Errorf("module must complete loading first")
	case ModuleStateLoaded:
	case ModuleStateErrorEnabling:
	case ModuleStateDisabled:
		// OK
		break
	}

	err := t.enableModule2(&t.modules[idx])
	if err != nil {
		return errors.Wrapf(err, "Could not enable '%s'", ident)
	}

	for _, v := range t.modules[idx].Dependencies {
		*v.Pointer = t.modules[idx].Instance
	}
	return nil
}

func (t *Team) DisableModule(ident marvin.ModuleID) error {
	var idx int = -1

	for i, ms := range t.modules {
		if ms.Identifier == ident {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errors.Errorf("No such module '%s'", ident)
	}

	switch t.modules[idx].State {
	case ModuleStateDisabled:
	case ModuleStateLoaded:
	case ModuleStateConstructed:
		// Do nothing
		return nil
	case ModuleStateErrorLoading:
	case ModuleStateErrorEnabling:
	default:
		return errors.Errorf("module must complete loading first")
	case ModuleStateEnabled:
		// OK
		break
	}

	for _, v := range t.modules[idx].Dependencies {
		*v.Pointer = nil
	}

	err := protectedCallT(t, t.modules[idx].Instance.Disable)

	if err != nil {
		return errors.Wrapf(err, "Failure disabling '%s'", ident)
	}
	return nil
}

// Loading

type ModuleState int

const (
	_ ModuleState = iota
	ModuleStateConstructed
	ModuleStateLoaded
	ModuleStateEnabled
	ModuleStateDisabled
	ModuleStateErrorLoading
	ModuleStateErrorEnabling
)

type oneModuleDependency struct {
	Identifier marvin.ModuleID
	Pointer    *marvin.Module
}

type moduleStatus struct {
	Identifier    marvin.ModuleID
	Instance      marvin.Module
	State         ModuleState
	DegradeReason error
	Dependencies  []oneModuleDependency
}

func (ms *moduleStatus) IsEnabled() bool {
	return ms.DegradeReason == nil && ms.State == ModuleStateEnabled
}

func (ms *moduleStatus) Degraded() bool {
	return ms.DegradeReason != nil
}

func (t *Team) constructModules() {
	var modList []moduleStatus
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
		modList = append(modList, moduleStatus{
			Instance:     mod,
			Identifier:   mod.Identifier(),
			State:        ModuleStateConstructed,
			Dependencies: nil,
		})
	}
	t.modules = modList
}

type sortModules []moduleStatus

func (sm sortModules) Len() int      { return len(sm) }
func (sm sortModules) Swap(i, j int) { var tmp moduleStatus; tmp = sm[i]; sm[i] = sm[j]; sm[j] = tmp }
func (sm sortModules) Less(i, j int) bool {
	for _, v := range sm[j].Dependencies {
		if v.Identifier == sm[i].Identifier {
			return true
		}
	}
	return false
}

func (t *Team) loadModules() {
	for i, v := range t.modules {
		err := protectedCallT(t, v.Instance.Load)
		if err != nil {
			t.modules[i].State = ModuleStateErrorLoading
			t.modules[i].DegradeReason = err
			util.LogBadf("Module %s failed to load: %v\n", t.modules[i].Identifier, err)
			continue
		}
		util.LogGood("Loaded module", t.modules[i].Identifier)
		t.modules[i].State = ModuleStateLoaded
	}

	sort.Sort(sortModules(t.modules))
}

func (t *Team) enableModules() {
	conf := t.ModuleConfig("modules")
	for i, ms := range t.modules {
		desired, _ := conf.Get(fmt.Sprintf("%s.enabled", ms.Identifier), "on")
		if desired == ConfTurnOffModule {
			t.modules[i].State = ModuleStateDisabled
			util.LogWarn("Left disabled module", t.modules[i].Identifier)
			continue
		}

		// Set dependency pointers
		util.LogDebug(ms.Identifier, "depends:", ms.Dependencies)
		ok := true
		for _, v := range ms.Dependencies {
			dependMS := t.GetModuleStatus(v.Identifier)
			if !dependMS.IsEnabled() {
				util.LogWarnf("Enabling module %s failed: dependency %s is not enabled\n%v", ms.Identifier, dependMS.Identifier, dependMS)
				ok = false
				break
			}
			*v.Pointer = dependMS.Instance
		}
		if !ok {
			continue
		}

		t.enableModule2(&t.modules[i])
	}
}

func (t *Team) enableModule2(ms *moduleStatus) error {
	err := protectedCallT(t, ms.Instance.Enable)
	if err != nil {
		ms.State = ModuleStateErrorEnabling
		ms.DegradeReason = err
		protectedCallT(t, ms.Instance.Disable)
		util.LogBadf("Enabling module %s failed: %+v\n", ms.Identifier, err)
		return err
	}
	util.LogGood("Enabled module", ms.Identifier)
	ms.State = ModuleStateEnabled
	ms.DegradeReason = nil
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
