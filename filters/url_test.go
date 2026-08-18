package filters

import (
	"strings"
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/liquid"
	"github.com/stretchr/testify/require"
)

func TestRelativeURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty base relative input", input: "assets/app.css", want: "/assets/app.css"},
		{name: "empty base rooted input", input: "/assets/app.css", want: "/assets/app.css"},
		{name: "rooted base relative input", baseURL: "/docs", input: "assets/app.css", want: "/docs/assets/app.css"},
		{name: "rooted base rooted input", baseURL: "/docs", input: "/assets/app.css", want: "/docs/assets/app.css"},
		{name: "slashless base relative input", baseURL: "docs", input: "assets/app.css", want: "/docs/assets/app.css"},
		{name: "trailing slash base", baseURL: "/docs/", input: "/assets/app.css", want: "/docs/assets/app.css"},
		{name: "empty input", baseURL: "/docs", want: "/docs"},
		{name: "root input", baseURL: "/docs", input: "/", want: "/docs/"},
		{name: "root base root input", baseURL: "/", input: "/", want: "/"},
		{name: "multiple trailing base slashes", baseURL: "/docs//", input: "/assets/app.css", want: "/docs//assets/app.css"},
		{name: "unicode input", input: "错误.html", want: "/%E9%94%99%E8%AF%AF.html"},
		{name: "escaped input", input: "assets/%E2%9C%93.html", want: "/assets/%E2%9C%93.html"},
		{name: "query and fragment", input: "assets/app.css?version=1#main", want: "/assets/app.css?version=1#main"},
		{name: "protocol relative input", baseURL: "/docs", input: "//example.com/", want: "/docs//example.com/"},
		{name: "https absolute input", baseURL: "/docs", input: "https://example.com/path", want: "https://example.com/path"},
		{name: "file absolute input", baseURL: "/docs", input: "file:///tmp/file.html", want: "file:///tmp/file.html"},
		{name: "mailto absolute input", baseURL: "/docs", input: "mailto:user@example.com", want: "mailto:user@example.com"},
		{name: "malformed percent escape", input: "assets/%zz", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := relativeURL(tt.baseURL, tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestURLFilterRegistration(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		absoluteURL string
		template    string
		want        string
	}{
		{name: "empty base relative input", template: `{{ "assets/js/search-data.json" | relative_url }}`, want: "/assets/js/search-data.json"},
		{name: "rooted base relative input", baseURL: "/docs", template: `{{ "assets/js/search-data.json" | relative_url }}`, want: "/docs/assets/js/search-data.json"},
		{name: "rooted input", baseURL: "/docs", template: `{{ "/assets/js/search-data.json" | relative_url }}`, want: "/docs/assets/js/search-data.json"},
		{name: "absolute relative URL input", baseURL: "/docs", template: `{{ "https://example.com/path" | relative_url }}`, want: "https://example.com/path"},
		{name: "absolute URL filter", baseURL: "/docs", absoluteURL: "https://example.com", template: `{{ "/assets/style.css" | absolute_url }}`, want: "https://example.com/docs/assets/style.css"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireURLTemplateRender(t, tt.baseURL, tt.absoluteURL, tt.template, tt.want)
		})
	}

	t.Run("invalid URI returns filter error", func(t *testing.T) {
		engine := liquid.NewEngine()
		cfg := config.Default()
		AddJekyllFilters(engine, &cfg)

		_, err := engine.ParseAndRender([]byte(`{{ "assets/%zz" | relative_url }}`), nil)
		require.Error(t, err)
	})
}

func requireURLTemplateRender(t *testing.T, baseURL, absoluteURL, tmpl, want string) {
	t.Helper()

	engine := liquid.NewEngine()
	cfg := config.Default()
	cfg.BaseURL = baseURL
	cfg.AbsoluteURL = absoluteURL
	AddJekyllFilters(engine, &cfg)

	data, err := engine.ParseAndRender([]byte(tmpl), nil)
	require.NoErrorf(t, err, tmpl)
	require.Equalf(t, want, strings.TrimSpace(string(data)), tmpl)
}
