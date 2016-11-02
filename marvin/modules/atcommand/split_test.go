package atcommand

import (
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
