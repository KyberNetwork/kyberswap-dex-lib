package metavault_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/metavault"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The router receives simulators through pkg/msgpack, so the vault accounting and the real balances have to survive
// the trip: a decoded simulator quotes, rejects and updates like the original.
func TestMsgpackRoundTrip(t *testing.T) {
	const (
		a0, a2 = "A0", "A2"
		poolA0 = "351500182590784658632430"
		poolA2 = "582500526946365638607"
	)
	newSim := func(t *testing.T, tokenBalances, realBalances string) *metavault.PoolSimulator {
		extra := `{"vault":{"hasDynamicFees":true,"includeAmmPrice":false,"isSwapEnabled":true,"stableSwapFeeBasisPoints":25,"stableTaxBasisPoints":5,"swapFeeBasisPoints":30,"taxBasisPoints":50,"totalTokenWeights":36000,
"whitelistedTokens":["A0","A2"],
"poolAmounts":{"A0":` + poolA0 + `,"A2":` + poolA2 + `},
"bufferAmounts":{"A0":1,"A2":1},
"reservedAmounts":{"A0":10076951923665922051838,"A2":177431951598853399981},
"tokenDecimals":{"A0":18,"A2":18},
"stableTokens":{"A0":false,"A2":false},
"usdmAmounts":{"A0":253813004375598635984875,"A2":1093840701705446620699791},
"maxUsdmAmounts":{"A0":400000000000000000000000,"A2":1400000000000000000000000},
"tokenWeights":{"A0":8000,"A2":28000},` + tokenBalances + realBalances + `
"priceFeed":{"isSecondaryPriceEnabled":false,"maxStrictPriceDeviation":15000000000000000000000000000,"priceSampleSpace":1,
"priceDecimals":{"A0":8,"A2":8},"spreadBasisPoints":{"A0":8,"A2":0},"adjustmentBasisPoints":{"A0":0,"A2":0},
"strictStableTokens":{"A0":false,"A2":false},"isAdjustmentAdditive":{"A0":false,"A2":false},
"secondaryPriceFeedVersion":2,"secondaryPriceFeed":{},
"priceFeeds":{"A0":{"roundId":1,"answer":69050000,"answers":{"1":69050000}},"A2":{"roundId":1,"answer":196345780000,"answers":{"1":196345780000}}}},
"usdm":{"address":"0x533403a3346cA31D67c380917ffaF185c24e7333","totalSupply":3389201190535341442726377}}}`
		sim, err := metavault.NewPoolSimulator(entity.Pool{
			Tokens:   []*entity.PoolToken{{Address: a0}, {Address: a2}},
			Reserves: entity.PoolReserves{"1", "1"},
			Extra:    extra,
		})
		require.NoError(t, err)
		return sim
	}
	healthy := `"tokenBalances":{"A0":` + poolA0 + `,"A2":` + poolA2 + `},`
	realHealthy := `"realBalances":{"A0":` + poolA0 + `,"A2":` + poolA2 + `},`
	// the vault holds none of A2 while its accounting still shows funds
	realDrained := `"realBalances":{"A0":` + poolA0 + `,"A2":0},`

	quote := func(sim pool.IPoolSimulator, in, out string, amount int64) (*pool.CalcAmountOutResult, error) {
		return sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: in, Amount: big.NewInt(amount)}, TokenOut: out,
		})
	}
	roundTrip := func(t *testing.T, sim pool.IPoolSimulator) pool.IPoolSimulator {
		enc, err := msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{"p": sim})
		require.NoError(t, err)
		dec, err := msgpack.DecodePoolSimulatorsMap(enc)
		require.NoError(t, err)
		return dec["p"]
	}

	t.Run("with real balances, after an UpdateBalance", func(t *testing.T) {
		sim := newSim(t, healthy, realHealthy)
		res, err := quote(sim, a0, a2, 1e15)
		require.NoError(t, err)
		sim.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: a0, Amount: big.NewInt(1e15)},
			TokenAmountOut: *res.TokenAmountOut,
			Fee:            *res.Fee,
		})

		decoded := roundTrip(t, sim)
		for _, dir := range [][2]string{{a0, a2}, {a2, a0}} {
			want, err := quote(sim, dir[0], dir[1], 1e15)
			require.NoError(t, err)
			got, err := quote(decoded, dir[0], dir[1], 1e15)
			require.NoError(t, err)
			require.Equal(t, want.TokenAmountOut, got.TokenAmountOut)
			require.Equal(t, want.Fee, got.Fee)
		}
	})

	t.Run("drained token is still rejected", func(t *testing.T) {
		sim := newSim(t, healthy, realDrained)
		_, err := quote(sim, a0, a2, 1e15)
		require.ErrorIs(t, err, metavault.ErrInsufficientRealBalance)

		_, err = quote(roundTrip(t, sim), a0, a2, 1e15)
		require.ErrorIs(t, err, metavault.ErrInsufficientRealBalance)
	})

	t.Run("without balances (state saved before they existed)", func(t *testing.T) {
		sim := newSim(t, "", "")
		want, err := quote(sim, a0, a2, 1e15)
		require.NoError(t, err)

		got, err := quote(roundTrip(t, sim), a0, a2, 1e15)
		require.NoError(t, err)
		require.Equal(t, want.TokenAmountOut, got.TokenAmountOut)
	})
}
