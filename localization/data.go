// Package localization contains the project-level localization seams shared by
// discovery, prepared locale sites, and Liquid registration.
package localization

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/jigyll/utils"
)

// DataCatalog owns immutable shared data and locale overlays discovered from a
// project's data directory. Its methods always return independent values, so a
// locale overlay cannot mutate the data exposed to another locale.
type DataCatalog struct {
	shared         map[string]interface{}
	overlays       map[string]map[string]interface{}
	fallbackChains map[string][]string
	missingMode    string
}

// DiscoverData reads a project's data directory. A top-level shared data file
// contributes its mapping directly to common data. The locales directory is
// reserved for locale overlays and is never included in shared data.
func DiscoverData(dir string, locales *config.LocalizationConfig) (*DataCatalog, error) {
	if locales == nil {
		return nil, fmt.Errorf("discovering localized data: localization is not enabled")
	}
	if err := locales.Validate(); err != nil {
		return nil, fmt.Errorf("discovering localized data: %w", err)
	}

	shared, overlays, err := discoverData(dir, locales)
	if err != nil {
		return nil, err
	}
	return newDataCatalog(shared, overlays, locales), nil
}

// Shared returns a fresh copy of data common to every locale. It never
// contains the reserved locales subtree.
func (c *DataCatalog) Shared() map[string]interface{} {
	if c == nil {
		return nil
	}
	return cloneMap(c.shared)
}

// Data returns the data visible to locale. It merges shared data, then each
// fallback from farthest to nearest, then the active locale overlay.
func (c *DataCatalog) Data(locale string) (map[string]interface{}, error) {
	chain, err := c.chain(locale)
	if err != nil {
		return nil, err
	}

	data := cloneMap(c.shared)
	for i := len(chain) - 1; i >= 0; i-- {
		data = mergeMaps(data, c.overlays[chain[i]])
	}
	return data, nil
}

// Messages returns the message catalog bound to locale. The returned module is
// the seam consumed by Liquid's translate filter; it deliberately does not
// expose generic data merging details to filter registration.
func (c *DataCatalog) Messages(locale string) (*MessageCatalog, error) {
	chain, err := c.chain(locale)
	if err != nil {
		return nil, err
	}
	return &MessageCatalog{
		overlays:    c.overlays,
		chain:       append([]string(nil), chain...),
		activeLocale: locale,
		missingMode: c.missingMode,
	}, nil
}

func (c *DataCatalog) chain(locale string) ([]string, error) {
	if c == nil {
		return nil, fmt.Errorf("localized data is not configured")
	}
	chain, ok := c.fallbackChains[locale]
	if !ok {
		return nil, fmt.Errorf("unknown locale %q", locale)
	}
	return append([]string(nil), chain...), nil
}

func newDataCatalog(shared map[string]interface{}, overlays map[string]map[string]interface{}, locales *config.LocalizationConfig) *DataCatalog {
	fallbackChains := make(map[string][]string, len(locales.Locales))
	for key := range locales.Locales {
		fallbackChains[key] = fallbackChain(key, locales.Locales)
	}
	return &DataCatalog{
		shared:         cloneMap(shared),
		overlays:       cloneOverlays(overlays),
		fallbackChains: fallbackChains,
		missingMode:    locales.MissingMessages,
	}
}

// fallbackChain returns the active locale followed by its nearest fallbacks.
// A depth-first traversal preserves each locale's declared fallback order and
// reaches a fallback's own prerequisites before moving to a farther sibling.
func fallbackChain(locale string, locales map[string]config.Locale) []string {
	chain := make([]string, 0, len(locales))
	seen := make(map[string]bool, len(locales))
	var visit func(string)
	visit = func(key string) {
		if seen[key] {
			return
		}
		seen[key] = true
		chain = append(chain, key)
		for _, fallback := range locales[key].Fallbacks {
			visit(fallback)
		}
	}
	visit(locale)
	return chain
}

func discoverData(dir string, locales *config.LocalizationConfig) (map[string]interface{}, map[string]map[string]interface{}, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]interface{}{}, map[string]map[string]interface{}{}, nil
		}
		return nil, nil, fmt.Errorf("reading data directory %q: %w", dir, err)
	}

	shared := make(map[string]interface{})
	claims := make(map[string]string)
	var problems []string
	var localeDir string
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.Name() == "locales" && entry.IsDir() {
			claims["locales"] = "locales"
			localeDir = path
			continue
		}
		if dataExtension(filepath.Ext(entry.Name())) && strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())) == "locales" {
			if prior, exists := claims["locales"]; exists {
				problems = append(problems, fmt.Sprintf("data key %q is defined by both %q and %q", "locales", prior, entry.Name()))
			} else {
				problems = append(problems, fmt.Sprintf("reserved locale data path %q must be a directory", entry.Name()))
			}
			continue
		}
		if dataExtension(filepath.Ext(entry.Name())) && strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())) == "shared" {
			if prior, exists := claims["shared"]; exists {
				problems = append(problems, fmt.Sprintf("data key %q is defined by both %q and %q", "shared", prior, entry.Name()))
				continue
			}
			claims["shared"] = entry.Name()
			value, err := readDataFile(path)
			if err != nil {
				problems = append(problems, fmt.Sprintf("reading data file %q: %v", entry.Name(), err))
				continue
			}
			values, ok := value.(map[string]interface{})
			if !ok {
				problems = append(problems, fmt.Sprintf("shared data file %q must contain a mapping", entry.Name()))
				continue
			}
			shared = mergeMaps(shared, values)
			continue
		}
		readDataEntry(path, entry, entry.Name(), shared, claims, &problems)
	}

	overlays := make(map[string]map[string]interface{})
	if localeDir != "" {
		discoverLocaleOverlays(localeDir, locales, overlays, &problems)
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return nil, nil, &DataValidationError{Problems: problems}
	}
	return shared, overlays, nil
}

func discoverLocaleOverlays(dir string, locales *config.LocalizationConfig, overlays map[string]map[string]interface{}, problems *[]string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		*problems = append(*problems, fmt.Sprintf("reading reserved locale data directory %q: %v", filepath.Base(dir), err))
		return
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		display := filepath.Join("locales", entry.Name())
		if !entry.IsDir() {
			*problems = append(*problems, fmt.Sprintf("reserved locale data path %q must be a directory", display))
			continue
		}
		if _, ok := locales.Locales[entry.Name()]; !ok {
			*problems = append(*problems, fmt.Sprintf("locale data %q names an unknown locale", display))
			continue
		}
		data, err := readLocaleOverlay(path, display)
		if err != nil {
			appendDataTreeError(problems, err)
			continue
		}
		overlays[entry.Name()] = data
	}
}

// readLocaleOverlay reads locale data using the same filename-keyed tree as
// ordinary data. In particular, locales/de/settings.yml overlays
// site.data.settings, while locales/de/messages.yml is the message module.
func readLocaleOverlay(dir, display string) (map[string]interface{}, error) {
	return readDataTree(dir, display)
}

func readDataEntry(path string, entry os.DirEntry, display string, data map[string]interface{}, claims map[string]string, problems *[]string) {
	key, value, recognized, err := readDataPath(path, entry, display)
	if err != nil {
		appendDataTreeError(problems, err)
		return
	}
	if !recognized {
		return
	}
	if prior, exists := claims[key]; exists {
		*problems = append(*problems, fmt.Sprintf("data key %q is defined by both %q and %q", key, prior, display))
		return
	}
	claims[key] = display
	data[key] = value
}

func readDataTree(dir, display string) (map[string]interface{}, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading data directory %q: %w", display, err)
	}
	data := make(map[string]interface{})
	claims := make(map[string]string)
	var problems []string
	for _, entry := range entries {
		entryDisplay := filepath.Join(display, entry.Name())
		readDataEntry(filepath.Join(dir, entry.Name()), entry, entryDisplay, data, claims, &problems)
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return nil, &DataValidationError{Problems: problems}
	}
	return data, nil
}

func readDataPath(path string, entry os.DirEntry, display string) (string, interface{}, bool, error) {
	if entry.IsDir() {
		value, err := readDataTree(path, display)
		return entry.Name(), value, true, err
	}

	ext := filepath.Ext(entry.Name())
	if !dataExtension(ext) {
		return "", nil, false, nil
	}
	value, err := readDataFile(path)
	if err != nil {
		return "", nil, false, fmt.Errorf("reading data file %q: %w", display, err)
	}
	return strings.TrimSuffix(entry.Name(), ext), value, true, nil
}

func appendDataTreeError(problems *[]string, err error) {
	if validation, ok := err.(*DataValidationError); ok {
		*problems = append(*problems, validation.Problems...)
		return
	}
	*problems = append(*problems, err.Error())
}

func dataExtension(ext string) bool {
	switch ext {
	case ".csv", ".json", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func readDataFile(filename string) (interface{}, error) {
	switch filepath.Ext(filename) {
	case ".csv":
		file, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer file.Close() // nolint:errcheck
		return csv.NewReader(file).ReadAll()
	case ".json":
		contents, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		var value interface{}
		if err := json.Unmarshal(contents, &value); err != nil {
			return nil, err
		}
		return value, nil
	case ".yaml", ".yml":
		contents, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		var value interface{}
		if err := utils.UnmarshalYAMLInterface(contents, &value); err != nil {
			return nil, err
		}
		return value, nil
	default:
		return nil, fmt.Errorf("unsupported data file %q", filename)
	}
}

// DataValidationError reports independent data-discovery failures in a stable
// order, independent of filesystem enumeration order.
type DataValidationError struct {
	Problems []string
}

func (e *DataValidationError) Error() string {
	return "invalid localized data:\n - " + strings.Join(e.Problems, "\n - ")
}

func mergeMaps(base, overlay map[string]interface{}) map[string]interface{} {
	merged := cloneMap(base)
	for key, overlayValue := range overlay {
		if baseValue, exists := merged[key]; exists {
			baseMap, baseIsMap := baseValue.(map[string]interface{})
			overlayMap, overlayIsMap := overlayValue.(map[string]interface{})
			if baseIsMap && overlayIsMap {
				merged[key] = mergeMaps(baseMap, overlayMap)
				continue
			}
		}
		merged[key] = cloneValue(overlayValue)
	}
	return merged
}

func cloneOverlays(overlays map[string]map[string]interface{}) map[string]map[string]interface{} {
	cloned := make(map[string]map[string]interface{}, len(overlays))
	for locale, data := range overlays {
		cloned[locale] = cloneMap(data)
	}
	return cloned
}

func cloneMap(source map[string]interface{}) map[string]interface{} {
	cloned := make(map[string]interface{}, len(source))
	for key, value := range source {
		cloned[key] = cloneValue(value)
	}
	return cloned
}

func cloneValue(value interface{}) interface{} {
	switch value := value.(type) {
	case map[string]interface{}:
		return cloneMap(value)
	case []interface{}:
		cloned := make([]interface{}, len(value))
		for index, item := range value {
			cloned[index] = cloneValue(item)
		}
		return cloned
	case [][]string:
		cloned := make([][]string, len(value))
		for index, row := range value {
			cloned[index] = append([]string(nil), row...)
		}
		return cloned
	default:
		return value
	}
}
