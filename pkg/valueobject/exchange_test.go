package valueobject

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExchangeAmmalgamDLEXIsNotPropAMM(t *testing.T) {
	require.Equal(t, "ammalgam-dlex", ExchangeAmmalgamDLEX)
	require.False(t, IsPropAMMSource(ExchangeAmmalgamDLEX))
}

// clear-rfq is not a pmm-N slot, so the prefix rule does not cover it: it needs its RFQSourceSet entry.
func TestExchangeClearRFQIsRFQSource(t *testing.T) {
	require.Equal(t, "clear-rfq", ExchangeClearRFQ)
	require.True(t, IsRFQSource(ExchangeClearRFQ))
	require.True(t, NeedsFallbackSource(ExchangeClearRFQ))
	require.False(t, IsRFQSource(ExchangeClear), "on-chain clear must not become an RFQ source")
}
