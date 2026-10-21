package woofiv21

import (
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	testWeth = "0x82af49447d8a07e3bd95bd0d56f35241523fbab1"
	testUsdc = "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8"
)

func newTestPool(t *testing.T, oraclePaused bool) *PoolSimulator {
	t.Helper()
	extra := Extra{
		QuoteToken: testUsdc,
		TokenInfos: map[string]TokenInfo{
			testWeth: {Reserve: u("306097831372356871541"), FeeRate: 25, MaxNotionalSwap: u("1000000000000"),
				MaxGamma: u("3000000000000000"), CapBal: u("1000000000000000000000000")},
			testUsdc: {Reserve: u("403770676421"), MaxNotionalSwap: u("1000000000000"),
				MaxGamma: u("3000000000000000"), CapBal: u("1000000000000000")},
		},
		Wooracle: Wooracle{
			States: map[string]State{
				testWeth: {Price: u("159709047746"), Spread: 270000000000000, Coeff: 1550000000, WoFeasible: true},
				testUsdc: {Price: u("100000000"), WoFeasible: true},
			},
			Decimals:      map[string]uint8{testWeth: 8, testUsdc: 8},
			Timestamp:     time.Now().Unix(),
			StaleDuration: 300,
			Bound:         10000000000000000,
		},
		OraclePaused: oraclePaused,
	}
	b, err := json.Marshal(extra)
	require.NoError(t, err)
	sim, err := NewPoolSimulator(entity.Pool{
		Address: "0x5520385bfcf07ec87c4c53a7d8d65595dff69fa4", Exchange: "woofi-v3", Type: DexTypeWooFiV21,
		Reserves: entity.PoolReserves{"306097831372356871541", "403770676421"},
		Tokens: []*entity.PoolToken{
			{Address: testWeth, Decimals: 18, Swappable: true},
			{Address: testUsdc, Decimals: 8, Swappable: true},
		},
		Extra: string(b),
	})
	require.NoError(t, err)
	return sim
}

func u(s string) *uint256.Int { return number.NewUint256(s) }

// A paused Wooracle reverts state()/price(), so every on-chain swap reverts; the
// simulator must not quote while the same snapshot quotes fine when unpaused.
func TestPoolSimulator_OraclePaused(t *testing.T) {
	t.Parallel()
	params := poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{Token: testWeth, Amount: bignumber.NewBig("304999404452284472")},
		TokenOut:      testUsdc,
	}

	res, err := newTestPool(t, false).CalcAmountOut(params)
	require.NoError(t, err)
	assert.Equal(t, "48685801162", res.TokenAmountOut.Amount.String())

	res, err = newTestPool(t, true).CalcAmountOut(params)
	assert.ErrorIs(t, err, ErrOracleIsPaused)
	assert.Nil(t, res)
}

// The tracker treats a reverting paused() as "not paused" only because the pre-pause oracle
// lacks the selector. Guard the selector we call.
func TestWooracleABI_PausedSelector(t *testing.T) {
	t.Parallel()
	m, ok := WooracleV2ABI.Methods[wooracleMethodPaused]
	require.True(t, ok)
	assert.Equal(t, "0x5c975abb", hexutil.Encode(m.ID))
}
