package utils

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// LeftPad left-pads s with spaces to n wide. It's an alternative to http://left-pad.io.
func LeftPad(s string, n int) string {
	if n <= len(s) {
		return s
	}
	ws := make([]byte, n-len(s))
	for i := range ws {
		ws[i] = ' '
	}
	return string(ws) + s
}

type replaceStringFuncError error

// SafeReplaceAllStringFunc is like regexp.ReplaceAllStringFunc but passes an
// an error back from the replacement function.
func SafeReplaceAllStringFunc(re *regexp.Regexp, src string, repl func(m string) (string, error)) (out string, err error) {
	// The ReplaceAllStringFunc callback signals errors via panic.
	// Turn them into return values.
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(replaceStringFuncError); ok {
				err = e.(error)
			} else {
				panic(r)
			}
		}
	}()
	return re.ReplaceAllStringFunc(src, func(m string) string {
		out, err := repl(m)
		if err != nil {
			panic(replaceStringFuncError(err))
		}
		return out
	}), nil
}

// Matches Ruby Jekyll's SLUGIFY_DEFAULT_REGEXP: Unicode letters, digits, and
// combining marks survive; everything else collapses to a hyphen.
var nonAlphanumericSequenceMatcher = regexp.MustCompile(`[^\p{M}\p{L}\p{Nd}]+`)
var leadingOrTrailingHyphenMatcher = regexp.MustCompile(`(^-|-$)`)

// Slugify replaces each sequence of non-alphanumerics by a single hyphen,
// and lowercases the result. This matches Ruby Jekyll's default slugify mode
// (and therefore the Liquid `slugify` filter).
func Slugify(s string) string {
	slug := strings.ToLower(nonAlphanumericSequenceMatcher.ReplaceAllString(s, "-"))

	// remove leading and trailing hyphen
	slug = leadingOrTrailingHyphenMatcher.ReplaceAllString(slug, "")
	return slug
}

// SlugifyPermalink replaces each sequence of non-alphanumerics by a single
// hyphen, but preserves the original case. Used for the permalink :title
// variable, which Ruby Jekyll slugifies with cased: true.
func SlugifyPermalink(s string) string {
	slug := nonAlphanumericSequenceMatcher.ReplaceAllString(s, "-")
	slug = leadingOrTrailingHyphenMatcher.ReplaceAllString(slug, "")
	return slug
}

// StringArrayToMap creates a map for use as a set.
func StringArrayToMap(a []string) map[string]bool {
	m := map[string]bool{}
	for _, s := range a {
		m[s] = true
	}
	return m
}

// StringArrayContains returns a bool indicating whether the array contains the string.
func StringArrayContains(a []string, s string) bool {
	for _, item := range a {
		if item == s {
			return true
		}
	}
	return false
}

// Titleize splits at ` `, capitalizes, and joins.
func Titleize(s string) string {
	a := strings.Split(s, "-")
	for i, s := range a {
		if r, size := utf8.DecodeRuneInString(s); size > 0 {
			a[i] = string(unicode.ToUpper(r)) + s[size:]
		}
	}
	return strings.Join(a, " ")
}
