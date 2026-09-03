package config

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	yaml "gopkg.in/yaml.v2"
)

// LocalizationConfig is the opt-in configuration for a localized project.
// Its locale map is keyed by the project's stable locale identifiers.
type LocalizationConfig struct {
	DefaultLanguage         string            `yaml:"default_language"`
	DefaultLanguageInSubdir bool              `yaml:"default_language_in_subdir"`
	MissingMessages         string            `yaml:"missing_messages"`
	RequiredTranslations    []string          `yaml:"required_translations"`
	Locales                 map[string]Locale `yaml:"locales"`
}

// Locale describes one configured locale. Key and Default are derived from the
// enclosing LocalizationConfig and are available to Liquid-facing callers.
type Locale struct {
	Key       string                 `yaml:"-"`
	Tag       string                 `yaml:"tag"`
	Label     string                 `yaml:"label"`
	Direction string                 `yaml:"direction"`
	Weight    *int                   `yaml:"weight"`
	Fallbacks []string               `yaml:"fallbacks"`
	Variables map[string]interface{} `yaml:"variables"`
	Default   bool                   `yaml:"-"`
}

// Enabled reports whether the project opted into localization.
func (c *Config) Enabled() bool {
	return c.Localization != nil
}

// OrderedLocales returns fresh locale records in presentation order: weighted
// locales first by weight, then unweighted locales, with the locale key as the
// deterministic tie-breaker in both groups.
func (l *LocalizationConfig) OrderedLocales() []Locale {
	if l == nil {
		return nil
	}
	locales := make([]Locale, 0, len(l.Locales))
	for _, locale := range l.Locales {
		locales = append(locales, cloneLocale(locale))
	}
	sort.Slice(locales, func(i, j int) bool {
		left, right := locales[i], locales[j]
		switch {
		case left.Weight == nil && right.Weight != nil:
			return false
		case left.Weight != nil && right.Weight == nil:
			return true
		case left.Weight != nil && right.Weight != nil && *left.Weight != *right.Weight:
			return *left.Weight < *right.Weight
		default:
			return left.Key < right.Key
		}
	})
	return locales
}

// Locale returns a fresh locale record for key.
func (l *LocalizationConfig) Locale(key string) (Locale, bool) {
	if l == nil {
		return Locale{}, false
	}
	locale, ok := l.Locales[key]
	return cloneLocale(locale), ok
}

// Validate applies localization defaults and validates the complete locale
// registry. Its errors are collected and sorted so invalid configuration has a
// stable, actionable diagnostic independent of YAML map iteration order.
func (l *LocalizationConfig) Validate() error {
	if l == nil {
		return nil
	}

	problems := l.validateSettings()
	keys := sortedLocaleKeys(l.Locales)
	seenTags := make(map[string]string, len(keys))
	for _, key := range keys {
		locale := l.Locales[key]
		problems = append(problems, validateLocale(key, l.DefaultLanguage, &locale, seenTags)...)
		l.Locales[key] = locale
	}
	problems = append(problems, l.validateFallbacks(keys)...)
	problems = append(problems, l.validateRequiredTranslations()...)
	problems = append(problems, fallbackCycleProblems(l.Locales)...)
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return &LocalizationValidationError{Problems: problems}
}

func (l *LocalizationConfig) validateSettings() []string {
	var problems []string
	if len(l.Locales) == 0 {
		problems = append(problems, "locales must contain at least one locale")
	}
	if l.DefaultLanguage == "" {
		problems = append(problems, "default_language is required")
	} else if _, ok := l.Locales[l.DefaultLanguage]; !ok {
		problems = append(problems, fmt.Sprintf("default_language %q does not name a configured locale", l.DefaultLanguage))
	}
	if l.MissingMessages == "" {
		l.MissingMessages = "error"
	} else if l.MissingMessages != "error" && l.MissingMessages != "key" {
		problems = append(problems, fmt.Sprintf("missing_messages must be \"error\" or \"key\", got %q", l.MissingMessages))
	}
	return problems
}

func (l *LocalizationConfig) validateRequiredTranslations() []string {
	seen := make(map[string]struct{}, len(l.RequiredTranslations))
	var problems []string
	for index, locale := range l.RequiredTranslations {
		if _, duplicate := seen[locale]; duplicate {
			problems = append(problems, fmt.Sprintf("required_translations[%d]: duplicate locale %q", index, locale))
			continue
		}
		seen[locale] = struct{}{}
		if _, exists := l.Locales[locale]; !exists {
			problems = append(problems, fmt.Sprintf("required_translations[%d]: unknown locale %q", index, locale))
		}
		if locale == l.DefaultLanguage {
			problems = append(problems, fmt.Sprintf("required_translations[%d]: default locale %q cannot be required", index, locale))
		}
	}
	return problems
}

func validateLocale(key, defaultLanguage string, locale *Locale, seenTags map[string]string) []string {
	locale.Key = key
	locale.Default = key == defaultLanguage
	problems := validateLocaleIdentity(key, locale, seenTags)
	if strings.TrimSpace(locale.Label) == "" {
		problems = append(problems, fmt.Sprintf("locales.%s.label: label is required", key))
	}
	if locale.Direction == "" {
		locale.Direction = "ltr"
	} else if locale.Direction != "ltr" && locale.Direction != "rtl" {
		problems = append(problems, fmt.Sprintf("locales.%s.direction: must be \"ltr\" or \"rtl\", got %q", key, locale.Direction))
	}
	for _, variable := range sortedStringMapKeys(locale.Variables) {
		if operationalLocaleVariable(variable) {
			problems = append(problems, fmt.Sprintf("locales.%s.variables.%s: operational configuration cannot be overridden per locale", key, variable))
		}
	}
	return problems
}

func validateLocaleIdentity(key string, locale *Locale, seenTags map[string]string) []string {
	var problems []string
	if !validLocaleKey(key) {
		problems = append(problems, fmt.Sprintf("locales.%s: locale key must be a lowercase URL-safe slug", key))
	}
	if !validBCP47(locale.Tag) {
		return append(problems, fmt.Sprintf("locales.%s.tag: %q is not a valid BCP 47 tag", key, locale.Tag))
	}
	canonicalTag := strings.ToLower(locale.Tag)
	if other, exists := seenTags[canonicalTag]; exists {
		return append(problems, fmt.Sprintf("locales.%s.tag: %q duplicates locales.%s.tag under case-insensitive comparison", key, locale.Tag, other))
	}
	seenTags[canonicalTag] = key
	return problems
}

func (l *LocalizationConfig) validateFallbacks(keys []string) []string {
	defaultExists := l.defaultLocaleExists()
	var problems []string
	for _, key := range keys {
		locale := l.Locales[key]
		valid, fallbackProblems := validateFallbacks(key, locale.Fallbacks, l.Locales)
		problems = append(problems, fallbackProblems...)
		if key != l.DefaultLanguage && defaultExists && !containsLocale(locale.Fallbacks, l.DefaultLanguage) {
			valid = append(valid, l.DefaultLanguage)
		}
		locale.Fallbacks = valid
		l.Locales[key] = locale
	}
	return problems
}

func (l *LocalizationConfig) defaultLocaleExists() bool {
	_, exists := l.Locales[l.DefaultLanguage]
	return exists
}

func validateFallbacks(key string, fallbacks []string, locales map[string]Locale) ([]string, []string) {
	seen := make(map[string]struct{}, len(fallbacks))
	valid := make([]string, 0, len(fallbacks)+1)
	var problems []string
	for index, fallback := range fallbacks {
		if _, duplicate := seen[fallback]; duplicate {
			problems = append(problems, fmt.Sprintf("locales.%s.fallbacks[%d]: duplicate fallback %q", key, index, fallback))
			continue
		}
		seen[fallback] = struct{}{}
		if fallback == key {
			problems = append(problems, fmt.Sprintf("locales.%s.fallbacks[%d]: locale cannot fall back to itself", key, index))
			continue
		}
		if _, exists := locales[fallback]; !exists {
			problems = append(problems, fmt.Sprintf("locales.%s.fallbacks[%d]: unknown locale %q", key, index, fallback))
			continue
		}
		valid = append(valid, fallback)
	}
	return valid, problems
}

func containsLocale(locales []string, key string) bool {
	for _, locale := range locales {
		if locale == key {
			return true
		}
	}
	return false
}

// LocalizationValidationError reports all independent validation failures in
// deterministic order.
type LocalizationValidationError struct {
	Problems []string
}

func (e *LocalizationValidationError) Error() string {
	return "invalid localization configuration:\n - " + strings.Join(e.Problems, "\n - ")
}

// DeriveLocale returns an entirely independent configuration for locale. It
// applies that locale's Liquid-facing variable overrides without changing the
// shared configuration or any other locale's derived configuration.
func (c Config) DeriveLocale(key string) (Config, error) {
	if c.Localization == nil {
		return Config{}, fmt.Errorf("cannot derive locale %q: localization is not enabled", key)
	}
	locale, ok := c.Localization.Locales[key]
	if !ok {
		return Config{}, fmt.Errorf("cannot derive locale %q: unknown locale", key)
	}

	derived := c.Clone()
	for _, variable := range sortedStringMapKeys(locale.Variables) {
		derived.Set(variable, cloneValue(locale.Variables[variable]))
	}
	return derived, nil
}

// Clone returns a configuration with no mutable maps or slices shared with the
// receiver. It is the base for every locale-specific configuration.
func (c Config) Clone() Config {
	clone := c
	clone.Include = append([]string(nil), c.Include...)
	clone.Exclude = append([]string(nil), c.Exclude...)
	clone.KeepFiles = append([]string(nil), c.KeepFiles...)
	clone.Plugins = append([]string(nil), c.Plugins...)
	clone.Collections = cloneCollections(c.Collections)
	clone.Defaults = cloneDefaults(c.Defaults)
	clone.m = cloneStringMap(c.m)
	clone.ms = cloneMapSlice(c.ms)
	clone.RequireFrontMatterExclude = cloneBoolMap(c.RequireFrontMatterExclude)
	clone.Localization = cloneLocalization(c.Localization)
	return clone
}

func cloneLocalization(l *LocalizationConfig) *LocalizationConfig {
	if l == nil {
		return nil
	}
	clone := *l
	clone.RequiredTranslations = append([]string(nil), l.RequiredTranslations...)
	clone.Locales = make(map[string]Locale, len(l.Locales))
	for key, locale := range l.Locales {
		clone.Locales[key] = cloneLocale(locale)
	}
	return &clone
}

func cloneLocale(locale Locale) Locale {
	clone := locale
	if locale.Weight != nil {
		weight := *locale.Weight
		clone.Weight = &weight
	}
	clone.Fallbacks = append([]string(nil), locale.Fallbacks...)
	clone.Variables = cloneStringMap(locale.Variables)
	return clone
}

func cloneCollections(collections map[string]map[string]interface{}) map[string]map[string]interface{} {
	if collections == nil {
		return nil
	}
	clone := make(map[string]map[string]interface{}, len(collections))
	for key, values := range collections {
		clone[key] = cloneStringMap(values)
	}
	return clone
}

func cloneDefaults(defaults []struct {
	Scope struct {
		Path string
		Type string
	}
	Values map[string]interface{}
}) []struct {
	Scope struct {
		Path string
		Type string
	}
	Values map[string]interface{}
} {
	if defaults == nil {
		return nil
	}
	clone := make([]struct {
		Scope struct {
			Path string
			Type string
		}
		Values map[string]interface{}
	}, len(defaults))
	for index, entry := range defaults {
		clone[index] = entry
		clone[index].Values = cloneStringMap(entry.Values)
	}
	return clone
}

func cloneMapSlice(values yaml.MapSlice) yaml.MapSlice {
	if values == nil {
		return nil
	}
	clone := make(yaml.MapSlice, len(values))
	for index, item := range values {
		clone[index] = yaml.MapItem{Key: cloneValue(item.Key), Value: cloneValue(item.Value)}
	}
	return clone
}

func cloneStringMap(values map[string]interface{}) map[string]interface{} {
	if values == nil {
		return nil
	}
	clone := make(map[string]interface{}, len(values))
	for key, value := range values {
		clone[key] = cloneValue(value)
	}
	return clone
}

func cloneBoolMap(values map[string]bool) map[string]bool {
	if values == nil {
		return nil
	}
	clone := make(map[string]bool, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func cloneValue(value interface{}) interface{} {
	switch value := value.(type) {
	case map[string]interface{}:
		return cloneStringMap(value)
	case map[interface{}]interface{}:
		clone := make(map[interface{}]interface{}, len(value))
		for key, item := range value {
			clone[cloneValue(key)] = cloneValue(item)
		}
		return clone
	case []interface{}:
		clone := make([]interface{}, len(value))
		for index, item := range value {
			clone[index] = cloneValue(item)
		}
		return clone
	case []string:
		return append([]string(nil), value...)
	case yaml.MapSlice:
		return cloneMapSlice(value)
	default:
		return value
	}
}

func sortedLocaleKeys(locales map[string]Locale) []string {
	keys := make([]string, 0, len(locales))
	for key := range locales {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedStringMapKeys(values map[string]interface{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func validLocaleKey(key string) bool {
	if key == "" || key[0] == '-' || key[len(key)-1] == '-' {
		return false
	}
	for _, char := range key {
		if char == '-' {
			continue
		}
		if char < 'a' || char > 'z' {
			if char < '0' || char > '9' {
				return false
			}
		}
	}
	return true
}

func operationalLocaleVariable(key string) bool {
	_, operational := map[string]struct{}{
		"baseurl": {}, "collections": {}, "data_dir": {}, "destination": {}, "drafts": {}, "dry_run": {},
		"exclude": {}, "force_polling": {}, "future": {}, "host": {}, "include": {}, "includes_dir": {},
		"incremental": {}, "keep_files": {}, "layouts_dir": {}, "liquid": {}, "markdown_ext": {}, "paginate": {},
		"paginate_path": {}, "permalink": {}, "plugins": {}, "port": {}, "remote_theme": {}, "sass": {}, "show_drafts": {},
		"source": {}, "theme": {}, "timezone": {}, "unpublished": {}, "url": {}, "verbose": {}, "watch": {}, "localization": {},
		"gems": {},
	}[key]
	return operational
}

func fallbackCycleProblems(locales map[string]Locale) []string {
	state := make(map[string]uint8, len(locales))
	path := make([]string, 0, len(locales))
	var problems []string
	var visit func(string)
	visit = func(key string) {
		state[key] = 1
		path = append(path, key)
		for _, fallback := range locales[key].Fallbacks {
			switch state[fallback] {
			case 0:
				visit(fallback)
			case 1:
				start := 0
				for path[start] != fallback {
					start++
				}
				cycle := append(append([]string(nil), path[start:]...), fallback)
				problems = append(problems, "fallback cycle: "+strings.Join(cycle, " -> "))
			}
		}
		path = path[:len(path)-1]
		state[key] = 2
	}
	for _, key := range sortedLocaleKeys(locales) {
		if state[key] == 0 {
			visit(key)
		}
	}
	return problems
}

func validBCP47(tag string) bool {
	if grandfatheredBCP47Tags[strings.ToLower(tag)] {
		return true
	}
	parts, valid := bcp47Parts(tag)
	if !valid {
		return false
	}
	if strings.EqualFold(parts[0], "x") {
		return validPrivateUse(parts[1:])
	}
	return validRegularBCP47(parts)
}

func bcp47Parts(tag string) ([]string, bool) {
	if tag == "" || strings.Contains(tag, "_") {
		return nil, false
	}
	parts := strings.Split(tag, "-")
	for _, part := range parts {
		if part == "" {
			return nil, false
		}
	}
	return parts, true
}

func validRegularBCP47(parts []string) bool {
	index, valid := consumeLanguage(parts)
	if !valid {
		return false
	}
	index = consumeScript(parts, index)
	index = consumeRegion(parts, index)
	index, valid = consumeVariants(parts, index)
	if !valid {
		return false
	}
	index, valid = consumeExtensions(parts, index)
	if !valid {
		return false
	}
	if index < len(parts) && strings.EqualFold(parts[index], "x") {
		return validPrivateUse(parts[index+1:])
	}
	return index == len(parts)
}

func consumeLanguage(parts []string) (int, bool) {
	language := parts[0]
	if !allLetters(language) || len(language) < 2 || len(language) > 8 {
		return 0, false
	}
	index := 1
	if len(language) <= 3 {
		for index < len(parts) && len(parts[index]) == 3 && allLetters(parts[index]) && index <= 3 {
			index++
		}
	}
	return index, true
}

func consumeScript(parts []string, index int) int {
	if index < len(parts) && len(parts[index]) == 4 && allLetters(parts[index]) {
		return index + 1
	}
	return index
}

func consumeRegion(parts []string, index int) int {
	if index < len(parts) && ((len(parts[index]) == 2 && allLetters(parts[index])) || (len(parts[index]) == 3 && allDigits(parts[index]))) {
		return index + 1
	}
	return index
}

func consumeVariants(parts []string, index int) (int, bool) {
	seen := make(map[string]struct{})
	for index < len(parts) && validVariant(parts[index]) {
		variant := strings.ToLower(parts[index])
		if _, found := seen[variant]; found {
			return index, false
		}
		seen[variant] = struct{}{}
		index++
	}
	return index, true
}

func consumeExtensions(parts []string, index int) (int, bool) {
	seen := make(map[string]struct{})
	for index < len(parts) && validExtensionSingleton(parts[index]) {
		singleton := strings.ToLower(parts[index])
		if _, found := seen[singleton]; found {
			return index, false
		}
		seen[singleton] = struct{}{}
		index++
		start := index
		for index < len(parts) && len(parts[index]) >= 2 && len(parts[index]) <= 8 && allAlphaNumeric(parts[index]) {
			index++
		}
		if start == index {
			return index, false
		}
	}
	return index, true
}

func validPrivateUse(parts []string) bool {
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if len(part) < 1 || len(part) > 8 || !allAlphaNumeric(part) {
			return false
		}
	}
	return true
}

func validVariant(part string) bool {
	return (len(part) >= 5 && len(part) <= 8 && allAlphaNumeric(part)) ||
		(len(part) == 4 && part[0] >= '0' && part[0] <= '9' && allAlphaNumeric(part))
}

func validExtensionSingleton(part string) bool {
	return len(part) == 1 && allAlphaNumeric(part) && !strings.EqualFold(part, "x")
}

func allLetters(value string) bool {
	for _, char := range value {
		if !unicode.IsLetter(char) || char > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func allDigits(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func allAlphaNumeric(value string) bool {
	for _, char := range value {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') {
			return false
		}
	}
	return true
}

var grandfatheredBCP47Tags = map[string]bool{
	"art-lojban": true, "cel-gaulish": true, "en-gb-oed": true, "i-ami": true, "i-bnn": true,
	"i-default": true, "i-enochian": true, "i-hak": true, "i-klingon": true, "i-lux": true,
	"i-mingo": true, "i-navajo": true, "i-pwn": true, "i-tao": true, "i-tay": true,
	"i-tsu": true, "no-bok": true, "no-nyn": true, "sgn-be-fr": true, "sgn-be-nl": true,
	"sgn-ch-de": true, "zh-guoyu": true, "zh-hakka": true, "zh-min": true, "zh-min-nan": true,
	"zh-xiang": true,
}
