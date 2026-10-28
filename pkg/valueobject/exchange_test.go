package valueobject

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExchangeAmmalgamDLEXIsNotPropAMM(t *testing.T) {
	require.Equal(t, "ammalgam-dlex", ExchangeAmmalgamDLEX)
	require.False(t, IsPropAMMSource(ExchangeAmmalgamDLEX))
}
