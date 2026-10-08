package uscoreprop

import (
	"math"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// ABI decode targets of the live checks.
type reservesResult struct{ ReserveQuote, ReserveBase *big.Int }
type quoteResult struct {
	AmountOut, Fee *big.Int
	Status         uint8
}
type ladderResult struct {
	Outs, Fees []*big.Int
	Status     uint8
}

var livePools = []string{testPool, "0xb9fa3bdfa88da2dc78c20ca03472e043992c6671"}

func liveClient(t *testing.T) *ethrpc.Client {
	url := os.Getenv("USCORE_RPC_URL")
	if url == "" {
		t.Skip("set USCORE_RPC_URL for live HyperEVM checks")
	}
	return ethrpc.New(url).SetMulticallContract(common.HexToAddress(multicall))
}

func livePoolEntities(t *testing.T, client *ethrpc.Client) []entity.Pool {
	pools, _, err := NewPoolsListUpdater(&Config{DexID: DexType, Pools: livePools}, client).GetNewPools(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 2)
	for _, p := range pools {
		p.Tokens[0].Decimals, p.Tokens[1].Decimals = 6, 18
	}
	return pools
}

func TestLiveUSCore(t *testing.T) {
	client := liveClient(t)
	for _, p := range livePoolEntities(t, client) {
		for cycle := 0; cycle < 3; cycle++ {
			var err error
			p, err = NewPoolTracker(nil, client).GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
			require.NoError(t, err)
			require.Positive(t, p.BlockNumber)
			_, err = NewPoolSimulatorWith(p, MaxAge)
			require.NoError(t, err)
			t.Logf("pool=%s block=%d cycle=%d", p.Address, p.BlockNumber, cycle)
			compareLiveQuotes(t, client, p)
		}
	}
}

func compareLiveQuotes(t *testing.T, client *ethrpc.Client, p entity.Pool) {
	t.Helper()
	var old Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &old))
	var samples, probes [2][]*big.Int
	var sampled, exact [2]ladderResult
	var checks [2][]*big.Int
	var checked [2][]quoteResult
	var reserves reservesResult
	req := client.R().SetContext(t.Context())
	req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: "getReserves"}, []any{&reserves})
	for dir := range 2 {
		for _, point := range old.Ladders[dir] {
			samples[dir] = append(samples[dir], point[0].ToBig())
		}
		for i := 0; i < len(samples[dir]); i++ {
			lo, hi := big.NewInt(0), samples[dir][i]
			if i > 0 {
				lo = samples[dir][i-1]
			}
			for _, numerator := range []int64{1, 2, 3} {
				x := new(big.Int).Sub(hi, lo)
				x.Mul(x, big.NewInt(numerator)).Div(x, big.NewInt(4)).Add(x, lo)
				probes[dir] = append(probes[dir], x)
			}
		}
		if len(probes[dir]) == 0 {
			continue
		}
		token := common.HexToAddress(p.Tokens[dir].Address)
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: "quoteLadder", Params: []any{token, samples[dir]}}, []any{&sampled[dir]})
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: "quoteLadder", Params: []any{token, probes[dir]}}, []any{&exact[dir]})
		unit := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(p.Tokens[dir].Decimals)), nil)
		checks[dir] = []*big.Int{big.NewInt(1), big.NewInt(10), big.NewInt(100)}
		for _, divisor := range []int64{1_000_000, 10_000, 100, 10} {
			checks[dir] = append(checks[dir], new(big.Int).Div(unit, big.NewInt(divisor)))
		}
		last := samples[dir][len(samples[dir])-1]
		for _, bps := range []int64{10_010, 10_100, 11_000, 15_000} {
			x := new(big.Int).Mul(last, big.NewInt(bps))
			checks[dir] = append(checks[dir], x.Div(x, big.NewInt(10_000)))
		}
		if dir == 0 {
			for _, amount := range []int64{3_500, 4_000} {
				checks[dir] = append(checks[dir], new(big.Int).Mul(unit, big.NewInt(amount)))
			}
		}
		checked[dir] = make([]quoteResult, len(checks[dir]))
		for i, amount := range checks[dir] {
			req.AddCall(&ethrpc.Call{ABI: poolABI, Target: p.Address, Method: "quoteExactIn", Params: []any{token, amount}}, []any{&checked[dir][i]})
		}
	}
	resp, err := req.Aggregate()
	require.NoError(t, err)
	var extra Extra
	for dir := range 2 {
		points := big256.MustFromBigs(samples[dir])
		q, err := toLadderQuote(points, sampled[dir].Outs, sampled[dir].Status)
		require.NoError(t, err)
		extra.Ladders[dir] = collectQuotes(points, q)
	}
	b, err := json.Marshal(extra)
	require.NoError(t, err)
	p.Extra = string(b)
	p.Reserves = entity.PoolReserves{reserves.ReserveQuote.String(), reserves.ReserveBase.String()}
	p.BlockNumber = resp.BlockNumber.Uint64()
	sim, err := NewPoolSimulatorWith(p, math.MaxInt64)
	require.NoError(t, err)
	for dir := range 2 {
		maxErrorBps := 0.0
		maxErrorBeyondRoundingBps := 0.0
		compared := 0
		check := func(amount, actual *big.Int, status uint8) {
			q, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[dir].Address, Amount: amount}, TokenOut: p.Tokens[1-dir].Address})
			if status != 0 || actual.Sign() == 0 {
				require.Error(t, err)
				return
			}
			if err != nil {
				require.ErrorIs(t, err, ladder.ErrAmountInTooLarge)
				points := extra.Ladders[dir]
				require.NotEmpty(t, points)
				best := points[len(points)-1][1].ToBig()
				bound := new(big.Int).Div(best, big.NewInt(1_000))
				bound.Add(bound, best).Add(bound, big.NewInt(1))
				require.LessOrEqual(t, actual.Cmp(bound), 0, "productive input omitted: pool=%s direction=%d amountIn=%s", p.Address, dir, amount)
				return
			}
			delta := new(big.Int).Sub(q.TokenAmountOut.Amount, actual)
			delta.Abs(delta)
			bound := new(big.Int).Div(actual, big.NewInt(100_000))
			rounding := new(big.Int).Add(actual, amount)
			rounding.Sub(rounding, big.NewInt(1)).Div(rounding, amount)
			bound.Add(bound, rounding)
			require.LessOrEqual(t, delta.Cmp(bound), 0, "quote error: pool=%s direction=%d amountIn=%s amountOut=%s delta=%s", p.Address, dir, amount, actual, delta)
			ratio, _ := new(big.Rat).SetFrac(delta, actual).Float64()
			maxErrorBps = math.Max(maxErrorBps, ratio*10_000)
			ratio, _ = new(big.Rat).SetFrac(new(big.Int).Sub(delta, rounding), actual).Float64()
			maxErrorBeyondRoundingBps = math.Max(maxErrorBeyondRoundingBps, ratio*10_000)
			compared++
		}
		for i, amount := range probes[dir] {
			if i >= len(exact[dir].Outs) || exact[dir].Outs[i].Sign() == 0 {
				continue
			}
			check(amount, exact[dir].Outs[i], 0)
		}
		for i, amount := range checks[dir] {
			check(amount, checked[dir][i].AmountOut, checked[dir][i].Status)
		}
		t.Logf("pool=%s direction=%d atomicBlock=%d compared=%d maxAbsErrorBps=%.6f maxErrorBeyondRoundingBps=%.6f", p.Address, dir, p.BlockNumber, compared, maxErrorBps, maxErrorBeyondRoundingBps)
		if len(extra.Ladders[dir]) > 1 {
			require.Positive(t, compared)
		}
	}
}
