package shellquote

import (
	"testing"
)

func TestTokenize(t *testing.T) {
	tokens, err := FullTokenize([]byte(`Hello, 'Example N"ame' and "Aa Bb'c"`))
	if err != nil {
		t.Error("bad eof")
	} else {
		if tokens[0] != "Hello," {
			t.Error("bad 0")
		}
		if tokens[1] != "Example N\"ame" {
			t.Error("bad 1")
		}
		if tokens[2] != "and" {
			t.Error("bad 2")
		}
		if tokens[3] != "Aa Bb'c" {
			t.Errorf("bad 3: %s", tokens[3])
		}
	}
}
