package localization

import (
	"fmt"
	"strings"
)

// MessageCatalog resolves interface messages for one active locale. It has no
// interpolation or formatting behavior: callers receive only scalar strings.
type MessageCatalog struct {
	overlays     map[string]map[string]interface{}
	chain        []string
	activeLocale string
	missingMode  string
}

// Translate resolves a dotted message key through the active locale and its
// validated fallback chain. A present non-string value is terminal and fails;
// it never silently falls through to a different locale.
func (c *MessageCatalog) Translate(key string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("message catalog is not configured")
	}
	parts := strings.Split(key, ".")
	if key == "" || hasEmptyPart(parts) {
		return "", fmt.Errorf("message key %q must be a non-empty dotted key", key)
	}

	for _, locale := range c.chain {
		value, found, err := messageValue(c.overlays[locale], parts)
		if err != nil {
			return "", fmt.Errorf("message %q for locale %q: %w", key, locale, err)
		}
		if !found {
			continue
		}
		message, ok := value.(string)
		if !ok {
			return "", fmt.Errorf("message %q for locale %q must resolve to a scalar string, got %s", key, locale, valueKind(value))
		}
		return message, nil
	}

	if c.missingMode == "key" {
		return key, nil
	}
	return "", fmt.Errorf("missing message %q for locale %q", key, c.activeLocale)
}

func messageValue(overlay map[string]interface{}, parts []string) (interface{}, bool, error) {
	messages, exists := overlay["messages"]
	if !exists {
		return nil, false, nil
	}
	current, ok := messages.(map[string]interface{})
	if !ok {
		return nil, false, fmt.Errorf("messages must be a mapping, got %s", valueKind(messages))
	}
	for index, part := range parts {
		value, exists := current[part]
		if !exists {
			return nil, false, nil
		}
		if index == len(parts)-1 {
			return value, true, nil
		}
		next, ok := value.(map[string]interface{})
		if !ok {
			return nil, false, fmt.Errorf("%s must be a mapping, got %s", strings.Join(parts[:index+1], "."), valueKind(value))
		}
		current = next
	}
	return nil, false, nil
}

func hasEmptyPart(parts []string) bool {
	for _, part := range parts {
		if part == "" {
			return true
		}
	}
	return false
}

func valueKind(value interface{}) string {
	if value == nil {
		return "null"
	}
	switch value.(type) {
	case map[string]interface{}:
		return "mapping"
	case []interface{}:
		return "sequence"
	default:
		return fmt.Sprintf("%T", value)
	}
}
