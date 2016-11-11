package factoid

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/riking/homeapi/marvin"
)

func (mod *FactoidModule) RunFactoid(info FactoidInfo, source marvin.ActionSource, args []string) (string, error) {
	if len(info.RawSource) == 0 || strings.HasPrefix(info.RawSource, "{noreply}") {
		return "", nil
	}

	tokens := info.Tokens()
	// Handle only directives in the first loop
	directiveIdx := -1
	for i, v := range tokens {
		if dt, ok := v.(DirectiveToken); ok {
			directiveIdx = i
			if dt.Directive == "noreply" {
				return "", nil
			}
		} else {
			break
		}
	}
	tokens = tokens[directiveIdx+1:]
	var buf bytes.Buffer
	for _, v := range tokens {
		str, err := v.Run(mod, source, args)
		if err != nil {
			return "", errors.Wrapf(err, "processing factoid")
		}
		buf.WriteString(str)
	}
	return buf.String(), nil
}

//WRITE TESTS

func (fi *FactoidInfo) Tokens() []Token {
	fi.tokenize.Do(func() {
		fi.tokens = tokenize(fi.RawSource, false)
	})
	return fi.tokens
}

func tokenize(source string, recursed bool) []Token {
	var tokens []Token

	fmt.Println("tokenizing:", source)

	// Get all directives
	// Directives are anchored to beginning of factoid
	m := DirectiveTokenRgx.FindStringSubmatchIndex(source)
	for recursed == false && m != nil {
		directive := source[m[2]:m[3]]
		tokens = append(tokens, DirectiveToken{Directive: directive})
		source = source[m[1]:]
		m = DirectiveTokenRgx.FindStringSubmatchIndex(source)
	}
	// Parameter directives
	m = ParameterTokenRgx.FindStringSubmatchIndex(source)
	for m != nil {
		fmt.Println(m)
		var opStr, startStr, rangeStr, endStr string
		if m[2] != -1 {
			opStr = source[m[2]:m[3]]
		}
		if m[4] != -1 {
			startStr = source[m[4]:m[5]]
		}
		if m[6] != -1 {
			rangeStr = source[m[6]:m[7]]
		}
		if m[8] != -1 {
			endStr = source[m[8]:m[9]]
		}
		t := NewParameterToken(source[m[0]:m[1]], opStr, startStr, rangeStr, endStr)
		prev := tokenize(source[:m[0]], true)
		tokens = append(tokens, prev...)
		tokens = append(tokens, t)
		source = source[m[1]:]
		m = ParameterTokenRgx.FindStringSubmatchIndex(source)
	}
	tokens = append(tokens, TextToken{Text: source})
	fmt.Println(tokens)
	return tokens
}
