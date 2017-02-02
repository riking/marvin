package util

import "github.com/pkg/errors"

type TriValue int

const (
	TriNo      = -1
	TriDefault = 0
	TriYes     = 1
)

func (t TriValue) MarshalJSON() ([]byte, error) {
	if t == TriNo {
		return []byte(`false`), nil
	} else if t == TriYes {
		return []byte(`true`), nil
	} else if t == TriDefault {
		return []byte(`null`), nil
	} else {
		return nil, errors.Errorf("cannot marshal int(%d) as a TriValue", int(t))
	}
}

func (t *TriValue) UnmarshalJSON(b []byte) error {
	if b == nil || string(b) == "null" {
		*t = TriDefault
	} else if string(b) == "true" {
		*t = TriYes
	} else if string(b) == "false" {
		*t = TriNo
	} else {
		return errors.Errorf("unrecognized encoding for TriValue: %.20s", string(b))
	}
	return nil
}
