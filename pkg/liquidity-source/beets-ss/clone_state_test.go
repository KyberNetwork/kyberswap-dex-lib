package beets_ss

import (
	"testing"

	"github.com/goccy/go-json"
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
	extra, err := json.Marshal(Extra{TotalSupply: bignumber.NewBig10("1000000000000000000000"),
		TotalAssets: bignumber.NewBig10("1050000000000000000000")})
	require.NoError(t, err)
	p, err := NewPoolSimulator(entity.Pool{Address: "0xbeets", Exchange: "beets-ss", Type: "beets-ss",
		Reserves: entity.PoolReserves{"1000000000000000000000", "1050000000000000000000"}, Tokens: cloneTestTokens(),
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
