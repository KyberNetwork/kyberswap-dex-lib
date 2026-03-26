package canonic

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func buildTestPool(t *testing.T) *PoolSimulator {
	t.Helper()

	staticExtra := StaticExtra{
		MAOB:          "0x23469683e25b780DFDC11410a8e83c923caDF125",
		Previewer:     "0xEaeD40cC4bA1e7A2A7CA3f1A22C815B628B074Ea",
		BaseToken:     "0xbase",
		QuoteToken:    "0xquote",
		BaseDecimals:  18,
		QuoteDecimals: 6,
		BaseScale:     "1000000000000000000",
		QuoteScale:    "1000000",
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	require.NoError(t, err)

	extra := Extra{
		MidPrice:       "250000",
		MidPrecision:   "100000",
		OracleUpdAt:    1710000000,
		TakerFee:       30,
		FeeDenom:       "10000",
		MinQuoteTaker:  "1000000",
		MarketState:    0,
		StateExpiresAt: 0,
		RungDenom:      "100000",
		PriceSigfigs:   "6",
		AskRungs:       []uint16{100, 200, 500},
		AskVolumes:     []string{"5000000000000000000", "3000000000000000000", "2000000000000000000"},
		BidRungs:       []uint16{100, 200, 500},
		BidVolumes:     []string{"12500000", "7500000", "5000000"},
	}
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)

	entityPool := entity.Pool{
		Address:  "0x23469683e25b780DFDC11410a8e83c923caDF125",
		Exchange: "canonic",
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: "0xbase", Decimals: 18, Swappable: true},
			{Address: "0xquote", Decimals: 6, Swappable: true},
		},
		Reserves:    entity.PoolReserves{"10000000000000000000", "25000000"},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}

	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)
	return sim
}

func TestCalcAmountOut_BuyBase(t *testing.T) {
	sim := buildTestPool(t)

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xquote", Amount: big.NewInt(5_000_000)},
		TokenOut:      "0xbase",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.TokenAmountOut.Amount.Sign() > 0, "should get positive base out")
	require.True(t, result.Fee.Amount.Sign() > 0, "should have fee")
	require.Equal(t, defaultGas, result.Gas)

	t.Logf("BuyBase: quoteIn=5000000, baseOut=%s, fee=%s",
		result.TokenAmountOut.Amount, result.Fee.Amount)
}

func TestCalcAmountOut_SellBase(t *testing.T) {
	sim := buildTestPool(t)

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xbase", Amount: big.NewInt(1_000_000_000_000_000_000)},
		TokenOut:      "0xquote",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.TokenAmountOut.Amount.Sign() > 0, "should get positive quote out")
	require.True(t, result.Fee.Amount.Sign() > 0, "should have fee")

	t.Logf("SellBase: baseIn=1e18, quoteOut=%s, fee=%s",
		result.TokenAmountOut.Amount, result.Fee.Amount)
}

func TestCalcAmountOut_InvalidToken(t *testing.T) {
	sim := buildTestPool(t)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xunknown", Amount: big.NewInt(1000)},
		TokenOut:      "0xbase",
	})
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestCalcAmountOut_SameToken(t *testing.T) {
	sim := buildTestPool(t)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xbase", Amount: big.NewInt(1000)},
		TokenOut:      "0xbase",
	})
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestCalcAmountOut_ZeroAmount(t *testing.T) {
	sim := buildTestPool(t)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xquote", Amount: big.NewInt(0)},
		TokenOut:      "0xbase",
	})
	require.ErrorIs(t, err, ErrInvalidAmountIn)
}

func TestCalcAmountOut_BelowMinQuoteTaker(t *testing.T) {
	sim := buildTestPool(t)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xquote", Amount: big.NewInt(100)},
		TokenOut:      "0xbase",
	})
	require.ErrorIs(t, err, ErrQuoteAmountTooLow)
}

func TestCalcAmountOut_MarketPaused(t *testing.T) {
	sim := buildTestPool(t)
	sim.marketState = marketStatePaused

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xquote", Amount: big.NewInt(5_000_000)},
		TokenOut:      "0xbase",
	})
	require.ErrorIs(t, err, ErrMarketPaused)
}

func TestCloneState_Isolation(t *testing.T) {
	sim := buildTestPool(t)

	clone := sim.CloneState().(*PoolSimulator)

	result, err := clone.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xquote", Amount: big.NewInt(5_000_000)},
		TokenOut:      "0xbase",
	})
	require.NoError(t, err)

	clone.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "0xquote", Amount: big.NewInt(5_000_000)},
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})

	require.Equal(t, "10000000000000000000", sim.Info.Reserves[0].String())
	require.Equal(t, "25000000", sim.Info.Reserves[1].String())

	require.NotEqual(t, sim.Info.Reserves[0].String(), clone.Info.Reserves[0].String())
}

func TestIdempotency(t *testing.T) {
	sim := buildTestPool(t)

	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xquote", Amount: big.NewInt(5_000_000)},
		TokenOut:      "0xbase",
	}

	r1, err := sim.CalcAmountOut(params)
	require.NoError(t, err)

	r2, err := sim.CalcAmountOut(params)
	require.NoError(t, err)

	require.Equal(t, r1.TokenAmountOut.Amount.Cmp(r2.TokenAmountOut.Amount), 0,
		"CalcAmountOut must be idempotent")
}

func TestMultiHop(t *testing.T) {
	sim := buildTestPool(t)

	r1, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xquote", Amount: big.NewInt(2_000_000)},
		TokenOut:      "0xbase",
	})
	require.NoError(t, err)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "0xquote", Amount: big.NewInt(2_000_000)},
		TokenAmountOut: *r1.TokenAmountOut,
		Fee:            *r1.Fee,
		SwapInfo:       r1.SwapInfo,
	})

	r2, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xbase", Amount: r1.TokenAmountOut.Amount},
		TokenOut:      "0xquote",
	})
	require.NoError(t, err)
	require.True(t, r2.TokenAmountOut.Amount.Sign() > 0, "hop 2 should produce output")

	t.Logf("MultiHop: quoteIn=2000000 -> baseOut=%s -> quoteOut=%s",
		r1.TokenAmountOut.Amount, r2.TokenAmountOut.Amount)
}
