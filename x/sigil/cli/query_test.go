package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetQueryCmd_RegistersPulses(t *testing.T) {
	cmd := GetQueryCmd()

	sub, _, err := cmd.Find([]string{"pulses"})
	require.NoError(t, err)
	require.NotNil(t, sub)
	require.Equal(t, "pulses", sub.Name())
	require.Equal(t, "pulses [sigil-id]", sub.Use)
}
