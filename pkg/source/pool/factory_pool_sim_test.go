package pool

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// legacyFake is a legacy big.Int pool. It returns out = in * 2 and records the update it receives.
type legacyFake struct {
	IPoolSimulator
	updated UpdateBalanceParams
}

func (f *legacyFake) CalcAmountOut(p CalcAmountOutParams) (*CalcAmountOutResult, error) {
	out := new(big.Int).Mul(p.TokenAmountIn.Amount, big.NewInt(2))
	return &CalcAmountOutResult{
		TokenAmountOut: &TokenAmount{Token: p.TokenOut, Amount: out},
		Fee:            &TokenAmount{Token: p.TokenOut, Amount: big.NewInt(3)},
		Gas:            7,
	}, nil
}

func (f *legacyFake) CloneState() IPoolSimulator { return f }

func (f *legacyFake) UpdateBalance(p UpdateBalanceParams) { f.updated = p }

func (f *legacyFake) CalculateLimit() map[string]*big.Int {
	return map[string]*big.Int{"a": big.NewInt(50)}
}

func (f *legacyFake) GetReserves() []*big.Int { return []*big.Int{big.NewInt(9)} }

func TestLegacyAdapterCalcAmtOutMatchesBig(t *testing.T) {
	u := &legacyAdapter{IPoolSimulator: &legacyFake{}}
	res, err := u.CalcAmtOut(CalcAmtOutParams{
		TokenAmountIn: TokenAmt{Token: "in", Amount: *uint256.NewInt(21)},
		TokenOut:      "out",
	})
	require.NoError(t, err)
	require.Equal(t, uint64(42), res.TokenAmountOut.Amount.Uint64(), "uint256 out must equal big out")
	require.Equal(t, uint64(3), res.Fee.Amount.Uint64())
	require.Equal(t, int64(7), res.Gas)
}

func TestLegacyAdapterUpdateStateReachesLegacyPool(t *testing.T) {
	fake := &legacyFake{}
	u := legacyAdapter{IPoolSimulator: fake}
	u.UpdateState(UpdateStateParams{
		TokenAmountIn:  TokenAmt{Token: "in", Amount: *uint256.NewInt(5)},
		TokenAmountOut: TokenAmt{Token: "out", Amount: *uint256.NewInt(8)},
	})
	require.Equal(t, int64(5), fake.updated.TokenAmountIn.Amount.Int64())
	require.Equal(t, int64(8), fake.updated.TokenAmountOut.Amount.Int64())
}

func TestLegacyAdapterLimitsAndReservesConvert(t *testing.T) {
	u := legacyAdapter{IPoolSimulator: &legacyFake{}}
	require.Equal(t, uint64(50), u.CalcLimit()["a"].Uint64())
	require.Equal(t, uint64(9), u.GetReserves()[0].Uint64())
}

// A legacy result with a nil or negative amount must not become a real value.
func TestBigToUOrZeroRejectsNilAndNegative(t *testing.T) {
	require.True(t, bigToU256OrZero(nil).IsZero())
	require.True(t, bigToU256OrZero(big.NewInt(-4)).IsZero())
	require.Equal(t, uint64(4), bigToU256OrZero(big.NewInt(4)).Uint64())
}
