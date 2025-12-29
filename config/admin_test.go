package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromDirectory_AdminYML(t *testing.T) {
	// Create a temporary directory with _admin.yml
	tmpDir, err := os.MkdirTemp("", "gojekyll-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	adminYML := `site:
  base:
    title: Base Title
    description: Base description
    form_action: https://base.example.com/submit
  
  prod:
    envid: prod
  
  stg:
    envid: stg
    form_action: https://stg.example.com/submit
  
  dev:
    envid: dev
    form_action: https://dev.example.com/submit
`
	adminPath := filepath.Join(tmpDir, "_admin.yml")
	err = os.WriteFile(adminPath, []byte(adminYML), 0644)
	require.NoError(t, err)

	// Test dev environment
	c := Default()
	err = c.FromDirectory(tmpDir, "dev")
	require.NoError(t, err)
	require.Contains(t, c.ConfigFile, "_admin.yml (env: dev)")
	devFormAction, ok := c.String("form_action")
	require.True(t, ok)
	require.Equal(t, "https://dev.example.com/submit", devFormAction)
	devEnvID, ok := c.String("envid")
	require.True(t, ok)
	require.Equal(t, "dev", devEnvID)
	// Verify base config is inherited
	title, ok := c.String("title")
	require.True(t, ok)
	require.Equal(t, "Base Title", title)

	// Test stg environment
	c = Default()
	err = c.FromDirectory(tmpDir, "stg")
	require.NoError(t, err)
	stgFormAction, ok := c.String("form_action")
	require.True(t, ok)
	require.Equal(t, "https://stg.example.com/submit", stgFormAction)

	// Test prod environment (only has envid override)
	c = Default()
	err = c.FromDirectory(tmpDir, "prod")
	require.NoError(t, err)
	prodFormAction, ok := c.String("form_action")
	require.True(t, ok)
	require.Equal(t, "https://base.example.com/submit", prodFormAction) // Should inherit from base
	prodEnvID, ok := c.String("envid")
	require.True(t, ok)
	require.Equal(t, "prod", prodEnvID)
}

func TestFromDirectory_AdminYML_NoEnv(t *testing.T) {
	// Create a temporary directory with both _admin.yml and _config.yml
	tmpDir, err := os.MkdirTemp("", "gojekyll-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	adminYML := `site:
  base:
    title: Admin Title
`
	adminPath := filepath.Join(tmpDir, "_admin.yml")
	err = os.WriteFile(adminPath, []byte(adminYML), 0644)
	require.NoError(t, err)

	configYML := `title: Config Title
`
	configPath := filepath.Join(tmpDir, "_config.yml")
	err = os.WriteFile(configPath, []byte(configYML), 0644)
	require.NoError(t, err)

	// When no environment is specified, should use _admin.yml base config if it exists
	c := Default()
	err = c.FromDirectory(tmpDir, "")
	require.NoError(t, err)
	require.Contains(t, c.ConfigFile, "_admin.yml (base)")
	title, ok := c.String("title")
	require.True(t, ok)
	require.Equal(t, "Admin Title", title)
}

func TestFromDirectory_AdminYML_MissingBase(t *testing.T) {
	// Create a temporary directory with _admin.yml but no base section
	tmpDir, err := os.MkdirTemp("", "gojekyll-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	adminYML := `site:
  prod:
    title: Prod Title
`
	adminPath := filepath.Join(tmpDir, "_admin.yml")
	err = os.WriteFile(adminPath, []byte(adminYML), 0644)
	require.NoError(t, err)

	// Should fail if base section is missing
	c := Default()
	err = c.FromDirectory(tmpDir, "prod")
	require.Error(t, err)
	require.Contains(t, err.Error(), "site.base")
}
