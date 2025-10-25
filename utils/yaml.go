package utils

import yaml "gopkg.in/yaml.v2"

// UnmarshalYAMLInterface is a wrapper for yaml.Unmarshal that
// knows how to unmarshal maps and lists.
func UnmarshalYAMLInterface(b []byte, i *interface{}) error {
	var m yaml.MapSlice
	err := yaml.Unmarshal(b, &m)
	switch err.(type) {
	case *yaml.TypeError:
		// Work around https://github.com/go-yaml/yaml/issues/20
		var s []interface{}
		err = yaml.Unmarshal(b, &s)
		if err != nil {
			return err
		}
		*i = convertYAMLValue(s)
	default:
		if err != nil {
			return err
		}
		// Check if this is actually an array that was incorrectly parsed as MapSlice
		// (happens when YAML array is unmarshaled into MapSlice - it creates nil keys)
		if len(m) > 0 && m[0].Key == nil {
			var s []interface{}
			err = yaml.Unmarshal(b, &s)
			if err != nil {
				return err
			}
			*i = convertYAMLValue(s)
		} else {
			*i = convertYAMLValue(m)
		}
	}
	return nil
}

// convertYAMLValue recursively converts yaml.MapSlice and map[interface{}]interface{}
// to map[string]interface{} and processes nested structures
func convertYAMLValue(v interface{}) interface{} {
	switch v := v.(type) {
	case yaml.MapSlice:
		m := make(map[string]interface{}, len(v))
		for _, item := range v {
			if key, ok := item.Key.(string); ok {
				m[key] = convertYAMLValue(item.Value)
			}
		}
		return m
	case map[interface{}]interface{}:
		m := make(map[string]interface{}, len(v))
		for key, value := range v {
			if keyStr, ok := key.(string); ok {
				m[keyStr] = convertYAMLValue(value)
			}
		}
		return m
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = convertYAMLValue(item)
		}
		return result
	default:
		return v
	}
}
