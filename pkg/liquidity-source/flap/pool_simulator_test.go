package flap

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	testQuoteToken = "0x205812cdbed920aff76c6580abd681a46d11efc7"
	testBaseToken  = "0xfc5e45df826761961dccda41f7a243be5b147777"
)

// buildTestPool mirrors real state read live from Portal.getTokenV8 on BSC for this token/curve.
func buildTestPool(t *testing.T) *PoolSimulator {
	t.Helper()
	return buildTestPoolWithExtra(t, Extra{
		Status: TokenStatusTradable,
		Curve: Curve{
			R: uint256.MustFromDecimal("5685925930000000000"),
			H: uint256.MustFromDecimal("107036751000000000000000000"),
			K: uint256.MustFromDecimal("6294528967973853430000000000"),
		},
		CirculatingSupply: uint256.MustFromDecimal("356411072277789739739740347"),
		DexSupplyThresh:   uint256.MustFromDecimal("800000000000000000000000000"),
		BuyFeeBps:         100,
		SellFeeBps:        100,
	})
}

// buildTestPoolWithTax mirrors a real tax-enabled curve token (buyTaxRate=sellTaxRate=100, matched
// live against the board API's tax.buyTaxBps/sellTaxBps for the same token).
func buildTestPoolWithTax(t *testing.T) *PoolSimulator {
	t.Helper()
	return buildTestPoolWithExtra(t, Extra{
		Status: TokenStatusTradable,
		Curve: Curve{
			R: uint256.MustFromDecimal("5685925930000000000"),
			H: uint256.MustFromDecimal("107036751000000000000000000"),
			K: uint256.MustFromDecimal("6294528967973853430000000000"),
		},
		CirculatingSupply:        uint256.MustFromDecimal("356411072277789739739740347"),
		DexSupplyThresh:          uint256.MustFromDecimal("800000000000000000000000000"),
		BuyFeeBps:                100,
		SellFeeBps:               100,
		BuyTaxBps:                100,
		SellTaxBps:               100,
		TaxOnBondingCurveEnabled: true,
	})
}

func buildTestPoolWithExtra(t *testing.T, extra Extra) *PoolSimulator {
	t.Helper()

	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)

	staticExtra := StaticExtra{
		QuoteToken:    testQuoteToken,
		PortalAddress: "0xe2cE6ab80874Fa9Fa2aAE65D277Dd6B8e65C9De0",
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	require.NoError(t, err)

	entityPool := entity.Pool{
		Address:     testBaseToken,
		Exchange:    DexType,
		Type:        DexType,
		Reserves:    entity.PoolReserves{"2699783680533211615", "643588927722210260260259653"},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
		Tokens: []*entity.PoolToken{
			{Address: testQuoteToken, Decimals: 18},
			{Address: testBaseToken, Decimals: 18},
		},
	}

	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)
	return sim
}

func TestCalcAmountOut_Buy(t *testing.T) {
	sim := buildTestPool(t)

	amountIn, _ := new(big.Int).SetString("1000000000000000000", 10) // 1 quote token
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testQuoteToken, Amount: amountIn},
		TokenOut:      testBaseToken,
	})
	require.NoError(t, err)
	require.True(t, result.IsValid())
	assert.Equal(t, testBaseToken, result.TokenAmountOut.Token)
	assert.Positive(t, result.TokenAmountOut.Amount.Sign())
	// 1% protocol fee on the 1e18 input.
	assert.Equal(t, "10000000000000000", result.Fee.Amount.String())

	swapInfo, ok := result.SwapInfo.(SwapInfo)
	require.True(t, ok)
	assert.True(t, swapInfo.NewCirculatingSupply.Gt(sim.circulatingSupply))
}

func TestCalcAmountOut_Sell(t *testing.T) {
	sim := buildTestPool(t)

	amountIn, _ := new(big.Int).SetString("1000000000000000000000", 10) // 1000 base tokens
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testBaseToken, Amount: amountIn},
		TokenOut:      testQuoteToken,
	})
	require.NoError(t, err)
	require.True(t, result.IsValid())
	assert.Equal(t, testQuoteToken, result.TokenAmountOut.Token)

	swapInfo, ok := result.SwapInfo.(SwapInfo)
	require.True(t, ok)
	assert.True(t, swapInfo.NewCirculatingSupply.Lt(sim.circulatingSupply))
}

func TestCalcAmountOut_SellMoreThanCirculatingSupply(t *testing.T) {
	sim := buildTestPool(t)

	amountIn := new(big.Int).Add(sim.circulatingSupply.ToBig(), big.NewInt(1))
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testBaseToken, Amount: amountIn},
		TokenOut:      testQuoteToken,
	})
	assert.ErrorIs(t, err, ErrInsufficientSupply)
}

func TestCalcAmountOut_NotTradable(t *testing.T) {
	sim := buildTestPool(t)
	sim.status = TokenStatusDEX

	amountIn := big.NewInt(1e18)
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testQuoteToken, Amount: amountIn},
		TokenOut:      testBaseToken,
	})
	assert.ErrorIs(t, err, ErrPoolNotTradable)
}

func TestCalcAmountOut_BuyCapAtDexSupplyThresh(t *testing.T) {
	sim := buildTestPool(t)

	// A huge buy should cap circulating supply at dexSupplyThresh and refund the unused input.
	amountIn, _ := new(big.Int).SetString("1000000000000000000000000", 10) // 1,000,000 quote tokens
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testQuoteToken, Amount: amountIn},
		TokenOut:      testBaseToken,
	})
	require.NoError(t, err)
	require.True(t, result.IsValid())
	require.NotNil(t, result.RemainingTokenAmountIn)
	assert.Positive(t, result.RemainingTokenAmountIn.Amount.Sign())

	swapInfo, ok := result.SwapInfo.(SwapInfo)
	require.True(t, ok)
	assert.True(t, swapInfo.NewCirculatingSupply.Eq(sim.dexSupplyThresh))
	assert.Equal(t, TokenStatusDEX, swapInfo.NewStatus)
}

func TestUpdateBalance_MovesCirculatingSupply(t *testing.T) {
	sim := buildTestPool(t)
	original := new(uint256.Int).Set(sim.circulatingSupply)

	amountIn := big.NewInt(1e18)
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testQuoteToken, Amount: amountIn},
		TokenOut:      testBaseToken,
	})
	require.NoError(t, err)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testQuoteToken, Amount: amountIn},
		TokenAmountOut: *result.TokenAmountOut,
		SwapInfo:       result.SwapInfo,
	})

	assert.False(t, sim.circulatingSupply.Eq(original))
	assert.True(t, sim.circulatingSupply.Gt(original))
}

func TestCloneState_Isolation(t *testing.T) {
	sim := buildTestPool(t)
	clone, ok := sim.CloneState().(*PoolSimulator)
	require.True(t, ok)

	clone.circulatingSupply.AddUint64(clone.circulatingSupply, 1)

	assert.False(t, sim.circulatingSupply.Eq(clone.circulatingSupply))
}

func TestCalcAmountOut_Purity(t *testing.T) {
	sim := buildTestPool(t)
	before := new(uint256.Int).Set(sim.circulatingSupply)

	amountIn := big.NewInt(1e18)
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testQuoteToken, Amount: amountIn},
		TokenOut:      testBaseToken,
	})
	require.NoError(t, err)

	assert.True(t, sim.circulatingSupply.Eq(before), "CalcAmountOut must not mutate state")
}

func TestGetApprovalAddress(t *testing.T) {
	sim := buildTestPool(t)
	assert.Equal(t, "0xe2cE6ab80874Fa9Fa2aAE65D277Dd6B8e65C9De0", sim.GetApprovalAddress(testQuoteToken, testBaseToken))
}

func TestCalcAmountOut_TokenTaxOnTopOfProtocolFee(t *testing.T) {
	noTax := buildTestPool(t)
	withTax := buildTestPoolWithTax(t)

	amountIn := big.NewInt(1e18)
	resNoTax, err := noTax.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testQuoteToken, Amount: amountIn},
		TokenOut:      testBaseToken,
	})
	require.NoError(t, err)
	resWithTax, err := withTax.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testQuoteToken, Amount: amountIn},
		TokenOut:      testBaseToken,
	})
	require.NoError(t, err)

	// Both protocol fee (1%) and token tax (1%) apply, so the taxed pool's output must be strictly
	// less than the untaxed pool's for the same input - and its recorded Fee strictly larger.
	assert.True(t, resWithTax.TokenAmountOut.Amount.Cmp(resNoTax.TokenAmountOut.Amount) < 0)
	assert.True(t, resWithTax.Fee.Amount.Cmp(resNoTax.Fee.Amount) > 0)
}

func TestCalcAmountOut_TaxDisabledGlobally(t *testing.T) {
	sim := buildTestPoolWithExtra(t, Extra{
		Status: TokenStatusTradable,
		Curve: Curve{
			R: uint256.MustFromDecimal("5685925930000000000"),
			H: uint256.MustFromDecimal("107036751000000000000000000"),
			K: uint256.MustFromDecimal("6294528967973853430000000000"),
		},
		CirculatingSupply: uint256.MustFromDecimal("356411072277789739739740347"),
		DexSupplyThresh:   uint256.MustFromDecimal("800000000000000000000000000"),
		BuyFeeBps:         100,
		SellFeeBps:        100,
		BuyTaxBps:         300,
		SellTaxBps:        300,
		// TaxOnBondingCurveEnabled left false: the token has a tax config but the global switch is
		// off, so only the protocol fee should apply.
	})
	baseline := buildTestPool(t)

	amountIn := big.NewInt(1e18)
	resTaxToken, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testQuoteToken, Amount: amountIn},
		TokenOut:      testBaseToken,
	})
	require.NoError(t, err)
	resBaseline, err := baseline.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testQuoteToken, Amount: amountIn},
		TokenOut:      testBaseToken,
	})
	require.NoError(t, err)

	assert.Equal(t, resBaseline.TokenAmountOut.Amount, resTaxToken.TokenAmountOut.Amount)
}

func TestCalcAmountIn_Buy_RoundTrip(t *testing.T) {
	sim := buildTestPool(t)

	desiredOut, _ := new(big.Int).SetString("1000000000000000000000", 10) // 1000 base tokens
	inResult, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: testBaseToken, Amount: desiredOut},
		TokenIn:        testQuoteToken,
	})
	require.NoError(t, err)
	require.NotNil(t, inResult.TokenAmountIn)
	assert.Positive(t, inResult.TokenAmountIn.Amount.Sign())

	// CalcAmountOut(CalcAmountIn(x)) >= x
	outResult, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testQuoteToken, Amount: inResult.TokenAmountIn.Amount},
		TokenOut:      testBaseToken,
	})
	require.NoError(t, err)
	assert.True(t, outResult.TokenAmountOut.Amount.Cmp(desiredOut) >= 0)
}

func TestCalcAmountIn_Sell_RoundTrip(t *testing.T) {
	sim := buildTestPool(t)

	desiredOut, _ := new(big.Int).SetString("1000000000000000", 10) // 0.001 quote token
	inResult, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: testQuoteToken, Amount: desiredOut},
		TokenIn:        testBaseToken,
	})
	require.NoError(t, err)
	require.NotNil(t, inResult.TokenAmountIn)
	assert.Positive(t, inResult.TokenAmountIn.Amount.Sign())

	outResult, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testBaseToken, Amount: inResult.TokenAmountIn.Amount},
		TokenOut:      testQuoteToken,
	})
	require.NoError(t, err)
	assert.True(t, outResult.TokenAmountOut.Amount.Cmp(desiredOut) >= 0)
}

func TestCalcAmountIn_Buy_ExceedsDexSupplyThresh(t *testing.T) {
	sim := buildTestPool(t)

	// Requesting more base tokens than remain until graduation must fail rather than silently cap.
	remaining := new(big.Int).Sub(sim.dexSupplyThresh.ToBig(), sim.circulatingSupply.ToBig())
	tooMuch := new(big.Int).Add(remaining, big.NewInt(1))

	_, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: testBaseToken, Amount: tooMuch},
		TokenIn:        testQuoteToken,
	})
	assert.ErrorIs(t, err, ErrSupplyExceedsTotalSupply)
}

func TestCalcAmountIn_NotTradable(t *testing.T) {
	sim := buildTestPool(t)
	sim.status = TokenStatusDEX

	_, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: testBaseToken, Amount: big.NewInt(1e18)},
		TokenIn:        testQuoteToken,
	})
	assert.ErrorIs(t, err, ErrPoolNotTradable)
}
