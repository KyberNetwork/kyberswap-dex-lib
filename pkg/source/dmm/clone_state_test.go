package dmm

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestCloneStateIsIndependent checks that UpdateBalance on a clone changes neither the original's reserves
// nor its virtual reserves.
func TestCloneStateIsIndependent(t *testing.T) {
	t.Parallel()
	orig := &PoolSimulator{VReserves: []*big.Int{big.NewInt(500), big.NewInt(600)}}
	orig.Info = pool.PoolInfo{Tokens: []string{"0xaaa", "0xbbb"},
		Reserves: []*big.Int{big.NewInt(1000), big.NewInt(2000)}, SwapFee: big.NewInt(0)}

	c := orig.CloneState().(*PoolSimulator)
	c.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "0xaaa", Amount: big.NewInt(10)},
		TokenAmountOut: pool.TokenAmount{Token: "0xbbb", Amount: big.NewInt(20)},
	})

	require.Equal(t, big.NewInt(1000).Cmp(orig.Info.Reserves[0]), 0, "original reserve 0 changed")
	require.Equal(t, big.NewInt(2000).Cmp(orig.Info.Reserves[1]), 0, "original reserve 1 changed")
	require.Equal(t, big.NewInt(500).Cmp(orig.VReserves[0]), 0, "original vreserve 0 changed")
	require.Equal(t, big.NewInt(600).Cmp(orig.VReserves[1]), 0, "original vreserve 1 changed")
	require.Equal(t, big.NewInt(510).Cmp(c.VReserves[0]), 0, "clone vreserve 0 not updated")
}
