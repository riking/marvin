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

// Tokens can panic, make sure it gets called with PCall() before saving a factoid to the database.
func (fi *FactoidInfo) Tokens() []Token {
	fi.tokenize.Do(func() {
		tokens := fi.mod.collectTokenize(fi.RawSource)
		fmt.Println("result:", tokens)
		fi.tokens = tokens
	})
	return fi.tokens
}

func (mod *FactoidModule) collectTokenize(source string) []Token {
	var tokens []Token
	tokenCh := make(chan Token)

	go mod.tokenize(source, false, tokenCh)
	for v := range tokenCh {
		tokens = append(tokens, v)
	}
	return tokens
}

func (mod *FactoidModule) tokenize(source string, recursed bool, tokenCh chan<- Token) {
	fmt.Println("tokenizing:", source)

	// Get all directives
	// Directives are anchored to beginning of factoid
	m := DirectiveTokenRgx.FindStringSubmatchIndex(source)
	for recursed == false && m != nil {
		directive := source[m[2]:m[3]]
		tokenCh <- DirectiveToken{Directive: directive}
		source = source[m[1]:]
		m = DirectiveTokenRgx.FindStringSubmatchIndex(source)
	}
	// Function directives
	m = FunctionTokenRgx.FindStringSubmatchIndex(source)
	for m != nil {
		fmt.Println(m)
		if source[m[2]:m[3]] != "" {
			m[0]++
		}
		mod.tokenize(source[:m[0]], true, tokenCh)
		func_name := source[m[4]:m[5]]
		// TODO getFunction(func_name)
		start := m[5]
		end := -999
		nesting := 0
		for i := start; i < len(source); i++ {
			var b byte = source[i]
			if b == '\\' {
				i++
				continue
			} else if b == '(' {
				nesting++
			} else if b == ')' {
				nesting--
			}
			if nesting == 0 {
				end = i
				break
			}
		}
		if end == -999 {
			panic(errors.Errorf("Unclosed function named '%s'", func_name))
		}
		params := mod.collectTokenize(source[start+1 : end])
		// TODO if function.multi_arg
		tokenCh <- FunctionToken{funcName: func_name, params: [][]Token{params}}
		source = source[end+1:]
		m = FunctionTokenRgx.FindStringSubmatchIndex(source)
	}
	// Parameter directives
	m = ParameterTokenRgx.FindStringSubmatchIndex(source)
	for m != nil {
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
		mod.tokenize(source[:m[0]], true, tokenCh)
		tokenCh <- t
		source = source[m[1]:]
		m = ParameterTokenRgx.FindStringSubmatchIndex(source)
	}
	tokenCh <- TextToken{Text: source}
	if !recursed {
		close(tokenCh)
	}
}
