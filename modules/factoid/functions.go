package factoid

import (
	"bytes"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"unicode/utf8"
)

type FactoidFunction struct {
	F func(args ...string) string

	MultiArg bool
}

func setupFunctions(mod *FactoidModule) {
	mod.RegisterFunctionSingleArg("ucase", funcUCase)
	mod.RegisterFunctionSingleArg("lcase", funcLCase)

	mod.RegisterFunctionSingleArg2("munge", funcMunge)
	mod.RegisterFunctionSingleArg2("flipraw", funcRawFlip)
	mod.RegisterFunctionSingleArg2("flip", funcFlip)
	mod.RegisterFunctionSingleArg2("reverse", funcReverse)

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

func funcMunge(arg string) string {
	return mungeReplacer.Replace(arg)
}

func funcRawFlip(arg string) string {
	return flipReplacer.Replace(arg)
}

func funcFlip(arg string) string {
	return funcReverse(funcRawFlip(arg))
}

func funcReverse(arg string) string {
	runes := []rune(arg)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

var (
	mungeReplacer = makeReplacer(false, "abcdefghijklmnoprstuwxyzABCDEGHIJKLMORSTUWYZ0123456789", "äḃċđëƒġħíĵķĺṁñöρŗšţüωχÿźÅḂÇĎĒĠĦÍĴĶĹṀÖŖŠŢŮŴỲŻ０１２３４５６７８９")
	flipFrom      = "!().12345679<>?ABCDEFGJKLMPQRTUVWY[]_abcdefghijklmnpqrtuvwy{},'\"┳（╰"
	flipTo        = "¡)(˙⇂ᄅƐㄣϛ9Ɫ6><¿∀ᗺƆᗡƎℲפᒋ丬˥WԀΌᴚ⊥∩ΛMλ][‾ɐqɔpǝɟɓɥıɾʞlɯudbɹʇnʌʍʎ}{',„┻）╯"
	flipReplacer  = makeReplacer(true, flipFrom, flipTo)
)

func makeReplacer(inverse bool, fromStr, toStr string) *strings.Replacer {
	size := 2 * len(fromStr) // XXX only works if from is ASCII-only
	if inverse {
		size = 2 * size
	}
	var args = make([]string, 0, size)
	fromIdx := 0
	toIdx := 0
	for fromIdx < len(fromStr) && toIdx < len(toStr) {
		_, fSize := utf8.DecodeRuneInString(fromStr[fromIdx:])
		_, tSize := utf8.DecodeRuneInString(toStr[toIdx:])
		args = append(args, fromStr[fromIdx:fromIdx+fSize], toStr[toIdx:toIdx+tSize])
		if inverse {
			args = append(args, toStr[toIdx:toIdx+tSize], fromStr[fromIdx:fromIdx+fSize])
		}
		fromIdx += fSize
		toIdx += tSize
	}
	return strings.NewReplacer(args...)
}
