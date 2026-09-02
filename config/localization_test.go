package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalizationConfigValidatesAndOrdersLocales(t *testing.T) {
	c := Default()
	require.NoError(t, Unmarshal([]byte(`
localization:
  default_language: en
  locales:
    en:
      tag: en-US
      label: English
      weight: 10
    de:
      tag: de-DE
      label: Deutsch
      weight: 10
      fallbacks: [fr]
    fr:
      tag: fr
      label: Français
      weight: 20
    ja:
      tag: ja
      label: 日本語
`), &c))

	require.True(t, c.Enabled())
	require.Equal(t, "error", c.Localization.MissingMessages)
	require.Equal(t, "ltr", c.Localization.Locales["en"].Direction)
	require.True(t, c.Localization.Locales["en"].Default)
	require.Equal(t, []string{"fr", "en"}, c.Localization.Locales["de"].Fallbacks)
	require.Equal(t, []string{}, c.Localization.Locales["en"].Fallbacks)

	ordered := c.Localization.OrderedLocales()
	require.Equal(t, []string{"de", "en", "fr", "ja"}, []string{
		ordered[0].Key,
		ordered[1].Key,
		ordered[2].Key,
		ordered[3].Key,
	})

	ordered[0].Fallbacks[0] = "mutated"
	require.Equal(t, "fr", c.Localization.Locales["de"].Fallbacks[0])
}

func TestLocalizationConfigRejectsRequiredAndLocaleRecordErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		want string
	}{
		{
			name: "empty locale map",
			src:  "localization: {default_language: en, locales: {}}",
			want: "locales must contain at least one locale",
		},
		{
			name: "unknown default locale",
			src: `
localization:
  default_language: en
  locales:
    de: {tag: de, label: Deutsch}
`,
			want: `default_language "en" does not name a configured locale`,
		},
		{
			name: "invalid locale key",
			src: `
localization:
  default_language: EN
  locales:
    EN: {tag: en, label: English}
`,
			want: "locales.EN: locale key must be a lowercase URL-safe slug",
		},
		{
			name: "invalid tag",
			src: `
localization:
  default_language: en
  locales:
    en: {tag: en_US, label: English}
`,
			want: `locales.en.tag: "en_US" is not a valid BCP 47 tag`,
		},
		{
			name: "duplicate tag under canonical comparison",
			src: `
localization:
  default_language: en
  locales:
    en: {tag: en-US, label: English}
    english: {tag: EN-us, label: English}
`,
			want: `locales.english.tag: "EN-us" duplicates locales.en.tag under case-insensitive comparison`,
		},
		{
			name: "missing label",
			src: `
localization:
  default_language: en
  locales:
    en: {tag: en}
`,
			want: "locales.en.label: label is required",
		},
		{
			name: "invalid direction",
			src: `
localization:
  default_language: en
  locales:
    en: {tag: en, label: English, direction: left}
`,
			want: `locales.en.direction: must be "ltr" or "rtl", got "left"`,
		},
		{
			name: "invalid missing message behavior",
			src: `
localization:
  default_language: en
  missing_messages: blank
  locales:
    en: {tag: en, label: English}
`,
			want: `missing_messages must be "error" or "key", got "blank"`,
		},
		{
			name: "operational locale variable",
			src: `
localization:
  default_language: en
  locales:
    en:
      tag: en
      label: English
      variables: {destination: translated-site}
`,
			want: "locales.en.variables.destination: operational configuration cannot be overridden per locale",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := Default()
			err := Unmarshal([]byte(tc.src), &c)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestLocalizationConfigRejectsFallbackFailuresDeterministically(t *testing.T) {
	src := []byte(`
localization:
  default_language: en
  locales:
    en: {tag: en, label: English}
    de: {tag: de, label: Deutsch, fallbacks: [fr, fr, xx]}
    fr: {tag: fr, label: Français, fallbacks: [de]}
`)

	var errors []string
	for range 2 {
		c := Default()
		err := Unmarshal(src, &c)
		require.Error(t, err)
		errors = append(errors, err.Error())
	}
	require.Equal(t, errors[0], errors[1])
	require.Contains(t, errors[0], `locales.de.fallbacks[1]: duplicate fallback "fr"`)
	require.Contains(t, errors[0], `locales.de.fallbacks[2]: unknown locale "xx"`)
	require.Contains(t, errors[0], "fallback cycle: de -> fr -> de")
}

func TestConfigDeriveLocaleDeeplySeparatesMutableConfiguration(t *testing.T) {
	c := Default()
	require.NoError(t, Unmarshal([]byte(`
include: [shared]
collections:
  guides:
    output: true
metadata:
  items: [shared]
localization:
  default_language: en
  locales:
    en: {tag: en, label: English}
    de:
      tag: de
      label: Deutsch
      variables:
        title: Deutsch
        custom:
          items: [de]
`), &c))

	de, err := c.DeriveLocale("de")
	require.NoError(t, err)
	en, err := c.DeriveLocale("en")
	require.NoError(t, err)
	require.Equal(t, "Deutsch", de.Variables()["title"])
	require.NotContains(t, en.Variables(), "title")

	de.Include[0] = "changed"
	de.Collections["guides"]["output"] = false
	de.m["metadata"].(map[interface{}]interface{})["items"].([]interface{})[0] = "changed"
	de.m["custom"].(map[interface{}]interface{})["items"].([]interface{})[0] = "changed"
	deLocale := de.Localization.Locales["de"]
	deLocale.Fallbacks = append(deLocale.Fallbacks, "en")
	de.Localization.Locales["de"] = deLocale

	require.Equal(t, "shared", c.Include[0])
	require.Equal(t, true, c.Collections["guides"]["output"])
	require.Equal(t, "shared", c.m["metadata"].(map[interface{}]interface{})["items"].([]interface{})[0])
	require.NotContains(t, en.m, "custom")
	require.Equal(t, []string{"en"}, c.Localization.Locales["de"].Fallbacks)
}

func TestLocalizationRemainsOptIn(t *testing.T) {
	c := Default()
	require.NoError(t, Unmarshal([]byte("title: Existing site\n"), &c))
	require.False(t, c.Enabled())
	require.Equal(t, "Existing site", c.Variables()["title"])
}
