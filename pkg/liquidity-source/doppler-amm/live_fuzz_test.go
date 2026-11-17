package doppleramm

import (
	"context"
	"math/big"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
)

type quoteResult struct {
	ActualInputAmount  *big.Int `abi:"actualInputAmount"`
	ActualOutputAmount *big.Int `abi:"actualOutputAmount"`
}

// TestLiveFuzzAgainstQuoter lists every Base market of both kinds, refreshes them with the
// tracker, then compares randomized exact-input quotes (both directions, 1 wei up to several times the
// reserve) with DopplerQuoter at the pool's block. A revert on chain must be an error here, and
// the other way round. Skipped unless LIVE_RPC points at a Base archive RPC.
func TestLiveFuzzAgainstQuoter(t *testing.T) {
	rpc := os.Getenv("LIVE_RPC")
	if rpc == "" {
		t.Skip("LIVE_RPC not set")
	}
	ctx := context.Background()
	client := ethrpc.New(rpc).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	pools := listLivePools(t, client)
	require.NotEmpty(t, pools)

	tracker, err := NewPoolTracker(testConfig(), client)
	require.NoError(t, err)
	rng := rand.New(rand.NewSource(1))
	samples := 200
	if v := os.Getenv("FUZZ_SAMPLES"); v != "" {
		n, _ := new(big.Int).SetString(v, 10)
		samples = int(n.Int64())
	}

	for _, ep := range pools {
		ep, err := tracker.GetNewPoolState(ctx, ep, pool.GetNewPoolStateParams{})
		if err != nil {
			t.Logf("market %s skipped: %v", ep.Address[:10], err)
			continue
		}
		sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: ep})
		require.NoError(t, err)
		var static StaticExtra
		require.NoError(t, json.Unmarshal([]byte(ep.StaticExtra), &static))
		marketID := new(big.Int).SetBytes(common.FromHex(static.MarketID))

		stats := map[string]int{}
		for _, xIn := range []bool{true, false} {
			reserveIn, _ := new(big.Int).SetString(ep.Reserves[map[bool]int{true: 0, false: 1}[xIn]], 10)
			maxBits := reserveIn.BitLen() + 3
			amounts := make([]*big.Int, samples)
			for i := range amounts {
				bits := 1 + rng.Intn(maxBits)
				amounts[i] = new(big.Int).Rand(rng, new(big.Int).Lsh(big.NewInt(1), uint(bits)))
				amounts[i].Add(amounts[i], big.NewInt(1))
			}
			if sim.launch {
				amounts = append(amounts, crossingAmounts(sim, xIn)...)
			}
			quotes := make([]quoteResult, len(amounts))
			req := client.NewRequest().SetContext(ctx).SetBlockNumber(new(big.Int).SetUint64(ep.BlockNumber))
			for i := range amounts {
				req.AddCall(&ethrpc.Call{ABI: quoterABI, Target: static.Quoter, Method: "quoteSwapExactInput",
					Params: []any{marketID, xIn, amounts[i], []byte{}, []byte{}}}, []any{&quotes[i]})
			}
			resp, err := req.TryAggregate()
			require.NoError(t, err)

			tokenIn, tokenOut := ep.Tokens[0].Address, ep.Tokens[1].Address
			if !xIn {
				tokenIn, tokenOut = tokenOut, tokenIn
			}
			for i, a := range amounts {
				res, simErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: a}, TokenOut: tokenOut})
				chainOK := resp.Result[i] && quotes[i].ActualOutputAmount != nil && quotes[i].ActualOutputAmount.Sign() > 0
				switch {
				case simErr != nil && !chainOK:
					stats["both fail"]++
				case simErr != nil:
					stats["sim fails, chain ok"]++
					t.Errorf("%s xIn=%v in=%s: sim %v, chain out %s", ep.Address[:10], xIn, a, simErr, quotes[i].ActualOutputAmount)
				case !chainOK:
					stats["sim ok, chain reverts"]++
					t.Errorf("%s xIn=%v in=%s: sim out %s, chain reverts", ep.Address[:10], xIn, a, res.TokenAmountOut.Amount)
				case quotes[i].ActualInputAmount == nil || quotes[i].ActualInputAmount.Cmp(a) != 0:
					stats["partial fill on chain"]++
					t.Errorf("%s xIn=%v in=%s: chain consumed %s", ep.Address[:10], xIn, a, quotes[i].ActualInputAmount)
				case res.TokenAmountOut.Amount.Cmp(quotes[i].ActualOutputAmount) != 0:
					stats["different"]++
					t.Errorf("%s xIn=%v in=%s: sim %s chain %s", ep.Address[:10], xIn, a, res.TokenAmountOut.Amount, quotes[i].ActualOutputAmount)
				default:
					stats["exact"]++
				}
			}
		}
		t.Logf("market %s block %d launch=%v: %v extra=%.200s", ep.Address[:10], ep.BlockNumber, static.Module != "", stats, ep.Extra)
	}
}

// crossingAmounts bisects the input that first reaches each of the next two level boundaries and
// returns the amounts within 3 wei of it, where a rounding difference in the crossing shows up.
func crossingAmounts(sim *PoolSimulator, xIn bool) []*big.Int {
	var out []*big.Int
	for k := 0; k < 2; k++ {
		// price after the swap on or past the k-th boundary in the swap direction
		reached := func(a *uint256.Int) bool {
			_, info, err := sim.launchOut(a, xIn)
			if err != nil {
				return true // the range ran out: the boundary was passed
			}
			n := 0
			for i := range sim.bounds {
				b := &sim.bounds[i]
				if (!xIn && b.Gt(&sim.sqrtP) && !b.Gt(info.SqrtPriceX96)) ||
					(xIn && !b.Gt(&sim.sqrtP) && !info.SqrtPriceX96.Gt(b) && info.SqrtPriceX96.Lt(&sim.sqrtP)) {
					n++
				}
			}
			return n > k
		}
		lo, hi := uint256.NewInt(1), uint256.NewInt(1)
		for i := 0; i < 255 && !reached(hi); i++ {
			hi.Lsh(hi, 1)
		}
		if !reached(hi) {
			continue
		}
		for lo.Lt(hi) {
			var mid uint256.Int
			mid.Add(lo, hi).Rsh(&mid, 1)
			if reached(&mid) {
				hi.Set(&mid)
			} else {
				lo.AddUint64(&mid, 1)
			}
		}
		for d := int64(-3); d <= 3; d++ {
			a := new(big.Int).Add(hi.ToBig(), big.NewInt(d))
			if a.Sign() > 0 {
				out = append(out, a)
			}
		}
	}
	return out
}

// listLivePools replays every MarketCreated log of the Base AMM through the backfiller, as
// pool-service does, and returns the pools of both kinds.
func listLivePools(t *testing.T, client *ethrpc.Client) []entity.Pool {
	t.Helper()
	ctx := context.Background()
	f, err := NewPoolFactory(poolfactory.PoolsDecoderParams[Config]{Cfg: testConfig()})
	require.NoError(t, err)
	bf, err := poolfactory.NewFilterLogsBackfiller(f, client, poolfactory.BackfillConfig{
		FactoryAddresses:     []common.Address{f.amm},
		StartBlock:           52_856_797,
		MaxBlockRangePerScan: 200_000,
		Exchange:             DexType,
	})
	require.NoError(t, err)

	var pools, ps []entity.Pool
	var meta []byte
	for done := false; !done; {
		for try := 0; try < 5; try++ { // load-balanced nodes lag behind each other
			if ps, meta, done, err = bf.Backfill(ctx, meta); err == nil {
				break
			}
			time.Sleep(time.Second)
		}
		require.NoError(t, err)
		pools = append(pools, ps...)
	}
	return pools
}
