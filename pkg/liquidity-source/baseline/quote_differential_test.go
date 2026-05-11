package baseline

import (
	"context"
	"errors"
	"math/big"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type baselineDifferentialEnv struct {
	rpcURL      string
	relay       string
	bToken      string
	reserve     string
	blockNumber *big.Int
}

func TestQuoteErrorParitySelectorMapping(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		wantErr  error
	}{
		{
			name:     "PriceMustChange",
			selector: "0x241b0fb3",
			wantErr:  errPriceMustChange,
		},
		{
			name:     "TradeExceedsLimit",
			selector: "0x8a313469",
			wantErr:  errTradeExceedsLimit,
		},
		{
			name:     "SolverFailed",
			selector: "0x308ab3c2",
			wantErr:  errSolverFailed,
		},
		{
			name:     "InvalidActivePrice",
			selector: "0x82975b38",
			wantErr:  errInvalidCurveState,
		},
		{
			name:     "BlockPricingLib_SellExceedsSameBlockCapacity",
			selector: "0x6d6b15dc",
			wantErr:  errTradeExceedsLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotErr, ok := quoteErrorParitySelectorMapping(
				errors.New("execution reverted with custom error selector " + tt.selector),
			)
			if !ok {
				t.Fatalf("expected selector %s to map to %s", tt.selector, tt.name)
			}
			if gotName != tt.name {
				t.Fatalf("selector %s mapped to name %s, want %s", tt.selector, gotName, tt.name)
			}
			if !errors.Is(gotErr, tt.wantErr) {
				t.Fatalf("selector %s mapped to error %v, want %v", tt.selector, gotErr, tt.wantErr)
			}
		})
	}
}

func loadBaselineDifferentialEnv(t *testing.T) baselineDifferentialEnv {
	t.Helper()

	env := baselineDifferentialEnv{
		rpcURL:  os.Getenv("BASELINE_RPC_URL"),
		relay:   os.Getenv("BASELINE_RELAY_ADDRESS"),
		bToken:  os.Getenv("BASELINE_BTOKEN_ADDRESS"),
		reserve: os.Getenv("BASELINE_RESERVE_ADDRESS"),
	}
	if env.rpcURL == "" || env.relay == "" || env.bToken == "" || env.reserve == "" {
		t.Skip("Set BASELINE_RPC_URL, BASELINE_RELAY_ADDRESS, BASELINE_BTOKEN_ADDRESS, and BASELINE_RESERVE_ADDRESS to run Baseline quote differential tests")
	}

	if rawBlock := os.Getenv("BASELINE_BLOCK_NUMBER"); rawBlock != "" {
		blockNumber, ok := new(big.Int).SetString(rawBlock, 10)
		if !ok {
			t.Fatalf("invalid BASELINE_BLOCK_NUMBER: %q", rawBlock)
		}
		env.blockNumber = blockNumber
	}

	return env
}

func newBaselineDifferentialClient(env baselineDifferentialEnv) *ethrpc.Client {
	return ethrpc.New(env.rpcURL)
}

func TestBaselineQuoteDifferential_ExactIn(t *testing.T) {
	env := loadBaselineDifferentialEnv(t)
	ethrpcClient := newBaselineDifferentialClient(env)
	ctx := context.Background()

	state := fetchDifferentialQuoteState(t, ctx, ethrpcClient, env)
	sim := newDifferentialSimulator(t, env, state)

	t.Run("quoteBuyExactIn", func(t *testing.T) {
		for _, amountIn := range reserveExactInAmounts(state) {
			t.Run(amountIn.String(), func(t *testing.T) {
				goQuote, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: env.reserve, Amount: amountIn},
					TokenOut:      env.bToken,
				})
				solidity, solidityErr := callQuoteBuyExactIn(t, ctx, ethrpcClient, env, amountIn)
				assertQuoteErrorParity(t, methodQuoteBuyExactIn, solidityErr, err)
				if solidityErr != nil {
					return
				}
				if err != nil {
					t.Fatalf("Go quoteBuyExactIn failed: %v", err)
				}
				assertBigEqual(t, "tokensOut", solidity.amount, goQuote.TokenAmountOut.Amount)

				optimized, optimizedErr := callQuoteBuyExactOut(t, ctx, ethrpcClient, env, solidity.amount)
				if optimizedErr != nil {
					t.Fatalf("Solidity quoteBuyExactOut(%s) failed: %v", solidity.amount, optimizedErr)
				}
				assertBigEqual(t, "feesReceived", optimized.fee, goQuote.Fee.Amount)
				assertOptionalRemainingAmount(t, "remainingTokenAmountIn", subBI(amountIn, optimized.amount), goQuote.RemainingTokenAmountIn)
			})
		}
	})

	t.Run("quoteSellExactIn", func(t *testing.T) {
		for _, amountIn := range sellExactInAmounts(state) {
			t.Run(amountIn.String(), func(t *testing.T) {
				goQuote, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: env.bToken, Amount: amountIn},
					TokenOut:      env.reserve,
				})
				solidity, solidityErr := callQuoteSellExactIn(t, ctx, ethrpcClient, env, amountIn)
				assertQuoteErrorParity(t, methodQuoteSellExactIn, solidityErr, err)
				if solidityErr != nil {
					return
				}
				if err != nil {
					t.Fatalf("Go quoteSellExactIn failed: %v", err)
				}
				assertBigEqual(t, "amountOut", solidity.amount, goQuote.TokenAmountOut.Amount)
				assertBigEqual(t, "feesReceived", solidity.fee, goQuote.Fee.Amount)
			})
		}
	})
}

func TestBaselineQuoteDifferential_ExactOut(t *testing.T) {
	env := loadBaselineDifferentialEnv(t)
	ethrpcClient := newBaselineDifferentialClient(env)
	ctx := context.Background()

	state := fetchDifferentialQuoteState(t, ctx, ethrpcClient, env)
	sim := newDifferentialSimulator(t, env, state)

	t.Run("quoteBuyExactOut", func(t *testing.T) {
		for _, amountOut := range buyExactOutAmounts(state) {
			t.Run(amountOut.String(), func(t *testing.T) {
				goQuote, err := sim.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: pool.TokenAmount{Token: env.bToken, Amount: amountOut},
					TokenIn:        env.reserve,
				})
				solidity, solidityErr := callQuoteBuyExactOut(t, ctx, ethrpcClient, env, amountOut)
				assertQuoteErrorParity(t, methodQuoteBuyExactOut, solidityErr, err)
				if solidityErr != nil {
					return
				}
				if err != nil {
					t.Fatalf("Go quoteBuyExactOut failed: %v", err)
				}
				assertBigEqual(t, "amountIn", solidity.amount, goQuote.TokenAmountIn.Amount)
				assertBigEqual(t, "feesReceived", solidity.fee, goQuote.Fee.Amount)
			})
		}
	})

	t.Run("quoteSellExactOut", func(t *testing.T) {
		for _, reservesOut := range sellExactOutAmounts(state) {
			t.Run(reservesOut.String(), func(t *testing.T) {
				goQuote, err := sim.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: pool.TokenAmount{Token: env.reserve, Amount: reservesOut},
					TokenIn:        env.bToken,
				})
				solidity, solidityErr := callQuoteSellExactOut(t, ctx, ethrpcClient, env, reservesOut)
				assertQuoteErrorParity(t, methodQuoteSellExactOut, solidityErr, err)
				if solidityErr != nil {
					return
				}
				if err != nil {
					t.Fatalf("Go quoteSellExactOut failed: %v", err)
				}
				assertBigEqual(t, "tokensIn", solidity.amount, goQuote.TokenAmountIn.Amount)
				assertBigEqual(t, "feesReceived", solidity.fee, goQuote.Fee.Amount)
			})
		}
	})
}

func TestBaselineQuoteDifferential_SequentialBuyState(t *testing.T) {
	env := loadBaselineDifferentialEnv(t)
	requireCast(t)

	amountIn := decimalUnit(18)

	fundSequentialTrader(t, env, mulBI(amountIn, big.NewInt(3)))

	ethrpcClient := newBaselineDifferentialClient(env)
	ctx := context.Background()
	state := fetchDifferentialQuoteState(t, ctx, ethrpcClient, env)
	sim := newDifferentialSimulator(t, env, state)

	goQuote, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: env.reserve, Amount: amountIn},
		TokenOut:      env.bToken,
	})
	if err != nil {
		t.Fatalf("Go quoteBuyExactIn failed: %v", err)
	}
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: env.reserve, Amount: amountIn},
		TokenAmountOut: *goQuote.TokenAmountOut,
		SwapInfo:       goQuote.SwapInfo,
	})

	sendBuyExactIn(t, env, amountIn, false)

	nextBuyAmountIn := decimalUnit(18)
	nextSolidityBuy, solidityErr := callQuoteBuyExactIn(t, ctx, ethrpcClient, env, nextBuyAmountIn)
	nextGoBuy, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: env.reserve, Amount: nextBuyAmountIn},
		TokenOut:      env.bToken,
	})
	assertQuoteErrorParity(t, methodQuoteBuyExactIn, solidityErr, err)
	if solidityErr == nil {
		if err != nil {
			t.Fatalf("Go post-buy quoteBuyExactIn failed: %v", err)
		}
		assertBigEqual(t, "post-buy tokensOut", nextSolidityBuy.amount, nextGoBuy.TokenAmountOut.Amount)
		assertBigEqual(t, "post-buy buy feesReceived", nextSolidityBuy.fee, nextGoBuy.Fee.Amount)
	}

	nextSellAmountIn := decimalUnit(18)
	nextSoliditySell, solidityErr := callQuoteSellExactIn(t, ctx, ethrpcClient, env, nextSellAmountIn)
	nextGoSell, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: env.bToken, Amount: nextSellAmountIn},
		TokenOut:      env.reserve,
	})
	assertQuoteErrorParity(t, methodQuoteSellExactIn, solidityErr, err)
	if solidityErr == nil {
		if err != nil {
			t.Fatalf("Go post-buy quoteSellExactIn failed: %v", err)
		}
		assertBigEqual(t, "post-buy amountOut", nextSoliditySell.amount, nextGoSell.TokenAmountOut.Amount)
		assertBigEqual(t, "post-buy sell feesReceived", nextSoliditySell.fee, nextGoSell.Fee.Amount)
	}
}

func TestBaselineQuoteDifferential_SellUsesQuoteEffectiveReserveLimit(t *testing.T) {
	env := loadBaselineDifferentialEnv(t)
	ethrpcClient := newBaselineDifferentialClient(env)
	ctx := context.Background()

	state := fetchDifferentialQuoteState(t, ctx, ethrpcClient, env)
	sim := newDifferentialSimulator(t, env, state)
	sim.Info.Reserves[0] = big.NewInt(0)

	amountIn := decimalUnit(18)
	solidity, solidityErr := callQuoteSellExactIn(t, ctx, ethrpcClient, env, amountIn)
	goQuote, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: env.bToken, Amount: amountIn},
		TokenOut:      env.reserve,
	})
	assertQuoteErrorParity(t, methodQuoteSellExactIn, solidityErr, err)
	if solidityErr != nil {
		return
	}
	if err != nil {
		t.Fatalf("Go quoteSellExactIn failed with raw reserves below output: %v", err)
	}
	assertBigEqual(t, "sell amountOut", solidity.amount, goQuote.TokenAmountOut.Amount)
	assertBigEqual(t, "sell feesReceived", solidity.fee, goQuote.Fee.Amount)
}

func TestBaselineQuoteDifferential_MixedSameBlockAndMultiBlockSequence(t *testing.T) {
	env := loadBaselineDifferentialEnv(t)
	requireCast(t)

	ctx := context.Background()
	ethrpcClient := newBaselineDifferentialClient(env)
	fundSequentialTrader(t, env, mulBI(decimalUnit(18), big.NewInt(10)))

	state := fetchDifferentialQuoteState(t, ctx, ethrpcClient, env)
	sim := newDifferentialSimulator(t, env, state)

	sameBlockSwaps := []sequentialSwap{
		{kind: sequentialSwapBuyExactIn, amount: decimalUnit(18)},
		{kind: sequentialSwapBuyExactOut, amount: divBI(decimalUnit(18), twoBI)},
		{kind: sequentialSwapSellExactIn, amount: decimalUnit(18)},
		{kind: sequentialSwapSellExactOut, amount: divBI(decimalUnit(18), big.NewInt(10))},
	}

	runCast(t, "rpc", "evm_setAutomine", "false", "--rpc-url", env.rpcURL)
	t.Cleanup(func() {
		runCast(t, "rpc", "evm_setAutomine", "true", "--rpc-url", env.rpcURL)
	})
	nonce := castNonce(t, env)
	for _, swap := range sameBlockSwaps {
		quoteAndApplySequentialSwap(t, sim, swap)
		sendSequentialSwapAsync(t, env, swap, nonce)
		nonce++
	}
	runCast(t, "rpc", "evm_mine", "--rpc-url", env.rpcURL)
	runCast(t, "rpc", "evm_setAutomine", "true", "--rpc-url", env.rpcURL)

	assertPostSequenceQuotes(t, ctx, ethrpcClient, env, sim, "same-block mixed sequence")

	runCast(t, "rpc", "evm_mine", "--rpc-url", env.rpcURL)
	freshState := fetchDifferentialQuoteState(t, ctx, ethrpcClient, env)
	if uToBI(freshState.QuoteBlockBuyDeltaCirc).Sign() != 0 || uToBI(freshState.QuoteBlockSellDeltaCirc).Sign() != 0 {
		t.Fatalf("expected stale same-block accumulators to be zero after mining a new block: buy=%s sell=%s",
			uToBI(freshState.QuoteBlockBuyDeltaCirc),
			uToBI(freshState.QuoteBlockSellDeltaCirc),
		)
	}
	sim = newDifferentialSimulator(t, env, freshState)

	multiBlockSwaps := []sequentialSwap{
		{kind: sequentialSwapBuyExactIn, amount: divBI(decimalUnit(18), big.NewInt(3))},
		{kind: sequentialSwapBuyExactOut, amount: divBI(decimalUnit(18), big.NewInt(4))},
		{kind: sequentialSwapSellExactIn, amount: divBI(decimalUnit(18), big.NewInt(5))},
		{kind: sequentialSwapSellExactOut, amount: divBI(decimalUnit(18), big.NewInt(12))},
	}
	for i, swap := range multiBlockSwaps {
		quoteAndApplySequentialSwap(t, sim, swap)
		sendSequentialSwap(t, env, swap)
		runCast(t, "rpc", "evm_mine", "--rpc-url", env.rpcURL)

		freshState = fetchDifferentialQuoteState(t, ctx, ethrpcClient, env)
		if uToBI(freshState.QuoteBlockBuyDeltaCirc).Sign() != 0 || uToBI(freshState.QuoteBlockSellDeltaCirc).Sign() != 0 {
			t.Fatalf("expected zero accumulators after multi-block step %d: buy=%s sell=%s",
				i,
				uToBI(freshState.QuoteBlockBuyDeltaCirc),
				uToBI(freshState.QuoteBlockSellDeltaCirc),
			)
		}
		sim = newDifferentialSimulator(t, env, freshState)
		assertPostSequenceQuotes(t, ctx, ethrpcClient, env, sim, "multi-block mixed sequence")
	}
}

type contractQuote struct {
	amount *big.Int
	fee    *big.Int
}

type sequentialSwap struct {
	kind   string
	amount *big.Int
}

const anvilPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

const (
	sequentialSwapBuyExactIn   = "buyExactIn"
	sequentialSwapBuyExactOut  = "buyExactOut"
	sequentialSwapSellExactIn  = "sellExactIn"
	sequentialSwapSellExactOut = "sellExactOut"
)

var sequentialTrader = common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")

func fetchDifferentialQuoteState(
	t *testing.T,
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	env baselineDifferentialEnv,
) *QuoteState {
	t.Helper()

	var result rpcGetQuoteStateResult
	req := ethrpcClient.NewRequest().SetContext(ctx)
	if env.blockNumber != nil {
		req.SetBlockNumber(env.blockNumber)
	}
	req.AddCall(&ethrpc.Call{
		ABI:    relayABI,
		Target: env.relay,
		Method: methodGetQuoteState,
		Params: []any{common.HexToAddress(env.bToken)},
	}, []any{&result})

	if _, err := req.Call(); err != nil {
		t.Fatalf("getQuoteState failed: %v", err)
	}

	return result.State.toQuoteState()
}

func callQuoteBuyExactIn(
	t *testing.T,
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	env baselineDifferentialEnv,
	amountIn *big.Int,
) (contractQuote, error) {
	t.Helper()

	var quote struct{ TokensOut, FeesReceived, Slippage *big.Int }
	if err := callDifferentialQuote(t, ctx, ethrpcClient, env, methodQuoteBuyExactIn, amountIn, &quote); err != nil {
		return contractQuote{}, err
	}
	return contractQuote{amount: nonNilBI(quote.TokensOut), fee: nonNilBI(quote.FeesReceived)}, nil
}

func callQuoteBuyExactOut(
	t *testing.T,
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	env baselineDifferentialEnv,
	amountOut *big.Int,
) (contractQuote, error) {
	t.Helper()

	var quote struct{ AmountIn, FeesReceived, Slippage *big.Int }
	if err := callDifferentialQuote(t, ctx, ethrpcClient, env, methodQuoteBuyExactOut, amountOut, &quote); err != nil {
		return contractQuote{}, err
	}
	return contractQuote{amount: nonNilBI(quote.AmountIn), fee: nonNilBI(quote.FeesReceived)}, nil
}

func callQuoteSellExactIn(
	t *testing.T,
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	env baselineDifferentialEnv,
	amountIn *big.Int,
) (contractQuote, error) {
	t.Helper()

	var quote struct{ AmountOut, FeesReceived, Slippage *big.Int }
	if err := callDifferentialQuote(t, ctx, ethrpcClient, env, methodQuoteSellExactIn, amountIn, &quote); err != nil {
		return contractQuote{}, err
	}
	return contractQuote{amount: nonNilBI(quote.AmountOut), fee: nonNilBI(quote.FeesReceived)}, nil
}

func callQuoteSellExactOut(
	t *testing.T,
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	env baselineDifferentialEnv,
	reservesOut *big.Int,
) (contractQuote, error) {
	t.Helper()

	var quote struct{ TokensIn, FeesReceived, Slippage *big.Int }
	if err := callDifferentialQuote(t, ctx, ethrpcClient, env, methodQuoteSellExactOut, reservesOut, &quote); err != nil {
		return contractQuote{}, err
	}
	return contractQuote{amount: nonNilBI(quote.TokensIn), fee: nonNilBI(quote.FeesReceived)}, nil
}

func callDifferentialQuote(
	t *testing.T,
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	env baselineDifferentialEnv,
	method string,
	amount *big.Int,
	output any,
) error {
	t.Helper()

	req := ethrpcClient.NewRequest().SetContext(ctx)
	if env.blockNumber != nil {
		req.SetBlockNumber(env.blockNumber)
	}
	req.AddCall(&ethrpc.Call{
		ABI:    relayABI,
		Target: env.relay,
		Method: method,
		Params: []any{common.HexToAddress(env.bToken), amount},
	}, []any{output})

	if _, err := req.Call(); err != nil {
		return err
	}
	return nil
}

func newDifferentialSimulator(t *testing.T, env baselineDifferentialEnv, state *QuoteState) *PoolSimulator {
	t.Helper()

	extraBytes, err := json.Marshal(Extra{
		RelayAddress: env.relay,
		QuoteState:   state,
	})
	if err != nil {
		t.Fatalf("marshal extra: %v", err)
	}

	sim, err := NewPoolSimulator(entity.Pool{
		Address:  env.bToken,
		Exchange: "baseline",
		Type:     DexType,
		Reserves: entity.PoolReserves{
			uToBI(state.TotalReserves).String(),
			uToBI(state.TotalBTokens).String(),
		},
		Tokens: []*entity.PoolToken{
			{Address: env.reserve, Decimals: state.ReserveDecimals, Swappable: true},
			{Address: env.bToken, Decimals: bTokenDecimals, Swappable: true},
		},
		Extra: string(extraBytes),
	})
	if err != nil {
		t.Fatalf("NewPoolSimulator failed: %v", err)
	}
	return sim
}

func reserveExactInAmounts(state *QuoteState) []*big.Int {
	unit := decimalUnit(state.ReserveDecimals)
	return uniquePositiveAmounts(
		big.NewInt(1),
		unit,
		divBI(uToBI(state.TotalReserves), big.NewInt(1_000_000)),
		divBI(uToBI(state.TotalReserves), big.NewInt(10_000)),
		divBI(uToBI(state.TotalReserves), big.NewInt(1_000)),
	)
}

func sellExactInAmounts(state *QuoteState) []*big.Int {
	maxSell := uToBI(state.MaxSellDelta)
	return uniquePositiveAmounts(
		big.NewInt(1),
		decimalUnit(bTokenDecimals),
		divBI(maxSell, big.NewInt(10_000)),
		divBI(maxSell, big.NewInt(1_000)),
		divBI(maxSell, big.NewInt(100)),
	)
}

func buyExactOutAmounts(state *QuoteState) []*big.Int {
	totalBTokens := uToBI(state.TotalBTokens)
	return uniquePositiveAmounts(
		big.NewInt(1),
		decimalUnit(bTokenDecimals),
		divBI(totalBTokens, big.NewInt(10_000)),
		divBI(totalBTokens, big.NewInt(1_000)),
		divBI(totalBTokens, big.NewInt(100)),
	)
}

func sellExactOutAmounts(state *QuoteState) []*big.Int {
	totalReserves := uToBI(state.TotalReserves)
	return uniquePositiveAmounts(
		big.NewInt(1),
		decimalUnit(state.ReserveDecimals),
		divBI(totalReserves, big.NewInt(1_000_000)),
		divBI(totalReserves, big.NewInt(10_000)),
		divBI(totalReserves, big.NewInt(1_000)),
	)
}

func uniquePositiveAmounts(candidates ...*big.Int) []*big.Int {
	amounts := make([]*big.Int, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if candidate == nil || candidate.Sign() <= 0 {
			continue
		}
		key := candidate.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		amounts = append(amounts, new(big.Int).Set(candidate))
	}
	return amounts
}

func decimalUnit(decimals uint8) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
}

func assertBigEqual(t *testing.T, label string, expected, actual *big.Int) {
	t.Helper()

	if expected.Cmp(actual) != 0 {
		t.Fatalf("%s mismatch: solidity=%s go=%s diff=%s", label, expected, actual, new(big.Int).Sub(actual, expected))
	}
}

func assertOptionalRemainingAmount(t *testing.T, label string, expected *big.Int, actual *pool.TokenAmount) {
	t.Helper()

	if expected.Sign() == 0 {
		if actual == nil || actual.Amount == nil || actual.Amount.Sign() == 0 {
			return
		}
		t.Fatalf("%s mismatch: expected zero or nil, got %s", label, actual.Amount)
	}
	if actual == nil || actual.Amount == nil {
		t.Fatalf("%s mismatch: expected %s, got nil", label, expected)
	}
	assertBigEqual(t, label, expected, actual.Amount)
}

func assertQuoteErrorParity(t *testing.T, method string, solidityErr, goErr error) {
	t.Helper()

	if solidityErr == nil {
		return
	}
	solidityErrorName, expectedGoErr, ok := quoteErrorParitySelectorMapping(solidityErr)
	if !ok {
		t.Fatalf("%s unexpected Solidity error: %v", method, solidityErr)
	}
	if !errors.Is(goErr, expectedGoErr) {
		t.Fatalf("%s Solidity reverted with %s, but Go error was %v", method, solidityErrorName, goErr)
	}
}

type quoteErrorParitySelector struct {
	selector string
	name     string
	goErr    error
}

var quoteErrorParitySelectors = []quoteErrorParitySelector{
	{
		selector: "0x241b0fb3",
		name:     "PriceMustChange",
		goErr:    errPriceMustChange,
	},
	{
		selector: "0x8a313469",
		name:     "TradeExceedsLimit",
		goErr:    errTradeExceedsLimit,
	},
	{
		selector: "0x308ab3c2",
		name:     "SolverFailed",
		goErr:    errSolverFailed,
	},
	{
		selector: "0x82975b38",
		name:     "InvalidActivePrice",
		goErr:    errInvalidCurveState,
	},
	{
		selector: "0x6d6b15dc",
		name:     "BlockPricingLib_SellExceedsSameBlockCapacity",
		goErr:    errTradeExceedsLimit,
	},
}

func quoteErrorParitySelectorMapping(solidityErr error) (string, error, bool) {
	if solidityErr == nil {
		return "", nil, false
	}

	errText := solidityErr.Error()
	for _, selector := range quoteErrorParitySelectors {
		if strings.Contains(errText, selector.selector) {
			return selector.name, selector.goErr, true
		}
	}

	return "", nil, false
}

// nolint: unused
func assertQuoteStateEqual(t *testing.T, expected, actual *QuoteState) {
	t.Helper()

	assertCurveParamsEqual(t, "snapshotCurveParams", expected.SnapshotCurveParams, actual.SnapshotCurveParams)
	assertU256Equal(t, "quoteBlockBuyDeltaCirc", expected.QuoteBlockBuyDeltaCirc, actual.QuoteBlockBuyDeltaCirc)
	assertU256Equal(t, "quoteBlockSellDeltaCirc", expected.QuoteBlockSellDeltaCirc, actual.QuoteBlockSellDeltaCirc)
	assertU256Equal(t, "totalSupply", expected.TotalSupply, actual.TotalSupply)
	assertU256Equal(t, "totalBTokens", expected.TotalBTokens, actual.TotalBTokens)
	assertU256Equal(t, "totalReserves", expected.TotalReserves, actual.TotalReserves)
	if expected.ReserveDecimals != actual.ReserveDecimals {
		t.Fatalf("reserveDecimals mismatch: local=%d onchain=%d", expected.ReserveDecimals, actual.ReserveDecimals)
	}
	assertU256Equal(t, "liquidityFeePct", expected.LiquidityFeePct, actual.LiquidityFeePct)
	assertU256Equal(t, "maxSellDelta", expected.MaxSellDelta, actual.MaxSellDelta)
	assertU256Equal(t, "snapshotActivePrice", expected.SnapshotActivePrice, actual.SnapshotActivePrice)
}

// nolint: unused
func assertCurveParamsEqual(t *testing.T, label string, expected, actual CurveParams) {
	t.Helper()

	assertU256Equal(t, label+".BLV", expected.BLV, actual.BLV)
	assertU256Equal(t, label+".Circ", expected.Circ, actual.Circ)
	assertU256Equal(t, label+".Supply", expected.Supply, actual.Supply)
	assertU256Equal(t, label+".SwapFee", expected.SwapFee, actual.SwapFee)
	assertU256Equal(t, label+".Reserves", expected.Reserves, actual.Reserves)
	assertU256Equal(t, label+".TotalSupply", expected.TotalSupply, actual.TotalSupply)
	assertU256Equal(t, label+".ConvexityExp", expected.ConvexityExp, actual.ConvexityExp)
	assertU256Equal(t, label+".LastInvariant", expected.LastInvariant, actual.LastInvariant)
}

// nolint: unused
func assertU256Equal(t *testing.T, label string, expected, actual *uint256.Int) {
	t.Helper()

	if uToBI(expected).Cmp(uToBI(actual)) != 0 {
		t.Fatalf("%s mismatch: local=%s onchain=%s", label, uToBI(expected), uToBI(actual))
	}
}

func fundSequentialTrader(t *testing.T, env baselineDifferentialEnv, reserveAmount *big.Int) {
	t.Helper()

	runCast(t, "send", env.reserve, "deposit()", "--value", reserveAmount.String(), "--private-key", anvilPrivateKey, "--rpc-url", env.rpcURL)
	runCast(t, "send", env.reserve, "approve(address,uint256)", env.relay, reserveAmount.String(), "--private-key", anvilPrivateKey, "--rpc-url", env.rpcURL)
	runCast(t, "send", env.bToken, "approve(address,uint256)", env.relay, maxUint256BI().String(), "--private-key", anvilPrivateKey, "--rpc-url", env.rpcURL)
}

func quoteAndApplySequentialSwap(t *testing.T, sim *PoolSimulator, swap sequentialSwap) {
	t.Helper()

	if swap.kind == sequentialSwapBuyExactOut {
		quote, err := sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: sim.Info.Tokens[1], Amount: swap.amount},
			TokenIn:        sim.Info.Tokens[0],
		})
		if err != nil {
			t.Fatalf("local sequential buy exact-out amount %s failed: %v", swap.amount, err)
		}
		sim.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  *quote.TokenAmountIn,
			TokenAmountOut: pool.TokenAmount{Token: sim.Info.Tokens[1], Amount: swap.amount},
			SwapInfo:       quote.SwapInfo,
		})
		return
	}

	if swap.kind == sequentialSwapSellExactOut {
		quote, err := sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: sim.Info.Tokens[0], Amount: swap.amount},
			TokenIn:        sim.Info.Tokens[1],
		})
		if err != nil {
			t.Fatalf("local sequential sell exact-out amount %s failed: %v", swap.amount, err)
		}
		sim.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  *quote.TokenAmountIn,
			TokenAmountOut: pool.TokenAmount{Token: sim.Info.Tokens[0], Amount: swap.amount},
			SwapInfo:       quote.SwapInfo,
		})
		return
	}

	tokenIn := sim.Info.Tokens[0]
	tokenOut := sim.Info.Tokens[1]
	if swap.kind == sequentialSwapSellExactIn {
		tokenIn = sim.Info.Tokens[1]
		tokenOut = sim.Info.Tokens[0]
	}
	quote, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: swap.amount},
		TokenOut:      tokenOut,
	})
	if err != nil {
		t.Fatalf("local sequential %s amount %s failed: %v", swap.kind, swap.amount, err)
	}
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: tokenIn, Amount: swap.amount},
		TokenAmountOut: *quote.TokenAmountOut,
		SwapInfo:       quote.SwapInfo,
	})
}

func sendSequentialSwap(t *testing.T, env baselineDifferentialEnv, swap sequentialSwap) {
	t.Helper()

	switch swap.kind {
	case sequentialSwapBuyExactIn:
		sendBuyExactIn(t, env, swap.amount, false)
	case sequentialSwapBuyExactOut:
		sendBuyExactOut(t, env, swap.amount, false)
	case sequentialSwapSellExactIn:
		sendSellExactIn(t, env, swap.amount, false)
	case sequentialSwapSellExactOut:
		sendSellExactOut(t, env, swap.amount, false)
	}
}

func sendSequentialSwapAsync(t *testing.T, env baselineDifferentialEnv, swap sequentialSwap, nonce uint64) {
	t.Helper()

	switch swap.kind {
	case sequentialSwapBuyExactIn:
		sendBuyExactIn(t, env, swap.amount, true, nonce)
	case sequentialSwapBuyExactOut:
		sendBuyExactOut(t, env, swap.amount, true, nonce)
	case sequentialSwapSellExactIn:
		sendSellExactIn(t, env, swap.amount, true, nonce)
	case sequentialSwapSellExactOut:
		sendSellExactOut(t, env, swap.amount, true, nonce)
	}
}

func sendBuyExactIn(t *testing.T, env baselineDifferentialEnv, amountIn *big.Int, async bool, nonce ...uint64) {
	t.Helper()

	args := []string{
		"send",
		"--from", sequentialTrader.Hex(),
		env.relay,
		"buyTokensExactIn(address,uint256,uint256)",
		env.bToken,
		amountIn.String(),
		"0",
		"--private-key", anvilPrivateKey,
		"--rpc-url", env.rpcURL,
	}
	if async {
		args = append(args, "--async")
	}
	if len(nonce) > 0 {
		args = append(args, "--nonce", strconv.FormatUint(nonce[0], 10))
	}
	runCast(t, args...)
}

func sendBuyExactOut(t *testing.T, env baselineDifferentialEnv, amountOut *big.Int, async bool, nonce ...uint64) {
	t.Helper()

	args := []string{
		"send",
		"--from", sequentialTrader.Hex(),
		env.relay,
		"buyTokensExactOut(address,uint256,uint256)",
		env.bToken,
		amountOut.String(),
		maxUint256BI().String(),
		"--private-key", anvilPrivateKey,
		"--rpc-url", env.rpcURL,
	}
	if async {
		args = append(args, "--async")
	}
	if len(nonce) > 0 {
		args = append(args, "--nonce", strconv.FormatUint(nonce[0], 10))
	}
	runCast(t, args...)
}

func sendSellExactIn(t *testing.T, env baselineDifferentialEnv, amountIn *big.Int, async bool, nonce ...uint64) {
	t.Helper()

	args := []string{
		"send",
		"--from", sequentialTrader.Hex(),
		env.relay,
		"sellTokensExactIn(address,uint256,uint256)",
		env.bToken,
		amountIn.String(),
		"0",
		"--private-key", anvilPrivateKey,
		"--rpc-url", env.rpcURL,
	}
	if async {
		args = append(args, "--async")
	}
	if len(nonce) > 0 {
		args = append(args, "--nonce", strconv.FormatUint(nonce[0], 10))
	}
	runCast(t, args...)
}

func sendSellExactOut(t *testing.T, env baselineDifferentialEnv, amountOut *big.Int, async bool, nonce ...uint64) {
	t.Helper()

	args := []string{
		"send",
		"--from", sequentialTrader.Hex(),
		env.relay,
		"sellTokensExactOut(address,uint256,uint256)",
		env.bToken,
		amountOut.String(),
		maxUint256BI().String(),
		"--private-key", anvilPrivateKey,
		"--rpc-url", env.rpcURL,
	}
	if async {
		args = append(args, "--async")
	}
	if len(nonce) > 0 {
		args = append(args, "--nonce", strconv.FormatUint(nonce[0], 10))
	}
	runCast(t, args...)
}

func assertPostSequenceQuotes(
	t *testing.T,
	ctx context.Context,
	ethrpcClient *ethrpc.Client,
	env baselineDifferentialEnv,
	sim *PoolSimulator,
	label string,
) {
	t.Helper()

	buyAmountIn := divBI(decimalUnit(18), big.NewInt(7))
	solidityBuy, solidityErr := callQuoteBuyExactIn(t, ctx, ethrpcClient, env, buyAmountIn)
	goBuy, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: env.reserve, Amount: buyAmountIn},
		TokenOut:      env.bToken,
	})
	assertQuoteErrorParity(t, methodQuoteBuyExactIn, solidityErr, err)
	if solidityErr == nil {
		if err != nil {
			t.Fatalf("%s local buy quote failed: %v", label, err)
		}
		assertBigEqual(t, label+" buy amount", solidityBuy.amount, goBuy.TokenAmountOut.Amount)
		assertBigEqual(t, label+" buy fee", solidityBuy.fee, goBuy.Fee.Amount)
	}

	sellAmountIn := divBI(decimalUnit(18), big.NewInt(9))
	soliditySell, solidityErr := callQuoteSellExactIn(t, ctx, ethrpcClient, env, sellAmountIn)
	goSell, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: env.bToken, Amount: sellAmountIn},
		TokenOut:      env.reserve,
	})
	assertQuoteErrorParity(t, methodQuoteSellExactIn, solidityErr, err)
	if solidityErr == nil {
		if err != nil {
			t.Fatalf("%s local sell quote failed: %v", label, err)
		}
		assertBigEqual(t, label+" sell amount", soliditySell.amount, goSell.TokenAmountOut.Amount)
		assertBigEqual(t, label+" sell fee", soliditySell.fee, goSell.Fee.Amount)
	}
}

func maxUint256BI() *big.Int {
	return new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
}

func castNonce(t *testing.T, env baselineDifferentialEnv) uint64 {
	t.Helper()

	cmd := exec.Command("cast", "nonce", sequentialTrader.Hex(), "--rpc-url", env.rpcURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cast nonce failed: %v\n%s", err, output)
	}
	fields := strings.Fields(string(output))
	if len(fields) == 0 {
		t.Fatalf("cast nonce returned empty output")
	}
	nonce, err := strconv.ParseUint(fields[len(fields)-1], 10, 64)
	if err != nil {
		t.Fatalf("invalid cast nonce output %q: %v", output, err)
	}
	return nonce
}

func requireCast(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("cast"); err != nil {
		t.Skip("cast is required for sequential Baseline fork differential test")
	}
}

func runCast(t *testing.T, args ...string) {
	t.Helper()

	cmd := exec.Command("cast", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cast %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func TestBaselineDifferentialEnvBlockNumberParse(t *testing.T) {
	t.Setenv("BASELINE_RPC_URL", "http://127.0.0.1:8545")
	t.Setenv("BASELINE_RELAY_ADDRESS", "0x0000000000000000000000000000000000000001")
	t.Setenv("BASELINE_BTOKEN_ADDRESS", "0x0000000000000000000000000000000000000002")
	t.Setenv("BASELINE_RESERVE_ADDRESS", "0x0000000000000000000000000000000000000003")
	t.Setenv("BASELINE_BLOCK_NUMBER", strconv.FormatUint(12345, 10))

	env := loadBaselineDifferentialEnv(t)
	if env.blockNumber == nil || env.blockNumber.Uint64() != 12345 {
		t.Fatalf("unexpected block number: %v", env.blockNumber)
	}
}
