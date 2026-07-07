package pools

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApproximateExtraDistinctTimeBitmapLookupsWordBoundaries(t *testing.T) {
	t.Parallel()

	base := uint64(0)
	sameWord := (uint64(1) << 16) - 1
	nextWord := uint64(1) << 16

	require.Equal(t, int64(0), approximateExtraDistinctTimeBitmapLookups(base, sameWord))
	require.Equal(t, int64(1), approximateExtraDistinctTimeBitmapLookups(base, nextWord))
}
