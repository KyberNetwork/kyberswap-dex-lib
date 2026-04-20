package nabla

import (
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/stretchr/testify/require"
)

func baseSnapshotPools() (NablaPoolMeta, NablaPoolState, NablaPoolState) {
	meta := NablaPoolMeta{
		CurveBeta:                 int256.MustFromDec("1000000000000000"),
		CurveC:                    int256.MustFromDec("33638584039112749"),
		LpFee:                     int256.NewInt(25),
		BackstopFee:               int256.NewInt(12),
		ProtocolFee:               int256.NewInt(12),
		MaxCoverageRatioForSwapIn: int256.NewInt(200),
	}

	usdcState := NablaPoolState{
		Reserve:             int256.MustFromDec("6668027803"),
		ReserveWithSlippage: int256.MustFromDec("6668068923"),
		TotalLiabilities:    int256.MustFromDec("6136347818"),
		Price:               int256.MustFromDec("1000000000000000000"),
	}
	eurcState := NablaPoolState{
		Reserve:             int256.MustFromDec("4818674874"),
		ReserveWithSlippage: int256.MustFromDec("4818713498"),
		TotalLiabilities:    int256.MustFromDec("5257931086"),
		Price:               int256.MustFromDec("1176482790000000000"),
	}

	return meta, usdcState, eurcState
}

// Test_sell_baseSnapshot reproduces a fixed Base USDC/EURC pool snapshot from
// the Nabla router on top of the contract semantics. The values here are not a
// live RPC lookup and should stay deterministic.
func Test_sell_baseSnapshot(t *testing.T) {
	meta, usdcState, eurcState := baseSnapshotPools()

	usdcPool := NablaPool{Meta: meta, State: usdcState}
	eurcPool := NablaPool{Meta: meta, State: eurcState}

	tests := []struct {
		name      string
		amountIn  string
		want      string
		expectErr bool
	}{
		{"1 USDC -> EURC", "1000000", "849667", false},
		{"100 USDC -> EURC", "100000000", "84963938", false},
		{"5600 USDC -> EURC (snapshot OK)", "5600000000", "4643936953", false},
		{"5610 USDC -> EURC (snapshot reverts)", "5610000000", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := sell(usdcPool, eurcPool, int256.MustFromDec(tt.amountIn), 6, 6)
			if tt.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got.Dec())
		})
	}
}

func Test_sell_baseSnapshotSwapInfo(t *testing.T) {
	meta, usdcState, eurcState := baseSnapshotPools()

	usdcPool := NablaPool{Meta: meta, State: usdcState}
	eurcPool := NablaPool{Meta: meta, State: eurcState}
	amountIn := int256.MustFromDec("1000000000")

	amountOut, swapInfo, err := sell(usdcPool, eurcPool, amountIn, 6, 6)
	require.NoError(t, err)

	curveIn := NewCurve(meta.CurveBeta, meta.CurveC)
	curveOut := NewCurve(meta.CurveBeta, meta.CurveC)

	effectiveAmountIn := curveIn.InverseHorizontal(
		usdcState.Reserve,
		usdcState.TotalLiabilities,
		new(int256.Int).Add(usdcState.ReserveWithSlippage, amountIn),
		6,
	)
	require.Equal(t,
		new(int256.Int).Add(usdcState.Reserve, effectiveAmountIn).Dec(),
		swapInfo.frPoolNewState.Reserve.Dec(),
	)
	require.Equal(t,
		new(int256.Int).Add(usdcState.ReserveWithSlippage, amountIn).Dec(),
		swapInfo.frPoolNewState.ReserveWithSlippage.Dec(),
	)

	rawAmountOut := new(int256.Int).Mul(effectiveAmountIn, usdcState.Price)
	rawAmountOut.Quo(rawAmountOut, eurcState.Price)

	backstopFeeAmount := new(int256.Int).Mul(rawAmountOut, meta.BackstopFee)
	backstopFeeAmount.Quo(backstopFeeAmount, feePrecision)

	protocolFeeAmount := new(int256.Int).Mul(rawAmountOut, meta.ProtocolFee)
	protocolFeeAmount.Quo(protocolFeeAmount, feePrecision)

	maxLpFeeAmount := new(int256.Int).Mul(rawAmountOut, meta.LpFee)
	maxLpFeeAmount.Quo(maxLpFeeAmount, feePrecision)

	reducedReserveOut := new(int256.Int).Sub(eurcState.Reserve, rawAmountOut)
	reducedReserveOut.Add(reducedReserveOut, backstopFeeAmount)
	reducedReserveOut.Add(reducedReserveOut, protocolFeeAmount)

	actualLpFeeAmount := curveOut.InverseDiagonal(
		reducedReserveOut,
		eurcState.TotalLiabilities,
		eurcState.ReserveWithSlippage,
		6,
	)
	if actualLpFeeAmount.Gt(maxLpFeeAmount) {
		actualLpFeeAmount = maxLpFeeAmount
	}

	actualReducedReserveOut := new(int256.Int).Add(reducedReserveOut, actualLpFeeAmount)
	actualTotalLiabilitiesOut := new(int256.Int).Add(eurcState.TotalLiabilities, actualLpFeeAmount)
	reserveWithSlippageAfterAmountOut := curveOut.Psi(actualReducedReserveOut, actualTotalLiabilitiesOut, 6)
	if reserveWithSlippageAfterAmountOut.Gt(eurcState.ReserveWithSlippage) {
		reserveWithSlippageAfterAmountOut = eurcState.ReserveWithSlippage
	}

	expectedOutputReserve := new(int256.Int).Sub(actualReducedReserveOut, protocolFeeAmount)
	expectedOutputReserveWithSlippage := curveOut.Psi(expectedOutputReserve, actualTotalLiabilitiesOut, 6)
	if expectedOutputReserveWithSlippage.Gt(reserveWithSlippageAfterAmountOut) {
		expectedOutputReserveWithSlippage = reserveWithSlippageAfterAmountOut
	}

	require.Equal(t,
		new(int256.Int).Sub(eurcState.ReserveWithSlippage, reserveWithSlippageAfterAmountOut).Dec(),
		amountOut.Dec(),
	)
	require.Equal(t, expectedOutputReserve.Dec(), swapInfo.toPoolNewState.Reserve.Dec())
	require.Equal(t, expectedOutputReserveWithSlippage.Dec(), swapInfo.toPoolNewState.ReserveWithSlippage.Dec())
	require.Equal(t, actualTotalLiabilitiesOut.Dec(), swapInfo.toPoolNewState.TotalLiabilities.Dec())
}
