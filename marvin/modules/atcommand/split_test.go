package atcommand

import (
	"fmt"
	"reflect"
	"testing"
)

func TestShellSplit(t *testing.T) {
	var tests = []struct {
		Name  string
		Raw   string
		Split []string
	}{
		{"simple", "Hello there everybody", []string{"Hello", "there", "everybody"}},
		{"empty", "", []string{}},
		{"one", "Hi", []string{"Hi"}},
		{"quoted-string", `Hi "Mr. Dent" and 'Ford Prefect'`, []string{"Hi", "Mr. Dent", "and", "Ford Prefect"}},
		{"remember-tricky", `remember -- "--" ; Several words that make up one string`, []string{"remember", "--", "--", "Several words that make up one string"}},
		{"quoted-semicolon", `Hey ";" there`, []string{"Hey", ";", "there"}},
	}
	for _, v := range tests {
		t.Run(v.Name, func(t *testing.T) {
			split := shellSplit(v.Raw)
			if len(split) != len(v.Split) {
				t.Fatalf("Length is wrong. Wanted (%v)[%d], got (%v)[%d]", v.Split, len(v.Split), split, len(split))
			}
			for i := range split {
				if split[i] != v.Split[i] {
					t.Errorf("Argument %d is wrong, wanted [%s] got [%s]", i, v.Split[i], split[i])
				}
			}
		})
	}
}

func TestCodeBlocks(t *testing.T) {
	var tests = []struct {
		Name  string
		Raw   string
		Split []string
	}{
		{"correct input", "@marvin echo &1\n```\nHello\nthere\n```", []string{"Hello\nthere"}},
		{"trailing newline", "@marvin echo &1\n```\nHello\nthere\n```\n", []string{"Hello\nthere"}},
		{"no inner leading newline", "@marvin echo &1\n```Hello\nthere\n```", []string{"Hello\nthere"}},

		{"only a code block", "```test```", []string{"test", "test2"}},
		{"only a multiline code block", "```test\nline 2```", []string{"test\nline 2"}},
		{"three newlines", "```\n\n\n\n\n```", []string{"\n\n\n"}},
		{"two code blocks on one line", "```test```   ```test2```", []string{"test", "test2"}},
	}
	for _, v := range tests {
		t.Run(v.Name, func(t *testing.T) {
			match := rgxCodeBlock.FindAllStringSubmatch(v.Raw, -1)
			filterMatch := make([]string, len(match))
			for i := range match {
				filterMatch[i] = match[i][1]
				fmt.Printf("[%#v]\n", match[i])
			}
			if !reflect.DeepEqual(filterMatch, v.Split) {
				t.Errorf("WANT: %v\nGOT : %#v", v.Split, filterMatch)
			}
		})
	}
}
