package uniswapv4

import (
	"context"
	stdjson "encoding/json"
	"maps"
	"math"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

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

// svCalls counts what a stub served: multicalls and the calls inside them.
type svCalls struct{ multicalls, inner int }

// svCount is what stub served so far.
func svCount(stub *testutil.MulticallStub) svCalls {
	var n svCalls
	for _, targets := range stub.Targets() {
		n.multicalls++
		n.inner += len(targets)
	}
	return n
}

// stateViewStub answers multicall aggregates of StateView getTickBitmap and getTickInfo from ticks, and counts them.
func stateViewStub(t *testing.T, ticks map[int64]svTick, spacing int64) *testutil.MulticallStub {
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
	return testutil.NewMulticallStub(t, func(_ common.Address, data []byte) (bool, []byte) {
		return true, answer(data)
	})
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

// svTicks reads p's ticks through pt, as a map, failing on an unsorted or duplicate tick.
func svTicks(t *testing.T, pt *PoolTracker, p entity.Pool, logs []types.Log) (map[int64]svTick, error) {
	t.Helper()
	resp, err := pt.getPoolTicksFromStateView(context.Background(), p, pool.GetNewPoolStateParams{Logs: logs})
	if err != nil {
		return nil, err
	}
	got, last := map[int64]svTick{}, int64(math.MinInt64)
	for _, r := range resp {
		idx, err := strconv.ParseInt(r.TickIdx, 10, 64)
		require.NoError(t, err)
		require.Greater(t, idx, last, "ticks sorted, each once")
		last = idx
		gross, _ := new(big.Int).SetString(r.LiquidityGross, 10)
		net, _ := new(big.Int).SetString(r.LiquidityNet, 10)
		require.True(t, gross != nil && net != nil, "tick %d liquidity", idx)
		got[idx] = svTick{gross.Int64(), net.Int64()}
	}
	return got, nil
}

func burnLog(lower, upper int) types.Log {
	return types.Log{Topics: []common.Hash{burnTopic, {}, tickTopic(lower), tickTopic(upper)}}
}

func modifyLiquidityLog(t *testing.T, poolID string, lower, upper int64) types.Log {
	t.Helper()
	ev := poolManagerABI.Events["ModifyLiquidity"]
	data, err := ev.Inputs.NonIndexed().Pack(big.NewInt(lower), big.NewInt(upper), big.NewInt(-1), [32]byte{})
	require.NoError(t, err)
	return types.Log{Topics: []common.Hash{ev.ID, common.HexToHash(poolID), {}}, Data: data}
}

// svFullScan is what reading svChainTicks in full costs at tickSpacing 60: one bitmap and one tick-info multicall,
// over 116 words and the initialized ticks.
func svFullScan(ticks int) svCalls { return svCalls{multicalls: 2, inner: 116 + ticks} }

// The routing switch: a read without logs, a pool never tracked, or more changed ticks than maxChangedTicks reads
// every initialized tick; a few changed ticks are read one by one; logs that change none keep the stored ticks.
func TestStateViewTicks_Routing(t *testing.T) {
	chain := svChainTicks()
	stale := map[int64]svTick{}
	for idx, tk := range chain {
		stale[idx] = svTick{tk.gross + 1, tk.net + 1}
	}
	patched := maps.Clone(chain) // -600 changed, 120 newly initialized, 60 cleared
	patched[-600], patched[120] = svTick{1, 1}, svTick{5, 5}
	delete(patched, 60)
	var many []types.Log
	for i := 0; i <= maxChangedTicks/2; i++ { // maxChangedTicks + 2 distinct ticks
		many = append(many, burnLog(-6000-60*i, 6000+60*i))
	}
	swap := types.Log{Topics: []common.Hash{poolManagerABI.Events["Swap"].ID, common.HexToHash(svPoolID), {}}}

	for _, tc := range []struct {
		name          string
		stored, onChn map[int64]svTick // stored nil: no extra yet
		block         uint64
		logs          []types.Log
		want          map[int64]svTick
		calls         svCalls
	}{
		{"bootstrap, new pool", nil, chain, 0, nil, chain, svFullScan(len(chain))},
		{"bootstrap, stale ticks", stale, chain, 100, nil, chain, svFullScan(len(chain))},
		{"many changed ticks", stale, chain, 100, many, chain, svFullScan(len(chain))},
		{"burn: two ticks", stale, chain, 100, []types.Log{burnLog(-600, 60)},
			mergeTicks(stale, chain, -600, 60), svCalls{multicalls: 1, inner: 2}},
		{"modify liquidity: changed, new, cleared", chain, patched, 100,
			[]types.Log{modifyLiquidityLog(t, svPoolID, -600, 120), modifyLiquidityLog(t, svPoolID, -600, 60)},
			patched, svCalls{multicalls: 1, inner: 3}},
		{"drained, tracked: patched", map[int64]svTick{}, map[int64]svTick{-600: {5, 5}, 120: {5, -5}}, 100,
			[]types.Log{modifyLiquidityLog(t, svPoolID, -600, 120)},
			map[int64]svTick{-600: {5, 5}, 120: {5, -5}}, svCalls{multicalls: 1, inner: 2}},
		{"drained, never tracked: full", map[int64]svTick{}, chain, 0,
			[]types.Log{modifyLiquidityLog(t, svPoolID, -600, 120)}, chain, svFullScan(len(chain))},
		{"no changed ticks: stored", stale, chain, 100,
			[]types.Log{swap, modifyLiquidityLog(t, "0x01", -600, 60)}, stale, svCalls{}},
	} {
		for reader, stateView := range map[string]string{"stateview": svStateView, "extsload": ""} {
			t.Run(tc.name+"/"+reader, func(t *testing.T) {
				stub := stateViewStub(t, tc.onChn, svTickSpacing)
				defer stub.Close()
				p := svPool(t, tc.stored)
				p.BlockNumber = tc.block
				got, err := svTicks(t, svTracker(stub.URL, stateView), p, tc.logs)
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
				require.Equal(t, tc.calls, svCount(stub))
			})
		}
	}
}

// mergeTicks is stored with the ticks at idx read from chain (a tick cleared on chain drops out).
func mergeTicks(stored, chain map[int64]svTick, idx ...int64) map[int64]svTick {
	out := maps.Clone(stored)
	for _, i := range idx {
		if tk, ok := chain[i]; ok {
			out[i] = tk
		} else {
			delete(out, i)
		}
	}
	return out
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
			stub := stateViewStub(t, chain, tc.spacing)
			defer stub.Close()
			p := svPool(t, nil)
			p.StaticExtra = `{"tS":` + strconv.FormatInt(tc.spacing, 10) + `}`
			got, err := svTicks(t, svTracker(stub.URL, svStateView), p, nil)
			require.NoError(t, err)
			require.Equal(t, chain, got)
			n := svCount(stub)
			require.Equal(t, svCalls{multicalls: tc.multicalls, inner: tc.words + len(chain)}, n)
			t.Logf("tickSpacing %d: %d multicalls, %d calls (%d words, %d ticks)", tc.spacing, n.multicalls, n.inner,
				tc.words, len(chain))
		})
	}
}

// A failed tick read says what failed instead of blaming the changed-tick count.
func TestStateViewTicks_ErrorsNameCause(t *testing.T) {
	t.Run("bad extra", func(t *testing.T) {
		stub := stateViewStub(t, svChainTicks(), svTickSpacing)
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
