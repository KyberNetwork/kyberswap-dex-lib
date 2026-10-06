package uscoreprop

import (
	"encoding/json"
	"math"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const testPool = "0xed2ef1b02f2d82d238d6af17e6404a4977b0fefa"
const testQuote = "0xb88339cb7199b77e23db6e890353e22632ba630f"
const testBase = "0x5555555555555555555555555555555555555555"

var multicallABI = lo.Must(abi.JSON(strings.NewReader(`[
 {"type":"function","name":"aggregate","inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"type":"uint256"},{"type":"bytes[]"}]},
 {"type":"function","name":"tryAggregate","inputs":[{"type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}
]`)))

type mockCall struct {
	Target   common.Address
	CallData []byte
}
type mockResult struct {
	Success    bool
	ReturnData []byte
}

func mockRPC(t *testing.T, paused, failBase bool, count *int, seenOverrides *bool, smallReserve ...int64) *ethrpc.Client {
	return mockRPCWithStatus(t, paused, failBase, count, seenOverrides, smallReserve, nil)
}

func mockRPCWithStatus(t *testing.T, paused, failBase bool, count *int, seenOverrides *bool, smallReserve []int64, snapshotStatus func(int) uint8) *ethrpc.Client {
	t.Helper()
	snapshots := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ID     json.RawMessage
			Method string
			Params []json.RawMessage
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "eth_call", body.Method)
		require.Equal(t, `"latest"`, string(body.Params[1]))
		if len(body.Params) == 3 && seenOverrides != nil {
			*seenOverrides = true
		}
		var call struct{ Data, Input hexutil.Bytes }
		require.NoError(t, json.Unmarshal(body.Params[0], &call))
		data := call.Input
		if len(data) == 0 {
			data = call.Data
		}
		method, err := multicallABI.MethodById(data[:4])
		require.NoError(t, err)
		args, err := method.Inputs.Unpack(data[4:])
		require.NoError(t, err)
		calls := *abi.ConvertType(args[len(args)-1], new([]mockCall)).(*[]mockCall)
		atomic := false
		for _, c := range calls {
			if c.Target == common.HexToAddress(multicall3) {
				atomic = true
			}
		}
		block := big.NewInt(100)
		if atomic {
			snapshots++
			block = big.NewInt(100 + int64(snapshots))
		}
		var outputs [][]byte
		var results []mockResult
		for _, c := range calls {
			var output []byte
			success := true
			if c.Target == common.HexToAddress(multicall3) {
				output, err = clockABI.Methods["getCurrentBlockTimestamp"].Outputs.Pack(big.NewInt(1_800_000_000 + int64(snapshots)))
			} else {
				m, e := poolABI.MethodById(c.CallData[:4])
				require.NoError(t, e)
				a, e := m.Inputs.Unpack(c.CallData[4:])
				require.NoError(t, e)
				switch m.Name {
				case "quote":
					output, err = m.Outputs.Pack(common.HexToAddress(testQuote))
				case "base":
					output, err = m.Outputs.Pack(common.HexToAddress(testBase))
					success = !failBase
				case "getReserves":
					if len(smallReserve) > 0 {
						base := big.NewInt(0)
						if len(smallReserve) > 1 {
							base.SetInt64(smallReserve[1])
						}
						output, err = m.Outputs.Pack(big.NewInt(smallReserve[0]), base)
					} else if atomic {
						output, err = m.Outputs.Pack(big.NewInt(900), big.NewInt(1100))
					} else {
						output, err = m.Outputs.Pack(big.NewInt(1000), big.NewInt(1000))
					}
				case "quoteExactIn":
					if len(smallReserve) > 0 {
						out, status := smallPoolQuote(a[0].(common.Address), a[1].(*big.Int), smallReserve[0])
						output, err = m.Outputs.Pack(out, big.NewInt(0), status)
					} else {
						output, err = m.Outputs.Pack(big.NewInt(2), big.NewInt(0), uint8(0))
					}
				case "quoteLadder":
					points := a[1].([]*big.Int)
					outs, fees := make([]*big.Int, len(points)), make([]*big.Int, len(points))
					status := uint8(0)
					for i, p := range points {
						outs[i], fees[i] = big.NewInt(0), big.NewInt(0)
						if len(smallReserve) > 0 {
							if status == 0 {
								outs[i], status = smallPoolQuote(a[0].(common.Address), p, smallReserve[0])
							}
						} else if paused {
							status = 4
						} else if snapshotStatus != nil && snapshotStatus(snapshots) != 0 {
							status = snapshotStatus(snapshots)
						} else if p.Cmp(big.NewInt(300)) > 0 {
							status = 3
						} else {
							outs[i].Mul(p, big.NewInt(2))
						}
					}
					output, err = m.Outputs.Pack(outs, fees, status)
				}
			}
			require.NoError(t, err)
			outputs = append(outputs, output)
			results = append(results, mockResult{success, output})
		}
		var encoded []byte
		if method.Name == "aggregate" {
			encoded, err = method.Outputs.Pack(block, outputs)
		} else {
			encoded, err = method.Outputs.Pack(results)
		}
		require.NoError(t, err)
		if count != nil {
			*count++
		}
		w.Header().Set("content-type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": body.ID, "result": hexutil.Encode(encoded)}))
	}))
	t.Cleanup(server.Close)
	return ethrpc.New(server.URL).SetMulticallContract(common.HexToAddress(multicall3))
}

func smallPoolQuote(token common.Address, amount *big.Int, reserve int64) (*big.Int, uint8) {
	if token != common.HexToAddress(testBase) {
		return big.NewInt(0), 3
	}
	out := new(big.Int).Mul(amount, big.NewInt(100_000_000))
	out.Div(out, big.NewInt(1_000_000_000_000_000_000))
	if out.Sign() == 0 || out.Cmp(big.NewInt(reserve)) >= 0 {
		return big.NewInt(0), 3
	}
	return out, 0
}

func TestTrackerSmallSingleSidedPool(t *testing.T) {
	for _, reserve := range []int64{1_000_000, 100} {
		for _, dust := range []int64{0, 1} {
			t.Run(big.NewInt(reserve).String()+"/"+big.NewInt(dust).String(), func(t *testing.T) {
				p := testEntity()
				p.Tokens[0].Decimals = 6
				p.Tokens[1].Decimals = 18
				count := 0
				tracked, err := NewPoolTracker(&Config{}, mockRPC(t, false, false, &count, nil, reserve, dust)).GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
				require.NoError(t, err)
				minimumCalls := 3
				if dust != 0 {
					minimumCalls = 4
				}
				require.GreaterOrEqual(t, count, minimumCalls)
				require.LessOrEqual(t, count, minimumCalls+maxRefinementRounds)
				var extra Extra
				require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
				require.Empty(t, extra.Ladders[0])
				require.NotEmpty(t, extra.Ladders[1])
				sim, err := NewPoolSimulatorWith(tracked, math.MaxInt64)
				require.NoError(t, err)
				amount := new(big.Int).Mul(big.NewInt(reserve), big.NewInt(1_000_000_000))
				out, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: testBase, Amount: amount}, TokenOut: testQuote})
				require.NoError(t, err)
				require.Positive(t, out.TokenAmountOut.Amount.Sign())
				require.Less(t, out.TokenAmountOut.Amount.Int64(), reserve)
			})
		}
	}
}

func testEntity() entity.Pool {
	return entity.Pool{Address: testPool, Exchange: DexType, Type: DexType, Reserves: entity.PoolReserves{"0", "0"}, Extra: "{}",
		Tokens: []*entity.PoolToken{{Address: testQuote, Swappable: true}, {Address: testBase, Swappable: true}}}
}

func TestRegisteredFactories(t *testing.T) {
	client := mockRPC(t, false, false, nil, nil)
	listerFactory := poollist.Factory(DexType)
	require.NotNil(t, listerFactory)
	lister, err := listerFactory(DexType, poollist.FactoryParams{
		Properties:   poollist.Properties{"pools": []string{testPool}},
		Dependencies: poollist.Dependencies{EthrpcClient: client},
	})
	require.NoError(t, err)
	pools, _, err := lister.GetNewPools(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.Equal(t, DexType, pools[0].Exchange)
	require.Equal(t, DexType, pools[0].Type)

	trackerFactory := pooltrack.Factory(DexType)
	require.NotNil(t, trackerFactory)
	tracker, err := trackerFactory(DexType, pooltrack.FactoryParams{
		Dependencies: pooltrack.Dependencies{EthrpcClient: client},
	})
	require.NoError(t, err)
	tracked, err := tracker.GetNewPoolState(t.Context(), pools[0], pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	simulatorFactory := pool.Factory(DexType)
	require.NotNil(t, simulatorFactory)
	sim, err := simulatorFactory(pool.FactoryParams{EntityPool: tracked})
	require.NoError(t, err)
	out, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testQuote, Amount: big.NewInt(25)},
		TokenOut:      testBase,
	})
	require.NoError(t, err)
	require.Equal(t, int64(50), out.TokenAmountOut.Amount.Int64())
	require.Equal(t, int64(defaultGas), out.Gas)
	require.Equal(t, ladder.PoolMeta{BlockNumber: tracked.BlockNumber}, sim.GetMetaInfo(testQuote, testBase))
}

func TestDiscoveryDeduplicatesAndRetriesFailures(t *testing.T) {
	cfg := &Config{DexId: DexType, Pools: []string{testPool, common.HexToAddress(testPool).Hex()}}
	failed := NewPoolsListUpdater(cfg, mockRPC(t, false, true, nil, nil))
	pools, meta, err := failed.GetNewPools(t.Context(), nil)
	require.NoError(t, err)
	require.Empty(t, pools)
	count := 0
	lister := NewPoolsListUpdater(cfg, mockRPC(t, false, false, &count, nil))
	pools, meta, err = lister.GetNewPools(t.Context(), meta)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.Equal(t, testQuote, pools[0].Tokens[0].Address)
	require.Equal(t, testBase, pools[0].Tokens[1].Address)
	require.Equal(t, entity.PoolReserves{"0", "0"}, pools[0].Reserves)
	pools, _, err = lister.GetNewPools(t.Context(), meta)
	require.NoError(t, err)
	require.Empty(t, pools)
	require.Equal(t, 1, count)
}

func TestTrackerAtomicStatePartialLadderAndOverrides(t *testing.T) {
	count := 0
	overrides := false
	tracker := NewPoolTracker(&Config{}, mockRPC(t, false, false, &count, &overrides))
	p, err := tracker.GetNewPoolStateWithOverrides(t.Context(), testEntity(), pool.GetNewPoolStateWithOverridesParams{
		Overrides: map[common.Address]gethclient.OverrideAccount{common.HexToAddress(testPool): {Nonce: 1}},
	})
	require.NoError(t, err)
	require.Equal(t, 5, count)
	require.True(t, overrides)
	require.Equal(t, uint64(104), p.BlockNumber)
	require.Equal(t, int64(1_800_000_004), p.Timestamp)
	require.Equal(t, entity.PoolReserves{"900", "1100"}, p.Reserves)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	for _, points := range extra.Ladders {
		require.NotEmpty(t, points)
		require.Equal(t, uint64(300), points[len(points)-1][0].Uint64())
	}
	s, err := NewPoolSimulatorWith(p, math.MaxInt64)
	require.NoError(t, err)
	q := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: testQuote, Amount: big.NewInt(10)}, TokenOut: testBase}
	first, err := s.CalcAmountOut(q)
	require.NoError(t, err)
	require.Equal(t, int64(20), first.TokenAmountOut.Amount.Int64())
	clone := s.CloneState().(*PoolSimulator)
	clone.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: q.TokenAmountIn, TokenAmountOut: *first.TokenAmountOut, Fee: *first.Fee})
	require.Equal(t, "900", s.GetReserves()[0].String())
	require.Equal(t, "910", clone.GetReserves()[0].String())
	second, err := s.CalcAmountOut(q)
	require.NoError(t, err)
	require.Equal(t, first.TokenAmountOut.Amount, second.TokenAmountOut.Amount)
	require.Equal(t, ladder.PoolMeta{BlockNumber: 104}, s.GetMetaInfo(testQuote, testBase))
	q.TokenAmountIn.Amount = big.NewInt(290)
	large, err := s.CalcAmountOut(q)
	require.NoError(t, err)
	require.Equal(t, int64(580), large.TokenAmountOut.Amount.Int64())
	p.Timestamp = time.Now().Add(-MaxAge - time.Second).Unix()
	_, err = NewPoolSimulatorWith(p, MaxAge)
	require.ErrorIs(t, err, ladder.ErrStale)
}

func TestTrackerPausedClearsLadders(t *testing.T) {
	p, err := NewPoolTracker(&Config{}, mockRPC(t, true, false, nil, nil)).GetNewPoolState(t.Context(), testEntity(), pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	require.Empty(t, extra.Ladders[0])
	require.Empty(t, extra.Ladders[1])
	s, err := NewPoolSimulatorWith(p, math.MaxInt64)
	require.NoError(t, err)
	_, err = s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: testQuote, Amount: big.NewInt(10)}, TokenOut: testBase})
	require.Error(t, err)
}

func TestCollectQuotesRejectsMalformedResponse(t *testing.T) {
	for _, result := range []ladderResult{
		{}, {Status: 6, Outs: []*big.Int{big.NewInt(1)}, Fees: []*big.Int{big.NewInt(0)}},
		{Outs: []*big.Int{nil}, Fees: []*big.Int{big.NewInt(0)}}, {Outs: []*big.Int{big.NewInt(1)}},
	} {
		_, err := toLadderQuote(testAmounts(1), result)
		require.ErrorIs(t, err, ErrInvalidState)
	}
}

func TestLiveUSCore(t *testing.T) {
	url := os.Getenv("USCORE_RPC_URL")
	if url == "" {
		t.Skip("set USCORE_RPC_URL for live HyperEVM checks")
	}
	client := ethrpc.New(url).SetMulticallContract(common.HexToAddress(multicall3))
	cfg := &Config{DexId: DexType, Pools: []string{testPool, "0xb9fa3bdfa88da2dc78c20ca03472e043992c6671"}}
	pools, _, err := NewPoolsListUpdater(cfg, client).GetNewPools(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 2)
	for _, p := range pools {
		p.Tokens[0].Decimals = 6
		p.Tokens[1].Decimals = 18
		for cycle := 0; cycle < 3; cycle++ {
			p, err = NewPoolTracker(cfg, client).GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
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
		q, err := toLadderQuote(points, sampled[dir])
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
