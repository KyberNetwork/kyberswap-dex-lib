package poe

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
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

func newClonePool(t *testing.T) pool.IPoolSimulator {
	t.Helper()
	extra, err := json.Marshal(Extra{Price: uint256.NewInt(1_000_000_000_000_000_000), FeeHbps: uint256.NewInt(3000),
		Alpha: uint256.NewInt(10500), Expiry: 1 << 40})
	require.NoError(t, err)
	p, err := NewPoolSimulator(entity.Pool{Address: "0xpoe", Exchange: "lfj-poe", Type: "lfj-poe",
		Reserves: entity.PoolReserves{"1000000000000000000000", "1000000000000000000000"}, Tokens: cloneTestTokens(),
		Extra: string(extra)})
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
