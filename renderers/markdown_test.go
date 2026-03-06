package renderers

import (
	"log"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderMarkdown(t *testing.T) {
	require.Equal(t, "<p><em>b</em></p>\n", mustMarkdownString("*b*"))
}

func TestRenderMarkdownWithHtml1(t *testing.T) {
	// goldmark (CommonMark) treats <div> as an HTML block, so it is not wrapped in <p> tags.
	require.Equal(t, "<div a=1><p><em>b</em></p>\n</div>", mustMarkdownString(`<div a=1 markdown="1">*b*</div>`))
	require.Equal(t, "<div a=1><p><em>b</em></p>\n</div>", mustMarkdownString(`<div a=1 markdown='1'>*b*</div>`))
	require.Equal(t, "<div a=1><p><em>b</em></p>\n</div>", mustMarkdownString(`<div a=1 markdown=1>*b*</div>`))
	require.Equal(t, "<div a=1><p></div>", mustMarkdownString(`<div a=1 markdown=1><p></div>`))
}

func TestRenderMarkdownWithHtml2(t *testing.T) {
	t.Skip("skipping broken test.")
	// FIXME for now, manually test against against site/testdata/site1/markdown.md.
	// These render correctly in the entire pipeline, but not in the test.
	require.Equal(t, "<p><div>*b*</div></p>\n", mustMarkdownString("<div>*b*</div>"))
	require.Contains(t, mustMarkdownString(`<div markdown=1><user@example.com></div>`), `<a href="mailto:user@example.com">user@example.com</a>`)
	require.Contains(t, mustMarkdownString(`<div markdown=1><http://example.com></div>`), `<a href="http://example.com">http://example.com</a>`)
}

func TestPreprocessIAL(t *testing.T) {
	// kramdown-style {: ...} should be converted to Pandoc-style {...}
	require.Equal(t, "{.class}", string(preprocessIAL([]byte("{: .class}"))))
	require.Equal(t, "{#my-id .class}", string(preprocessIAL([]byte("{: #my-id .class}"))))
	require.Equal(t, "{.class key=\"value\"}", string(preprocessIAL([]byte("{: .class key=\"value\"}"))))
	// Already Pandoc-style should not be affected
	require.Equal(t, "{.class}", string(preprocessIAL([]byte("{.class}"))))
	// No IAL should pass through unchanged
	require.Equal(t, "plain text", string(preprocessIAL([]byte("plain text"))))
}

func TestRenderMarkdownHeadingAttributes(t *testing.T) {
	// Pandoc-style heading attributes
	require.Contains(t, mustMarkdownString("## Heading {#custom-id}"), `id="custom-id"`)
}

func TestRenderMarkdownKramdownIAL(t *testing.T) {
	// kramdown-style heading IAL should be preprocessed and applied
	require.Contains(t, mustMarkdownString("## Heading {: #custom-id}"), `id="custom-id"`)
}

func TestRenderMarkdownTable(t *testing.T) {
	md := "| A | B |\n|---|---|\n| 1 | 2 |\n"
	out := mustMarkdownString(md)
	require.Contains(t, out, "<table>")
	require.Contains(t, out, "<td>1</td>")
}

func TestRenderMarkdownFootnote(t *testing.T) {
	md := "Text[^1]\n\n[^1]: Footnote content\n"
	out := mustMarkdownString(md)
	require.Contains(t, out, "Footnote content")
	require.Contains(t, out, "fn:1")
}

func TestRenderMarkdownDefinitionList(t *testing.T) {
	md := "Term\n:   Definition\n"
	out := mustMarkdownString(md)
	require.Contains(t, out, "<dl>")
	require.Contains(t, out, "<dt>Term</dt>")
	require.Contains(t, out, "<dd>Definition</dd>")
}

func TestRenderMarkdownStrikethrough(t *testing.T) {
	require.Contains(t, mustMarkdownString("~~deleted~~"), "<del>deleted</del>")
}

func mustMarkdownString(md string) string {
	s, err := renderMarkdown([]byte(md))
	if err != nil {
		log.Fatal(err)
	}
	return string(s)
}

// func renderMarkdownString(md string) (string, error) {
// 	s, err := renderMarkdown([]byte(md))
// 	if err != nil {
// 		return "", err
// 	}
// 	return string(s), err
// }
