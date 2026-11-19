package math

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpread_Add(t *testing.T) {
	sum, err := Spread(283).Add(Spread(17))
	require.NoError(t, err)
	require.Equal(t, Spread(300), sum)

	sum, err = Spread(^uint32(0) - 1).Add(Spread(1))
	require.NoError(t, err)
	require.Equal(t, Spread(^uint32(0)), sum)

	_, err = Spread(^uint32(0)).Add(Spread(1))
	require.ErrorIs(t, err, ErrOverflow)
}
