package doppler

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

func wad(f float64) *big.Int { return new(big.Int).Mul(big.NewInt(int64(f*1e6)), big.NewInt(1e12)) }

// The fee distributions of the measured pools (getFeeDistributionInfo on chain) map to the afterSwap work per fee
// currency.
func TestRehypeFeeLegsByCurrency(t *testing.T) {
	zero := big.NewInt(0)
	// robinhood 0x3623…11c2: asset fees 62.5% numeraire buyback + 37.5% LP, numeraire fees 62.5% direct buyback + 37.5% LP.
	robinhood := feeDistributionRPC{zero, wad(.625), zero, wad(.375), zero, wad(.625), zero, wad(.375)}
	// base 0x31af…4ce4: asset fees 100% beneficiary, numeraire fees 20% asset buyback + 80% beneficiary.
	base := feeDistributionRPC{zero, zero, wad(1), zero, wad(.2), zero, wad(.8), zero}

	assert.Equal(t, [2]uint8{feeLegLp | feeLegBuybackSwap, feeLegLp}, robinhood.feeLegsByCurrency(true))
	assert.Equal(t, [2]uint8{feeLegLp, feeLegLp | feeLegBuybackSwap}, robinhood.feeLegsByCurrency(false))
	assert.Equal(t, [2]uint8{feeLegBuybackSwap, 0}, base.feeLegsByCurrency(false))
	assert.Equal(t, [2]uint8{0, 0}, (&feeDistributionRPC{}).feeLegsByCurrency(true), "no distribution read: no extra gas")
}

// The fee is taken in the output token (currency1 when zeroForOne), and its legs add their gas; no fee, no extra work.
func TestRehypeGasByFeeLegs(t *testing.T) {
	hook := &RehypeDHook{StartFee: 10500, FeeLegs: [2]uint8{feeLegLp | feeLegBuybackSwap, feeLegBuybackSwap}}
	for _, tc := range []struct {
		name       string
		hook       *RehypeDHook
		zeroForOne bool
		gas        int64
	}{
		{"fee in currency1: buyback swap", hook, true, rehypeGas + rehypeBuybackSwapGas},
		{"fee in currency0: LP and buyback swap", hook, false, rehypeGas + rehypeLpGas + rehypeBuybackSwapGas},
		{"no legs", &RehypeDHook{StartFee: 10500}, true, rehypeGas},
		{"no fee", &RehypeDHook{FeeLegs: hook.FeeLegs}, false, rehypeGas},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, err := tc.hook.AfterSwap(&uniswapv4.AfterSwapParams{
				BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: tc.zeroForOne},
				AmountIn:         big.NewInt(1_000_000), AmountOut: big.NewInt(1_000_000),
			}, nil)
			require.NoError(t, err)
			assert.Equal(t, tc.gas, res.Gas)
		})
	}

	// FeeLegs survives the pool-service round trip.
	raw, err := json.Marshal(hook)
	require.NoError(t, err)
	assert.Equal(t, hook.FeeLegs, NewRehypeDHook(raw).(*RehypeDHook).FeeLegs)
}
