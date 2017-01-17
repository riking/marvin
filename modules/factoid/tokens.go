package factoid

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/riking/marvin"
	"github.com/riking/marvin/util"
)

type ErrUser struct{ error }
type ErrSource struct{ error }

type Token interface {
	Run(mod *FactoidModule, args []string, actionSource marvin.ActionSource) (string, error)
	Source() string
}

var DirectiveTokenRgx = regexp.MustCompile(`^{([a-z]+)}`)

type DirectiveToken struct {
	Directive string
}

func (d DirectiveToken) Source() string { return "{" + d.Directive + "}" }

type TextToken struct {
	Text string
}

func (t TextToken) Source() string { return t.Text }

func (t TextToken) Run(mod *FactoidModule, args []string, actionSource marvin.ActionSource) (string, error) {
	return t.Text, nil
}

// parenthesis matching performed separately
// 1 = backslash check
// 2 = name
// do parencheck starting at end of 2, ignore end of 0
var FunctionTokenRgx = regexp.MustCompile(`(^|[^\\])\$([a-zA-Z_][a-zA-Z0-9_]*)\(.*?\)`)

type FunctionToken struct {
	Function
	params [][]Token
	raw    string
}

func (p FunctionToken) Source() string { return p.raw }

func (p FunctionToken) Run(mod *FactoidModule, args []string, actionSource marvin.ActionSource) (string, error) {
	funcParams := make([]string, len(p.params))
	var err error

	for i, v := range p.params {
		funcParams[i], err = mod.exec_processTokens(v, args, actionSource)
		if err != nil {
			return "", err
		}
	}
	return p.F(funcParams...), nil
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

func (p ParameterToken) Source() string { return p.raw }

func (p ParameterToken) Run(mod *FactoidModule, args []string, actionSource marvin.ActionSource) (string, error) {
	switch p.op {
	default:
		return p.raw, nil
	case "bot":
		return fmt.Sprintf("<@%s>", mod.team.BotUser()), nil
	case "chan":
		return fmt.Sprintf("<#%s>", actionSource.ChannelID()), nil
	case "user":
		return fmt.Sprintf("<@%s>", actionSource.UserID()), nil
	case "uname":
		return mod.team.UserName(actionSource.UserID()), nil
	case "ioru":
		if len(args) > 0 {
			return strings.Join(args, " "), nil
		} else {
			return fmt.Sprintf("<@%s>", actionSource.UserID()), nil
		}
	case "ioruname":
		if len(args) > 0 {
			return strings.Join(args, " "), nil
		} else {
			return mod.team.UserName(actionSource.UserID()), nil
		}
	case "args", "inp":
		return strings.Join(args, " "), nil
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
				return "", ErrUser{errors.Errorf("Not enough args (wanted %d)", max(start, end)+1)}
			}
			return strings.Join(args[start:end+1], " "), nil
		} else {
			if len(args) > start {
				return args[start], nil
			}
			return "", ErrUser{errors.Errorf("Not enough args (wanted %d)", start+1)}
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
