package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const mapYaml = "a: 1\nb: 2"
const listYaml = "- a\n- b"

func TestUnmarshalYAML(t *testing.T) {
	var d interface{}
	err := UnmarshalYAMLInterface([]byte(mapYaml), &d)
	require.NoError(t, err)
	switch d := d.(type) {
	case map[string]interface{}:
		require.Len(t, d, 2)
		require.Equal(t, 1, d["a"])
		require.Equal(t, 2, d["b"])
	default:
		require.IsType(t, map[string]interface{}{}, d)
	}

	err = UnmarshalYAMLInterface([]byte(listYaml), &d)
	require.NoError(t, err)
	require.IsType(t, d, []interface{}{})
	switch d := d.(type) {
	case []interface{}:
		require.Len(t, d, 2)
		require.Equal(t, "a", d[0])
		require.Equal(t, "b", d[1])
	default:
		require.IsType(t, d, map[string]interface{}{})
	}
}
