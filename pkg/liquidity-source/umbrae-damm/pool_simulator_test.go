package umbraedamm

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	tokenX = "0x14a4e80d633af55ace1160c320f5a36d41cced3e" // U1
	tokenY = "0x4200000000000000000000000000000000000006" // WETH (feeToken)
)

func newSim(t *testing.T, reserveX, reserveY string, feeBps uint64) *PoolSimulator {
	t.Helper()
	extra, _ := json.Marshal(Extra{FeeBps: feeBps, FeeToken: tokenY})
	sim, err := NewPoolSimulator(entity.Pool{
		Address:  "0x296964c34a571fcf85d3f74fb815ee871f5a08d4",
		Exchange: DexType,
		Type:     DexType,
		Reserves: entity.PoolReserves{reserveX, reserveY},
		Tokens:   []*entity.PoolToken{{Address: tokenX}, {Address: tokenY}},
		Extra:    string(extra),
	})
	require.NoError(t, err)
	return sim
}

func cpRef(in, rIn, rOut *big.Int) *big.Int {
	return new(big.Int).Div(new(big.Int).Mul(in, rOut), new(big.Int).Add(rIn, in))
}
func feeRef(amt *big.Int, feeBps int64) *big.Int {
	return new(big.Int).Div(new(big.Int).Mul(amt, big.NewInt(feeBps)), big.NewInt(feeDenominator))
}

// TestCalcAmountOut_FeeTokenModel verifies the deployed feeToken fee model: fee is always charged
// in feeToken (WETH = tokenY) — on the input for Y->X, on the output for X->Y.
func TestCalcAmountOut_FeeTokenModel(t *testing.T) {
	rx, _ := new(big.Int).SetString("1000000000000000000000", 10)
	ry, _ := new(big.Int).SetString("500000000000000000000", 10)
	sim := newSim(t, rx.String(), ry.String(), 30)
	amountIn, _ := new(big.Int).SetString("10000000000000000000", 10) // 10

	// Y->X: feeToken (WETH) is the input -> fee on input.
	resYX, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenY, Amount: amountIn}, TokenOut: tokenX,
	})
	require.NoError(t, err)
	feeIn := feeRef(amountIn, 30)
	wantYX := cpRef(new(big.Int).Sub(amountIn, feeIn), ry, rx)
	require.Equal(t, wantYX, resYX.TokenAmountOut.Amount)
	require.Equal(t, feeIn, resYX.Fee.Amount)
	require.Equal(t, tokenY, resYX.Fee.Token) // fee always in feeToken

	// X->Y: feeToken (WETH) is the output -> fee on output.
	resXY, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: amountIn}, TokenOut: tokenY,
	})
	require.NoError(t, err)
	outFull := cpRef(amountIn, rx, ry)
	feeOut := feeRef(outFull, 30)
	require.Equal(t, new(big.Int).Sub(outFull, feeOut), resXY.TokenAmountOut.Amount)
	require.Equal(t, feeOut, resXY.Fee.Amount)
	require.Equal(t, tokenY, resXY.Fee.Token)
}

// TestUpdateBalance_KConstant checks the reserve deltas: reserveOut drops by the full pre-fee
// output, reserveIn rises by the post-fee input (fees exit into accumulators, K constant).
func TestUpdateBalance_KConstant(t *testing.T) {
	rx, _ := new(big.Int).SetString("1000000000000000000000", 10)
	ry, _ := new(big.Int).SetString("500000000000000000000", 10)
	sim := newSim(t, rx.String(), ry.String(), 30)
	amountIn, _ := new(big.Int).SetString("10000000000000000000", 10)

	// X->Y (fee on output): reserveX += amountIn; reserveY -= outFull.
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: amountIn}, TokenOut: tokenY,
	})
	require.NoError(t, err)

	clone := sim.CloneState()
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: tokenX, Amount: amountIn},
		TokenAmountOut: pool.TokenAmount{Token: tokenY, Amount: res.TokenAmountOut.Amount},
		Fee:            *res.Fee,
		SwapInfo:       res.SwapInfo,
	})

	outFull := cpRef(amountIn, rx, ry)
	require.Equal(t, new(big.Int).Add(rx, amountIn), sim.reserves[0].ToBig())
	require.Equal(t, new(big.Int).Sub(ry, outFull), sim.reserves[1].ToBig())
	require.Equal(t, rx, clone.GetReserves()[0]) // clone untouched
}

func TestCalcAmountOut_Errors(t *testing.T) {
	sim := newSim(t, "1000000000000000000000", "500000000000000000000", 30)
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xdead", Amount: big.NewInt(1)}, TokenOut: tokenY,
	})
	require.ErrorIs(t, err, ErrInvalidToken)
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenX, Amount: big.NewInt(0)}, TokenOut: tokenY,
	})
	require.ErrorIs(t, err, ErrInvalidAmountIn)
}
