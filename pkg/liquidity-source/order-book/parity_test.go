package orderbook

// Parity of native CalcAmtOut and legacy CalcAmountOut. Remove with the legacy methods and the golden data.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmtOutParity(t *testing.T) {
	t.Parallel()
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entityPool})
	require.NoError(t, err)
	testutil.TestCalcAmtOutParity(t, sim, []string{"1", "1000", "100000", "500000000", "3000000000",
		"9000000000", "100000000000", "400000000000"}, nil)
}
