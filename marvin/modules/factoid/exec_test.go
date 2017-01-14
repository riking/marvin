package factoid

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/util/mock"
)

type MockFactoidModule struct {
	*FactoidModule
}

func GetMockFactoidModule() *FactoidModule {
	fm := &FactoidModule{
		team: nil,
		functions: map[string]FactoidFunction{
			"add1": {
				F: func(args ...string) string {
					return "1" + strings.Join(args, "")
				},
				MultiArg: false,
			},
		},
	}
	setupFunctions(fm)
	return fm
}

var argsEmpty []string

func testFactoidArgs(t *testing.T, rawSource string, args []string, as marvin.ActionSource, expect string) {
	mod := GetMockFactoidModule()
	fi := &Factoid{
		Mod:        mod,
		IsBareInfo: true,
		RawSource:  rawSource,
	}
	if args == nil {
		args = argsEmpty
	}
	var of OutputFlags
	result, err := mod.exec_parse(context.Background(), fi, rawSource, args, &of, as)
	if err != nil {
		t.Errorf("Unexpected error running factoid [%s]: %+v", rawSource, err)
	} else if expect != result {
		t.Errorf("Wrong output running [%s]:\nEXP: %s\nGOT: %s\n", rawSource, expect, result)
	}
}

type stackTracer interface {
	error
	StackTrace() errors.StackTrace
}

func testFactoidArgsErr(t *testing.T, rawSource string, args []string, as marvin.ActionSource, errMatch string) {
	mod := GetMockFactoidModule()
	fi := &Factoid{
		Mod:        mod,
		IsBareInfo: true,
		RawSource:  rawSource,
	}
	if args == nil {
		args = argsEmpty
	}
	var of OutputFlags
	_, err := mod.exec_parse(context.Background(), fi, rawSource, args, &of, as)
	if err == nil {
		t.Errorf("Expected error '%s' but got none: [%s]", errMatch, rawSource)
	} else if !strings.Contains(err.Error(), errMatch) {
		if stErr, ok := err.(stackTracer); ok {
			fmt.Printf("[ERR] %s %#v\n", stErr.Error(), stErr.StackTrace())
		} else {
			fmt.Printf("[ERR] %+v\n", err)
		}
		t.Errorf("Wrong error running [%s]:\nEXP: %s\nGOT: %s+v\n", rawSource, errMatch, err)
	}
}

func TestPlainText(t *testing.T) {
	s := mock.ActionSource{}
	testFactoidArgs(t, "Hello, World!", nil, s, "Hello, World!")
	testFactoidArgs(t, "Hello, {World!", nil, s, "Hello, {World!")
	testFactoidArgs(t, "{Hello, World!", nil, s, "{Hello, World!")
	testFactoidArgs(t, "Hello, {}World!", nil, s, "Hello, {}World!")
	testFactoidArgs(t, "{noreply}Hello, World!", nil, s, "")
}

func TestArgParam(t *testing.T) {
	s := mock.ActionSource{}
	testFactoidArgs(t, "Hello, %arg0%!", []string{"World"}, s, "Hello, World!")
	testFactoidArgs(t, "Hello, %args%!", []string{"World"}, s, "Hello, World!")
	testFactoidArgs(t, "Hello, %args%", []string{"World!"}, s, "Hello, World!")
	testFactoidArgs(t, "Hello, %args%!", []string{"big", "wide", "World"}, s, "Hello, big wide World!")
	testFactoidArgs(t, "%arg0% slaps %arg1-% with a giant trout",
		[]string{"Fred", "Barney", "Rubble"}, s,
		"Fred slaps Barney Rubble with a giant trout")
	testFactoidArgsErr(t, "%arg0% slaps %arg1-% with a giant trout",
		[]string{}, s, "Not enough args")
	testFactoidArgsErr(t, "%arg0% slaps %arg1-% with a giant trout",
		[]string{"Fred"}, s, "Not enough args")
}

func TestFunctions(t *testing.T) {
	s := mock.ActionSource{}

	testFactoidArgs(t, "$add1($add1($add1($add1(1))))", nil, s, "11111")
	testFactoidArgs(t, "$add1($notafunction($add1($add1(1))))", nil, s, "1$notafunction(111)")
	testFactoidArgs(t, "$notafunction($add1($add1(1))", nil, s, "$notafunction(111")
	testFactoidArgs(t, "$$$cashmoney$add1(00)$$$", nil, s, "$$$cashmoney100$$$")
}

func TestLua(t *testing.T) {
	s := mock.ActionSource{}

	testFactoidArgsErr(t, `{lua}"hello"`, nil, s, "syntax error")
	testFactoidArgs(t, `{lua}return "hello"`, nil, s, "hello")
	testFactoidArgs(t, `{lua}return 42`, nil, s, "42")
	testFactoidArgs(t, `{lua}print("hello") print(", ") print("world")`, nil, s, "hello, world")
	testFactoidArgs(t, `{lua}return "hello" .. " world"`, nil, s, "hello world")
}

func TestFlipMunge(t *testing.T) {
	s := mock.ActionSource{}

	testFactoidArgs(t, `$munge(jamie)`, nil, s, "ĵäṁíë")
	testFactoidArgs(t, `$munge($munge(jamie))`, nil, s, "ĵäṁíë")
	testFactoidArgs(t, `$munge(ĵäṁíë)`, nil, s, "ĵäṁíë")
	testFactoidArgs(t, `$flip(World)`, nil, s, "plɹoM")
	testFactoidArgs(t, `$flip($flip(World))`, nil, s, "World")
	testFactoidArgs(t, `$flip(plɹoM)`, nil, s, "World")
	testFactoidArgs(t, `$reverse($flipraw(World))`, nil, s, "plɹoM")
}

func BenchmarkPlainFactoidParse(b *testing.B) {
	s := mock.ActionSource{}
	mod := GetMockFactoidModule()
	fi := &Factoid{
		Mod:        mod,
		IsBareInfo: true,
		RawSource:  "Hello, World!",
	}
	args := argsEmpty
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var of OutputFlags
		result, err := mod.exec_parse(ctx, fi, fi.RawSource, args, &of, s)
		if err != nil {
			b.FailNow()
		}
		if result != "Hello, World!" {
			b.FailNow()
		}
	}
}

func BenchmarkLuaFactoid(b *testing.B) {
	s := mock.ActionSource{}
	mod := GetMockFactoidModule()
	fi := &Factoid{
		Mod:        mod,
		IsBareInfo: true,
		RawSource:  "{lua}return \"Hello, World!\"",
	}
	args := argsEmpty
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var of OutputFlags
		result, err := mod.exec_parse(ctx, fi, fi.RawSource, args, &of, s)
		if err != nil {
			b.FailNow()
		}
		if result != "Hello, World!" {
			b.FailNow()
		}
	}
}
