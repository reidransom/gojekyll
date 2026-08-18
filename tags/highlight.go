package tags

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma"
	chromahtml "github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/reidransom/liquid/render"
)

var highlightArgsRE = regexp.MustCompile(`^\s*([a-zA-Z0-9.+#_-]+)(\s+linenos)?\s*$`)

func highlightTag(rc render.Context) (string, error) {
	argStr, err := rc.ExpandTagArg()
	if err != nil {
		return "", err
	}
	args := highlightArgsRE.FindStringSubmatch(argStr)
	if args == nil {
		return "", fmt.Errorf("syntax error")
	}
	source, err := rc.InnerString()
	if err != nil {
		return "", err
	}
	language := strings.ToLower(args[1])
	classLanguage := strings.ReplaceAll(language, "+", "-")
	source = strings.Trim(source, "\r\n")

	// Determine lexer.
	l := lexers.Get(language)
	if l == nil {
		l = lexers.Analyse(source) //nolint:misspell // chroma API name
	}
	if l == nil {
		l = lexers.Fallback
	}
	l = chroma.Coalesce(l)

	lineNum := args[2] != ""

	// Determine formatter.
	f := chromahtml.New(
		chromahtml.WithClasses(true),
		chromahtml.WithLineNumbers(lineNum),
		chromahtml.LineNumbersInTable(true),
		chromahtml.WithPreWrapper(highlightPreWrapper{
			language:      language,
			classLanguage: classLanguage,
		}),
	)

	// Determine style.
	s := styles.Get("")
	if s == nil {
		s = styles.Fallback
	}

	it, err := l.Tokenise(nil, source)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	buf.WriteString(`<figure class="highlight">`)
	if err = f.Format(buf, s, it); err != nil {
		return "", err
	}
	buf.WriteString(`</figure>`)
	return buf.String(), nil
}

type highlightPreWrapper struct {
	language      string
	classLanguage string
}

func (w highlightPreWrapper) Start(code bool, _ string) string {
	if !code {
		return "<pre>"
	}
	return `<pre><code class="language-` + w.classLanguage + `" data-lang="` + w.language + `">`
}

func (highlightPreWrapper) End(code bool) string {
	if !code {
		return "</pre>"
	}
	return "</code></pre>"
}
