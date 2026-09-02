package renderers

import (
	"fmt"
	"testing"

	"github.com/reidransom/jigyll/config"
	"github.com/reidransom/jigyll/filters"
	"github.com/stretchr/testify/require"
)

type rendererLocalizationFake struct{}

func (rendererLocalizationFake) ActiveLocale() string { return "en" }
func (rendererLocalizationFake) Translation(interface{}, string) (interface{}, error) {
	return nil, nil
}
func (rendererLocalizationFake) LocalizedPageURL(interface{}, string) (string, error) {
	return "", fmt.Errorf("not used")
}
func (rendererLocalizationFake) LocalizedRouteURL(string, string) (string, error) {
	return "", fmt.Errorf("not used")
}
func (rendererLocalizationFake) IsSharedAsset(string) bool { return false }
func (rendererLocalizationFake) Translate(key string) (string, error) {
	if key != "nav.home" {
		return "", fmt.Errorf("missing message %q", key)
	}
	return "Home", nil
}

var _ filters.LocalizationContext = rendererLocalizationFake{}

func TestRendererRegistersLocalizedFiltersFromOptions(t *testing.T) {
	manager := Manager{
		cfg: config.Default(),
		Options: Options{
			Localization: rendererLocalizationFake{},
		},
	}
	output, err := manager.makeLiquidEngine().ParseAndRender([]byte(`{{ "nav.home" | translate }}`), nil)
	require.NoError(t, err)
	require.Equal(t, "Home", string(output))
}
