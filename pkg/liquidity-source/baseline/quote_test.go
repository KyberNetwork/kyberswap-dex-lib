package baseline

import (
	"errors"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	testReserveToken = "0x0000000000000000000000000000000000000001"
	testBToken       = "0x0000000000000000000000000000000000000002"
)

func TestCalcAmountOut_BuyExactInReportsOptimizedExecutionFee(t *testing.T) {
	state := newBaselineQuoteTestState()
	sim := newBaselineQuoteTestSimulator(t, state)
	amountInCandidates := []*big.Int{
		big.NewInt(1),
		mustTestBI(t, "1000000000000"),
		mustTestBI(t, "1000000000000000"),
		mustTestBI(t, "1000000000000000000"),
		mustTestBI(t, "10000000000000000000"),
		mustTestBI(t, "123456789012345678901"),
	}

	result, exactOutQuote, dust := quoteBuyExactInWithDust(t, state, sim, amountInCandidates)

	if result.Fee.Amount.Cmp(exactOutQuote.Fee.ToBig()) != 0 {
		t.Fatalf("router fee mismatch with %s max-input dust: want optimized exact-out fee %s, got %s", dust, exactOutQuote.Fee, result.Fee.Amount)
	}
	if result.RemainingTokenAmountIn == nil {
		t.Fatal("expected buy exact-in optimized execution to expose unspent max-input dust")
	}
	if result.RemainingTokenAmountIn.Amount.Cmp(dust) != 0 {
		t.Fatalf("remaining input mismatch: want %s got %s", dust, result.RemainingTokenAmountIn.Amount)
	}
}

func TestCalcAmountOutRejectsSameTokenDirection(t *testing.T) {
	sim := newBaselineQuoteTestSimulator(t, newBaselineQuoteTestState())

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testReserveToken, Amount: big.NewInt(1)},
		TokenOut:      testReserveToken,
	})
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for reserve->reserve quote, got %v", err)
	}

	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testBToken, Amount: big.NewInt(1)},
		TokenOut:      testBToken,
	})
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for bToken->bToken quote, got %v", err)
	}
}

func TestCalcAmountInRejectsSameTokenDirection(t *testing.T) {
	sim := newBaselineQuoteTestSimulator(t, newBaselineQuoteTestState())

	_, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: testReserveToken, Amount: big.NewInt(1)},
		TokenIn:        testReserveToken,
	})
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for reserve->reserve quote, got %v", err)
	}

	_, err = sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: testBToken, Amount: big.NewInt(1)},
		TokenIn:        testBToken,
	})
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for bToken->bToken quote, got %v", err)
	}
}

func quoteBuyExactInWithDust(
	t *testing.T,
	state *QuoteState,
	sim *PoolSimulator,
	amountInCandidates []*big.Int,
) (*pool.CalcAmountOutResult, *quoteResult, *big.Int) {
	t.Helper()

	for _, baseAmountIn := range amountInCandidates {
		for _, offset := range []int64{0, 1, 7, 13, 123, 997, 10_000, 1_000_000} {
			amountIn := addBI(baseAmountIn, big.NewInt(offset))
			result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: testReserveToken, Amount: amountIn},
				TokenOut:      testBToken,
			})
			if err != nil {
				continue
			}

			exactOutQuote, err := quoteBuyExactOut(cloneQuoteState(state), result.TokenAmountOut.Amount)
			if err != nil {
				t.Fatalf("quoteBuyExactOut for optimized execution amount %s failed: %v", result.TokenAmountOut.Amount, err)
			}
			dust := new(big.Int).Sub(amountIn, exactOutQuote.AmountOut.ToBig())
			if dust.Sign() > 0 {
				return result, exactOutQuote, dust
			}
		}
	}

	for offset := int64(0); offset < 10_000; offset++ {
		amountIn := addBI(amountInCandidates[len(amountInCandidates)-1], big.NewInt(offset))
		result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: testReserveToken, Amount: amountIn},
			TokenOut:      testBToken,
		})
		if err != nil {
			t.Fatalf("CalcAmountOut buy exact-in failed for amount %s: %v", amountIn, err)
		}

		exactOutQuote, err := quoteBuyExactOut(cloneQuoteState(state), result.TokenAmountOut.Amount)
		if err != nil {
			t.Fatalf("quoteBuyExactOut for optimized execution amount %s failed: %v", result.TokenAmountOut.Amount, err)
		}
		dust := new(big.Int).Sub(amountIn, exactOutQuote.AmountOut.ToBig())
		if dust.Sign() > 0 {
			return result, exactOutQuote, dust
		}
	}

	t.Fatalf("test setup must find positive max-input dust")
	return nil, nil, nil
}

func newBaselineQuoteTestSimulator(t *testing.T, state *QuoteState) *PoolSimulator {
	t.Helper()

	extra, err := json.Marshal(Extra{QuoteState: state})
	if err != nil {
		t.Fatalf("marshal extra: %v", err)
	}

	sim, err := NewPoolSimulator(entity.Pool{
		Address:  testBToken,
		Exchange: "baseline",
		Type:     DexType,
		Reserves: entity.PoolReserves{
			uToBI(state.TotalReserves).String(),
			uToBI(state.TotalBTokens).String(),
		},
		Tokens: []*entity.PoolToken{
			{Address: testReserveToken, Decimals: state.ReserveDecimals, Swappable: true},
			{Address: testBToken, Decimals: bTokenDecimals, Swappable: true},
		},
		Extra: string(extra),
	})
	if err != nil {
		t.Fatalf("NewPoolSimulator failed: %v", err)
	}
	return sim
}

func newBaselineQuoteTestState() *QuoteState {
	return &QuoteState{
		SnapshotCurveParams: CurveParams{
			BLV:           uint256.MustFromDecimal("2000000000000000000"),
			Circ:          uint256.MustFromDecimal("500000000000000000000000"),
			Supply:        uint256.MustFromDecimal("500000000000000000000000"),
			SwapFee:       uint256.MustFromDecimal("3000000000000000"),
			Reserves:      uint256.MustFromDecimal("1500000000000000000000000"),
			TotalSupply:   uint256.MustFromDecimal("1000000000000000000000000"),
			ConvexityExp:  uint256.MustFromDecimal("2000000000000000000"),
			LastInvariant: uint256.MustFromDecimal("500000000000000000000000"),
		},
		QuoteBlockBuyDeltaCirc:  uint256.NewInt(0),
		QuoteBlockSellDeltaCirc: uint256.NewInt(0),
		TotalSupply:             uint256.MustFromDecimal("1000000000000000000000000"),
		TotalBTokens:            uint256.MustFromDecimal("500000000000000000000000"),
		TotalReserves:           uint256.MustFromDecimal("1500000000000000000000000"),
		ReserveDecimals:         18,
		LiquidityFeePct:         uint256.MustFromDecimal("1000000000000000000"),
		PendingSurplus:          uint256.NewInt(0),
		MaxSellDelta:            uint256.MustFromDecimal("100000000000000000000000"),
	}
}
