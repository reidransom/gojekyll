package renderers

import (
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

	// Test markdown="block" (same as markdown="1")
	require.Contains(t, mustMarkdownString("\n<div markdown=\"block\">\n*b*\n</div>\n"), "<em>b</em>")

	// Test markdown="span" (no paragraphs, just inline elements)
	result := mustMarkdownString("\n<div markdown=\"span\">\n*b*\n</div>\n")
	require.Contains(t, result, "<em>b</em>")
	require.NotContains(t, result, "<p><em>b</em></p>")

	// Test markdown="0" (no markdown processing)
	require.NotContains(t, mustMarkdownString("\n<div markdown=\"0\">\n*b*\n</div>\n"), "<em>")
	require.Contains(t, mustMarkdownString("\n<div markdown=\"0\">\n*b*\n</div>\n"), "*b*")
}

func TestRenderMarkdownWithHtml2(t *testing.T) {
	// Test autolink processing with different markdown modes (block-level HTML)
	require.Contains(t, mustMarkdownString("\n<div markdown=1>\n<user@example.com>\n</div>\n"), `<a href="mailto:user@example.com">`)
	require.Contains(t, mustMarkdownString("\n<div markdown=\"block\">\n<user@example.com>\n</div>\n"), `<a href="mailto:user@example.com">`)
	require.Contains(t, mustMarkdownString("\n<div markdown=\"span\">\n<user@example.com>\n</div>\n"), `<a href="mailto:user@example.com">`)

	emailResult := mustMarkdownString("\n<div markdown=\"0\">\n<user@example.com>\n</div>\n")
	require.NotContains(t, emailResult, `<a href="mailto:user@example.com">`)
	require.Contains(t, emailResult, "user@example.com")

	// Test URL autolink processing with different markdown modes (block-level HTML)
	require.Contains(t, mustMarkdownString("\n<div markdown=1>\n<http://example.com>\n</div>\n"), `<a href="http://example.com">`)
	require.Contains(t, mustMarkdownString("\n<div markdown=\"block\">\n<http://example.com>\n</div>\n"), `<a href="http://example.com">`)
	require.Contains(t, mustMarkdownString("\n<div markdown=\"span\">\n<http://example.com>\n</div>\n"), `<a href="http://example.com">`)

	urlResult := mustMarkdownString("\n<div markdown=\"0\">\n<http://example.com>\n</div>\n")
	require.NotContains(t, urlResult, `<a href="http://example.com">`)
	require.Contains(t, urlResult, "http://example.com")
}

func TestRenderMarkdownUnclosedTag(t *testing.T) {
	// An unclosed tag inside markdown="1" is an error (with a helpful message),
	// not a silent partial render.
	_, err := renderMarkdown([]byte(`<div a=1 markdown=1><p></div>`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected EOF")
}

func TestRenderMarkdownIndentedHTML(t *testing.T) {
	// Regression test for upstream issue #113: indented HTML inside HTML blocks
	// should not be rendered as code blocks
	input := "<ul>\n    <li class=\"post-item\">\n        <a href=\"/post1/\">Post 1</a>\n    </li>\n</ul>\n"
	result := mustMarkdownString(input)
	require.NotContains(t, result, "<pre>", "indented HTML should not become a code block")
	require.NotContains(t, result, "<code>", "indented HTML should not become a code block")
	require.Contains(t, result, "<li", "list items should be preserved")
}

func TestDeIndentHTMLBlocks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(t *testing.T, result string)
	}{
		{
			name:  "indented HTML block",
			input: "<ul>\n    <li>item</li>\n</ul>\n",
			check: func(t *testing.T, result string) {
				require.Contains(t, result, "<li>item</li>")
				require.NotContains(t, result, "    <li>")
			},
		},
		{
			name:  "non-HTML content preserved",
			input: "regular paragraph\n\n    indented code\n",
			check: func(t *testing.T, result string) {
				require.Contains(t, result, "    indented code")
			},
		},
		{
			name:  "HTML block ends at blank line",
			input: "<div>\n    inside\n\n    outside\n",
			check: func(t *testing.T, result string) {
				require.Contains(t, result, "inside")
				require.Contains(t, result, "    outside")
			},
		},
		{
			name:  "HTML inside fenced code blocks untouched",
			input: "```\n<ul>\n    <li>item</li>\n</ul>\n```\n",
			check: func(t *testing.T, result string) {
				require.Contains(t, result, "    <li>item</li>")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(deIndentHTMLBlocks([]byte(tt.input)))
			tt.check(t, result)
		})
	}
}

func TestRenderMarkdownVoidElements(t *testing.T) {
	// Upstream issue #66: <br> tags inside markdown="1" blocks should not cause
	// EOF errors. Void elements like <br>, <hr>, <img> don't have end tags, so
	// the depth tracker must not increment for them.
	result := mustMarkdownString("\n<div markdown=\"1\">\n<br>\n<br>\n</div>\n")
	require.Contains(t, result, "<br")

	result = mustMarkdownString("\n<div markdown=\"1\">\n<hr>\n</div>\n")
	require.Contains(t, result, "<hr")

	result = mustMarkdownString("\n<div markdown=\"1\">\n<img src=\"test.png\">\n</div>\n")
	require.Contains(t, result, "img")

	// Self-closing variants should also work
	result = mustMarkdownString("\n<div markdown=\"1\">\n<br/>\n</div>\n")
	require.Contains(t, result, "<br")

	// markdown="0" with void elements should also not error
	result = mustMarkdownString("\n<div markdown=\"0\">\n<br>\n<br>\ntext\n</div>\n")
	require.Contains(t, result, "text")
}

func TestRenderMarkdownHeadingAttributes(t *testing.T) {
	// Pandoc-style heading attributes
	require.Contains(t, mustMarkdownString("## Heading {#custom-id}"), `id="custom-id"`)
}

func TestRenderMarkdownKramdownIAL(t *testing.T) {
	// kramdown-style heading IAL should be preprocessed and applied
	require.Contains(t, mustMarkdownString("## Heading {: #custom-id}"), `id="custom-id"`)
	require.Contains(t, mustMarkdownString("## Heading {: .special}"), `class="special"`)
}

func TestRenderMarkdownKramdownBlockIALs(t *testing.T) {
	t.Run("following block", func(t *testing.T) {
		result := mustMarkdownString("{: .warning }\nWarning text")
		require.Equal(t, "<p class=\"warning\">Warning text</p>\n", result)
		require.NotContains(t, result, "{:")
	})

	t.Run("preceding paragraph", func(t *testing.T) {
		result := mustMarkdownString("Lead text\n{: .fs-6 .fw-300 }")
		require.Equal(t, "<p class=\"fs-6 fw-300\">Lead text</p>\n", result)
		require.NotContains(t, result, "{:")
	})

	t.Run("preceding heading", func(t *testing.T) {
		result := mustMarkdownString("## Heading\n{: #custom-id .special }")
		require.Equal(t, "<h2 id=\"custom-id\" class=\"special\">Heading</h2>\n", result)
		require.NotContains(t, result, "{:")
	})

	t.Run("following blockquote", func(t *testing.T) {
		result := mustMarkdownString("Before\n\n{: .warning }\n> Quoted")
		require.Contains(t, result, "<blockquote class=\"warning\">")
		require.NotContains(t, result, "<p>{: .warning }</p>")
	})

	t.Run("preceding list", func(t *testing.T) {
		result := mustMarkdownString("- One\n- Two\n{: .compact }")
		require.Contains(t, result, "<ul class=\"compact\">")
		require.NotContains(t, result, "{:")
	})
}

func TestRenderMarkdownKramdownInlineIALs(t *testing.T) {
	result := mustMarkdownString("[Get started](#start){: .btn .btn-primary target=\"_blank\"}")
	require.Equal(t,
		"<p><a href=\"#start\" class=\"btn btn-primary\" target=\"_blank\">Get started</a></p>\n",
		result)

	result = mustMarkdownString("![Alt](image.png){: .rounded width=\"32\"}")
	require.Equal(t,
		"<p><img src=\"image.png\" alt=\"Alt\" class=\"rounded\" width=\"32\" /></p>\n",
		result)
}

func TestRenderMarkdownKramdownIALBoundaries(t *testing.T) {
	t.Run("fenced code remains literal", func(t *testing.T) {
		result := mustMarkdownString("```\n{: .warning }\n```")
		require.Contains(t, result, "{: .warning }")
		require.NotContains(t, result, "class=\"warning\"")
	})

	t.Run("inline code remains literal", func(t *testing.T) {
		result := mustMarkdownString("`{: .warning }`")
		require.Equal(t, "<p><code>{: .warning }</code></p>\n", result)
	})

	t.Run("toc marker remains available to toc processing", func(t *testing.T) {
		result := mustMarkdownString("* TOC\n{:toc}\n\n## Heading")
		require.NotContains(t, result, "{:toc}")
		require.Contains(t, result, `id="markdown-toc"`)
	})
}

func TestPreprocessHeadingIALs(t *testing.T) {
	// Same-line heading IALs are rewritten to Pandoc-style
	require.Equal(t, "## Heading {#my-id .class}",
		string(preprocessHeadingIALs([]byte("## Heading {: #my-id .class}"))))
	// TOC markers (no whitespace after the colon) are left alone for the TOC pass
	require.Equal(t, "* TOC\n{:toc}",
		string(preprocessHeadingIALs([]byte("* TOC\n{:toc}"))))
	require.Equal(t, "## Heading\n{:.no_toc}",
		string(preprocessHeadingIALs([]byte("## Heading\n{:.no_toc}"))))
	// IALs on non-heading lines are left alone
	require.Equal(t, "text {: .class}",
		string(preprocessHeadingIALs([]byte("text {: .class}"))))
	// Heading lines inside fenced code blocks are left alone
	fenced := "```\n## Heading {: #id}\n```"
	require.Equal(t, fenced, string(preprocessHeadingIALs([]byte(fenced))))
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
		panic(err)
	}
	return string(s)
}
