package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJEKYLL_URL_OverrideInRenderedOutput(t *testing.T) {
	// Create a temporary test site with _config.yml
	tmpDir, err := os.MkdirTemp("", "gojekyll-integration-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create _config.yml with a url
	configYML := `title: Test Site
url: https://example.com
`
	err = os.WriteFile(filepath.Join(tmpDir, "_config.yml"), []byte(configYML), 0644)
	require.NoError(t, err)

	// Create a simple page that references site.url
	pageContent := `---
title: Test Page
---
site.url is: {{ site.url }}
`
	err = os.WriteFile(filepath.Join(tmpDir, "index.md"), []byte(pageContent), 0644)
	require.NoError(t, err)

	// Set JEKYLL_URL environment variable to override the config
	origURL := os.Getenv("JEKYLL_URL")
	defer func() { _ = os.Setenv("JEKYLL_URL", origURL) }()
	err = os.Setenv("JEKYLL_URL", "https://override.example.com")
	require.NoError(t, err)

	// Build the site
	err = ParseAndRun([]string{"build", "-s", tmpDir, "-q"})
	require.NoError(t, err)

	// Read the rendered output
	outputPath := filepath.Join(tmpDir, "_site", "index.html")
	output, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	outputStr := string(output)
	
	// Verify that the output contains the overridden URL, not the config URL
	require.Contains(t, outputStr, "https://override.example.com", 
		"Rendered output should contain the JEKYLL_URL override value")
	require.NotContains(t, outputStr, "https://example.com", 
		"Rendered output should NOT contain the original config url value")
}

func TestJEKYLL_URL_NotSet_UsesConfigValue(t *testing.T) {
	// Create a temporary test site with _config.yml
	tmpDir, err := os.MkdirTemp("", "gojekyll-integration-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create _config.yml with a url
	configYML := `title: Test Site
url: https://config.example.com
`
	err = os.WriteFile(filepath.Join(tmpDir, "_config.yml"), []byte(configYML), 0644)
	require.NoError(t, err)

	// Create a simple page that references site.url
	pageContent := `---
title: Test Page
---
site.url is: {{ site.url }}
`
	err = os.WriteFile(filepath.Join(tmpDir, "index.md"), []byte(pageContent), 0644)
	require.NoError(t, err)

	// Make sure JEKYLL_URL is NOT set
	origURL := os.Getenv("JEKYLL_URL")
	defer func() { _ = os.Setenv("JEKYLL_URL", origURL) }()
	err = os.Unsetenv("JEKYLL_URL")
	require.NoError(t, err)

	// Build the site
	err = ParseAndRun([]string{"build", "-s", tmpDir, "-q"})
	require.NoError(t, err)

	// Read the rendered output
	outputPath := filepath.Join(tmpDir, "_site", "index.html")
	output, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	outputStr := string(output)
	
	// Verify that the output contains the config URL
	require.Contains(t, outputStr, "https://config.example.com", 
		"Rendered output should contain the config url value when JEKYLL_URL is not set")
}

func TestJEKYLL_URL_WithAdminYML(t *testing.T) {
	// Create a temporary test site with _admin.yml
	tmpDir, err := os.MkdirTemp("", "gojekyll-integration-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create _admin.yml with base and dev configs
	adminYML := `site:
  base:
    title: Test Site
    url: https://base.example.com
  
  dev:
    envid: dev
`
	err = os.WriteFile(filepath.Join(tmpDir, "_admin.yml"), []byte(adminYML), 0644)
	require.NoError(t, err)

	// Create a simple page that references site.url
	pageContent := `---
title: Test Page
---
site.url is: {{ site.url }}
envid is: {{ site.envid }}
`
	err = os.WriteFile(filepath.Join(tmpDir, "index.md"), []byte(pageContent), 0644)
	require.NoError(t, err)

	// Set JEKYLL_URL environment variable to override the admin config
	origURL := os.Getenv("JEKYLL_URL")
	defer func() { _ = os.Setenv("JEKYLL_URL", origURL) }()
	err = os.Setenv("JEKYLL_URL", "https://local.example.com")
	require.NoError(t, err)

	// Build the site with dev environment
	err = ParseAndRun([]string{"build", "-s", tmpDir, "-q", "--env=dev"})
	require.NoError(t, err)

	// Read the rendered output
	outputPath := filepath.Join(tmpDir, "_site", "index.html")
	output, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	outputStr := string(output)
	
	// Verify that the output contains the overridden URL
	require.Contains(t, outputStr, "https://local.example.com", 
		"Rendered output should contain the JEKYLL_URL override value")
	require.NotContains(t, outputStr, "https://base.example.com", 
		"Rendered output should NOT contain the original admin base url value")
	
	// Verify that the environment-specific config is still applied
	require.True(t, strings.Contains(outputStr, "envid is: dev") || strings.Contains(outputStr, "envid is:\ndev"),
		"Rendered output should contain the dev environment config")
}
