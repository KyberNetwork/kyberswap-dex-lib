package uniswapv4

import (
	"context"
	stdjson "encoding/json"
	"math"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// avaxDustPool is the router-api payload for the Avalanche BTC.b/WETH.e 0.01% V4 pool, verbatim.
//
// It holds a single full-range position of 4787383 liquidity, which at the current price works out
// to reserves of 2 units of BTC.b and 1.05e13 of WETH.e — roughly five cents of inventory. That is
// what makes it the sharpest case for word-bounded stepping: one bitmap word of price movement
// costs a fraction of one unit of token0, so the on-chain per-word round-up of amountIn dominates
// everything else in the swap.
const avaxDustPool = `{
  "address": "0x19644199c9a151af99960a39a43686bac3522857b16a8f537a162e791dab0871",
  "swapFee": 100,
  "exchange": "uniswap-v4",
  "type": "uniswap-v4",
  "timestamp": 1772817016,
  "reserves": ["2", "10466736059725"],
  "tokens": [
    {"address":"0x152b9d0fdc40c096757f570a51e494bd4b943e50","symbol":"BTC.b","decimals":8,"swappable":true},
    {"address":"0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab","symbol":"WETH.e","decimals":18,"swappable":true}
  ],
  "extra": "{\"liquidity\":4787383,\"sqrtPriceX96\":173217865696950102619138552042419888,\"tickSpacing\":1,\"tick\":291969,\"ticks\":[{\"index\":-887272,\"liquidityGross\":4787383,\"liquidityNet\":4787383},{\"index\":887272,\"liquidityGross\":4787383,\"liquidityNet\":-4787383}]}",
  "staticExtra": "{\"0x0\":[false,false],\"fee\":100,\"tS\":1,\"hooks\":\"0x0000000000000000000000000000000000000000\",\"uR\":\"0x94b75331ae8d42c1b61065089b7d48fe14aa73b7\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}",
  "blockNumber": 79724099
}`

const (
	avaxBTCb  = "0x152b9d0fdc40c096757f570a51e494bd4b943e50"
	avaxWETHe = "0x49d5c2bdffac6ce2bfdb6640f4f80f226bc10bab"
)

// TestDustPoolWordBoundedSteps pins the simulator to a PoolManager.swap that actually ran.
//
// Before word-bounded stepping the simulator jumped straight from tick 291969 to the only
// initialized tick below it (-887272), spent all 18 post-fee units of input on moving the price,
// and quoted 9331551613090 — 89% of the pool's entire token1 balance. The chain instead walked 9
// bitmap words, each costing 1 unit of input rounded up plus 1 unit of fee, exhausted the 19 units
// after 2178 ticks, and paid out 1079518536551. Routing on the old quote is what made the real
// transaction revert with "Return amount is not enough".
//
// Ground truth is Tenderly simulation 9a8a0d70-28e2-4222-b54f-0d04392f167a, whose Swap event reads
// amount0 -19, amount1 1079518536551, sqrtPriceX96 155352516287591836139164025014435689,
// tick 289791, liquidity 4787383.
func TestDustPoolWordBoundedSteps(t *testing.T) {
	t.Parallel()

	var poolEnt entity.Pool
	require.NoError(t, json.Unmarshal([]byte(avaxDustPool), &poolEnt))
	sim, err := NewPoolSimulator(poolEnt, valueobject.ChainIDAvalancheCChain)
	require.NoError(t, err)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: avaxBTCb, Amount: bignumber.NewBig10("19")},
		TokenOut:      avaxWETHe,
	})
	require.NoError(t, err)
	require.Equal(t, "1079518536551", res.TokenAmountOut.Amount.String())

	// Gas has to see the same work. This swap crosses no initialized tick at all, so under the
	// old model — which priced only initialized crossings — it came out as bare base gas, making
	// the pool look as cheap as it looked deep. The nine bitmap words it really walks are what
	// PoolManager charged 92773 gas for.
	require.Equal(t, DefaultGas.BaseGas+9*CrossEmptyWordGas, res.Gas,
		"nine empty-word crossings, no initialized ticks")
}

func TestPolygonEmptyWordGasIsV4AndChainScoped(t *testing.T) {
	t.Parallel()

	var poolEnt entity.Pool
	require.NoError(t, json.Unmarshal([]byte(avaxDustPool), &poolEnt))
	defaultGas := DefaultGas

	polygonSim, err := NewPoolSimulator(poolEnt, valueobject.ChainIDPolygon)
	require.NoError(t, err)
	avaxSim, err := NewPoolSimulator(poolEnt, valueobject.ChainIDAvalancheCChain)
	require.NoError(t, err)

	amountIn := pool.TokenAmount{Token: avaxBTCb, Amount: bignumber.NewBig10("19")}
	polygonOut, err := polygonSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: amountIn, TokenOut: avaxWETHe,
	})
	require.NoError(t, err)
	avaxOut, err := avaxSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: amountIn, TokenOut: avaxWETHe,
	})
	require.NoError(t, err)
	require.Equal(t, avaxOut.TokenAmountOut.Amount, polygonOut.TokenAmountOut.Amount)
	require.Equal(t, DefaultGas.BaseGas+9*CrossEmptyWordGas, avaxOut.Gas)
	require.Equal(t, DefaultGas.BaseGas+9*polygonCrossEmptyWordGas, polygonOut.Gas)

	// Exact-out quotes use the same chain-specific scan cost and retain it across cloning.
	polygonClone := polygonSim.CloneState().(*PoolSimulator)
	polygonIn, err := polygonClone.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: *polygonOut.TokenAmountOut, TokenIn: avaxBTCb,
	})
	require.NoError(t, err)
	avaxIn, err := avaxSim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: *avaxOut.TokenAmountOut, TokenIn: avaxBTCb,
	})
	require.NoError(t, err)
	require.Equal(t, avaxIn.TokenAmountIn.Amount, polygonIn.TokenAmountIn.Amount)
	wordCount := (avaxIn.Gas - DefaultGas.BaseGas) / CrossEmptyWordGas
	require.Greater(t, wordCount, int64(0))
	require.Equal(t, wordCount*(polygonCrossEmptyWordGas-CrossEmptyWordGas),
		polygonIn.Gas-avaxIn.Gas)
	require.Equal(t, polygonCrossEmptyWordGas, polygonClone.Gas.CrossEmptyWordGas)

	// Constructor overrides must not mutate the package default or alter non-Polygon V4 sims.
	require.Equal(t, defaultGas, DefaultGas)
	require.Equal(t, defaultGas, avaxSim.Gas)
}

// TestDustPoolOutputIsBoundedByWordCost checks the property the on-chain loop enforces and the
// single-jump simulator broke: a bitmap word costs at least one unit of input plus its fee, so the
// price can only travel as far as the input can pay for word by word.
//
// Without word-bounded stepping all of these collapse toward the pool's whole token1 balance —
// 10 units alone quoted 8418510341968, 80% of it — which is the shape of the bug rather than of
// the pool.
//
// The expectations come from an independent transcription of V4's swap loop, checked against the
// Tenderly trace pinned in TestDustPoolWordBoundedSteps at 19 units in.
func TestDustPoolOutputIsBoundedByWordCost(t *testing.T) {
	t.Parallel()

	var poolEnt entity.Pool
	require.NoError(t, json.Unmarshal([]byte(avaxDustPool), &poolEnt))

	for _, tc := range []struct {
		amountIn string
		want     string
	}{
		{"2", "67380838795"},       // 1 word crossed, ends at tick 291840
		{"5", "199637723176"},      // 2 words, tick 291584
		{"10", "586401573504"},     // 5 words, tick 290816
		{"19", "1079518536551"},    // 9 words, tick 289792
		{"100", "4912420720935"},   // 50 words, tick 279296
		{"1000", "10399346476185"}, // 394 words, tick 191232
	} {
		t.Run(tc.amountIn, func(t *testing.T) {
			t.Parallel()

			sim, err := NewPoolSimulator(poolEnt, valueobject.ChainIDAvalancheCChain)
			require.NoError(t, err)

			res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: avaxBTCb, Amount: bignumber.NewBig10(tc.amountIn)},
				TokenOut:      avaxWETHe,
			})
			require.NoError(t, err)
			require.Equal(t, tc.want, res.TokenAmountOut.Amount.String())
			require.Less(t, res.TokenAmountOut.Amount.Uint64(), uint64(10466736059725),
				"output must stay inside the pool's token1 balance")
		})
	}
}

// v4 swaps of sampled routes that walked 50 or more tick-bitmap words, measured on an Ethereum fork at block
// 26307077 (2026-10-30) by a debug_traceCall of the route's build: the PoolManager's swap and the pool's transfers,
// the executor's own share left out. A word costs 5,200-6,500 gas on-chain here (the PoolManager's loop reads the
// bitmap from its own storage), against the 2,500 the first estimate charged.
func TestEmptyWordGasMatchesTheChain(t *testing.T) {
	for _, w := range []struct {
		pool         string
		ticks, words int64
		spent        int64
	}{
		{"0x30011472", 0, 278, 1_623_950}, // USDC -> 0x66a1
		{"0x0c371c47", 1, 56, 431_639},    // held out of the fit
	} {
		got := DefaultGas.BaseGas + DefaultGas.CrossInitTickGas*w.ticks + DefaultGas.CrossEmptyWordGas*w.words
		if off := math.Abs(float64(got-w.spent)) / float64(w.spent); off > 0.10 {
			t.Errorf("%s (%d ticks, %d words): priced %d, spent %d on-chain (%.0f%% off)", w.pool, w.ticks, w.words, got,
				w.spent, off*100)
		}
	}
}

const (
	svPoolID      = "0x70bf44c3a9b6b047bf60e5a05968225dbf3d6a5b9e8a95a73727e48921e889c1"
	svStateView   = "0x76fd297e2d437cd7f76d50f01afe6160f86e9990"
	svTickSpacing = 60
)

var burnTopic = common.HexToHash("0x0c396cd989a39f4459b5fa1aed6a9a8dcdbc45908acfd67e028cd568da98982c")

type svTick struct{ gross, net int64 }

// svChainTicks is the pool's on-chain tick state: initialized ticks in several bitmap words, both signs.
func svChainTicks() map[int64]svTick {
	return map[int64]svTick{
		-887220: {100, 100}, -60000: {7, 7}, -600: {50, -20}, -60: {3, 3},
		0: {9, 9}, 60: {4, -4}, 15420: {11, -1}, 60000: {7, -7}, 887220: {111, -94},
	}
}

var multicallAggregateABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(`[{"name":"aggregate","type":"function","stateMutability":"view",
		"inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
		"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}]}]`))
	if err != nil {
		panic(err)
	}
	return a
}()

// svCalls counts what a stub served: multicalls and the calls inside them.
type svCalls struct{ multicalls, inner int }

// stateViewStub answers multicall aggregates of StateView getTickBitmap and getTickInfo from ticks, and counts them.
func stateViewStub(t *testing.T, ticks map[int64]svTick, spacing int64, calls *svCalls) *httptest.Server {
	t.Helper()
	bitmaps := map[int16]*big.Int{}
	for tick := range ticks {
		compressed := tick / spacing
		word, bit := int16(compressed>>8), uint(compressed&0xff)
		if bitmaps[word] == nil {
			bitmaps[word] = new(big.Int)
		}
		bitmaps[word].SetBit(bitmaps[word], int(bit), 1)
	}
	// PoolManager storage for the same pool, with the bitmap slots derived here from the Solidity mapping rule
	// (keccak256(abi.encode(int16 key, slot))) rather than from the tracker's helpers.
	base := stateSlot(common.HexToHash(svPoolID))
	storage := map[common.Hash][32]byte{}
	int16Slot := abi.Arguments{{Type: mustType("int16")}, {Type: mustType("uint256")}}
	for word, bm := range bitmaps {
		enc, err := int16Slot.Pack(word, new(big.Int).Add(base.Big(), big.NewInt(5)))
		require.NoError(t, err)
		var v [32]byte
		bm.FillBytes(v[:])
		storage[crypto.Keccak256Hash(enc)] = v
	}
	for tick, tk := range ticks {
		var v [32]byte
		big.NewInt(tk.gross).FillBytes(v[16:])
		net := big.NewInt(tk.net)
		if net.Sign() < 0 {
			net.Add(net, new(big.Int).Lsh(big.NewInt(1), 128))
		}
		net.FillBytes(v[:16])
		storage[tickSlot(base, tick)] = v
	}
	answer := func(data []byte) []byte {
		m, err := stateViewABI.MethodById(data[:4])
		if err != nil {
			m, err = poolManagerABI.MethodById(data[:4])
		}
		require.NoError(t, err)
		args, err := m.Inputs.Unpack(data[4:])
		require.NoError(t, err)
		var out []byte
		switch m.Name {
		case "getTickBitmap":
			bm := bitmaps[args[1].(int16)]
			if bm == nil {
				bm = new(big.Int)
			}
			out, err = m.Outputs.Pack(bm)
		case "getTickInfo":
			tk := ticks[args[1].(*big.Int).Int64()]
			out, err = m.Outputs.Pack(big.NewInt(tk.gross), big.NewInt(tk.net), new(big.Int), new(big.Int))
		case "extsload":
			out, err = m.Outputs.Pack(storage[args[0].([32]byte)])
		default:
			t.Fatalf("unexpected StateView call %s", m.Name)
		}
		require.NoError(t, err)
		return out
	}
	counts := calls
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     stdjson.RawMessage   `json:"id"`
			Params []stdjson.RawMessage `json:"params"`
		}
		require.NoError(t, stdjson.NewDecoder(r.Body).Decode(&req))
		var call struct {
			Input string `json:"input"`
			Data  string `json:"data"`
		}
		require.NoError(t, stdjson.Unmarshal(req.Params[0], &call))
		raw := call.Input
		if raw == "" {
			raw = call.Data
		}
		data := hexutil.MustDecode(raw)
		args, err := multicallAggregateABI.Methods["aggregate"].Inputs.Unpack(data[4:])
		require.NoError(t, err)
		calls := args[0].([]struct {
			Target   common.Address `json:"target"`
			CallData []byte         `json:"callData"`
		})
		rets := make([][]byte, len(calls))
		for i, c := range calls {
			rets[i] = answer(c.CallData)
		}
		counts.multicalls++
		counts.inner += len(calls)
		ret, err := multicallAggregateABI.Methods["aggregate"].Outputs.Pack(big.NewInt(100), rets)
		require.NoError(t, err)
		resp, _ := stdjson.Marshal(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": hexutil.Encode(ret)})
		_, _ = w.Write(resp)
	}))
}

func mustType(name string) abi.Type {
	ty, err := abi.NewType(name, "", nil)
	if err != nil {
		panic(err)
	}
	return ty
}

// svTracker reads ticks through StateView, or through PoolManager.extsload when stateView is empty.
func svTracker(url, stateView string) *PoolTracker {
	return &PoolTracker{
		config: &Config{DexID: DexType, StateViewAddress: stateView, FetchTickFromStateView: true,
			PoolManagerAddress: "0x000000000004444c5dc75cb358380d2e3de08a90"},
		ethrpcClient: ethrpc.New(url).
			SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
	}
}

func svPool(t *testing.T, ticks map[int64]svTick) entity.Pool {
	t.Helper()
	p := entity.Pool{Address: svPoolID, StaticExtra: `{"tS":60}`}
	if ticks == nil { // a pool the lister just added: no extra yet
		return p
	}
	extra := Extra{Extra: &uniswapv3.Extra{}}
	for idx, tk := range ticks {
		extra.Ticks = append(extra.Ticks, Tick{Index: int(idx), LiquidityGross: big.NewInt(tk.gross),
			LiquidityNet: big.NewInt(tk.net)})
	}
	b, err := stdjson.Marshal(extra)
	require.NoError(t, err)
	p.Extra = string(b)
	return p
}

// svResult turns the tracker's ticks into the same map shape, failing on a duplicate or unsorted tick.
func svResult(t *testing.T, got []ticklensTick) map[int64]svTick {
	t.Helper()
	res := map[int64]svTick{}
	require.True(t, sort.SliceIsSorted(got, func(i, j int) bool { return got[i].idx < got[j].idx }), "ticks sorted")
	for _, g := range got {
		_, dup := res[g.idx]
		require.False(t, dup, "tick %d twice", g.idx)
		res[g.idx] = svTick{g.gross, g.net}
	}
	return res
}

type ticklensTick struct{ idx, gross, net int64 }

func svTicks(t *testing.T, pt *PoolTracker, p entity.Pool, logs []types.Log) (map[int64]svTick, error) {
	t.Helper()
	resp, err := pt.getPoolTicksFromStateView(context.Background(), p, pool.GetNewPoolStateParams{Logs: logs})
	if err != nil {
		return nil, err
	}
	got := make([]ticklensTick, len(resp))
	for i, r := range resp {
		idx, err := strconv.ParseInt(r.TickIdx, 10, 64)
		require.NoError(t, err)
		gross, ok := new(big.Int).SetString(r.LiquidityGross, 10)
		require.True(t, ok)
		net, ok := new(big.Int).SetString(r.LiquidityNet, 10)
		require.True(t, ok)
		got[i] = ticklensTick{idx, gross.Int64(), net.Int64()}
	}
	return svResult(t, got), nil
}

func burnLog(lower, upper int) types.Log {
	return types.Log{Topics: []common.Hash{burnTopic, {}, tickTopic(lower), tickTopic(upper)}}
}

// A bootstrap has no logs. With FetchTickFromStateView it must read every initialized tick, for a new pool (no
// extra yet) and for a pool whose stored ticks are stale.
func TestStateViewTicks_BootstrapScansAllTicks(t *testing.T) {
	chain := svChainTicks()
	for name, extra := range map[string]string{
		"new pool":    "",
		"extra {}":    "{}",
		"stale ticks": svPool(t, map[int64]svTick{-600: {1, 1}, 60: {1, 1}, 120: {5, 5}}).Extra,
		"empty ticks": svPool(t, map[int64]svTick{}).Extra,
	} {
		for reader, stateView := range map[string]string{"stateview": svStateView, "extsload": ""} {
			t.Run(name+"/"+reader, func(t *testing.T) {
				var n svCalls
				stub := stateViewStub(t, chain, svTickSpacing, &n)
				defer stub.Close()
				p := svPool(t, nil)
				p.Extra = extra
				got, err := svTicks(t, svTracker(stub.URL, stateView), p, nil)
				require.NoError(t, err)
				require.Equal(t, chain, got)
			})
		}
	}
}

// An update whose logs change more than maxChangedTicks ticks must still give the pool's ticks, read in full.
func TestStateViewTicks_ManyChangedTicksScansAllTicks(t *testing.T) {
	chain := svChainTicks()
	stored := map[int64]svTick{}
	for idx, tk := range chain {
		stored[idx] = svTick{tk.gross + 1, tk.net + 1} // stale everywhere
	}
	var logs []types.Log
	for i := 0; i <= maxChangedTicks/2; i++ { // maxChangedTicks + 2 distinct ticks
		logs = append(logs, burnLog(-6000-60*i, 6000+60*i))
	}
	var n svCalls
	stub := stateViewStub(t, chain, svTickSpacing, &n)
	defer stub.Close()
	got, err := svTicks(t, svTracker(stub.URL, svStateView), svPool(t, stored), logs)
	require.NoError(t, err)
	require.Equal(t, chain, got)
}

func modifyLiquidityLog(t *testing.T, poolID string, lower, upper int64) types.Log {
	t.Helper()
	ev := poolManagerABI.Events["ModifyLiquidity"]
	data, err := ev.Inputs.NonIndexed().Pack(big.NewInt(lower), big.NewInt(upper), big.NewInt(-1), [32]byte{})
	require.NoError(t, err)
	return types.Log{Topics: []common.Hash{ev.ID, common.HexToHash(poolID), {}}, Data: data}
}

// v4 ModifyLiquidity logs carry their ticks in data. An update with a few of them reads just those ticks: a changed
// tick, a newly initialized one and a cleared one.
func TestStateViewTicks_ModifyLiquidityIsIncremental(t *testing.T) {
	chain := svChainTicks()
	stored := map[int64]svTick{}
	for idx, tk := range chain {
		stored[idx] = tk
	}
	stored[-600] = svTick{1, 1} // stale
	chain[120] = svTick{5, 5}   // new on chain
	delete(chain, 60)           // cleared on chain
	var n svCalls
	stub := stateViewStub(t, chain, svTickSpacing, &n)
	defer stub.Close()
	logs := []types.Log{modifyLiquidityLog(t, svPoolID, -600, 120), modifyLiquidityLog(t, svPoolID, -600, 60)}
	got, err := svTicks(t, svTracker(stub.URL, svStateView), svPool(t, stored), logs)
	require.NoError(t, err)
	require.Equal(t, chain, got)
	require.Equal(t, svCalls{multicalls: 1, inner: 3}, n)
}

// Logs that change none of the pool's ticks (a swap, another pool's ModifyLiquidity) keep the stored ticks, with no
// call.
func TestStateViewTicks_NoChangedTicksKeepsStoredTicks(t *testing.T) {
	stored := map[int64]svTick{-600: {1, 1}, 60: {4, -4}}
	var n svCalls
	stub := stateViewStub(t, svChainTicks(), svTickSpacing, &n)
	defer stub.Close()
	swap := types.Log{Topics: []common.Hash{poolManagerABI.Events["Swap"].ID, common.HexToHash(svPoolID), {}}}
	other := modifyLiquidityLog(t, "0x01", -600, 60)
	got, err := svTicks(t, svTracker(stub.URL, svStateView), svPool(t, stored), []types.Log{swap, other})
	require.NoError(t, err)
	require.Equal(t, stored, got)
	require.Zero(t, n)
}

// The calls of one bootstrap: ceil(words/500) bitmap multicalls over the words covering [minTick, maxTick], and
// ceil(ticks/500) tick-info multicalls.
func TestStateViewTicks_BootstrapCallCount(t *testing.T) {
	for _, tc := range []struct {
		spacing           int64
		words, multicalls int
	}{
		{1, 6932, 14 + 1},
		{10, 694, 2 + 1},
		{60, 116, 1 + 1},
	} {
		t.Run(strconv.FormatInt(tc.spacing, 10), func(t *testing.T) {
			chain := map[int64]svTick{}
			for _, tk := range []int64{-887220, -60000, -600, 0, 60, 60000, 887220} {
				chain[tk/tc.spacing*tc.spacing] = svTick{1, 1}
			}
			var n svCalls
			stub := stateViewStub(t, chain, tc.spacing, &n)
			defer stub.Close()
			p := svPool(t, nil)
			p.StaticExtra = `{"tS":` + strconv.FormatInt(tc.spacing, 10) + `}`
			got, err := svTicks(t, svTracker(stub.URL, svStateView), p, nil)
			require.NoError(t, err)
			require.Equal(t, chain, got)
			require.Equal(t, svCalls{multicalls: tc.multicalls, inner: tc.words + len(chain)}, n)
			t.Logf("tickSpacing %d: %d multicalls, %d calls (%d words, %d ticks)", tc.spacing, n.multicalls, n.inner,
				tc.words, len(chain))
		})
	}
}

// A few changed ticks are still read one by one (the existing incremental path).
func TestStateViewTicks_FewChangedTicksReadsThem(t *testing.T) {
	chain := svChainTicks()
	stored := map[int64]svTick{}
	for idx, tk := range chain {
		stored[idx] = tk
	}
	stored[-600] = svTick{1, 1}
	stored[60] = svTick{1, 1}
	var n svCalls
	stub := stateViewStub(t, chain, svTickSpacing, &n)
	defer stub.Close()
	got, err := svTicks(t, svTracker(stub.URL, svStateView), svPool(t, stored), []types.Log{burnLog(-600, 60)})
	require.NoError(t, err)
	require.Equal(t, chain, got)
	require.Equal(t, svCalls{multicalls: 1, inner: 2}, n)
}

// A failed tick read says what failed instead of blaming the changed-tick count.
func TestStateViewTicks_ErrorsNameCause(t *testing.T) {
	t.Run("bad extra", func(t *testing.T) {
		var n svCalls
		stub := stateViewStub(t, svChainTicks(), svTickSpacing, &n)
		defer stub.Close()
		p := svPool(t, nil)
		p.Extra = "{"
		_, err := svTicks(t, svTracker(stub.URL, svStateView), p, nil)
		require.ErrorContains(t, err, "failed to unmarshal pool extra: ") // the decoder's reason follows
	})
	t.Run("rpc down", func(t *testing.T) {
		stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "down", http.StatusBadGateway)
		}))
		defer stub.Close()
		_, err := svTicks(t, svTracker(stub.URL, svStateView), svPool(t, nil), nil)
		require.ErrorContains(t, err, "tick bitmap")
		require.NotErrorIs(t, err, ErrTooManyChangedTicks)
	})
}

// Vectors from the Ethereum v4 ETH/USDC 0.05% pool at block 23000000: StateView getTickBitmap(id, -347) and
// PoolManager extsload at this slot both returned 2^105 (the full-range tick -887270).
func TestTickBitmapSlotMatchesMainnet(t *testing.T) {
	base := stateSlot(common.HexToHash("0x21c67e77068de97969ba93d4aab21826d33ca12bb9f565d8496e8fda8a82ca27"))
	require.Equal(t, "0xda8cac368d67cd2f2d8aaa5cc531768e0fa3b1d205c5c5de60da078e1f59bdfc", base.Hex())
	require.Equal(t, "0x19ba78b67e2b236ca0d2b269d1e37eda197aa6f1a02f37488f31ef7f1a5364d3",
		tickBitmapSlot(base, -347).Hex())
	require.NotEqual(t, tickBitmapSlot(base, -1), tickBitmapSlot(base, 1))
}
