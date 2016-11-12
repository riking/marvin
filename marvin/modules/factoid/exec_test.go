package factoid

import (
	"testing"

	"strings"

	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/util/mock"
)

func MockFactoidModule() *FactoidModule {
	return &FactoidModule{
		team:    nil,
		onReact: nil,
	}
}

func testFactoidSimple(t *testing.T, rawSource string, expect string) {
	fi := FactoidInfo{
		IsBareInfo: true,
		RawSource:  rawSource,
	}
	as := mock.ActionSource{}
	result, err := MockFactoidModule().RunFactoid(fi, as, nil)
	if err != nil {
		t.Errorf("Unexpected error running factoid [%s]: %+v", rawSource, err)
	} else if expect != result {
		t.Errorf("Wrong output running [%s]:\nEXP: %s\nGOT: %s\n", rawSource, expect, result)
	}
}

func testFactoidArgs(t *testing.T, rawSource string, args []string, source marvin.ActionSource, expect string) {
	fi := FactoidInfo{
		IsBareInfo: true,
		RawSource:  rawSource,
	}
	as := mock.ActionSource{}
	result, err := MockFactoidModule().RunFactoid(fi, as, args)
	if err != nil {
		t.Errorf("Unexpected error running factoid [%s]: %+v", rawSource, err)
	} else if expect != result {
		t.Errorf("Wrong output running [%s]:\nEXP: %s\nGOT: %s\n", rawSource, expect, result)
	}
}

func testFactoidArgsErr(t *testing.T, rawSource string, args []string, source marvin.ActionSource, errMatch string) {
	fi := FactoidInfo{
		IsBareInfo: true,
		RawSource:  rawSource,
	}
	as := mock.ActionSource{}
	_, err := MockFactoidModule().RunFactoid(fi, as, args)
	if err == nil {
		t.Errorf("Expected error '%s' but got none: [%s]", errMatch, rawSource)
	} else if !strings.Contains(err.Error(), errMatch) {
		t.Errorf("Wrong error running [%s]:\nEXP: %s\nGOT: %s\n", rawSource, errMatch, err.Error())
	}
}

func TestPlainText(t *testing.T) {
	testFactoidSimple(t, "Hello, World!", "Hello, World!")
	testFactoidSimple(t, "Hello, {World!", "Hello, {World!")
	testFactoidSimple(t, "{Hello, World!", "{Hello, World!")
	testFactoidSimple(t, "Hello, {}World!", "Hello, {}World!")
	testFactoidSimple(t, "{noreply}Hello, World!", "")
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
	mod := MockFactoidModule()
	// TODO mod.DefineFactoidFunction

	mod.collectTokenize("$add1($add1($add1($add1(1))))")
}
