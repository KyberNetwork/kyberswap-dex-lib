package ringswap

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestCloneStateIsIndependent checks that UpdateBalance on a clone does not change the original reserves.
func TestCloneStateIsIndependent(t *testing.T) {
	t.Parallel()
	orig := &PoolSimulator{
		fee:          uint256.NewInt(3),
		feePrecision: uint256.NewInt(1000),
	}
	orig.Info.Tokens = []string{"0xaaa", "0xbbb"}
	orig.Info.Reserves = []*big.Int{big.NewInt(1000), big.NewInt(2000)}

	c := orig.CloneState().(*PoolSimulator)
	c.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "0xaaa", Amount: big.NewInt(10)},
		TokenAmountOut: pool.TokenAmount{Token: "0xbbb", Amount: big.NewInt(20)},
	})

	require.Equal(t, big.NewInt(1980).Cmp(c.Info.Reserves[1]), 0, "clone reserve not updated")
	require.Equal(t, big.NewInt(1000).Cmp(orig.Info.Reserves[0]), 0, "original reserve 0 changed")
	require.Equal(t, big.NewInt(2000).Cmp(orig.Info.Reserves[1]), 0, "original reserve 1 changed")
}
