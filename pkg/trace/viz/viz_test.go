package viz

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlot(t *testing.T) {
	err := PlotBlockSizeOverTime(
		"../../data/trace",
		"t-control-2")
	require.NoError(t, err)
}
