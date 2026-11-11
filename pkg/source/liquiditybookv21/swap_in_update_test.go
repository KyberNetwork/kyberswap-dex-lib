package liquiditybookv21

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// TestSwapInUpdatesBinsAsSwapOut: a swap quoted by the amount out leaves the bins as the same swap
// quoted by the amount in does, whenever the two quotes agree. Each bin gets what came into it, less
// the protocol's share of its fee, and gives what went out of it, the last bin included.
func TestSwapInUpdatesBinsAsSwapOut(t *testing.T) {
	const active = 8388608
	e18 := bignumber.TenPowInt(18)
	var bins []Bin
	for i := uint32(5); i >= 1; i-- {
		bins = append(bins, Bin{ID: active - i, ReserveX: new(big.Int), ReserveY: e18})
	}
	bins = append(bins, Bin{ID: active, ReserveX: e18, ReserveY: e18})
	for i := uint32(1); i <= 5; i++ {
		bins = append(bins, Bin{ID: active + i, ReserveX: e18, ReserveY: new(big.Int)})
	}
	extra, err := json.Marshal(Extra{
		RpcBlockTimestamp: 1713521596,
		StaticFeeParams: staticFeeParams{BaseFactor: 10000, FilterPeriod: 30, DecayPeriod: 600,
			ReductionFactor: 5000, VariableFeeControl: 20000, ProtocolShare: 1000, MaxVolatilityAccumulator: 350000},
		VariableFeeParams: variableFeeParams{IdReference: 8388608, TimeOfLastUpdate: 1713521596},
		ActiveBinID:       active,
		BinStep:           20,
		Bins:              bins,
	})
	require.NoError(t, err)
	newSim := func() *PoolSimulator {
		sim, err := NewPoolSimulator(entity.Pool{Address: "0x1111111111111111111111111111111111111111",
			Exchange: "traderjoe-v21", Type: DexTypeLiquidityBookV21,
			Reserves: entity.PoolReserves{"6000000000000000000", "6000000000000000000"},
			Tokens:   []*entity.PoolToken{{Address: "0xa"}, {Address: "0xb"}}, Extra: string(extra)})
		require.NoError(t, err)
		return sim
	}

	compared := 0
	for _, in := range []string{"0xa", "0xb"} {
		out := map[string]string{"0xa": "0xb", "0xb": "0xa"}[in]
		for _, amountOut := range []string{"1000", "500000000000000000", "1000000000000000000",
			"2500000000000000000", "4999999999999999999"} {
			byOut, byIn := newSim(), newSim()
			x, _ := new(big.Int).SetString(amountOut, 10)
			resIn, err := byOut.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{Token: out, Amount: x}, TokenIn: in})
			require.NoError(t, err)
			resOut, err := byIn.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: *resIn.TokenAmountIn, TokenOut: out})
			require.NoError(t, err)
			if resOut.TokenAmountOut.Amount.Cmp(x) != 0 {
				continue // rounding gave the exact-in swap a little more out
			}
			compared++
			byOut.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: *resIn.TokenAmountIn,
				TokenAmountOut: pool.TokenAmount{Token: out, Amount: x}, SwapInfo: resIn.SwapInfo})
			byIn.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: *resIn.TokenAmountIn,
				TokenAmountOut: *resOut.TokenAmountOut, SwapInfo: resOut.SwapInfo})
			assert.Equal(t, byIn.activeBinID, byOut.activeBinID, "%s out of %s", amountOut, out)
			assert.Equal(t, byIn.bins, byOut.bins, "%s out of %s", amountOut, out)
		}
	}
	require.Positive(t, compared)
}
