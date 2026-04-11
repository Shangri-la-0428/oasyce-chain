package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVisibleModuleVersionsIncludesSigilAndUpgradeVisibleModules(t *testing.T) {
	visible := visibleModuleVersions(map[string]uint64{
		"settlement": 1,
		"anchor":     1,
		"delegate":   1,
		"sigil":      2,
		"auth":       99,
	})

	require.Equal(t, uint64(2), visible["sigil"])
	require.Equal(t, uint64(1), visible["anchor"])
	require.Equal(t, uint64(1), visible["delegate"])
	require.Equal(t, uint64(1), visible["settlement"])
	require.NotContains(t, visible, "auth")
}
