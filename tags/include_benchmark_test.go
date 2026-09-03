package tags

import (
	"fmt"
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/liquid"
)

func BenchmarkIncludeFileCacheNavigation(b *testing.B) {
	includeDir := b.TempDir()
	writeIncludeCachedFixture(b, includeDir, "navigation.html", `{% for item in site.navigation %}<a href="{{ item.url }}"{% if page.url == item.url %} aria-current="page"{% endif %}>{{ item.title }}</a>{% endfor %}`)

	pages := make([]liquid.Bindings, 32)
	for i := range pages {
		pages[i] = liquid.Bindings{
			"page": map[string]interface{}{"url": fmt.Sprintf("/section-%d/", i)},
			"site": map[string]interface{}{
				"navigation": []map[string]string{
					{"title": "Home", "url": "/"},
					{"title": "Guides", "url": "/guides/"},
					{"title": "Reference", "url": "/reference/"},
					{"title": "Section", "url": fmt.Sprintf("/section-%d/", i)},
				},
			},
		}
	}

	for _, cacheEnabled := range []bool{false, true} {
		name := "cache_disabled"
		if cacheEnabled {
			name = "cache_enabled"
		}
		b.Run(name, func(b *testing.B) {
			engine := liquid.NewEngine()
			cfg := config.Default()
			AddJekyllTags(engine, &cfg, []string{includeDir}, func(string) (string, bool) { return "", false })
			if cacheEnabled {
				engine.EnableFileCache()
			}

			tpl, err := engine.ParseTemplateLocation([]byte(`{% include navigation.html %}`), "layouts/default.html", 1)
			if err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()
			b.ResetTimer()
			outputBytes := 0
			for i := range b.N {
				output, err := tpl.Render(pages[i%len(pages)])
				if err != nil {
					b.Fatal(err)
				}
				outputBytes += len(output)
			}
			if outputBytes == 0 {
				b.Fatal("navigation render produced no output")
			}
		})
	}
}
