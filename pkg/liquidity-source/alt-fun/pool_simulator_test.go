package altfun

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	bouncetech "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bounce-tech"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	testUSDC = "0x0000000000000000000000000000000000000001"
	testLT   = "0x0000000000000000000000000000000000000004"
	testMeme = "0x0000000000000000000000000000000000000002"
	testZap  = "0x0000000000000000000000000000000000000003"
)

// newBtPool creates a bounce-tech PoolSimulator for use as the btPool in tests.
func newBtPool(t *testing.T, exchangeRate, redemptionFee, targetLeverage, minTxSize, baseBalance *uint256.Int, mintPaused bool) *bouncetech.PoolSimulator {
	t.Helper()
	btExtra := bouncetech.Extra{
		ExchangeRate:       exchangeRate,
		RedemptionFee:      redemptionFee,
		TargetLeverage:     targetLeverage,
		MinTransactionSize: minTxSize,
		MintPaused:         mintPaused,
	}
	btExtraBytes, err := json.Marshal(btExtra)
	require.NoError(t, err)

	ep := entity.Pool{
		Address:  testLT,
		Exchange: bouncetech.DexType,
		Type:     bouncetech.DexType,
		Tokens: []*entity.PoolToken{
			{Address: testUSDC, Swappable: true},
			{Address: testLT, Swappable: true},
		},
		Reserves: entity.PoolReserves{
			baseBalance.ToBig().String(),
			"1000000000000000000000000000",
		},
		Extra:       string(btExtraBytes),
		StaticExtra: `{"usdc":"` + testUSDC + `"}`,
	}
	sim, err := bouncetech.NewPoolSimulator(ep)
	require.NoError(t, err)
	return sim
}

func TestCalcAmountOutBuyPresizesAgainstGraduationAndProratesFee(t *testing.T) {
	reserveToken := mustU256("1000000000000000000000000000") // 1B tokens
	reserveAsset := mustU256("3000000000000000000000")       // 3,000 LT
	k := new(uint256.Int).Mul(reserveToken, reserveAsset)

	// graduationThresholdUsd = 50e18 so that at exchangeRate=1e18:
	// thresholdRealLt = 50 LT; launchVirtual = k/TOTAL_SUPPLY = reserveAsset => realLtRaised=0
	// => ltUntilGrad = 50 LT = 50e18
	graduationThresholdUsd := mustU256("50000000000000000000") // 50 USD

	btPool := newBtPool(t,
		uint256.NewInt(1e18),  // exchangeRate 1:1
		uint256.NewInt(1e16),  // 1% redemptionFee
		uint256.NewInt(3e18),  // 3x leverage
		uint256.NewInt(10_000_000), // 10 USDC min
		uint256.NewInt(1_000_000_000), // 1000 USDC base balance
		false,
	)

	s := newAltFunTestSimulator(t, altFunTestState{
		reserveToken:           reserveToken,
		reserveAsset:           reserveAsset,
		k:                      k,
		tokenBalance:           reserveToken,
		graduationThresholdUsd: graduationThresholdUsd,
		buyFeeBps:              100,
	}, btPool)

	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testUSDC, Amount: big.NewInt(100_000_000)},
		TokenOut:      testMeme,
	})
	require.NoError(t, err)

	ltUntilGrad := mustU256("50000000000000000000") // 50 LT
	wantTokensOut := new(uint256.Int).Sub(
		reserveToken,
		new(uint256.Int).Div(k, new(uint256.Int).Add(reserveAsset, ltUntilGrad)),
	)
	require.Equal(t, wantTokensOut.ToBig().String(), res.TokenAmountOut.Amount.String())
	require.Equal(t, "505050", res.Fee.Amount.String())
	require.NotNil(t, res.RemainingTokenAmountIn)
	require.Equal(t, "49494950", res.RemainingTokenAmountIn.Amount.String())

	swapInfo := res.SwapInfo.(SwapInfo)
	require.Equal(t, ltUntilGrad.ToBig().String(), swapInfo.AmountInUsed.ToBig().String())
	require.Equal(t, "50000000", swapInfo.BaseToConvert.ToBig().String())

	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testUSDC, Amount: big.NewInt(100_000_000)},
		TokenAmountOut: *res.TokenAmountOut,
		Fee:            *res.Fee,
		SwapInfo:       res.SwapInfo,
	})

	require.Equal(t, "3050000000000000000000", s.reserveAsset.ToBig().String())
	require.Equal(t,
		new(uint256.Int).Div(k, mustU256("3050000000000000000000")).ToBig().String(),
		s.reserveToken.ToBig().String(),
	)
	// After consuming all ltUntilGrad, lifecycle transitions to Graduating.
	require.Equal(t, LifecycleGraduating, s.lifecycle)
}

func TestCalcAmountOutBuyRejectsPostFeeAmountBelowMin(t *testing.T) {
	btPool := newBtPool(t,
		uint256.NewInt(1e18), uint256.NewInt(1e16), uint256.NewInt(3e18),
		uint256.NewInt(10_000_000), uint256.NewInt(1_000_000_000), false,
	)
	s := newAltFunTestSimulator(t, altFunDefaultState(), btPool)

	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testUSDC, Amount: big.NewInt(10_000_000)},
		TokenOut:      testMeme,
	})
	require.ErrorIs(t, err, ErrBelowMinAmount)
}

func TestCalcAmountOutSellRejectsWhenRedeemGrossExceedsBaseBalance(t *testing.T) {
	// Set very low base balance so the sell will exceed it.
	btPool := newBtPool(t,
		uint256.NewInt(1e18), uint256.NewInt(1e16), uint256.NewInt(3e18),
		uint256.NewInt(10_000_000), uint256.NewInt(20_000_000), false,
	)
	s := newAltFunTestSimulator(t, altFunDefaultState(), btPool)

	_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testMeme, Amount: mustBig("10000000000000000000000000")},
		TokenOut:      testUSDC,
	})
	// Error comes from the bounce-tech base pool (insufficient USDC reserve in LT contract).
	require.Error(t, err)
}

func TestUpdateBalanceSellUpdatesCurveReserves(t *testing.T) {
	btPool := newBtPool(t,
		uint256.NewInt(1e18), uint256.NewInt(1e16), uint256.NewInt(3e18),
		uint256.NewInt(10_000_000), uint256.NewInt(1_000_000_000), false,
	)
	s := newAltFunTestSimulator(t, altFunDefaultState(), btPool)

	tokenIn := mustBig("10000000000000000000000000")
	beforeReserveToken := new(uint256.Int).Set(s.reserveToken)
	beforeReserveAsset := new(uint256.Int).Set(s.reserveAsset)

	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testMeme, Amount: tokenIn},
		TokenOut:      testUSDC,
	})
	require.NoError(t, err)

	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testMeme, Amount: tokenIn},
		TokenAmountOut: *res.TokenAmountOut,
		Fee:            *res.Fee,
		SwapInfo:       res.SwapInfo,
	})

	// Curve reserves updated: reserveToken increases, reserveAsset decreases.
	require.True(t, s.reserveToken.Gt(beforeReserveToken))
	require.True(t, s.reserveAsset.Lt(beforeReserveAsset))
}

type altFunTestState struct {
	reserveToken           *uint256.Int
	reserveAsset           *uint256.Int
	k                      *uint256.Int
	tokenBalance           *uint256.Int
	graduationThresholdUsd *uint256.Int
	buyFeeBps              uint64
	sellFeeBps             uint64
}

func altFunDefaultState() altFunTestState {
	reserveToken := mustU256("1000000000000000000000000000")
	reserveAsset := mustU256("3000000000000000000000")
	return altFunTestState{
		reserveToken:           reserveToken,
		reserveAsset:           reserveAsset,
		k:                      new(uint256.Int).Mul(reserveToken, reserveAsset),
		tokenBalance:           reserveToken,
		graduationThresholdUsd: mustU256("1000000000000000000000000"), // large threshold
		buyFeeBps:              100,
		sellFeeBps:             50,
	}
}

func newAltFunTestSimulator(t *testing.T, state altFunTestState, btPool *bouncetech.PoolSimulator) *PoolSimulator {
	t.Helper()

	extraBytes, err := json.Marshal(Extra{
		ReserveToken: state.reserveToken,
		ReserveAsset: state.reserveAsset,
		K:            state.k,
		TokenBalance: state.tokenBalance,
		Lifecycle:    LifecycleCurve,
	})
	require.NoError(t, err)

	staticExtraBytes, err := json.Marshal(StaticExtra{
		LTAddress:              testLT,
		USDC:                   testUSDC,
		ZapAddress:             testZap,
		BuyFeeBps:              state.buyFeeBps,
		SellFeeBps:             state.sellFeeBps,
		BasePool:               testLT,
		GraduationThresholdUsd: state.graduationThresholdUsd,
	})
	require.NoError(t, err)

	basePoolMap := map[string]pool.IPoolSimulator{
		testLT: btPool,
	}

	s, err := NewPoolSimulator(entity.Pool{
		Address:  testMeme,
		Exchange: DexType,
		Type:     DexType,
		Reserves: entity.PoolReserves{
			state.reserveToken.ToBig().String(),
			state.tokenBalance.ToBig().String(),
		},
		Tokens: []*entity.PoolToken{
			{Address: testUSDC, Swappable: true},
			{Address: testMeme, Swappable: true},
		},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}, basePoolMap)
	require.NoError(t, err)
	return s
}

func mustU256(s string) *uint256.Int {
	return uint256.MustFromDecimal(s)
}

func mustBig(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic("invalid big int")
	}
	return v
}
