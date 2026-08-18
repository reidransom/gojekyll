package filters

import (
	"net/url"
	"strings"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/jigyll/utils"
	"github.com/reidransom/liquid"
)

func registerURLFilters(e *liquid.Engine, c *config.Config) {
	e.RegisterFilter("absolute_url", func(s string) string {
		return utils.URLJoin(c.AbsoluteURL, c.BaseURL, s)
	})
	e.RegisterFilter("relative_url", func(s string) (string, error) {
		return relativeURL(c.BaseURL, s)
	})
}

func relativeURL(baseURL, input string) (string, error) {
	inputURL, err := url.Parse(input)
	if err != nil {
		return "", err
	}
	if inputURL.IsAbs() {
		return input, nil
	}

	baseURL = ensureLeadingSlash(strings.TrimSuffix(baseURL, "/"))
	input = ensureLeadingSlash(input)
	var combined strings.Builder
	combined.Grow(len(baseURL) + len(input))
	combined.WriteString(baseURL)
	combined.WriteString(input)
	combinedURL, err := url.Parse(combined.String())
	if err != nil {
		return "", err
	}
	return combinedURL.String(), nil
}

func ensureLeadingSlash(s string) string {
	if s == "" || strings.HasPrefix(s, "/") {
		return s
	}
	return "/" + s
}
