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

func TestParseArgsEscapedQuotes(t *testing.T) {
	const anchorBody = `<svg viewBox="0 0 16 16" aria-hidden="true"><use xlink:href="#svg-link"></use></svg>`
	args, err := ParseArgs(
		`vendor/anchor_headings.html html=content beforeHeading="true" anchorBody="<svg viewBox=\"0 0 16 16\" aria-hidden=\"true\"><use xlink:href=\"#svg-link\"></use></svg>" anchorClass="anchor-heading" anchorAttrs="aria-labelledby=\"%html_id%\""`,
	)
	require.NoError(t, err)
	require.Equal(t, []string{"vendor/anchor_headings.html"}, args.Args)
	require.Equal(t, optionRecord{value: anchorBody, quoted: true}, args.Options["anchorBody"])
	require.Equal(t, optionRecord{value: `aria-labelledby="%html_id%"`, quoted: true}, args.Options["anchorAttrs"])
	require.Equal(t, optionRecord{value: "anchor-heading", quoted: true}, args.Options["anchorClass"])

	tests := []struct {
		name  string
		input string
		key   string
		value string
	}{
		{
			name:  "double quoted delimiter",
			input: `value="a \"quote\"" following="value"`,
			key:   "value",
			value: `a "quote"`,
		},
		{
			name:  "single quoted delimiter",
			input: `value='a \'quote\'' following='value'`,
			key:   "value",
			value: `a 'quote'`,
		},
		{
			name:  "unrelated backslash sequences",
			input: `value="keep \n and \\ unchanged" following="value"`,
			key:   "value",
			value: `keep \n and \\ unchanged`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := ParseArgs(test.input)
			require.NoError(t, err)
			require.Equal(t, optionRecord{value: test.value, quoted: true}, actual.Options[test.key])
			require.Equal(t, optionRecord{value: "value", quoted: true}, actual.Options["following"])
		})
	}

	for _, input := range []string{
		`value="unterminated`,
		`value="escaped closing quote \"`,
		`value="quoted"trailing`,
	} {
		_, err := ParseArgs(input)
		require.EqualError(t, err, fmt.Sprintf("parse error in tag parameters %q", input))
	}
}
