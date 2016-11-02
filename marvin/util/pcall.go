package util

import "github.com/pkg/errors"

func PCall(f func() error) (err error) {
	defer func() {
		rec := recover()
		if rec != nil {
			if recErr, ok := rec.(error); ok {
				err = errors.Wrap(recErr, "panic")
			} else if recStr, ok := rec.(string); ok {
				err = errors.Errorf(recStr)
			} else {
				panic(errors.Errorf("Unrecognized panic object type=[%T] val=[%#v]", rec, rec))
			}
		}
	}()
	return f()
}
