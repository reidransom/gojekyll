package site

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/jigyll/filters"
	"github.com/reidransom/jigyll/localization"
	"github.com/reidransom/jigyll/pages"
	"github.com/reidransom/jigyll/utils"
)

// localizedSiteContext is the one runtime object shared by a prepared locale
// site's page drops and Liquid filters. Its lookup tables are completed after
// every locale site is prepared, before any site renders.
type localizedSiteContext struct {
	site        *Site
	locale      config.Locale
	registry    *config.LocalizationConfig
	messages    *localization.MessageCatalog
	pageInfo    map[pages.Page]localizedPageInfo
	routePages  map[string]pages.Page
	sharedAssets map[string]struct{}
}

type localizedPageInfo struct {
	identity localization.Identity
	locale   config.Locale
	page     pages.Page
	all      map[string]pages.Page
}

var _ filters.LocalizationContext = (*localizedSiteContext)(nil)
var _ pages.LocalizationContext = (*localizedSiteContext)(nil)

// LocalizationContext makes the optional page-drop seam available only to
// prepared locale sites. Ordinary Site values return no localization metadata.
func (s *Site) LocalizationContext() pages.LocalizationContext {
	if s.localizationContext == nil {
		return nil
	}
	return s.localizationContext
}

func (c *localizedSiteContext) ActiveLocale() string { return c.locale.Key }

func (c *localizedSiteContext) PageLocalization(page pages.Page) (pages.PageLocalization, bool) {
	info, found := c.pageInfo[page]
	if !found {
		return pages.PageLocalization{}, false
	}
	all := c.orderedPages(info)
	translations := make([]pages.Page, 0, len(all))
	alternates := make([]pages.Alternate, 0, len(all))
	for _, candidate := range all {
		if candidate != page {
			translations = append(translations, candidate)
		}
		candidateInfo := c.pageInfo[candidate]
		alternates = append(alternates, pages.Alternate{
			Language: candidateInfo.locale,
			URL:      canonicalLocalizedURL(c.site.cfg, candidate.URL()),
			XDefault: candidateInfo.locale.Default,
		})
	}
	return pages.PageLocalization{
		Language:        info.locale,
		TranslationKey:  info.identity.TranslationKey,
		Translations:    translations,
		AllTranslations: all,
		Alternates:      alternates,
	}, true
}

func (c *localizedSiteContext) Translation(value interface{}, targetLocale string) (interface{}, error) {
	if _, exists := c.registry.Locale(targetLocale); !exists {
		return nil, fmt.Errorf("unknown locale %q", targetLocale)
	}
	info, err := c.pageInfoFor(value)
	if err != nil {
		return nil, err
	}
	if info.identity.TranslationKey == "" {
		return nil, nil
	}
	return info.all[targetLocale], nil
}

func (c *localizedSiteContext) LocalizedPageURL(value interface{}, targetLocale string) (string, error) {
	edition, err := c.Translation(value, targetLocale)
	if err != nil {
		return "", err
	}
	page, ok := edition.(pages.Page)
	if !ok || page == nil {
		return "", fmt.Errorf("no published %q edition", targetLocale)
	}
	return page.URL(), nil
}

func (c *localizedSiteContext) LocalizedRouteURL(route, targetLocale string) (string, error) {
	if _, exists := c.registry.Locale(targetLocale); !exists {
		return "", fmt.Errorf("unknown locale %q", targetLocale)
	}
	page, found := c.routePages[route]
	if !found {
		return "", fmt.Errorf("unknown localized content route %q", route)
	}
	return c.LocalizedPageURL(page, targetLocale)
}

func (c *localizedSiteContext) IsSharedAsset(route string) bool {
	_, found := c.sharedAssets[route]
	return found
}

func (c *localizedSiteContext) Translate(key string) (string, error) {
	return c.messages.Translate(key)
}

func (c *localizedSiteContext) pageInfoFor(value interface{}) (localizedPageInfo, error) {
	if page, ok := value.(pages.Page); ok {
		if info, found := c.pageInfo[page]; found {
			return info, nil
		}
	}
	fields := liquidFields(value)
	key, keyFound := fields["translation_key"].(string)
	if !keyFound || key == "" {
		return localizedPageInfo{}, fmt.Errorf("translation filter requires a localized page")
	}
	namespace := localization.PagesNamespace
	if collection, ok := fields["collection"].(string); ok && collection != "" {
		namespace = collection
	}
	for _, info := range c.pageInfo {
		if info.identity == (localization.Identity{Namespace: namespace, TranslationKey: key}) {
			return info, nil
		}
	}
	return localizedPageInfo{}, fmt.Errorf("unknown localized page with translation key %q", key)
}

func (c *localizedSiteContext) orderedPages(info localizedPageInfo) []pages.Page {
	ordered := make([]pages.Page, 0, len(info.all))
	for _, locale := range c.registry.OrderedLocales() {
		if page, found := info.all[locale.Key]; found {
			ordered = append(ordered, page)
		}
	}
	return ordered
}

func liquidFields(value interface{}) map[string]interface{} {
	if fields, ok := value.(map[string]interface{}); ok {
		return fields
	}
	result := map[string]interface{}{}
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() || reflected.Kind() != reflect.Map || reflected.Type().Key().Kind() != reflect.String {
		return result
	}
	iterator := reflected.MapRange()
	for iterator.Next() {
		result[iterator.Key().String()] = iterator.Value().Interface()
	}
	return result
}

func canonicalLocalizedURL(cfg config.Config, route string) string {
	return utils.URLJoin(cfg.AbsoluteURL, cfg.BaseURL, route)
}

func localeRelativeRoute(prefix, route string) string {
	prefix = "/" + strings.Trim(prefix, "/")
	if prefix != "/" && (route == prefix || strings.HasPrefix(route, prefix+"/")) {
		route = strings.TrimPrefix(route, prefix)
	}
	if route == "" {
		return "/"
	}
	return route
}

func localeLiquidValue(locale config.Locale) map[string]interface{} {
	return map[string]interface{}{
		"key":       locale.Key,
		"tag":       locale.Tag,
		"label":     locale.Label,
		"direction": locale.Direction,
		"weight":    locale.Weight,
		"default":   locale.Default,
	}
}
