package navjit

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const multicallStubABI = `[
 {"type":"function","name":"aggregate","stateMutability":"view","inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}]},
 {"type":"function","name":"tryAggregate","stateMutability":"view","inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}
]`

// nodeGasCap is the node's eth_call gas cap (Robinhood RPCs: 50M).
const nodeGasCap = 50_000_000

// Measured on Robinhood (TestLive_QuoteGas): gas of a filling quote on the thin LOT-oldy venue, and of
// the NAV and float reads.
const (
	fillQuoteGas = 2_913_996
	navGas       = 269_661
	floatGas     = 3_265
)

// oogStub serves the multicalls the way a node does for a thin-constituent NavJit venue: a quote
// above the fill limit fails and burns ALL the gas its sub-call is given. With a per-sub-call gas
// limit that is bounded; the stub also fails a sub-call whose limit is below what it needs, and
// errors "out of gas" for the whole eth_call once the sub-calls exceed nodeGasCap.
type oogStub struct {
	poolID     common.Hash
	usdg, lot  common.Address
	buyLimit   *big.Int // largest USDG-in quote that fills
	sellLimit  *big.Int // largest LOT-in quote that fills
	calls      atomic.Int32
	maxSubGas  atomic.Uint64 // largest gas limit of any sub-call
	badOverlay atomic.Bool   // a multicall request lacked the injected capped-multicall code
}

// inner runs one sub-call; need is the gas it uses when it succeeds.
func (s *oogStub) inner(t *testing.T, data []byte) (ret []byte, need uint64, ok bool) {
	for _, a := range []abi.ABI{hookABI, navGuardABI, erc20ABI, quoterABI} {
		m, err := a.MethodById(data[:4])
		if err != nil {
			continue
		}
		var out []any
		var need uint64
		switch m.Name {
		case "usdg":
			out = []any{s.usdg}
		case "navGuard", "poolManager":
			out = []any{common.HexToAddress("0x01")}
		case "getParams":
			out = []any{hookParams{BandBps: 200, WidthTicks: big.NewInt(1)}}
		case "venueOf":
			var tok common.Address
			args, err := m.Inputs.Unpack(data[4:])
			require.NoError(t, err)
			tok = args[0].(common.Address)
			out = []any{common.Hash{}}
			if tok == s.lot {
				out = []any{s.poolID}
			}
		case "checkedNavPerUnit18":
			out, need = []any{big.NewInt(1e18)}, navGas
		case "balanceOf":
			out, need = []any{new(big.Int).Mul(big.NewInt(1e6), big.NewInt(1e18))}, floatGas
		case "quoteExactInputSingle":
			// ABI head: offset word, then PoolKey (5 words), zeroForOne, exactAmount.
			body := data[4+32:]
			zeroForOne := new(big.Int).SetBytes(body[5*32:6*32]).Sign() != 0
			amt := new(big.Int).SetBytes(body[6*32 : 7*32])
			limit := s.sellLimit
			if zeroForOne { // USDG is currency0: zeroForOne is a buy
				limit = s.buyLimit
			}
			if amt.Cmp(limit) > 0 {
				return nil, 0, false
			}
			out, need = []any{new(big.Int).Set(amt), big.NewInt(3_000_000)}, fillQuoteGas
		}
		ret, err = m.Outputs.Pack(out...)
		require.NoError(t, err, m.Name)
		return ret, need, true
	}
	t.Fatalf("unexpected selector %x", data[:4])
	return nil, 0, false
}

func (s *oogStub) client(t *testing.T) *ethrpc.Client {
	mc, err := abi.JSON(strings.NewReader(multicallStubABI))
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     any               `json:"id"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.Unmarshal(body, &req))
		var msg struct{ Data, Input hexutil.Bytes }
		require.NoError(t, json.Unmarshal(req.Params[0], &msg))
		data := msg.Input
		if len(data) == 0 {
			data = msg.Data
		}
		s.calls.Add(1)

		w.Header().Set("Content-Type", "application/json")
		if m, err := cappedMulticallABI.MethodById(data[:4]); err == nil {
			s.serveCapped(t, w, req.ID, req.Params, m, data)
			return
		}
		m, err := mc.MethodById(data[:4])
		require.NoError(t, err)
		args, err := m.Inputs.Unpack(data[4:])
		require.NoError(t, err)
		calls := args[len(args)-1].([]struct {
			Target   common.Address `json:"target"`
			CallData []byte         `json:"callData"`
		})
		type result struct {
			Success    bool   `json:"success"`
			ReturnData []byte `json:"returnData"`
		}
		var results []result
		var rets [][]byte
		for _, c := range calls {
			ret, _, ok := s.inner(t, c.CallData)
			results = append(results, result{ok, ret})
			rets = append(rets, ret)
		}
		var packed []byte
		if m.Name == "aggregate" {
			packed, err = m.Outputs.Pack(big.NewInt(86_000_000), rets)
		} else {
			packed, err = m.Outputs.Pack(results)
		}
		require.NoError(t, err)
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": hexutil.Encode(packed)})
	}))
	t.Cleanup(srv.Close)
	return ethrpc.New(srv.URL)
}

// serveCapped models UniswapInterfaceMulticall.multicall: each sub-call gets min(gasLimit, what is
// left) and a failing one burns all of it. The whole eth_call errors once nodeGasCap is exhausted.
func (s *oogStub) serveCapped(t *testing.T, w http.ResponseWriter, id any, params []json.RawMessage,
	m *abi.Method, data []byte) {
	var overrides map[common.Address]struct{ Code hexutil.Bytes }
	require.GreaterOrEqual(t, len(params), 3, "the multicall code must be injected by state override")
	require.NoError(t, json.Unmarshal(params[2], &overrides))
	if !bytes.Equal(overrides[cappedMulticallAddr].Code, cappedMulticallCode) {
		s.badOverlay.Store(true)
	}
	args, err := m.Inputs.Unpack(data[4:])
	require.NoError(t, err)
	calls := args[0].([]struct {
		Target   common.Address `json:"target"`
		GasLimit *big.Int       `json:"gasLimit"`
		CallData []byte         `json:"callData"`
	})
	type result struct {
		Success    bool     `json:"success"`
		GasUsed    *big.Int `json:"gasUsed"`
		ReturnData []byte   `json:"returnData"`
	}
	results := make([]result, 0, len(calls))
	left := uint64(nodeGasCap)
	for _, c := range calls {
		limit := min(c.GasLimit.Uint64(), left)
		if limit > s.maxSubGas.Load() {
			s.maxSubGas.Store(limit)
		}
		ret, need, ok := s.inner(t, c.CallData)
		used := limit // a failing sub-call burns everything it was given
		if ok && need <= limit {
			used = need
		} else {
			ret, ok = nil, false
		}
		results = append(results, result{ok, new(big.Int).SetUint64(used), ret})
		if left -= used; left == 0 {
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id,
				"error": map[string]any{"code": -32000, "message": "out of gas"}})
			return
		}
	}
	packed, err := m.Outputs.Pack(big.NewInt(86_000_000), results)
	require.NoError(t, err)
	_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": hexutil.Encode(packed)})
}

// The extra is pure chain state: two Tracks of the same chain state at different times give
// byte-identical output, so a quiet pool's Extra does not change and pool-service skips the save.
func TestTrack_ExtraHasNoClock(t *testing.T) {
	newStub := func() *oogStub {
		return &oogStub{buyLimit: big.NewInt(100e6), sellLimit: new(big.Int).Mul(big.NewInt(100), big.NewInt(1e18))}
	}
	first, err := json.Marshal(trackWithStub(t, newStub()))
	require.NoError(t, err)
	time.Sleep(1100 * time.Millisecond) // a Unix-second clock in the extra would differ now
	second, err := json.Marshal(trackWithStub(t, newStub()))
	require.NoError(t, err)
	assert.Equal(t, string(first), string(second))
	assert.NotContains(t, string(first), `"ts"`)
	assert.NotContains(t, string(first), `"bn"`)
}

func trackWithStub(t *testing.T, s *oogStub) Extra {
	s.poolID = common.HexToHash("0x85d90dac41ad4205b80b4c4e14f4831805fc7fe3e84068a59c7a2618e4db1817")
	s.usdg = common.HexToAddress(usdgAddr)
	s.lot = common.HexToAddress("0x8bb6f07aba0dae22943adc96338cd297000442c2")
	staticExtra, err := json.Marshal(uniswapv4.StaticExtra{TickSpacing: 1})
	require.NoError(t, err)
	p := &entity.Pool{Address: s.poolID.Hex(), StaticExtra: string(staticExtra),
		Tokens: []*entity.PoolToken{{Address: strings.ToLower(s.usdg.Hex()), Decimals: usdgDecimals},
			{Address: strings.ToLower(s.lot.Hex()), Decimals: 18}}}
	h := &Hook{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4NavJit}}
	raw, err := h.Track(context.Background(), &uniswapv4.HookParam{
		Cfg: &uniswapv4.Config{ChainID: valueobject.ChainIDRobinhood}, RpcClient: s.client(t), Pool: p,
		HookAddress: HookAddresses[0]})
	require.NoError(t, err)
	var got Hook
	require.NoError(t, json.Unmarshal(raw, &got))
	return got.Extra
}

// LOT-oldy (prod pool 0x85d90dac..., 2026-10): rungs from $500 up fail and each burns ~all the gas it
// is given. Without a per-sub-call cap the 5+ failing rungs ran the node's eth_call out of gas and the
// pool was never tracked. With the cap, the rungs that fill are kept.
func TestTrack_ThinConstituentPoolDoesNotRunOutOfGas(t *testing.T) {
	s := &oogStub{buyLimit: big.NewInt(100e6), sellLimit: new(big.Int).Mul(big.NewInt(100), big.NewInt(1e18))}
	got := trackWithStub(t, s)

	require.True(t, got.Tracked)
	assert.Len(t, got.Buy, 3, "buy rungs $1, $10 and $100 fill; $500 and up do not")
	assert.Len(t, got.Sell, 3, "sell rungs 1, 10 and 100 LOT fill")
	assert.Zero(t, got.TrackedAt, "Track writes no clock; the tracker stamps Extra lv")
	assert.False(t, s.badOverlay.Load())
	assert.EqualValues(t, 3, s.calls.Load(), "hook surface, nav+float+buy ladder, sell ladder: no per-rung fallback")
	assert.EqualValues(t, quoteGasCap, s.maxSubGas.Load())
}

// The cap must stay above the gas of a rung that fills: a filling quote costs fillQuoteGas on the
// thinnest live venue, and a cap below it would silently truncate healthy ladders.
func TestQuoteGasCapCoversAFillingRungAndFitsTheLadder(t *testing.T) {
	assert.Greater(t, quoteGasCap, fillQuoteGas)
	longest := len(rungsUsd) + len(floatRungsBps) // the sell round
	assert.Less(t, longest*quoteGasCap, nodeGasCap, "a round of failing rungs must fit the node cap")
	assert.Less(t, len(rungsUsd)*quoteGasCap+navGas+floatGas, nodeGasCap)
}

// A healthy venue keeps one multicall per round: no extra RPCs.
func TestTrack_HealthyVenueStaysOnTheFastPath(t *testing.T) {
	big1 := new(big.Int).Lsh(big.NewInt(1), 100)
	s := &oogStub{buyLimit: big1, sellLimit: big1}
	got := trackWithStub(t, s)

	assert.Len(t, got.Buy, len(rungsUsd))
	assert.EqualValues(t, 3, s.calls.Load(), "hook surface, nav+float+buy ladder, sell ladder")
}
