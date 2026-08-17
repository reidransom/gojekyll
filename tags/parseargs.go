package tags

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/reidransom/liquid/render"
)

var argPattern = regexp.MustCompile(`^([^=\s]+)(?:\s+|$)`)
var optionPattern = regexp.MustCompile(`^(\w+)\s*=\s*("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|[^'"\s]*)(?:\s+|$)`)

// ParsedArgs holds the parsed arguments from ParseArgs.
type ParsedArgs struct {
	Args    []string
	Options map[string]optionRecord
}

type optionRecord struct {
	value  string
	quoted bool
}

// ParseArgs parses a tag argument line {% include arg1 arg2 opt=a opt2='b' %}
func ParseArgs(argsline string) (*ParsedArgs, error) {
	args := ParsedArgs{
		[]string{},
		map[string]optionRecord{},
	}
	// Ranging over FindAllStringSubmatch would be better golf but got out of hand
	// maintenance-wise.
	for r, i := argsline, 0; len(r) > 0; r = r[i:] {
		am := argPattern.FindStringSubmatch(r)
		om := optionPattern.FindStringSubmatch(r)
		switch {
		case om != nil:
			k, v, quoted := om[1], om[2], false
			if v == "" && len(om[0]) < len(r) {
				return nil, fmt.Errorf("parse error in tag parameters %q", argsline)
			}
			if v != "" && (v[0] == '\'' || v[0] == '"') {
				v, quoted = decodeQuotedOption(v), true
			}
			args.Options[k] = optionRecord{v, quoted}
			i = len(om[0])
		case am != nil:
			args.Args = append(args.Args, am[1])
			i = len(am[0])
		default:
			return nil, fmt.Errorf("parse error in tag parameters %q", argsline)
		}
	}
	return &args, nil
}

func decodeQuotedOption(token string) string {
	quote := token[0]
	value := token[1 : len(token)-1]
	if quote == '"' {
		return strings.ReplaceAll(value, `\"`, `"`)
	}

	return strings.ReplaceAll(value, `\'`, `'`)
}

// EvalOptions evaluates unquoted options.
func (r *ParsedArgs) EvalOptions(ctx render.Context) (map[string]interface{}, error) {
	options := map[string]interface{}{}
	for k, v := range r.Options {
		if v.quoted {
			options[k] = v.value
		} else {
			value, err := ctx.EvaluateString(v.value)
			if err != nil {
				return nil, err
			}
			options[k] = value
		}
	}
	return options, nil
}
