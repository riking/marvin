package factoid

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/riking/homeapi/marvin"
)

var directiveRgx = regexp.MustCompile(`{([a-z]+)}`)

func RunFactoid(info FactoidInfo, source marvin.ActionSource, args []string) (string, error) {
	if len(info.RawSource) == 0 || strings.HasPrefix(info.RawSource, "{noreply}") {
		return "", nil
	}

	var leftover string

	leftover = info.RawSource[:]
	for {
		match := directiveRgx.FindStringSubmatch(leftover)
		directive := match[1]
		leftover = leftover[len(match[0]):]
		switch directive {
		case "alias":

		default:
			return "", errors.Errorf("Unknown directive '%s'", directive)
		}
	}

	return info.RawSource, nil
}

func (fi *FactoidInfo) Tokens(rawSource string) []Token {
	fi.tokenize.Do(func() {
		var tokens []Token

		tokens = append(tokens, TextToken{Text: rawSource})
	})
	return fi.tokens
}
