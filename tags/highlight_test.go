package tags

import (
	"html"
	"regexp"
	"strings"
	"testing"

	"github.com/reidransom/jigyll/cache"
	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/liquid"
	"github.com/stretchr/testify/require"
)

func TestHighlightTagStandardShell(t *testing.T) {
	rendered := renderHighlight(t, `{% highlight ruby %}
def foo
  puts "foo"
end
{% endhighlight %}`)

	require.True(t, strings.HasPrefix(
		rendered,
		`<figure class="highlight"><pre><code class="language-ruby" data-lang="ruby">`,
	))
	require.True(t, strings.HasSuffix(rendered, "</code></pre></figure>"))
	require.Equal(t, 1, strings.Count(rendered, `<figure class="highlight">`))
	require.NotContains(t, rendered, `<pre tabindex="0" class="chroma">`)
	require.Equal(t, "def foo\n  puts \"foo\"\nend", highlightCodeText(t, rendered))
}

func TestHighlightTagLanguageMetadata(t *testing.T) {
	for _, tc := range []struct {
		name          string
		language      string
		classLanguage string
		dataLanguage  string
	}{
		{name: "uppercase", language: "YAML", classLanguage: "yaml", dataLanguage: "yaml"},
		{name: "hash", language: "c#", classLanguage: "c#", dataLanguage: "c#"},
		{name: "plus", language: "xml+cheetah", classLanguage: "xml-cheetah", dataLanguage: "xml+cheetah"},
		{name: "unknown valid language", language: "unknown_language", classLanguage: "unknown_language", dataLanguage: "unknown_language"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rendered := renderHighlight(t, `{% highlight `+tc.language+` %}value{% endhighlight %}`)

			require.Contains(
				t,
				rendered,
				`<code class="language-`+tc.classLanguage+`" data-lang="`+tc.dataLanguage+`">`,
			)
		})
	}
}

func TestHighlightTagRejectsInvalidLanguage(t *testing.T) {
	_, err := renderHighlightResult(t, `{% highlight ruby^ %}value{% endhighlight %}`)

	require.Error(t, err)
}

func TestHighlightTagTrimsOnlyLineTerminators(t *testing.T) {
	for _, tc := range []struct {
		name     string
		source   string
		expected string
	}{
		{name: "leading and trailing LF", source: "\n\nputs :first\n\n", expected: "puts :first"},
		{name: "leading and trailing CRLF", source: "\r\nputs :first\r\n", expected: "puts :first"},
		{name: "leading indentation", source: "\n  puts :first\n", expected: "  puts :first"},
		{name: "internal blank line", source: "\nputs :first\n\nputs :second\n", expected: "puts :first\n\nputs :second"},
		{name: "only line terminators", source: "\r\n\n\r", expected: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rendered := renderHighlight(t, `{% highlight ruby %}`+tc.source+`{% endhighlight %}`)

			require.Equal(t, tc.expected, highlightCodeText(t, rendered))
		})
	}
}

func TestHighlightTagLineNumbers(t *testing.T) {
	rendered := renderHighlight(t, `{% highlight ruby linenos %}
puts :first
{% endhighlight %}`)

	require.Equal(t, 1, strings.Count(rendered, `<figure class="highlight">`))
	require.Contains(t, rendered, `<code class="language-ruby" data-lang="ruby">`)
	require.Contains(t, rendered, `class="lntable"`)
	require.Contains(t, rendered, `class="lntd"`)
	require.Less(t, strings.LastIndex(rendered, "</table>"), strings.LastIndex(rendered, "</figure>"))
}

func renderHighlight(t *testing.T, source string) string {
	t.Helper()

	rendered, err := renderHighlightResult(t, source)
	require.NoError(t, err)
	return rendered
}

func renderHighlightResult(t *testing.T, source string) (string, error) {
	t.Helper()

	cache.Disable()
	engine := liquid.NewEngine()
	cfg := config.Default()
	AddJekyllTags(engine, &cfg, []string{}, func(string) (string, bool) { return "", false })
	return engine.ParseAndRenderString(source, liquid.Bindings{})
}

func highlightCodeText(t *testing.T, rendered string) string {
	t.Helper()

	match := regexp.MustCompile(`(?s)<code\b[^>]*>(.*)</code>`).FindStringSubmatch(rendered)
	require.Len(t, match, 2)
	return html.UnescapeString(regexp.MustCompile(`(?s)<[^>]+>`).ReplaceAllString(match[1], ""))
}
