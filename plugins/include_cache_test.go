package plugins

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIncludeCachePluginRegistration(t *testing.T) {
	registered, ok := Lookup("jekyll-include-cache")
	require.True(t, ok)
	require.IsType(t, plugin{}, registered)
	require.Empty(t, registered.ModifyPluginList(nil))
}
