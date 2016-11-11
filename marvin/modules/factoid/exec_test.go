package factoid

import (
	"testing"

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

func testFactoidArgs(t *testing.T, rawSource string, args []string, expect string) {
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

func TestPlainText(t *testing.T) {
	testFactoidSimple(t, "Hello, World!", "Hello, World!")
	testFactoidSimple(t, "Hello, {World!", "Hello, {World!")
	testFactoidSimple(t, "{Hello, World!", "{Hello, World!")
	testFactoidSimple(t, "Hello, {}World!", "Hello, {}World!")
	testFactoidSimple(t, "{noreply}Hello, World!", "")
}

func TestArgParam(t *testing.T) {
	testFactoidArgs(t, "Hello, %arg0%!", []string{"World"}, "Hello, World!")
	testFactoidArgs(t, "Hello, %args%!", []string{"World"}, "Hello, World!")
	testFactoidArgs(t, "Hello, %args%!", []string{"big", "wide", "World"}, "Hello, big wide World!")
	testFactoidArgs(t, "%arg0% slaps %arg1-% with a giant trout", []string{"Fred", "Barney", "Rubble"}, "Fred slaps Barney Rubble with a giant trout")
}
