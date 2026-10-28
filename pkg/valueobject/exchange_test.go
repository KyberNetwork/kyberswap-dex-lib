package valueobject

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExchangeAmmalgamIsNotPropAMM(t *testing.T) {
	require.Equal(t, "ammalgam-dlex", ExchangeAmmalgam)
	require.False(t, IsPropAMMSource(ExchangeAmmalgam))
}
