package altfun

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	bouncetech "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/bounce-tech"
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
		uint256.NewInt(1e18),          // exchangeRate 1:1
		uint256.NewInt(1e16),          // 1% redemptionFee
		uint256.NewInt(3e18),          // 3x leverage
		uint256.NewInt(10_000_000),    // 10 USDC min
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

// TestSellSplitChunkBehavior mô phỏng splitAmountIn của pathfinder với real config
// trên Hyperliquid (finderOptions: distributionPercent=4, minPartUSD=40).
//
// Real data: SELLOR/USDC pool (block 36891843) + BTC5S LT bounce-tech (block 36893817).
//
// Key insight từ splitAmountIn logic:
//   - Nếu tokenInPrice = 0 (SELLOR không có giá onchain) → amountInPrice = 0 < minPartUSD
//   - → pathfinder trả về [fullAmount] (single chunk = full amount)
//   - → splits[last] = fullAmount → CalcAmountOut(fullAmount) → no split-chunk problem
//
// Case mà split-chunk problem xảy ra: token CÓ giá, amount vừa đủ để pass
// khi dùng full amount nhưng chunk lại dưới minTxSize của BT contract.
func TestSellSplitChunkBehavior(t *testing.T) {
	const (
		realUSDC = "0xb88339cb7199b77e23db6e890353e22632ba630f"
		realLT   = "0x42e4e7e81bcbb648282d5fa261b2c0840747c8f3"
		realMeme = "0xb6bf399001fe71e01f5a63866b93fc3074700000"
		realZap  = "0x693f12e9e6b35b34458793546065e8b08e0299d6"
	)

	btPool := newBtPoolWithAddrs(t,
		mustU256("2679844238188618895"), // exchangeRate ~2.68 USDC/LT (1e18-scaled)
		mustU256("3000000000000000"),    // redemptionFee 0.3% (1e18-scaled)
		mustU256("5000000000000000000"), // targetLeverage 5x (1e18-scaled)
		mustU256("10000000"),            // minTxSize = 10 USDC (6 dec)
		mustU256("761696667"),           // baseAssetBalance ~761 USDC (6 dec)
		false, realUSDC, realLT,
	)

	s := newAltFunTestSimulatorWithAddrs(t, altFunTestState{
		reserveToken:           mustU256("933413308093566079934095628"),
		reserveAsset:           mustU256("3497288862904305241296"),
		k:                      mustU256("3264415966882293652019000000000000000000000000000"),
		tokenBalance:           mustU256("683413308093566079934095628"),
		graduationThresholdUsd: mustU256("9000000000000000000000"),
		sellFeeBps:             75,
		buyFeeBps:              75,
	}, btPool, realUSDC, realMeme, realLT, realZap)

	// --- Case 1: SELLOR price = 0 (không có giá onchain) ---
	// splitAmountIn: tokenInPrice=0, amountInPrice=0 < minPartUSD=40
	// → trả về [fullAmount] → splits[last] = fullAmount
	t.Run("price=0 → single chunk = full amount (SELLOR thực tế)", func(t *testing.T) {
		fullAmount := mustBig("10000000000000000000000000") // 10M SELLOR
		// splits[last] = fullAmount (pathfinder không split khi không có price)
		res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: realMeme, Amount: fullAmount},
			TokenOut:      realUSDC,
		})
		require.NoError(t, err, "full amount phải pass khi price=0 → pathfinder dùng full amount làm chunk")
		t.Logf("10M SELLOR → %s USDC out", res.TokenAmountOut.Amount)
	})

	// --- Case 2: Token có giá → pathfinder split thành chunks nhỏ ---
	// Reproduce: full amount pass nhưng chunk (splits[last]) fail BelowMinAmount.
	// Ví dụ: full = 10M tokens pass, chunk 1/25 = 400K tokens fail (< 10 USDC output).
	t.Run("price>0 → chunk nhỏ fail nhưng full pass (split-chunk problem)", func(t *testing.T) {
		// distributionPercent=4 → splitNumber=25 → splits[last] = fullAmount/25
		fullAmount := mustBig("10000000000000000000000000") // 10M SELLOR
		chunkAmount := new(big.Int).Div(fullAmount, big.NewInt(25))

		_, chunkErr := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: realMeme, Amount: chunkAmount},
			TokenOut:      realUSDC,
		})
		fullRes, fullErr := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: realMeme, Amount: fullAmount},
			TokenOut:      realUSDC,
		})

		t.Logf("chunk (1/25 = 400K SELLOR) → err: %v", chunkErr)
		if fullErr == nil {
			t.Logf("full  (10M SELLOR)        → %s USDC out", fullRes.TokenAmountOut.Amount)
		}

		// Chunk phải fail, full phải pass → đây là split-chunk problem
		require.Error(t, chunkErr, "chunk nhỏ phải fail BelowMinAmount")
		require.NoError(t, fullErr, "full amount phải pass")
	})

	// --- Case 3: Amount thật sự quá nhỏ → cả chunk lẫn full đều fail (correct behavior) ---
	t.Run("amount too small → full amount cũng fail (onchain cũng revert)", func(t *testing.T) {
		tinyAmount := mustBig("1000000000000000000000") // 1K SELLOR → output << minTxSize
		_, err := s.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: realMeme, Amount: tinyAmount},
			TokenOut:      realUSDC,
		})
		require.Error(t, err, "genuinely tiny amount phải fail")
		t.Logf("1K SELLOR → ErrBelowMinAmount (correct: onchain cũng revert)")
	})
}

func newBtPoolWithAddrs(t *testing.T, exchangeRate, redemptionFee, targetLeverage, minTxSize, baseBalance *uint256.Int, mintPaused bool, usdcAddr, ltAddr string) *bouncetech.PoolSimulator {
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
		Address:  ltAddr,
		Exchange: bouncetech.DexType,
		Type:     bouncetech.DexType,
		Tokens: []*entity.PoolToken{
			{Address: usdcAddr, Swappable: true},
			{Address: ltAddr, Swappable: true},
		},
		Reserves: entity.PoolReserves{
			baseBalance.ToBig().String(),
			"1000000000000000000000000000",
		},
		Extra:       string(btExtraBytes),
		StaticExtra: `{"usdc":"` + usdcAddr + `"}`,
	}
	sim, err := bouncetech.NewPoolSimulator(ep)
	require.NoError(t, err)
	return sim
}

// newAltFunTestSimulatorWithAddrs tạo simulator với token addresses thực tế.
func newAltFunTestSimulatorWithAddrs(t *testing.T, state altFunTestState, btPool *bouncetech.PoolSimulator,
	usdcAddr, memeAddr, ltAddr, zapAddr string,
) *PoolSimulator {
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
		LTAddress:              ltAddr,
		USDC:                   usdcAddr,
		ZapAddress:             zapAddr,
		BuyFeeBps:              state.buyFeeBps,
		SellFeeBps:             state.sellFeeBps,
		BasePool:               ltAddr,
		GraduationThresholdUsd: state.graduationThresholdUsd,
	})
	require.NoError(t, err)

	basePoolMap := map[string]pool.IPoolSimulator{ltAddr: btPool}

	sim, err := NewPoolSimulator(entity.Pool{
		Address:  memeAddr,
		Exchange: DexType,
		Type:     DexType,
		Reserves: entity.PoolReserves{
			state.reserveToken.ToBig().String(),
			state.tokenBalance.ToBig().String(),
		},
		Tokens: []*entity.PoolToken{
			{Address: usdcAddr, Swappable: true},
			{Address: memeAddr, Swappable: true},
		},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}, basePoolMap)
	require.NoError(t, err)
	return sim
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
