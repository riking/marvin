package controller

import (
	"sort"

	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/riking/homeapi/marvin"
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

	for i, val := range t.modules {
		if val.Identifier == dependID {
			t.modules[i].Dependencies = append(t.modules[i].Dependencies, oneModuleDependency{
				Identifier: self.Identifier(),
				Pointer:    ptr,
			})
			if val.Degraded() {
				return -2
			}
			if val.State == ModuleStateEnabled {
				*ptr = t.modules[i].Instance
				return 1
			}
			return 0
		}
	}
	return -1
}

func (t *Team) GetModule(ident marvin.ModuleID) marvin.Module {
	for _, ms := range t.modules {
		if ms.Identifier == ident {
			return ms.Instance
		}
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
	return ms.DegradeReason != nil && ms.State == ModuleStateEnabled
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
			// TODO central logging
			fmt.Fprintf(os.Stderr, "[WARN] Could not construct module: %s",
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
			continue
		}
		t.modules[i].State = ModuleStateLoaded
	}

	sort.Sort(sortModules(t.modules))
}

func (t *Team) enableModules() {
	conf := t.ModuleConfig("modules")
	for i, ms := range t.modules {
		desired, err := conf.Get(fmt.Sprintf("%s.enabled", ms.Identifier))
		if err != nil {
			t.modules[i].DegradeReason = errors.Wrap(err, "Could not determine desired state")
			t.modules[i].State = ModuleStateErrorEnabling
			continue
		}
		if desired == "false" || desired == ConfTurnOffModule {
			t.modules[i].State = ModuleStateDisabled
			continue
		}

		// Set dependency pointers
		for _, v := range ms.Dependencies {
			for _, m2 := range t.modules {
				if m2.Identifier == v.Identifier && m2.IsEnabled() {
					*v.Pointer = m2.Instance
					break
				}
			}
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
		return err
	}
	ms.State = ModuleStateEnabled
	ms.DegradeReason = nil
	return nil
}

func protectedCallT(t marvin.Team, f func(t marvin.Team)) (err error) {
	defer func() {
		rec := recover()
		if rec != nil {
			if recErr, ok := rec.(error); ok {
				err = recErr
			} else if recStr, ok := rec.(string); ok {
				err = errors.Errorf(recStr)
			} else {
				panic(errors.Errorf("Unrecognized panic object type=[%T] val=[%#v]", rec, rec))
			}
		}
	}()

	f(t)
	return nil
}
