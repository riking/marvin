package factoid

import (
	"regexp"
	"strconv"

	"fmt"
	"github.com/pkg/errors"
	"github.com/riking/homeapi/marvin"
	"strings"
)

type ErrUser error
type ErrSource error

type Token interface {
	Run(mod *FactoidModule, source marvin.ActionSource, args []string) (string, error)
}

type TextToken struct {
	Text string
}

func (t TextToken) Run(mod *FactoidModule, source marvin.ActionSource, args []string) (string, error) {
	return t.Text, nil
}

var ParameterTokenRgx = regexp.MustCompile("%([A-Za-z]+)([0-9]+)?(-)?([0-9]+)?%")

type ParameterToken struct {
	raw     string
	op      string
	start   int
	end     int
	isRange bool
}

func NewParameterToken(match []string) Token {
	start, startErr := strconv.Atoi(match[2])
	if startErr != nil {
		start = -1
	}
	end, endErr := strconv.Atoi(match[4])
	if endErr != nil {
		end = -1
	}
	return ParameterToken{
		raw:     match[0],
		op:      match[1],
		start:   start,
		end:     end,
		isRange: match[3] == "-",
	}
}

func (p ParameterToken) Run(mod *FactoidModule, source marvin.ActionSource, args []string) (string, error) {
	switch p.op {
	default:
		return p.raw, nil
	case "inp":
		return strings.Join(args, " "), nil
	case "bot":
		return fmt.Sprintf("<@%s>", mod.team.BotUser()), nil
	case "chan":
		return fmt.Sprintf("<#%s>", source.ChannelID()), nil
	case "user":
		return fmt.Sprintf("<@%s>", source.UserID()), nil
	case "uname":
		return mod.team.UserName(source.UserID()), nil
	case "ioru":
		if len(args) > 0 {
			return strings.Join(args, " "), nil
		} else {
			return fmt.Sprintf("<@%s>", source.UserID()), nil
		}
	case "ioruname":
		if len(args) > 0 {
			return strings.Join(args, " "), nil
		} else {
			return mod.team.UserName(source.UserID()), nil
		}
	case "arg":
		start := p.start
		if start == -1 {
			start = 0
		}
		if p.isRange {
			end := p.end
			if end == -1 {
				end = len(args) - 1
			}
			if len(args) <= max(start, end) {
				return "", ErrUser(errors.Errorf("Not enough args (wanted %d)", max(start, end)+1))
			}
			return strings.Join(args[start:end+1], " "), nil
		} else {
			if len(args) > start {
				return args[start], nil
			}
			return "", ErrUser(errors.Errorf("Not enough args (wanted %d)", start+1))
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
