package factoid

import (
	"testing"

	"strings"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/util/mock"
)

var mockFactoidModule = &FactoidModule{
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

func MockFactoidModule() *FactoidModule {
	return mockFactoidModule
}

func testFactoidArgs(t *testing.T, rawSource string, args []string, as marvin.ActionSource, expect string) {
	mod := MockFactoidModule()
	fi := Factoid{
		mod:        mod,
		IsBareInfo: true,
		RawSource:  rawSource,
	}
	var of OutputFlags
	line := append([]string{"__mock_factoid_name"}, args...)
	result, err := mod.exec_parse(fi, rawSource, line, &of, as)
	if err != nil {
		t.Errorf("Unexpected error running factoid [%s]: %+v", rawSource, err)
	} else if expect != result {
		t.Errorf("Wrong output running [%s]:\nEXP: %s\nGOT: %s\n", rawSource, expect, result)
	}
}

func testFactoidArgsErr(t *testing.T, rawSource string, args []string, as marvin.ActionSource, errMatch string) {
	mod := MockFactoidModule()
	fi := Factoid{
		mod:        mod,
		IsBareInfo: true,
		RawSource:  rawSource,
	}
	var of OutputFlags
	line := append([]string{"__mock_factoid_name"}, args...)
	_, err := mod.exec_parse(fi, rawSource, line, &of, as)
	if err == nil {
		t.Errorf("Expected error '%s' but got none: [%s]", errMatch, rawSource)
	} else if !strings.Contains(err.Error(), errMatch) {
		t.Errorf("Wrong error running [%s]:\nEXP: %s\nGOT: %s\n", rawSource, errMatch, err.Error())
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

	testFactoidArgs(t, "$add1($add1($add1($add1(1))))", []string{}, s, "11111")
	testFactoidArgs(t, "$add1($notafunction($add1($add1(1))))", []string{}, s, "1$notafunction(111)")
	testFactoidArgs(t, "$notafunction($add1($add1(1))", []string{}, s, "$notafunction(111")
	testFactoidArgs(t, "$$$cashmoney$add1(00)$$$", []string{}, s, "$$$cashmoney100$$$")
}
