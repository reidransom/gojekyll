package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/osteele/gojekyll/utils"
	yaml "gopkg.in/yaml.v2"
)

// Config is the Jekyll site configuration, typically read from _config.yml.
// See https://jekyllrb.com/docs/configuration/#default-configuration
type Config struct {
	// Where things are:
	Source      string
	Destination string
	LayoutsDir  string                            `yaml:"layouts_dir"`
	DataDir     string                            `yaml:"data_dir"`
	IncludesDir string                            `yaml:"includes_dir"`
	Collections map[string]map[string]interface{} `yaml:"-"`
	Theme       string

	// Handling Reading
	Include     []string
	Exclude     []string
	KeepFiles   []string `yaml:"keep_files"`
	MarkdownExt string   `yaml:"markdown_ext"`

	// Filtering Content
	Drafts      bool `yaml:"show_drafts"`
	Future      bool
	Unpublished bool

	// Plugins
	Plugins []string

	// Conversion
	ExcerptSeparator string `yaml:"excerpt_separator"`
	Incremental      bool
	Sass             struct {
		Dir string `yaml:"sass_dir"`
		// TODO Style string // compressed
	}

	// Serving
	Host        string
	Port        int
	AbsoluteURL string `yaml:"url"`
	BaseURL     string

	// Outputting
	Permalink string
	Timezone  string
	Verbose   bool
	Defaults  []struct {
		Scope struct {
			Path string
			Type string
		}
		Values map[string]interface{}
	}

	// CLI-only
	DryRun       bool `yaml:"-"`
	ForcePolling bool `yaml:"-"`
	Watch        bool `yaml:"-"`

	// Meta
	ConfigFile string                 `yaml:"-"`
	m          map[string]interface{} `yaml:"-"` // config file, as map
	ms         yaml.MapSlice          `yaml:"-"` // config file, as MapSlice

	// Plugins
	RequireFrontMatter        bool            `yaml:"-"`
	RequireFrontMatterExclude map[string]bool `yaml:"-"`
}

// FromDirectory updates the config from the config file in
// the directory, if such a file exists.
func (c *Config) FromDirectory(dir string, environment string, adminFile string, configFiles string) error {
	// If explicit config files are specified, use those and skip admin file logic
	if configFiles != "" {
		return c.loadConfigFiles(dir, configFiles)
	}
	// Determine admin file path
	var adminPath string
	explicitAdminFile := adminFile != ""
	
	if explicitAdminFile {
		// Use explicit admin file path (can be absolute or relative to dir)
		if filepath.IsAbs(adminFile) {
			adminPath = adminFile
		} else {
			adminPath = filepath.Join(dir, adminFile)
		}
	} else {
		// Default to _admin.yml in the directory
		adminPath = filepath.Join(dir, "_admin.yml")
	}
	
	// Try to read admin file
	if bytes, err := os.ReadFile(adminPath); err == nil {
			// Parse _admin.yml structure
			var adminConfig struct {
				Site map[string]interface{} `yaml:"site"`
			}
			if err := yaml.Unmarshal(bytes, &adminConfig); err != nil {
				return utils.WrapPathError(err, adminPath)
			}

			// Get base and environment-specific config
			baseConfig, hasBase := adminConfig.Site["base"].(map[interface{}]interface{})

			if !hasBase {
				// If admin file was explicitly specified, require site.base
				if explicitAdminFile {
					return utils.WrapPathError(
						os.ErrInvalid,
						adminPath+": must have a site.base section",
					)
				}
				// Otherwise, fall back to _config.yml if site.base is missing
				// Continue to the _config.yml section below
			} else {
				// Merge base with environment overrides if environment specified
				mergedConfig := baseConfig
				if environment != "" {
					envConfig, hasEnv := adminConfig.Site[environment].(map[interface{}]interface{})
					if hasEnv {
						mergedConfig = mergeYAMLMaps(baseConfig, envConfig)
					}
				}

				// Convert merged config to YAML bytes and unmarshal into Config
				mergedBytes, err := yaml.Marshal(mergedConfig)
				if err != nil {
					return err
				}

				// Save the existing c.ms (from Default()) so we can preserve default values
				existingMS := c.ms
				existingM := c.m

				if err = Unmarshal(mergedBytes, c); err != nil {
					return utils.WrapPathError(err, adminPath)
				}
				
				// Merge the admin config into existing maps instead of replacing them
				// First update c.m with values from existing that aren't in the new config
				for k, v := range existingM {
					if _, exists := c.m[k]; !exists {
						c.m[k] = v
					}
				}
				
				// Then update c.ms - keep existing items that aren't overridden, append new ones
				adminKeys := make(map[interface{}]bool)
				for _, item := range c.ms {
					adminKeys[item.Key] = true
				}
				
				// Build new c.ms: existing items updated with admin values, plus new admin items
				newMS := yaml.MapSlice{}
				for _, item := range existingMS {
					if adminKeys[item.Key] {
						// This key is in admin config, use the admin value
						for _, adminItem := range c.ms {
							if adminItem.Key == item.Key {
								newMS = append(newMS, adminItem)
								break
							}
						}
					} else {
						// This key is not in admin config, keep the default
						newMS = append(newMS, item)
					}
				}
				// Add any new keys from admin that weren't in defaults
				for _, adminItem := range c.ms {
					found := false
					for _, item := range existingMS {
						if item.Key == adminItem.Key {
							found = true
							break
						}
					}
					if !found {
						newMS = append(newMS, adminItem)
					}
				}
				c.ms = newMS
				if environment != "" {
					c.ConfigFile = adminPath + " (env: " + environment + ")"
				} else {
					c.ConfigFile = adminPath + " (base)"
				}
				c.Source = dir
				
				// Override URL from JEKYLL_URL environment variable if set
				if jekyllURL := os.Getenv("JEKYLL_URL"); jekyllURL != "" {
					c.AbsoluteURL = jekyllURL
					c.Set("url", jekyllURL)
				}
				
				return nil
			}
	} else if explicitAdminFile {
		// If admin file was explicitly specified but not found, return error
		return utils.WrapPathError(err, adminPath)
	}

	// Fall back to _config.yml (only if admin file was not explicitly specified)
	path := filepath.Join(dir, "_config.yml")
	bytes, err := os.ReadFile(path)
	switch {
	case os.IsNotExist(err):
		// break
	case err != nil:
		return err
	default:
		if err = Unmarshal(bytes, c); err != nil {
			return utils.WrapPathError(err, path)
		}
		c.ConfigFile = path
	}
	c.Source = dir
	
	// Override URL from JEKYLL_URL environment variable if set
	if jekyllURL := os.Getenv("JEKYLL_URL"); jekyllURL != "" {
		c.AbsoluteURL = jekyllURL
		c.Set("url", jekyllURL)
	}
	
	return nil
}

// loadConfigFiles loads one or more config files separated by commas.
// Later files override earlier ones.
func (c *Config) loadConfigFiles(dir string, configFiles string) error {
	// Split by comma and trim whitespace
	files := strings.Split(configFiles, ",")
	for i, f := range files {
		files[i] = strings.TrimSpace(f)
	}
	
	if len(files) == 0 {
		return nil
	}
	
	// Track config file names for display
	configFileNames := []string{}
	
	// Merged YAML data
	var mergedData map[string]interface{}
	
	// Load and merge config files in order
	for _, configFile := range files {
		if configFile == "" {
			continue
		}
		
		// Determine full path
		var configPath string
		if filepath.IsAbs(configFile) {
			configPath = configFile
		} else {
			configPath = filepath.Join(dir, configFile)
		}
		
		// Read config file
		bytes, err := os.ReadFile(configPath)
		if err != nil {
			return utils.WrapPathError(err, configPath)
		}
		
		// Parse YAML into a map
		var fileData map[string]interface{}
		if err := yaml.Unmarshal(bytes, &fileData); err != nil {
			return utils.WrapPathError(err, configPath)
		}
		
		// Merge into accumulated data
		if mergedData == nil {
			mergedData = fileData
		} else {
			// Override with values from this file
			for k, v := range fileData {
				mergedData[k] = v
			}
		}
		
		configFileNames = append(configFileNames, configPath)
	}
	
	// Convert merged data back to YAML and unmarshal into config
	mergedBytes, err := yaml.Marshal(mergedData)
	if err != nil {
		return err
	}
	
	if err = Unmarshal(mergedBytes, c); err != nil {
		return err
	}
	
	// Set the config file display string
	if len(configFileNames) == 1 {
		c.ConfigFile = configFileNames[0]
	} else {
		c.ConfigFile = strings.Join(configFileNames, ", ")
	}
	c.Source = dir
	
	// Override URL from JEKYLL_URL environment variable if set
	if jekyllURL := os.Getenv("JEKYLL_URL"); jekyllURL != "" {
		c.AbsoluteURL = jekyllURL
		c.Set("url", jekyllURL)
	}
	
	return nil
}

type configCompat struct {
	Gems []string
}

type collectionsList struct {
	Collections []string
}

type collectionsMap struct {
	Collections map[string]map[string]interface{}
}

// IsConfigPath returns true if its arguments is a site configuration file.
func (c *Config) IsConfigPath(rel string) bool {
	return rel == "_config.yml"
}

// SassDir returns the relative path of the SASS directory.
func (c *Config) SassDir() string {
	return "_sass"
}

// SourceDir returns the source directory as an absolute path.
func (c *Config) SourceDir() string {
	return utils.MustAbs(c.Source)
}

// GetFrontMatterDefaults implements https://jekyllrb.com/docs/configuration/#front-matter-defaults
func (c *Config) GetFrontMatterDefaults(typename, rel string) (m map[string]interface{}) {
	for _, entry := range c.Defaults {
		scope := &entry.Scope
		hasPrefix := strings.HasPrefix(rel, scope.Path)
		hasType := scope.Type == "" || scope.Type == typename
		if hasPrefix && hasType {
			m = utils.MergeStringMaps(m, entry.Values)
		}
	}
	return
}

// RequiresFrontMatter returns a bool indicating whether the file requires front matter in order to recognize as a page.
func (c *Config) RequiresFrontMatter(rel string) bool {
	switch {
	case c.RequireFrontMatter:
		return true
	case !c.IsMarkdown(rel):
		return true
	case utils.StringArrayContains(c.Include, rel):
		return false
	case c.RequireFrontMatterExclude[strings.ToUpper(utils.TrimExt(filepath.Base(rel)))]:
		return true
	default:
		return false
	}
}

// Unmarshal updates site from a YAML configuration file.
func Unmarshal(bytes []byte, c *Config) error {
	var (
		compat configCompat
		cList  collectionsList
	)
	if err := yaml.Unmarshal(bytes, &c); err != nil {
		return err
	}
	if err := yaml.Unmarshal(bytes, &c.ms); err != nil {
		return err
	}
	if err := yaml.Unmarshal(bytes, &c.m); err != nil {
		return err
	}
	if err := yaml.Unmarshal(bytes, &cList); err == nil {
		if len(c.Collections) == 0 {
			c.Collections = make(map[string]map[string]interface{})
		}
		for _, name := range cList.Collections {
			c.Collections[name] = map[string]interface{}{}
		}
	}
	cMap := collectionsMap{c.Collections}
	if err := yaml.Unmarshal(bytes, &cMap); err == nil {
		c.Collections = cMap.Collections
	}
	if err := yaml.Unmarshal(bytes, &compat); err != nil {
		return err
	}
	if len(c.Plugins) == 0 {
		c.Plugins = compat.Gems
	}
	return nil
}

// Variables returns the configuration as a Liquid variable map.
func (c *Config) Variables() map[string]interface{} {
	m := map[string]interface{}{}
	for _, item := range c.ms {
		if s, ok := item.Key.(string); ok {
			m[s] = item.Value
		}
	}
	return m
}

// Set sets a value in the Liquid variable map.
// This does not update the corresponding value in the Config struct.
func (c *Config) Set(key string, val interface{}) {
	c.m[key] = val
	for i := range c.ms {
		if c.ms[i].Key == key {
			c.ms[i].Value = val
			return
		}
	}
	c.ms = append(c.ms, yaml.MapItem{Key: key, Value: val})
}

// Map returns the config indexed by key, if it's a map.
func (c *Config) Map(key string) (map[string]interface{}, bool) {
	if m, ok := c.m[key]; ok {
		if m, ok := m.(map[string]interface{}); ok {
			return m, ok
		}
	}
	return nil, false
}

// String returns the config indexed by key, if it's a string.
func (c *Config) String(key string) (string, bool) {
	if m, ok := c.m[key]; ok {
		if m, ok := m.(string); ok {
			return m, ok
		}
	}
	return "", false
}

// mergeYAMLMaps performs a deep merge of two YAML maps.
// Values from override will overwrite values in base.
func mergeYAMLMaps(base, override map[interface{}]interface{}) map[interface{}]interface{} {
	result := make(map[interface{}]interface{})

	// Copy all values from base
	for k, v := range base {
		result[k] = v
	}

	// Merge/override with values from override
	for k, v := range override {
		if vMap, ok := v.(map[interface{}]interface{}); ok {
			// If both base and override have a map at this key, merge them recursively
			if baseMap, ok := result[k].(map[interface{}]interface{}); ok {
				result[k] = mergeYAMLMaps(baseMap, vMap)
				continue
			}
		}
		// Otherwise, just overwrite
		result[k] = v
	}

	return result
}
