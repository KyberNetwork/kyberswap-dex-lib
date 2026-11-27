package maverickv2

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

const (
	cloneTokenA = "0x00000000000000000000000000000000000000a1"
	cloneTokenB = "0x00000000000000000000000000000000000000b2"
)

// newClonePool is a one-bin pool: enough for CloneState and a small swap.
func newClonePool(t *testing.T) pool.IPoolSimulator {
	t.Helper()
	extra := `{"feeAIn":100000000000000,"feeBIn":100000000000000,"protocolFeeRatio":0,"activeTick":0,"lastTwaD8":0,` +
		`"bins":{"1":{"tick":0,"totalSupply":"1000000000000000000000","tickBalance":"1000000000000000000000"}},` +
		`"ticks":{"0":{"ReserveA":"1000000000000000000000","ReserveB":"1000000000000000000000",` +
		`"TotalSupply":"1000000000000000000000","BinIdsByTick":{"0":1}}}}`
	p, err := NewPoolSimulator(entity.Pool{Address: "0xmav", Exchange: "maverick-v2", Type: "maverick-v2",
		Reserves: entity.PoolReserves{"1000000000000000000000", "1000000000000000000000"}, Tokens: cloneTestTokens(),
		Extra: extra, StaticExtra: `{"tickSpacing":10}`})
	require.NoError(t, err)
	return p
}

func cloneTestTokens() []*entity.PoolToken {
	return []*entity.PoolToken{{Address: cloneTokenA, Decimals: 18, Swappable: true},
		{Address: cloneTokenB, Decimals: 18, Swappable: true}}
}

// A swap on a clone changes only the clone, also while clones swap on several goroutines and the original is read.
func TestCloneStateIsolatesSwaps(t *testing.T) {
	testutil.TestCloneIsolation(t, newClonePool(t), map[string]testutil.CloneOp{
		"swap": testutil.SwapOp(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: cloneTokenA, Amount: bignumber.TenPowInt(18)}, TokenOut: cloneTokenB}),
	})
}
