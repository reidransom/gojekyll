package plugins

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIncludeCachePluginRegistration(t *testing.T) {
	factory, ok := Lookup("jekyll-include-cache")
	require.True(t, ok)
	registered := factory()
	require.IsType(t, plugin{}, registered)
	require.Empty(t, registered.ModifyPluginList(nil))
}
