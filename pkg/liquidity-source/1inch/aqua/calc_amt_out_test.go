package oneinchaqua

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

// Native CalcAmtOut must match legacy CalcAmountOut on amount out, fee, gas and error, for the same limit.
func TestCalcAmtOutMatchesLegacy(t *testing.T) {
	t.Parallel()
	s := newSim(t, poolOpts{})
	inv := swaplimit.NewInventory(DexType, swaplimit.ToU256Balances(s.CalculateLimit()))
	for in := range 2 {
		for _, amount := range []*uint256.Int{e18(1), e18(10), e18(900)} {
			legacy, lerr := quoteOut(s, in, amount, pool.AsSwapLimitBig(inv))
			native, nerr := s.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: s.Info.Tokens[in], Amount: *amount},
				TokenOut:      s.Info.Tokens[1-in],
				Limit:         inv,
			})
			require.Equal(t, lerr, nerr, "error in=%d amount=%s", in, amount.Dec())
			if lerr != nil {
				continue
			}
			require.Equal(t, legacy.TokenAmountOut.Amount.String(), native.TokenAmountOut.Amount.Dec(), "amountOut in=%d amount=%s", in, amount.Dec())
			require.Equal(t, legacy.Fee.Amount.String(), native.Fee.Amount.Dec(), "fee in=%d amount=%s", in, amount.Dec())
			require.Equal(t, legacy.Gas, native.Gas, "gas in=%d amount=%s", in, amount.Dec())
		}
	}
}

// Native CalcAmtOut must match legacy when a protocol fee is skipped, where FeeIn is set but FeeCollected is zero.
func TestCalcAmtOutFeeSkippedMatchesLegacy(t *testing.T) {
	t.Parallel()
	thin := poolOpts{program: prog(iProtocol(10_000_000, testRecipient), iSwap(), iSalt(1)), raw: [2]*uint256.Int{uint256.NewInt(1000), e18(2000)}}
	s := newSim(t, thin)
	legacy, lerr := quoteOut(s, 0, e18(1), nil)
	require.NoError(t, lerr)
	info := legacy.SwapInfo.(*SwapInfo)
	require.False(t, info.FeeIn.IsZero(), "fixture must owe a fee")
	require.True(t, info.FeeCollected.IsZero(), "fixture must skip the fee")
	native, nerr := s.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: s.Info.Tokens[0], Amount: *e18(1)},
		TokenOut:      s.Info.Tokens[1],
	})
	require.NoError(t, nerr)
	require.Equal(t, legacy.Fee.Amount.String(), native.Fee.Amount.Dec())
	require.Equal(t, legacy.Gas, native.Gas)
}

// A shared limit below the quoted amount out must fail the same way on both paths.
func TestCalcAmtOutLimitBindsLikeLegacy(t *testing.T) {
	t.Parallel()
	s := newSim(t, poolOpts{})
	probe, err := quoteOut(s, 0, e18(10), nil)
	require.NoError(t, err)
	out := probe.TokenAmountOut.Amount
	key := s.limitKeys[1]
	tight := swaplimit.NewInventory(DexType, swaplimit.ToU256Balances(map[string]*big.Int{key: new(big.Int).Sub(out, big.NewInt(1))}))
	_, lerr := quoteOut(s, 0, e18(10), pool.AsSwapLimitBig(tight))
	_, nerr := s.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: s.Info.Tokens[0], Amount: *e18(10)},
		TokenOut:      s.Info.Tokens[1],
		Limit:         tight,
	})
	require.ErrorIs(t, lerr, ErrInsufficientWallet)
	require.Equal(t, lerr, nerr)
}
