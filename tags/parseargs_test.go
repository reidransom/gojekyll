package tags

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

var argTests = []struct {
	in          string
	optionCount int
	positional  []string
}{
	{`filename`, 0, []string{"filename"}},
	{`filename a=1`, 1, []string{"filename"}},
	{`filename a=1 b=2`, 2, []string{"filename"}},
	{`filename a='1' b=2`, 2, []string{"filename"}},
	{`filename a='1 b=' c`, 1, []string{"filename", "c"}},
	{`a=1 b=2`, 2, []string{}},
	{`a='1' b=2`, 2, []string{}},
	{`arg1 arg2`, 0, []string{"arg1", "arg2"}},
}

func TestFilters(t *testing.T) {
	for i, test := range argTests {
		t.Run(fmt.Sprintf("%02d", i+1), func(t *testing.T) {
			actual, err := ParseArgs(test.in)
			require.NoError(t, err)
			require.Equal(t, test.optionCount, len(actual.Options), "options count in %q -> #%v", test.in, actual)
			require.Equal(t, test.positional, actual.Args, "args in %q -> #%v", test.in, actual)
		})
	}
}

func TestParseArgsOptionWhitespace(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		options map[string]optionRecord
	}{
		{
			name:  "no whitespace",
			input: `filename color_scheme=include.color_scheme`,
			options: map[string]optionRecord{
				"color_scheme": {value: "include.color_scheme"},
			},
		},
		{
			name:  "whitespace before equals",
			input: `filename color_scheme =include.color_scheme`,
			options: map[string]optionRecord{
				"color_scheme": {value: "include.color_scheme"},
			},
		},
		{
			name:  "whitespace after equals",
			input: `filename color_scheme= include.color_scheme`,
			options: map[string]optionRecord{
				"color_scheme": {value: "include.color_scheme"},
			},
		},
		{
			name:  "whitespace around equals",
			input: `filename color_scheme = include.color_scheme`,
			options: map[string]optionRecord{
				"color_scheme": {value: "include.color_scheme"},
			},
		},
		{
			name:  "mixed options and quoted value",
			input: `filename color_scheme = include.color_scheme label= "dark mode"`,
			options: map[string]optionRecord{
				"color_scheme": {value: "include.color_scheme"},
				"label":        {value: "dark mode", quoted: true},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := ParseArgs(test.input)
			require.NoError(t, err)
			require.Equal(t, []string{"filename"}, actual.Args)
			require.Equal(t, test.options, actual.Options)
		})
	}

	t.Run("malformed spaced assignment", func(t *testing.T) {
		_, err := ParseArgs(`filename color_scheme = "unterminated`)
		require.EqualError(t, err, `parse error in tag parameters "filename color_scheme = \"unterminated"`)
	})
}
