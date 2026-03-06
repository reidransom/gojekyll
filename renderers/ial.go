package renderers

import "regexp"

// kramdown uses {: .class #id key="value"} for inline attribute lists (IALs).
// goldmark's attribute parser expects Pandoc-style {.class #id key="value"} (no colon).
// This preprocessor converts kramdown IALs to Pandoc-style before goldmark parses.

// Matches block-level IALs: a line containing only {: ...}
// and inline IALs: {: ...} appearing after content.
var kramdownIALRE = regexp.MustCompile(`\{:\s*([^}]+)\}`)

// preprocessIAL rewrites kramdown-style {: ...} attribute lists to
// Pandoc-style {...} that goldmark understands.
func preprocessIAL(md []byte) []byte {
	return kramdownIALRE.ReplaceAll(md, []byte("{$1}"))
}
