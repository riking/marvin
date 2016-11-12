package factoid

import (
	"bytes"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

type FactoidFunction struct {
	F func(args ...string) string

	MultiArg bool
}

func setupFunctions(mod *FactoidModule) {
	mod.RegisterFunctionSingleArg("ucase", funcUCase)
	mod.RegisterFunctionSingleArg("lcase", funcLCase)
	mod.RegisterFunctionMultiArg("repeat", funcRepeat)
	mod.RegisterFunctionMultiArg("first", funcFirst)
	mod.RegisterFunctionMultiArg("if", funcIf)
	mod.RegisterFunctionMultiArg("rnd", funcRand)
}

func (mod *FactoidModule) RegisterFunctionMultiArg(name string, f func(args ...string) string) {
	mod.functions[name] = FactoidFunction{F: f, MultiArg: true}
}
func (mod *FactoidModule) RegisterFunctionSingleArg(name string, f func(args ...string) string) {
	mod.functions[name] = FactoidFunction{F: f, MultiArg: false}
}
func (mod *FactoidModule) RegisterFunctionSingleArg2(name string, f func(arg string) string) {
	mod.functions[name] = FactoidFunction{F: func(args ...string) string { return f(args[0]) }, MultiArg: false}
}

func funcUCase(args ...string) string {
	return strings.ToUpper(args[0])
}

func funcLCase(args ...string) string {
	return strings.ToLower(args[0])
}

func funcRepeat(args ...string) string {
	if len(args) != 2 {
		return fmt.Sprint("[Wrong number of arguments to repeat, expected 2, got ", len(args), "]")
	}
	c, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Sprint("[Argument #2 to repeat must be integer, got ", args[1], "]")
	}
	var buf bytes.Buffer
	for i := 0; i < c; i++ {
		buf.WriteString(args[0])
	}
	return buf.String()
}

func funcFirst(args ...string) string {
	for i := range args {
		if len(strings.TrimSpace(args[i])) != 0 {
			return args[i]
		}
	}
	return ""
}

func funcIf(args ...string) string {
	if len(args) != 3 && len(args) != 2 {
		return fmt.Sprint("[Wrong number of arguments to if, expected 3, got ", len(args), "]")
	}
	if len(strings.TrimSpace(args[0])) != 0 {
		if len(args) == 2 {
			return args[0]
		}
		return args[1]
	} else {
		if len(args) == 2 {
			return args[1]
		}
		return args[2]
	}
}

func funcRand(args ...string) string {
	return args[rand.Intn(len(args))]
}
