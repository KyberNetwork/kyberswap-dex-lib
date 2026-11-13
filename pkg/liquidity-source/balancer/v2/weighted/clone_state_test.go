package weighted

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v2/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestCloneStateIsIndependent checks that UpdateBalance on a clone changes neither the original's reserves
// nor the totals. Both are slices UpdateBalance writes by index.
func TestCloneStateIsIndependent(t *testing.T) {
	t.Parallel()
	orig := &PoolSimulator{
		totalAmountsIn: []*uint256.Int{uint256.NewInt(10), uint256.NewInt(20)},
		basePools:      map[string]shared.IBasePool{},
	}
	orig.Info = pool.PoolInfo{Address: "parent", Tokens: []string{"0xaaa", "0xbbb"},
		Reserves: []*big.Int{big.NewInt(1000), big.NewInt(2000)}}

	c := orig.CloneState().(*PoolSimulator)
	c.updateBalance("0xaaa", "0xbbb", big.NewInt(10), big.NewInt(20))

	require.Equal(t, big.NewInt(1000).Cmp(orig.Info.Reserves[0]), 0, "original reserve 0 changed")
	require.Equal(t, big.NewInt(2000).Cmp(orig.Info.Reserves[1]), 0, "original reserve 1 changed")
	require.Equal(t, uint64(10), orig.totalAmountsIn[0].Uint64(), "original total 0 changed")
	require.Equal(t, uint64(20), orig.totalAmountsIn[1].Uint64(), "original total 1 changed")
	require.Equal(t, big.NewInt(1010).Cmp(c.Info.Reserves[0]), 0, "clone reserve 0 not updated")
}
