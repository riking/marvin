package intra

import "reflect"

type luaProxy struct {
	value reflect.Value
	t     reflect.Type
}
