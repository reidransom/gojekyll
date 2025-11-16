package plugins

import (
	"time"

	yaml "gopkg.in/yaml.v2"
)

func init() {
	register("jekyll-inherit-frontmatter", inheritFrontmatterPlugin{})
}

type inheritFrontmatterPlugin struct{ plugin }

// PostReadSite implements the Plugin interface.
// It runs after all pages have been read and copies inheritable front matter
// from base language posts to translated posts.
func (p inheritFrontmatterPlugin) PostReadSite(s Site) error {
	cfg := s.Config()
	
	// Get default language from config (default to "en" if not specified)
	defaultLang := "en"
	if dl, ok := cfg.String("default_lang"); ok {
		defaultLang = dl
	}
	
	// Get configuration for which fields to inherit
	// By default, inherit all fields except "lang" and "title"
	excludeFields := map[string]bool{
		"lang":  true,
		"title": true,
	}
	
	var includeFields []string // If specified, only inherit these fields
	inheritAll := true         // By default, inherit all (except excluded)
	
	// Get config from Variables() which properly handles nested structures
	vars := cfg.Variables()
	if inheritableCfgRaw, ok := vars["inherit_frontmatter"]; ok {
		if inheritableCfg, ok := inheritableCfgRaw.(yaml.MapSlice); ok {
			for _, item := range inheritableCfg {
				switch item.Key {
				case "fields":
					if fields, ok := item.Value.([]interface{}); ok {
						inheritAll = false
						includeFields = make([]string, 0, len(fields))
						for _, field := range fields {
							if fieldStr, ok := field.(string); ok {
								includeFields = append(includeFields, fieldStr)
							}
						}
					}
				case "exclude":
					if fields, ok := item.Value.([]interface{}); ok {
						excludeFields = make(map[string]bool)
						for _, field := range fields {
							if fieldStr, ok := field.(string); ok {
								excludeFields[fieldStr] = true
							}
						}
					}
				}
			}
		}
	}
	
	// Get all posts
	posts := s.Posts()
	if posts == nil {
		return nil
	}
	
	// Build a map of base language posts indexed by date
	basePosts := make(map[string]Page)
	for _, post := range posts {
		fm := post.FrontMatter()
		lang := fm.String("lang", "")
		
		// Only process posts with explicit language markers
		if lang == "" {
			continue
		}
		
		if lang == defaultLang {
			// Store base language posts by date
			date := post.PostDate()
			key := date.Format("2006-01-02")
			basePosts[key] = post
		}
	}
	
	// Process non-default language posts
	for _, post := range posts {
		fm := post.FrontMatter()
		lang := fm.String("lang", "")
		
		// Skip posts without language or base language posts
		if lang == "" || lang == defaultLang {
			continue
		}
		
		// Find the base language counterpart by date
		date := post.PostDate()
		key := date.Format("2006-01-02")
		basePost, found := basePosts[key]
		if !found {
			continue
		}
		
		baseFM := basePost.FrontMatter()
		
		// Copy fields from base post to translated post
		if inheritAll {
			// Inherit all fields except excluded ones
			for field, value := range baseFM {
				// Skip excluded fields
				if excludeFields[field] {
					continue
				}
				
				// Skip if the translated post already has this field (allow overrides)
				if _, exists := fm[field]; exists {
					continue
				}
				
				// Copy the field
				fm[field] = copyValue(value)
			}
		} else {
			// Only inherit specified fields
			for _, field := range includeFields {
				// Skip if the translated post already has this field (allow overrides)
				if _, exists := fm[field]; exists {
					continue
				}
				
				// Copy the field from base post if it exists
				if value, exists := baseFM[field]; exists {
					fm[field] = copyValue(value)
				}
			}
		}
	}
	
	return nil
}

// copyValue creates a copy of a value to avoid sharing references
func copyValue(value interface{}) interface{} {
	switch v := value.(type) {
	case []interface{}:
		// Deep copy for slices
		copied := make([]interface{}, len(v))
		copy(copied, v)
		return copied
	case []string:
		// Deep copy for string slices
		copied := make([]string, len(v))
		copy(copied, v)
		return copied
	case map[string]interface{}:
		// Deep copy for maps
		copied := make(map[string]interface{}, len(v))
		for k, val := range v {
			copied[k] = copyValue(val)
		}
		return copied
	case time.Time:
		// Time is immutable but for consistency
		return v
	default:
		// For basic types (string, int, bool, etc.), return as-is
		return v
	}
}
