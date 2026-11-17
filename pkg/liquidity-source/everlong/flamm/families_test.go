package everlongflamm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Every family the protocol lists for routing states the shape that settles its ladders, and a front's states none:
// this build routes no front, whatever its family says.
func TestDefaultFamiliesStateTheirShape(t *testing.T) {
	var routed int
	for _, f := range defaultHookFamilies {
		if f.Front {
			require.Empty(t, f.Shape, "%s: a front states no shape", f.Name)
			continue
		}
		require.Equal(t, shapeEverlong, f.Shape, f.Name)
		routed++
	}
	require.Positive(t, routed)
	require.Equal(t, defaultHookFamilies, (&Config{}).families(), "an empty list takes the defaults")
}
