package rangepool

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// THE ACCEPTANCE GATE (reference §7, PROGRESS-kyber Stage 3): the connector's offline
// quote must equal the on-chain Router.querySwapSingleTokenExactIn / …ExactOut to the wei,
// for both live pools, both directions, and both sides, across an amount sweep.
//
// RPC-gated (test.SkipCI + ETH_RPC_URL); see range-pool_integration_test.go for RPC setup.
// Every querySwap is issued as its OWN eth_call pinned to the tracked block — NEVER batched
// into a multicall: query mode mutates _virtualBalances in-call, which would silently
// corrupt all but the first result (reference §5.4, gotcha #5).

// fracDenom is the denominator for the amount sweep; numerators below are parts per this.
const fracDenom = 1_000_000

// sweepNumerators sizes each test amount as reserve * num / fracDenom, i.e. 1e-6 .. 0.1 of
// the relevant fact balance — a spread that stays comfortably under the fact cap for small
// fractions and reaches/exceeds it for large ones (where EXACT_IN caps, matching on-chain,
// and EXACT_OUT reverts, which the sim mirrors by erroring).
var sweepNumerators = []int64{1, 10, 100, 1_000, 5_000, 10_000, 50_000, 100_000}

type ParitySuite struct {
	suite.Suite

	client  *ethrpc.Client
	lister  *PoolsListUpdater
	tracker *PoolTracker
}

func (ts *ParitySuite) SetupSuite() {
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		rpcURL = mainnetDefaultRPC
	}
	ts.client = ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress(mainnetMulticall3))

	cfg := &Config{
		DexID:          DexType,
		FactoryAddress: FactoryAddress.Hex(),
		NewPoolLimit:   100,
	}
	ts.lister = NewPoolsListUpdater(cfg, ts.client)

	tracker, err := NewPoolTracker(cfg, ts.client)
	ts.Require().NoError(err)
	ts.tracker = tracker
}

// buildSimulator discovers + tracks a pool and constructs the Range simulator from the
// exact tracked entity.Pool, returning it alongside the tracked pool (for tokens/reserves/
// block).
func (ts *ParitySuite) buildSimulator(poolAddr string) (*base.PoolSimulator, entity.Pool) {
	all := make(map[string]entity.Pool)
	var meta []byte
	for round := 0; round < 20; round++ {
		pools, next, err := ts.lister.GetNewPools(context.Background(), meta)
		ts.Require().NoError(err)
		for _, p := range pools {
			all[strings.ToLower(p.Address)] = p
		}
		if len(pools) == 0 {
			break
		}
		meta = next
	}

	seed, ok := all[poolAddr]
	ts.Require().Truef(ok, "pool %s not discovered", poolAddr)

	tracked, err := ts.tracker.GetNewPoolState(context.Background(), seed, pool.GetNewPoolStateParams{})
	ts.Require().NoError(err)

	sim, err := NewPoolSimulator(pool.FactoryParams{
		EntityPool: tracked,
		ChainID:    valueobject.ChainIDEthereum,
	})
	ts.Require().NoError(err)

	return sim, tracked
}

// TestParity_ROMEUSDT sweeps both directions and both sides of the 2-token 50/50 pool.
func (ts *ParitySuite) TestParity_ROMEUSDT() {
	sim, tracked := ts.buildSimulator(romeUSDTPool)
	ts.assertPairParity(sim, tracked, 0, 1)
	ts.assertPairParity(sim, tracked, 1, 0)
}

// TestParity_TopCryptoWETH sweeps the only live WETH pair in the 8-token pool: WETH
// (index 6) against the most-liquid other token, both directions and both sides.
func (ts *ParitySuite) TestParity_TopCryptoWETH() {
	sim, tracked := ts.buildSimulator(topCryptoPool)

	const wethIdx = 6
	counterpart := ts.mostLiquidExcept(tracked, wethIdx)
	ts.T().Logf("[top-crypto] WETH(%d) paired with most-liquid token index %d", wethIdx, counterpart)

	ts.assertPairParity(sim, tracked, wethIdx, counterpart)
	ts.assertPairParity(sim, tracked, counterpart, wethIdx)
}

// TestOversizedExactIn_TopCryptoWETH is the on-chain half of bugbot finding #4
// (adapters/docs/bugbot-findings-crosscheck.md). The weighted-product curve runs
// LogExpMath.pow BEFORE the fact-balance cap, so on the 8-token pool's WETH(weight 0.20)
// -> most-liquid low-weight token direction (non-short-circuit exponent > 1e18) an
// oversized exact-in overflows pow and reverts on-chain. The sim must return NO PRICE for
// exactly that amount (revert <=> no-price parity), and a normal amount on the same pool
// in the same test must still price to the wei (per-amount failure isolation — Kyber gets
// it by construction since CalcAmountOut is per-amount, but we prove it end-to-end here).
func (ts *ParitySuite) TestOversizedExactIn_TopCryptoWETH() {
	sim, tracked := ts.buildSimulator(topCryptoPool)

	const wethIdx = 6 // weight 0.20 (among the highest) => wIn > wOut for the pair below
	out := ts.mostLiquidExcept(tracked, wethIdx)
	tokenIn := tracked.Tokens[wethIdx].Address
	tokenOut := tracked.Tokens[out].Address
	block := tracked.BlockNumber

	// ~1e12 WETH — absurdly oversized, drives the curve base far past LogExpMath's bound.
	oversized := new(big.Int).Mul(big.NewInt(1_000_000_000_000_000_000), big.NewInt(1_000_000_000_000))

	_, oracleErr := ts.querySwapExactIn(block, tracked.Address, tokenIn, tokenOut, oversized)
	_, simErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: oversized},
		TokenOut:      tokenOut,
	})
	ts.Require().Errorf(oracleErr, "expected on-chain querySwap to revert on the oversized exact-in (idx %d->%d)", wethIdx, out)
	ts.Require().Errorf(simErr, "sim priced an oversized amount the chain reverts (%v) — pow overflow not handled as no-price", oracleErr)

	// Isolation: a normal amount still prices to the wei right after the failing one.
	normal := fracOf(mustBig(tracked.Reserves[wethIdx]), 100)
	oracleOut, oracleErr2 := ts.querySwapExactIn(block, tracked.Address, tokenIn, tokenOut, normal)
	ts.Require().NoError(oracleErr2, "normal amount should price on-chain — adjust sizing if this fails")
	res, simErr2 := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: normal},
		TokenOut:      tokenOut,
	})
	ts.Require().NoError(simErr2, "sim errored on a normal amount after an oversized one — failure isolation broken")
	ts.Require().Equal(oracleOut.String(), res.TokenAmountOut.Amount.String(), "normal-amount wei parity after an oversized-amount rejection")
	ts.T().Logf("oversized #4: idx %d->%d oversized=no-price<->revert OK; normal amount still prices to the wei", wethIdx, out)
}

// assertPairParity runs the full EXACT_IN and EXACT_OUT sweep for a single ordered token
// pair and asserts wei-exact agreement with the on-chain oracle for every non-reverting
// amount. It requires at least one successful match per side so a pool that happened to
// revert everything can't pass vacuously.
func (ts *ParitySuite) assertPairParity(sim *base.PoolSimulator, tracked entity.Pool, in, out int) {
	tokenIn := tracked.Tokens[in].Address
	tokenOut := tracked.Tokens[out].Address
	reserveIn := mustBig(tracked.Reserves[in])
	reserveOut := mustBig(tracked.Reserves[out])
	block := tracked.BlockNumber

	// ---- EXACT_IN: CalcAmountOut == querySwapSingleTokenExactIn ----
	exactInMatches := 0
	for _, num := range sweepNumerators {
		amountIn := fracOf(reserveIn, num)
		if amountIn.Sign() == 0 {
			continue
		}

		oracleOut, oracleErr := ts.querySwapExactIn(block, tracked.Address, tokenIn, tokenOut, amountIn)
		res, simErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
			TokenOut:      tokenOut,
		})

		if oracleErr != nil {
			// On-chain rejected this size (e.g. below MINIMUM_TRADE_AMOUNT); the sim must
			// reject too. A sim that prices where the chain reverts is a real divergence.
			ts.Require().Errorf(simErr, "EXACT_IN %d->%d amt=%s: chain reverted (%v) but sim succeeded",
				in, out, amountIn, oracleErr)
			continue
		}

		ts.Require().NoErrorf(simErr, "EXACT_IN %d->%d amt=%s: chain priced %s but sim errored",
			in, out, amountIn, oracleOut)
		ts.Require().Equalf(oracleOut.String(), res.TokenAmountOut.Amount.String(),
			"EXACT_IN %d->%d amt=%s: wei-parity mismatch", in, out, amountIn)
		exactInMatches++
	}
	ts.Require().Positivef(exactInMatches, "EXACT_IN %d->%d: no amount priced on-chain — bad sizing", in, out)

	// ---- EXACT_OUT: CalcAmountIn == querySwapSingleTokenExactOut ----
	exactOutMatches := 0
	for _, num := range sweepNumerators {
		amountOut := fracOf(reserveOut, num)
		if amountOut.Sign() == 0 {
			continue
		}

		oracleIn, oracleErr := ts.querySwapExactOut(block, tracked.Address, tokenIn, tokenOut, amountOut)
		res, simErr := sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: tokenOut, Amount: amountOut},
			TokenIn:        tokenIn,
		})

		if oracleErr != nil {
			// The EXACT_OUT guards (fact cap / virtual-out) revert on-chain; the sim must
			// mirror with an error.
			ts.Require().Errorf(simErr, "EXACT_OUT %d->%d out=%s: chain reverted (%v) but sim succeeded",
				in, out, amountOut, oracleErr)
			continue
		}

		ts.Require().NoErrorf(simErr, "EXACT_OUT %d->%d out=%s: chain priced %s but sim errored",
			in, out, amountOut, oracleIn)
		ts.Require().Equalf(oracleIn.String(), res.TokenAmountIn.Amount.String(),
			"EXACT_OUT %d->%d out=%s: wei-parity mismatch", in, out, amountOut)
		exactOutMatches++
	}
	ts.Require().Positivef(exactOutMatches, "EXACT_OUT %d->%d: no amount priced on-chain — bad sizing", in, out)

	ts.T().Logf("parity %d->%d OK: exactIn=%d exactOut=%d matches", in, out, exactInMatches, exactOutMatches)
}

// querySwapExactIn issues ONE eth_call to Router.querySwapSingleTokenExactIn pinned to the
// given block (never batched — reference §5.4). The zero `from` (default) keeps tx.origin
// zero so Balancer's query mode activates.
func (ts *ParitySuite) querySwapExactIn(block uint64, poolAddr, tokenIn, tokenOut string, amountIn *big.Int) (*big.Int, error) {
	var amountOut *big.Int
	req := ts.client.NewRequest().SetContext(context.Background()).
		SetBlockNumber(new(big.Int).SetUint64(block))
	req.AddCall(&ethrpc.Call{
		ABI:    rangeRouterABI,
		Target: RouterAddress.Hex(),
		Method: routerMethodQuerySwapExactIn,
		Params: []any{
			common.HexToAddress(poolAddr),
			common.HexToAddress(tokenIn),
			common.HexToAddress(tokenOut),
			amountIn,
			common.Address{}, // sender (unused for pricing)
			[]byte{},         // userData
		},
	}, []any{&amountOut})
	if _, err := req.Call(); err != nil {
		return nil, err
	}
	return amountOut, nil
}

// querySwapExactOut issues ONE eth_call to Router.querySwapSingleTokenExactOut pinned to
// the given block.
func (ts *ParitySuite) querySwapExactOut(block uint64, poolAddr, tokenIn, tokenOut string, amountOut *big.Int) (*big.Int, error) {
	var amountIn *big.Int
	req := ts.client.NewRequest().SetContext(context.Background()).
		SetBlockNumber(new(big.Int).SetUint64(block))
	req.AddCall(&ethrpc.Call{
		ABI:    rangeRouterABI,
		Target: RouterAddress.Hex(),
		Method: routerMethodQuerySwapExactOut,
		Params: []any{
			common.HexToAddress(poolAddr),
			common.HexToAddress(tokenIn),
			common.HexToAddress(tokenOut),
			amountOut,
			common.Address{}, // sender
			[]byte{},         // userData
		},
	}, []any{&amountIn})
	if _, err := req.Call(); err != nil {
		return nil, err
	}
	return amountIn, nil
}

// mostLiquidExcept returns the token index with the largest fact reserve, skipping `skip`.
func (ts *ParitySuite) mostLiquidExcept(tracked entity.Pool, skip int) int {
	best, bestVal := -1, big.NewInt(0)
	for i, r := range tracked.Reserves {
		if i == skip {
			continue
		}
		v := mustBig(r)
		if v.Cmp(bestVal) > 0 {
			best, bestVal = i, v
		}
	}
	ts.Require().GreaterOrEqualf(best, 0, "no counterpart token with reserves found (skip=%d)", skip)
	return best
}

// GetMetaInfo must expose the standard Router (0x8726…) as the approval/target address the
// Stage-5 executor will need (reference §6, gotcha #6).
func (ts *ParitySuite) TestGetMetaInfo_RouterApproval() {
	sim, tracked := ts.buildSimulator(romeUSDTPool)
	metaAny := sim.GetMetaInfo(tracked.Tokens[0].Address, tracked.Tokens[1].Address)

	raw, err := json.Marshal(metaAny)
	ts.Require().NoError(err)
	var meta struct {
		ApprovalAddress string `json:"approvalAddress"`
	}
	ts.Require().NoError(json.Unmarshal(raw, &meta))
	ts.Require().Equal(strings.ToLower(RouterAddress.Hex()), strings.ToLower(meta.ApprovalAddress),
		"GetMetaInfo must approve/target the standard Router")
}

func fracOf(x *big.Int, num int64) *big.Int {
	out := new(big.Int).Mul(x, big.NewInt(num))
	return out.Div(out, big.NewInt(fracDenom))
}

func mustBig(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return big.NewInt(0)
	}
	return v
}

func TestParitySuite(t *testing.T) {
	test.SkipCI(t)
	suite.Run(t, new(ParitySuite))
}
