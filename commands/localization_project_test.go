package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildCommandBuildsLocalizedAcceptanceFixture(t *testing.T) {
	source := copyLocalizedAcceptanceFixture(t)

	require.NoError(t, ParseAndRun([]string{"build", "-s", source, "-q"}))

	destination := filepath.Join(source, "_site")
	for _, filename := range []string{
		"index.html",
		"about/index.html",
		"page2/index.html",
		"guides/getting-started.html",
		"2020/01/02/english-post.html",
		"2020/01/03/second-english-post.html",
		"de/index.html",
		"de/uber-uns/index.html",
		"de/guides/erste-schritte.html",
		"de/2020/01/02/deutscher-post.html",
		"fr/bonjour/index.html",
		"assets/site.css",
		"sitemap.xml",
		"robots.txt",
	} {
		require.FileExists(t, filepath.Join(destination, filename))
	}
	for _, filename := range []string{
		"de/page2/index.html",
		"draft/index.html",
		"9999/12/31/future.html",
	} {
		require.NoFileExists(t, filepath.Join(destination, filename))
	}

	englishAbout := readLocalizedAcceptanceOutput(t, destination, "about/index.html")
	require.Contains(t, englishAbout, "site-language=en")
	require.Contains(t, englishAbout, "languages=de,en,fr")
	require.Contains(t, englishAbout, "default-language=en")
	require.Contains(t, englishAbout, "page-language=en")
	require.Contains(t, englishAbout, "translation-key=about")
	require.Contains(t, englishAbout, "translations=de=/de/uber-uns/")
	require.Contains(t, englishAbout, "all-translations=de=/de/uber-uns/,en=/about/")
	require.Contains(t, englishAbout, "alternates=de=https://example.test/handbook/de/uber-uns/,en=https://example.test/handbook/about/=x-default")
	require.Contains(t, englishAbout, "shared=common")
	require.Contains(t, englishAbout, "locale-value=English")
	require.Contains(t, englishAbout, "nested=common/English")
	require.Contains(t, englishAbout, "message=Home")
	require.Contains(t, englishAbout, "page-lookup=/de/uber-uns/")
	require.Contains(t, englishAbout, "route-lookup=/de/uber-uns/")
	require.Contains(t, englishAbout, "route-query-fragment=/de/uber-uns/?source=fixture#section")
	require.Contains(t, englishAbout, "foreign-route-lookup=/about/")
	require.Contains(t, englishAbout, "shared-asset=/assets/site.css")
	require.Contains(t, englishAbout, "external=https://cdn.example.test/app.js")
	require.Contains(t, englishAbout, "fragment=#section")
	require.Contains(t, englishAbout, "relative=/handbook/about/")
	require.Contains(t, englishAbout, "absolute=https://example.test/handbook/about/")
	require.Contains(t, englishAbout, "posts=Second English post,English post")
	require.Contains(t, englishAbout, "guides=Getting started")

	germanAbout := readLocalizedAcceptanceOutput(t, destination, "de/uber-uns/index.html")
	require.Contains(t, germanAbout, "site-language=de")
	require.Contains(t, germanAbout, "locale-value=Deutsch")
	require.Contains(t, germanAbout, "message=Startseite")
	require.Contains(t, germanAbout, "posts=Deutscher Post")
	require.Contains(t, germanAbout, "guides=Erste Schritte")

	sitemap := readLocalizedAcceptanceOutput(t, destination, "sitemap.xml")
	require.Contains(t, sitemap, "<loc>https://example.test/handbook/about/</loc>")
	require.Contains(t, sitemap, "<loc>https://example.test/handbook/de/uber-uns/</loc>")
	require.Contains(t, sitemap, `hreflang="en" href="https://example.test/handbook/about/"`)
	require.Contains(t, sitemap, `hreflang="de" href="https://example.test/handbook/de/uber-uns/"`)
	require.Contains(t, sitemap, `hreflang="x-default" href="https://example.test/handbook/about/"`)
	require.Contains(t, readLocalizedAcceptanceOutput(t, destination, "robots.txt"), "Sitemap: https://example.test/handbook/sitemap.xml")
}

func copyLocalizedAcceptanceFixture(t *testing.T) string {
	t.Helper()
	destination := t.TempDir()
	source := filepath.Join("testdata", "localization-acceptance")
	require.NoError(t, filepath.Walk(source, func(filename string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, filename)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		contents, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		return os.WriteFile(target, contents, info.Mode())
	}))
	return destination
}

func readLocalizedAcceptanceOutput(t *testing.T, destination, relative string) string {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(destination, relative))
	require.NoError(t, err)
	return string(contents)
}
