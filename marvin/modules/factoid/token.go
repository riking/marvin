package factoid

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/riking/homeapi/marvin"
	"github.com/riking/homeapi/marvin/util"
)

type ErrUser error
type ErrSource error

type Token interface {
	Run(mod *FactoidModule, source marvin.ActionSource, args []string) (string, error)
}

var DirectiveTokenRgx = regexp.MustCompile(`^{([a-z]+)}`)

type DirectiveToken struct {
	Directive string
}

func (t DirectiveToken) Run(mod *FactoidModule, source marvin.ActionSource, args []string) (string, error) {
	return "", nil
}

type TextToken struct {
	Text string
}

func (t TextToken) Run(mod *FactoidModule, source marvin.ActionSource, args []string) (string, error) {
	return t.Text, nil
}

// parenthesis matching performed separately
// 1 = backslash check
// 2 = name
// do parencheck starting at end of 2, ignore end of 0
var FunctionTokenRgx = regexp.MustCompile(`(^|[^\\])\$([a-zA-Z_][a-zA-Z0-9_]*)\(.*?\)`)

type FunctionToken struct {
	funcName string // TODO switch to func object
	params   [][]Token
}

func (p FunctionToken) Run(mod *FactoidModule, source marvin.ActionSource, args []string) (string, error) {
	// TODO switch to func object
	return fmt.Sprintf("(%%!FunctionToken.Run not implemented [%s] [%#v])", p.funcName, p.params), nil
}

var ParameterTokenRgx = regexp.MustCompile("%([A-Za-z]+)([0-9]+)?(-)?([0-9]+)?%")

type ParameterToken struct {
	raw     string
	op      string
	start   int
	end     int
	isRange bool
}

// Pass match[0], match[1], ... match[4]
func NewParameterToken(rawStr, opStr, startStr, rangeStr, endStr string) Token {
	start, startErr := strconv.Atoi(startStr)
	if startErr != nil {
		start = -1
	}
	end, endErr := strconv.Atoi(endStr)
	if endErr != nil {
		end = -1
	}
	return ParameterToken{
		raw:     rawStr,
		op:      opStr,
		start:   start,
		end:     end,
		isRange: rangeStr == "-",
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
	case "args":
		p.isRange = true
		fallthrough
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
	case "date":
		return time.Now().In(util.TZ42USA()).Format("2006-01-02"), nil
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
