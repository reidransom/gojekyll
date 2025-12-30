package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromDirectory_JEKYLL_URL_Override_ConfigYML(t *testing.T) {
	// Create a temporary directory with _config.yml
	tmpDir, err := os.MkdirTemp("", "gojekyll-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configYML := `url: https://example.com
title: Test Site
`
	configPath := filepath.Join(tmpDir, "_config.yml")
	err = os.WriteFile(configPath, []byte(configYML), 0644)
	require.NoError(t, err)

	// Set JEKYLL_URL environment variable
	origURL := os.Getenv("JEKYLL_URL")
	defer func() { _ = os.Setenv("JEKYLL_URL", origURL) }()
	err = os.Setenv("JEKYLL_URL", "https://override.com")
	require.NoError(t, err)

	// Load config
	c := Default()
	err = c.FromDirectory(tmpDir, "")
	require.NoError(t, err)

	// Check that AbsoluteURL field is overridden
	require.Equal(t, "https://override.com", c.AbsoluteURL)

	// Check that the Variables() map also reflects the override
	// This is what templates actually use
	vars := c.Variables()
	urlValue, ok := vars["url"]
	require.True(t, ok, "url should be present in variables")
	require.Equal(t, "https://override.com", urlValue, "url in variables should be overridden")
}

func TestFromDirectory_JEKYLL_URL_Override_AdminYML(t *testing.T) {
	// Create a temporary directory with _admin.yml
	tmpDir, err := os.MkdirTemp("", "gojekyll-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	adminYML := `site:
  base:
    url: https://base.example.com
    title: Base Title
  
  dev:
    envid: dev
`
	adminPath := filepath.Join(tmpDir, "_admin.yml")
	err = os.WriteFile(adminPath, []byte(adminYML), 0644)
	require.NoError(t, err)

	// Set JEKYLL_URL environment variable
	origURL := os.Getenv("JEKYLL_URL")
	defer func() { _ = os.Setenv("JEKYLL_URL", origURL) }()
	err = os.Setenv("JEKYLL_URL", "https://override.com")
	require.NoError(t, err)

	// Load config with dev environment
	c := Default()
	err = c.FromDirectory(tmpDir, "dev")
	require.NoError(t, err)

	// Check that AbsoluteURL field is overridden
	require.Equal(t, "https://override.com", c.AbsoluteURL)

	// Check that the Variables() map also reflects the override
	// This is what templates actually use
	vars := c.Variables()
	urlValue, ok := vars["url"]
	require.True(t, ok, "url should be present in variables")
	require.Equal(t, "https://override.com", urlValue, "url in variables should be overridden")
}

func TestFromDirectory_JEKYLL_URL_NotSet(t *testing.T) {
	// Create a temporary directory with _config.yml
	tmpDir, err := os.MkdirTemp("", "gojekyll-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configYML := `url: https://example.com
title: Test Site
`
	configPath := filepath.Join(tmpDir, "_config.yml")
	err = os.WriteFile(configPath, []byte(configYML), 0644)
	require.NoError(t, err)

	// Ensure JEKYLL_URL is not set
	origURL := os.Getenv("JEKYLL_URL")
	defer func() { _ = os.Setenv("JEKYLL_URL", origURL) }()
	err = os.Unsetenv("JEKYLL_URL")
	require.NoError(t, err)

	// Load config
	c := Default()
	err = c.FromDirectory(tmpDir, "")
	require.NoError(t, err)

	// Check that AbsoluteURL field uses config value
	require.Equal(t, "https://example.com", c.AbsoluteURL)

	// Check that the Variables() map also has the original value
	vars := c.Variables()
	urlValue, ok := vars["url"]
	require.True(t, ok, "url should be present in variables")
	require.Equal(t, "https://example.com", urlValue)
}

func TestFromDirectory_AdminYML_PreservesDefaultConfig(t *testing.T) {
	// This test catches the bug where _admin.yml loading would overwrite
	// c.ms with only admin config values, losing all default config values
	tmpDir, err := os.MkdirTemp("", "gojekyll-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	adminYML := `site:
  base:
    url: https://example.com
    title: My Site
`
	adminPath := filepath.Join(tmpDir, "_admin.yml")
	err = os.WriteFile(adminPath, []byte(adminYML), 0644)
	require.NoError(t, err)

	// Load config
	c := Default()
	err = c.FromDirectory(tmpDir, "")
	require.NoError(t, err)

	vars := c.Variables()
	
	// Check that admin config values are present
	urlValue, ok := vars["url"]
	require.True(t, ok, "url from admin config should be present")
	require.Equal(t, "https://example.com", urlValue)
	
	titleValue, ok := vars["title"]
	require.True(t, ok, "title from admin config should be present")
	require.Equal(t, "My Site", titleValue)
	
	// Check that default config values are ALSO present (the bug was losing these)
	source, ok := vars["source"]
	require.True(t, ok, "source from default config should be present")
	require.Equal(t, ".", source)
	
	destination, ok := vars["destination"]
	require.True(t, ok, "destination from default config should be present")
	require.Equal(t, "./_site", destination)
	
	port, ok := vars["port"]
	require.True(t, ok, "port from default config should be present")
	require.Equal(t, 4000, port)
	
	baseurl, ok := vars["baseurl"]
	require.True(t, ok, "baseurl from default config should be present")
	require.Equal(t, "", baseurl)
	
	// Verify that Variables() returns a reasonable number of keys
	// Should be close to the default count (~24) plus admin overrides
	require.GreaterOrEqual(t, len(vars), 20, "Variables() should contain default config values, not just admin config")
}
