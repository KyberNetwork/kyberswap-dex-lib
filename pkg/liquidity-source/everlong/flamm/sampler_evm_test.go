package everlongflamm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// FlammSampler's and MaxOfTwoPool's code, pinned over the decoded runtime and over the source: a rebuild that moves a
// byte, or a source edit left unbuilt, fails here. testdata/build_sampler.py builds both; --check verifies them.
const (
	samplerRuntimeKeccak  = "0xdef8a5c6daa5ac664aff33e93625b8ef24d0b23fd041eb57193d3da960212ac5"
	samplerSourceKeccak   = "0xcf941e0e26687c978ca84ff807257393e5008bb1ce0e08bf9f0a5b6543e6e5fd"
	maxOfTwoRuntimeKeccak = "0xe9b47028abbbbde6b37871b0bc7383dcd45447c40f072dbd271019abbb678941"
	maxOfTwoSourceKeccak  = "0xed86138a409c1fa6cd6ddadc3f6e526b3709b563ffb8fdcffcd99c1657175d3c"
)

// readStarved and gasShort are the reverts the sampler reads as a starved probe.
var (
	readStarved = [4]byte(crypto.Keccak256([]byte("ReadStarved()"))[:4])
	gasShort    = [4]byte(crypto.Keccak256([]byte("GasShort()"))[:4])
)

// reasonOf is a revert reason as MaxOfTwoPool's m2Reason slot holds it.
func reasonOf(reason [4]byte) *big.Int { return new(big.Int).SetBytes(reason[:]) }

// testProbeGas is the per-probe cap a pool's sampler call starts at.
const testProbeGas = defaultProbeGas

// MaxOfTwoPool (testdata/MaxOfTwoPool.sol) is a test pool that is its own pair hook: two concave routes, the curve and
// the leverage route, each labelled by its fee, the better one filling, and switches that break the premises a
// ladder rests on. Its storage, as the source lays it out:
const (
	m2Faults   = 2 // the switch bits below
	m2Taker    = 3 // the caller the caller check wants
	m2BurnLeft = 4 // the gas a burner leaves
	m2BombSize = 5 // a bomb's size in bytes
	m2Reason   = 6 // a revert reason, in the low 4 bytes
	m2Band     = 7 // the live band, request word 10
	m2Usd0     = 8 // token 0's and token 1's USD prices, WAD
	m2Usd1     = 9
	m2P36      = 10 // token 1 units per token 0 unit, scaled by 1e36
	m2CurveFee = 11 // quotePair word 3
	m2Bid      = 12 // levState word 6
	m2CV       = 13 // levState word 0
	m2Legs0    = 16 // token 0's legs, held, supplied, posted and owed; token 1's at m2Legs1
	m2Legs1    = 20
	m2Sale     = 0x40 // a sale's routes, at m2Sale plus an offset below; a buy's at m2Buy
	m2Buy      = 0x50
	m2Params   = 0x100 // params() word k at m2Params + k
)

// A route's slots, past m2Sale or m2Buy.
const (
	rtCurveR = iota
	rtCurveX
	rtLevR
	rtLevX
	rtLevEdge // leverage refuses above this size
	rtHoleMod // leverage refuses where size % rtHoleMod == rtHoleRem
	rtHoleRem
	rtLottery   // leverage refuses at or under the band's far line, and over it by less than a size's hash mod this
	rtFlickerAt // leverage refuses on [rtFlickerAt, rtFlickerAt + rtFlickerLen)
	rtFlickerLen
	rtFallAt // leverage pays rtFallBy less above rtFallAt
	rtFallBy
	rtTokenCap  // leverage refuses a net over this
	rtCurveClip // the curve uses at most this input; the preview re-asks at the input used
)

// The row's params() words.
const (
	pmSpread   = 27
	pmSpreadDn = 28
	pmUpCap    = 29
	pmDnCap    = 30
	pmCeiling  = 31
	pmBand     = 32
	pmFeedGate = 33
	pmUpFloor  = 34
)

// The switch bits of m2Faults.
const (
	swCaller        = 1 << iota // previewSwap and pairRequest revert NotTaker() for any caller but m2Taker's
	swBurn                      // previewSwap burns its gas down to m2BurnLeft, then answers; zero: it runs out
	swReturnBomb                // previewSwap answers m2BombSize bytes
	swRevertBomb                // previewSwap reverts with m2BombSize bytes
	swShort                     // previewSwap answers 64 bytes
	swReason                    // previewSwap reverts with m2Reason's four bytes
	swBare                      // previewSwap reverts with no data
	swEqualFee                  // the leverage route carries the curve's fee
	swCoreBand                  // a fill under the live band's far line reverts: the dust windows
	swUsed1bp                   // used is the ask less ask / 10,000
	swRequestRevert             // pairRequest reverts
	swLevRevert                 // levState reverts
	swLevShort                  // levState answers 7 words
)

var (
	m2Addr   = common.HexToAddress("0x00000000000000000000000000000000000f0010")
	m2Token0 = common.HexToAddress("0x00000000000000000000000000000000000000c0")
	m2Token1 = common.HexToAddress("0x00000000000000000000000000000000000000c1")
	m2Other  = common.HexToAddress("0x000000000000000000000000000000000000dead") // not the taker
	m2Block  = big.NewInt(100)
)

const m2Time = 1_800_000_000

var (
	previewSelector     = []byte{0xbb, 0xba, 0xff, 0xab} // previewSwap(address,address,uint256)
	pairRequestSelector = []byte{0x6a, 0x19, 0x82, 0x09} // pairRequest(address,address,uint256)
	maxOfTwoRuntime     = sync.OnceValues(func() ([]byte, error) {
		hexCode, err := os.ReadFile("testdata/MaxOfTwoPool.bin")
		return common.FromHex(strings.TrimSpace(string(hexCode))), err
	})
)

// m2Slots is MaxOfTwoPool's storage by slot; a slot left out is zero, which turns its feature off.
type m2Slots map[uint64]*big.Int

// with is a copy of s with each slot, value pair set; a value is an int or a *big.Int.
func (s m2Slots) with(kv ...any) m2Slots {
	c := maps.Clone(s)
	if c == nil {
		c = m2Slots{}
	}
	for k := 0; k+1 < len(kv); k += 2 {
		v, ok := kv[k+1].(*big.Int)
		if !ok {
			v = big.NewInt(int64(kv[k+1].(int)))
		}
		c[uint64(kv[k].(int))] = v
	}
	return c
}

func (s m2Slots) at(slot uint64) *big.Int {
	if v, ok := s[slot]; ok {
		return v
	}
	return new(big.Int)
}

func ten(m int64, e int) *big.Int { return new(big.Int).Mul(big.NewInt(m), pow10(e)) }

// m2Base is a pool of a d0- and a d1-decimal token, token 0 at `price` dollars, and its hook's leverage row: spread
// 1.75 %, spreadDn 1.3 %, caps of 0.5 % up and 2 % down inside a 5 % band, at a live band of 8 %. The leverage route
// pays the small fills of both directions and the curve the large ones.
func m2Base(d0, d1 int, price int64) m2Slots {
	addr := func(a common.Address) *big.Int { return new(big.Int).SetBytes(a[:]) }
	p36 := new(big.Int).Mul(big.NewInt(price), pow10(d1+36))
	return m2Slots{}.with(0, addr(m2Token0), 1, addr(m2Token1), m2Taker, addr(defaultTaker), m2Band, ten(8, 16),
		m2Usd0, ten(price, 18), m2Usd1, ten(1, 18), m2P36, p36.Quo(p36, pow10(d0)), m2CurveFee, 12644526805792593,
		m2Bid, ten(736, 15), m2CV, ten(23, 22), m2Legs0, 117098252, m2Legs0+2, 192927801, m2Legs1+3, 80231847915,
		m2Sale+rtCurveR, ten(745, 10), m2Sale+rtCurveX, ten(1, 10), m2Sale+rtLevR, ten(750*229, 7),
		m2Sale+rtLevX, ten(229, 7),
		m2Buy+rtCurveR, ten(13156, 6), m2Buy+rtCurveX, ten(1, 13), m2Buy+rtLevR, ten(67, 8), m2Buy+rtLevX, ten(5, 12),
		m2Params, ten(2, 18), m2Params+1, ten(102, 16), m2Params+2, ten(6, 18), m2Params+pmSpread, ten(175, 14),
		m2Params+pmSpreadDn, ten(13, 15), m2Params+pmUpCap, ten(5, 15), m2Params+pmDnCap, ten(2, 16),
		m2Params+pmCeiling, ten(22, 17), m2Params+pmBand, ten(5, 16), m2Params+pmUpFloor, ten(17, 17))
}

// m2Row is the hook's leverage row, params() words 27-34.
func m2Row(s m2Slots) hookRow {
	w := func(k uint64) *big.Int { return s.at(m2Params + k) }
	return hookRow{spread: w(pmSpread), spreadDn: w(pmSpreadDn), upCap: w(pmUpCap), dnCap: w(pmDnCap),
		ceiling: w(pmCeiling), band: w(pmBand), feedGate: w(pmFeedGate), upFloor: w(pmUpFloor)}
}

// m2Pair is a direction's tokens: a sale sells token 0, a buy buys it.
func m2Pair(sale bool) (in, out common.Address) {
	if sale {
		return m2Token0, m2Token1
	}
	return m2Token1, m2Token0
}

// evmNode answers eth_call as a node does, on an in-process EVM (core/vm/runtime) over MaxOfTwoPool's state: the
// call's code overrides set first, its gas less the intrinsic cost of its call data, from its `from`, zero when
// unset. It keeps the last call's frames one level under the top call, and its gas.
type evmNode struct {
	client *ethrpc.Client

	mu       sync.Mutex
	state    *state.StateDB
	frames   []evmFrame
	gasUsed  uint64                    // the last eth_call's, intrinsic cost included
	answer   int                       // the last eth_call's answer, in bytes
	previews map[previewKey][]*big.Int // the pool's own previewSwap at this state: used, out, fee; nil: a revert
}

type evmFrame struct {
	op            vm.OpCode
	from, to      common.Address
	input, output []byte
	reverted      bool
}

type previewKey struct {
	sale bool
	a    string
}

func newEVMNode(t *testing.T) *evmNode {
	n := &evmNode{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		resp := map[string]any{"jsonrpc": "2.0"}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			resp["error"] = map[string]any{"code": -32700, "message": err.Error()}
		} else {
			resp["id"] = req.ID
			if result, fail := n.serve(req.Method, req.Params); fail != nil {
				resp["error"] = fail
			} else {
				resp["result"] = result
			}
		}
		w.Header().Set("content-type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	n.client = ethrpc.New(srv.URL)
	return n
}

func (n *evmNode) serve(method string, raw []json.RawMessage) (any, map[string]any) {
	fail := func(code int, msg string) map[string]any { return map[string]any{"code": code, "message": msg} }
	if method != "eth_call" {
		return nil, fail(-32601, "the method does not exist")
	}
	var call struct {
		From        *common.Address
		To          *common.Address
		Gas         *hexutil.Uint64
		Data, Input *hexutil.Bytes
	}
	var block string
	var overrides map[common.Address]struct{ Code *hexutil.Bytes }
	if len(raw) < 2 || json.Unmarshal(raw[0], &call) != nil || json.Unmarshal(raw[1], &block) != nil ||
		len(raw) > 2 && json.Unmarshal(raw[2], &overrides) != nil || call.To == nil || call.Gas == nil {
		return nil, fail(-32602, "invalid params")
	}
	if block != hexutil.EncodeBig(m2Block) {
		return nil, fail(-32000, "header not found")
	}
	var data []byte
	if call.Input != nil {
		data = *call.Input
	} else if call.Data != nil {
		data = *call.Data
	}
	var from common.Address
	if call.From != nil {
		from = *call.From
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	st := n.state.Copy()
	for a, o := range overrides {
		if o.Code != nil {
			st.SetCode(a, *o.Code, tracing.CodeChangeUnspecified)
		}
	}
	ret, used, err := n.run(st, from, *call.To, data, uint64(*call.Gas), true)
	n.gasUsed, n.answer = used, len(ret)
	switch {
	case errors.Is(err, vm.ErrExecutionReverted):
		return nil, map[string]any{"code": 3, "message": "execution reverted", "data": hexutil.Encode(ret)}
	case err != nil:
		return nil, fail(-32000, err.Error())
	}
	return hexutil.Encode(ret), nil
}

// run executes one call as a transaction's top call, with its gas less the intrinsic cost of its call data. traced:
// the node keeps the frames one level under it.
func (n *evmNode) run(st *state.StateDB, from, to common.Address, data []byte, gas uint64,
	traced bool) ([]byte, uint64, error) {
	intrinsic, err := core.IntrinsicGas(data, nil, nil, false, params.Rules{IsHomestead: true, IsIstanbul: true}, 0)
	if err != nil {
		return nil, 0, err
	}
	if gas < intrinsic.RegularGas {
		return nil, 0, core.ErrIntrinsicGas
	}
	cfg := runtime.Config{State: st, Origin: from, BlockNumber: m2Block, Time: m2Time,
		GasLimit: gas - intrinsic.RegularGas}
	if traced {
		n.frames = nil
		cfg.EVMConfig.Tracer = &tracing.Hooks{
			OnEnter: func(depth int, op byte, from, to common.Address, input []byte, _ uint64, _ *big.Int) {
				if depth == 1 {
					n.frames = append(n.frames,
						evmFrame{op: vm.OpCode(op), from: from, to: to, input: slices.Clone(input)})
				}
			},
			OnExit: func(depth int, output []byte, _ uint64, _ error, reverted bool) {
				if depth == 1 {
					f := &n.frames[len(n.frames)-1]
					f.output, f.reverted = slices.Clone(output), reverted
				}
			},
		}
	}
	ret, left, err := runtime.Call(to, data, &cfg)
	return ret, gas - left, err
}

// load puts MaxOfTwoPool's code at m2Addr on an empty state, its storage s.
func (n *evmNode) load(t *testing.T, s m2Slots) {
	t.Helper()
	code, err := maxOfTwoRuntime()
	require.NoError(t, err)
	st, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	require.NoError(t, err)
	st.SetCode(m2Addr, code, tracing.CodeChangeUnspecified)
	for k, v := range s {
		st.SetState(m2Addr, common.BigToHash(new(big.Int).SetUint64(k)), common.BigToHash(v))
	}
	n.mu.Lock()
	n.state, n.previews = st, map[previewKey][]*big.Int{}
	n.mu.Unlock()
}

// last is the last eth_call's frames, its gas and its answer's size.
func (n *evmNode) last() ([]evmFrame, uint64, int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	return slices.Clone(n.frames), n.gasUsed, n.answer
}

// view runs a call to the pool from the taker on the node's state.
func (n *evmNode) view(t *testing.T, data []byte) ([]byte, error) {
	t.Helper()
	n.mu.Lock()
	defer n.mu.Unlock()
	ret, _, err := n.run(n.state.Copy(), defaultTaker, m2Addr, data, 30_000_000, false)
	return ret, err
}

// preview is the pool's own previewSwap of `a` from the taker: used, out and fee; nil where it reverts.
func (n *evmNode) preview(t *testing.T, sale bool, a *big.Int) []*big.Int {
	t.Helper()
	k := previewKey{sale, a.String()}
	n.mu.Lock()
	p, seen := n.previews[k]
	n.mu.Unlock()
	if seen {
		return p
	}
	in, out := m2Pair(sale)
	data, err := poolABI.Pack("previewSwap", in, out, a)
	require.NoError(t, err)
	if ret, err := n.view(t, data); err == nil {
		require.Len(t, ret, 96)
		p = []*big.Int{word(ret, 0), word(ret, 1), word(ret, 2)}
	}
	n.mu.Lock()
	n.previews[k] = p
	n.mu.Unlock()
	return p
}

// m2Ask asks a direction of MaxOfTwoPool at `seeds`, with the pool as its hook and the row's spreads as its tiers.
func m2Ask(s m2Slots, sale bool, seeds []*big.Int) sampleAsk {
	in, out := m2Pair(sale)
	return sampleAsk{TokenIn: in, TokenOut: out, Hook: m2Addr, TierA: s.at(m2Params + pmSpread),
		TierB: s.at(m2Params + pmSpreadDn), MaxProbes: big.NewInt(maxProbesPerDirection), Seeds: seeds}
}

// firstSeeds are the tracker's sizes at a pool's first refresh for an input of d decimals and no notional cap: 32,
// geometric, from 10^(d-3) to 10^(d+9).
func firstSeeds(d int) []*big.Int {
	return seedsOf(d, nil, nil, firstRefreshSeeds, defaultRungs)
}

// samplerCall sends one eth_call as a node gets it: to `at`, with the sampler's runtime as a code override there, an
// explicit gas, `from` unset, at the pinned block.
func (n *evmNode) samplerCall(at common.Address, data []byte, gas uint64) ([]byte, error) {
	var ret hexutil.Bytes
	arg := map[string]any{"to": at, "data": hexutil.Bytes(data), "gas": hexutil.Uint64(gas)}
	overrides := map[common.Address]gethclient.OverrideAccount{at: {Code: samplerRuntime}}
	err := n.client.GetETHClient().Client().CallContext(context.Background(), &ret, "eth_call", arg,
		hexutil.EncodeBig(m2Block), overrides)
	return ret, err
}

// sample runs the sampler at `at` over MaxOfTwoPool and decodes its answer, which must never be a revert: the tag
// first, the pinned block, a walk per ask, each ascending by ask and holding at most its budget and one probe more.
func (n *evmNode) sample(t *testing.T, at common.Address, gas uint64, lim sampleLimits,
	asks ...sampleAsk) sampleAnswer {
	t.Helper()
	data, err := samplerABI.Pack("sample", m2Addr, asks, lim)
	require.NoError(t, err)
	ret, err := n.samplerCall(at, data, gas)
	require.NoError(t, err, "the sampler never reverts")
	var a sampleAnswer
	require.NoError(t, samplerABI.UnpackIntoInterface(&a, "sample", ret))
	require.Equal(t, samplerTag, a.Tag)
	require.Zero(t, a.BlockNumber.Cmp(m2Block))
	require.Len(t, a.Walks, len(asks))
	for k, w := range a.Walks {
		require.LessOrEqual(t, len(w.Probes), int(asks[k].MaxProbes.Int64())+1)
		for j := 1; j < len(w.Probes); j++ {
			require.Positive(t, w.Probes[j].Ask.Cmp(w.Probes[j-1].Ask), "a walk ascends by ask")
		}
	}
	return a
}

// callsOf are the frames calling sel on a direction: the input token is word 0 of a pool call, and word 3 of the
// request's words, which a hook call carries, is 1 on a buy.
func callsOf(frames []evmFrame, sel []byte, sale bool) []evmFrame {
	hook := bytes.Equal(sel, quotePairSelector) || bytes.Equal(sel, levStateSelector)
	var out []evmFrame
	for _, f := range frames {
		if !bytes.HasPrefix(f.input, sel) || len(f.input) < 4+32*3 {
			continue
		}
		isSale := common.BytesToAddress(f.input[4:36]) == m2Token0
		if hook {
			isSale = len(f.input) >= 4+32*requestWords && word(f.input[4:], 3).Sign() == 0
		}
		if isSale == sale {
			out = append(out, f)
		}
	}
	return out
}

func statusesOf(w sampleWalk) []probeStatus {
	var s []probeStatus
	for _, p := range w.Probes {
		s = append(s, probeStatus(p.Status.Uint64()))
	}
	return s
}

// topOf is a walk's top priced probe paying at least dustOut, the probe asking `skip` left out.
func topOf(t *testing.T, w sampleWalk, skip *big.Int) sampleProbe {
	t.Helper()
	var top sampleProbe
	for _, p := range w.Probes {
		if probeStatus(p.Status.Uint64()) == statusPriced && p.Out.Cmp(dust) >= 0 &&
			(skip == nil || p.Ask.Cmp(skip) != 0) {
			top = p
		}
	}
	require.NotNil(t, top.Ask, "the walk has a price over dust")
	return top
}

// aboveOf is where the probe above a top at p asks: used + (used >> aboveShift) + 1.
func aboveOf(p sampleProbe) *big.Int {
	v := new(big.Int).Rsh(p.Used, aboveShift)
	return v.Add(v, p.Used).Add(v, one)
}

func isTierOf(tiers []*big.Int, f *big.Int) bool {
	return slices.ContainsFunc(tiers, func(t *big.Int) bool { return t.Cmp(f) == 0 })
}

// settleOf is what the tracker gives settle for one walk over MaxOfTwoPool: the probes, the request's words where the
// walk read them, the row and its gate at the live band, and the curve's fee.
func settleOf(s m2Slots, sale bool, decIn, decOut int, w sampleWalk) settleIn {
	row := m2Row(s)
	in := settleIn{rule: ruleEverlong, tiers: row.tiers(), rowBand: row.band, sale: sale, decIn: uint8(decIn),
		decOut: uint8(decOut), curveFee: w.CurveFeeWad}
	for _, p := range w.Probes {
		in.probes = append(in.probes, sampled{ask: p.Ask, used: p.Used, out: p.Out, fee: p.FeeWad,
			status: probeStatus(p.Status.Uint64())})
	}
	if w.BandWad != 0 {
		in.words = pairWords{band: new(big.Int).SetUint64(w.BandWad), usdIn: w.UsdIn, usdOut: w.UsdOut, legs: w.Legs,
			bid: w.BidWad}
	}
	in.trusted = rowOk(row, in.words.band)
	return in
}

// withoutAbove is the walk less its probe above: its last preview, where that asked used + (used >> aboveShift) + 1
// of the walk's top priced probe paying at least dustOut and that probe is tier-labelled. above: the probe above's
// ask; nil: the walk made none.
func withoutAbove(in settleIn, frames []evmFrame, sale bool) (rest []sampled, above *big.Int) {
	previews := callsOf(frames, previewSelector, sale)
	if len(previews) == 0 {
		return in.probes, nil
	}
	last := word(previews[len(previews)-1].input[4:], 2)
	rest = slices.DeleteFunc(slices.Clone(in.probes), func(p sampled) bool { return p.ask.Cmp(last) == 0 })
	var top *sampled
	for k := range rest {
		if rest[k].status == statusPriced && rest[k].out.Cmp(dust) >= 0 {
			top = &rest[k]
		}
	}
	if top == nil || !isTierOf(in.tiers, top.fee) {
		return in.probes, nil
	}
	v := new(big.Int).Rsh(top.used, aboveShift)
	if v.Add(v, top.used).Add(v, one).Cmp(last) != 0 {
		return in.probes, nil
	}
	return rest, last
}

// ladderOf is a settled direction as the tracker publishes it.
func ladderOf(t *testing.T, s settled) Ladder {
	require.LessOrEqual(t, len(s.x), maxLadderPoints)
	l := Ladder{Capped: s.capped, Cells: quoteCells, Second: quoteSecond}
	for k := range s.x {
		l.Points = append(l.Points, Point{uint256.MustFromBig(s.x[k]), uint256.MustFromBig(s.y[k])})
		if s.flat[k] {
			l.Flat |= 1 << k
		}
	}
	return l
}

// audited is a ladder's quotes against the pool's own previewSwap at the same sizes: the sizes it quoted, those where
// the pool reverts (a failed route, never an over-quote), those where it quotes more than the pool pays, and the most
// it does, in units and in bp of the pool's output, and where.
type audited struct {
	points, sizes, reverted, over int
	worst, at                     *big.Int
	worstBp                       float64
}

// audit quotes l, with no haircut, on the grid of the tracker's over-quote test, inside l's range: 2,000 sizes
// log-spaced, each point and two units either side of it, 300 sizes across each flat span and 300 around each route
// switch, and `extra`. It compares each quote with the pool's own previewSwap at the input the quote uses.
func (n *evmNode) audit(t *testing.T, sale bool, l Ladder, switches, extra []*big.Int) audited {
	t.Helper()
	a := audited{points: len(l.Points), worst: new(big.Int)}
	if len(l.Points) == 0 {
		return a
	}
	c, err := newCurve(l)
	require.NoError(t, err)
	first, top := l.Points[0][0].ToBig(), l.Points[len(l.Points)-1][0].ToBig()
	sizes := map[string]*big.Int{}
	add := func(v *big.Int) {
		if v.Cmp(first) >= 0 && v.Cmp(top) <= 0 {
			sizes[v.String()] = v
		}
	}
	lf, _ := new(big.Float).SetInt(first).Float64()
	hf, _ := new(big.Float).SetInt(top).Float64()
	for k := range 2_000 {
		v, _ := big.NewFloat(lf * math.Pow(hf/lf, float64(k)/1_999)).Int(nil)
		add(v)
	}
	for k, p := range l.Points {
		x := p[0].ToBig()
		for d := int64(-2); d <= 2; d++ {
			add(new(big.Int).Add(x, big.NewInt(d)))
		}
		if k > 0 && l.Flat>>k&1 == 1 {
			x0 := l.Points[k-1][0].ToBig()
			for i := range int64(300) {
				v := new(big.Int).Sub(x, x0)
				add(v.Mul(v, big.NewInt(i)).Quo(v, big.NewInt(299)).Add(v, x0))
			}
		}
	}
	for _, s := range switches {
		for d := int64(-150); d < 150; d++ {
			add(new(big.Int).Add(s, big.NewInt(d)))
		}
	}
	for _, v := range extra {
		add(v)
	}
	grid := slices.SortedFunc(maps.Values(sizes), func(a, b *big.Int) int { return a.Cmp(b) })
	for _, v := range grid {
		var used, out uint256.Int
		_, err := c.quote(uint256.MustFromBig(v), 0, &used, &out)
		require.NoError(t, err, "a size inside the ladder quotes: %s", v)
		a.sizes++
		fill := n.preview(t, sale, used.ToBig())
		if fill == nil {
			a.reverted++
			continue
		}
		if d := new(big.Int).Sub(out.ToBig(), fill[1]); d.Sign() > 0 {
			a.over++
			if d.Cmp(a.worst) > 0 {
				bp, _ := new(big.Float).Quo(new(big.Float).SetInt(d), new(big.Float).SetInt(fill[1])).Float64()
				a.worst, a.at, a.worstBp = d, v, bp*10_000
			}
		}
	}
	return a
}

// switchesOf are the sizes where a walk's route changes: between two adjacent priced probes on different labels, the
// least size the pool fills on the upper one's label, bisected on its own previewSwap.
func (n *evmNode) switchesOf(t *testing.T, sale bool, w sampleWalk) []*big.Int {
	t.Helper()
	var at []*big.Int
	for k := 1; k < len(w.Probes); k++ {
		p, q := w.Probes[k-1], w.Probes[k]
		if probeStatus(p.Status.Uint64()) != statusPriced || probeStatus(q.Status.Uint64()) != statusPriced ||
			p.FeeWad.Cmp(q.FeeWad) == 0 {
			continue
		}
		lo, hi := p.Ask, q.Ask
		for new(big.Int).Sub(hi, lo).Cmp(one) > 0 {
			mid := new(big.Int).Rsh(new(big.Int).Add(lo, hi), 1)
			if f := n.preview(t, sale, mid); f != nil && f[2].Cmp(q.FeeWad) == 0 {
				hi = mid
			} else {
				lo = mid
			}
		}
		at = append(at, hi)
	}
	return at
}

func (a audited) String() string {
	s := fmt.Sprintf("%d points, %d sizes, %d where the pool reverts, %d over", a.points, a.sizes, a.reverted, a.over)
	if a.over > 0 {
		s += fmt.Sprintf(", worst %s units at %s, %.2f bp", a.worst, a.at, a.worstBp)
	}
	return s
}

// answerSize is the ABI size of an answer of two walks holding m probes in all: it follows the probes alone.
func answerSize(m int) int { return 96 + 32 + 2*32 + 2*(32*15+32) + 32*7*m }

// opcodeIn reports whether code, read as instructions, uses op.
func opcodeIn(code []byte, op vm.OpCode) bool {
	for pc := 0; pc < len(code); pc++ {
		c := vm.OpCode(code[pc])
		if c == op {
			return true
		}
		if c >= vm.PUSH1 && c <= vm.PUSH32 {
			pc += int(c-vm.PUSH1) + 1
		}
	}
	return false
}

func TestSamplerBytesArePinned(t *testing.T) {
	require.Equal(t, samplerRuntimeKeccak, crypto.Keccak256Hash(samplerRuntime).Hex(),
		"rebuild: testdata/build_sampler.py")
	source, err := os.ReadFile("abi/FlammSampler.sol")
	require.NoError(t, err)
	require.Equal(t, samplerSourceKeccak, crypto.Keccak256Hash(source).Hex())
	pool, err := maxOfTwoRuntime()
	require.NoError(t, err)
	require.Equal(t, maxOfTwoRuntimeKeccak, crypto.Keccak256Hash(pool).Hex())
	source, err = os.ReadFile("testdata/MaxOfTwoPool.sol")
	require.NoError(t, err)
	require.Equal(t, maxOfTwoSourceKeccak, crypto.Keccak256Hash(source).Hex())
	t.Logf("runtime: the sampler %d bytes, the test pool %d", len(samplerRuntime), len(pool))
	for _, code := range [][]byte{samplerRuntime, pool} {
		require.False(t, opcodeIn(code, vm.PUSH0), "built for paris: no PUSH0")
		require.False(t, opcodeIn(code, vm.MCOPY), "built for paris: no MCOPY")
	}
	require.Equal(t, common.FromHex("0x611b77fc"), samplerABI.Methods["sample"].ID)
	require.Equal(t, common.FromHex("0x47221ac7"), samplerABI.Methods["TAG"].ID)
	require.Equal(t, previewSelector, poolABI.Methods["previewSwap"].ID)
	require.Equal(t, pairRequestSelector, poolABI.Methods["pairRequest"].ID)

	n := newEVMNode(t)
	n.load(t, m2Base(8, 6, 75_600))
	ret, err := n.samplerCall(defaultTaker, samplerABI.Methods["TAG"].ID, 100_000)
	require.NoError(t, err)
	require.Equal(t, common.RightPadBytes(samplerTag[:], 32), ret)
}

func TestSamplerCallsThePoolFromTheTaker(t *testing.T) {
	n := newEVMNode(t)
	s := m2Base(8, 6, 75_600).with(m2Faults, swCaller)
	n.load(t, s)
	asks := []sampleAsk{m2Ask(s, true, firstSeeds(8)), m2Ask(s, false, firstSeeds(6))}
	notTaker := [4]byte(crypto.Keccak256([]byte("NotTaker()"))[:4])
	for _, at := range []common.Address{defaultTaker, m2Other} {
		a := n.sample(t, at, defaultSampleGas, samplerLimits(testProbeGas), asks...)
		frames, _, _ := n.last()
		require.NotEmpty(t, frames)
		for _, f := range frames {
			require.Equal(t, vm.STATICCALL, f.op, "the sampler only reads")
			require.Equal(t, at, f.from, "the pool sees the sampler's own address")
			require.Equal(t, m2Addr, f.to)
		}
		for k, w := range a.Walks {
			if at == defaultTaker {
				require.NotZero(t, w.BandWad, "at the taker the walk prices and reads")
				for _, st := range statusesOf(w) {
					require.Equal(t, statusPriced, st)
				}
				request := callsOf(frames, pairRequestSelector, k == 0)
				require.Len(t, request, 1)
				require.Equal(t, common.LeftPadBytes(defaultTaker[:], 32), request[0].output[:32],
					"the request's word 0")
				continue
			}
			require.Zero(t, w.BandWad, "elsewhere nothing is read")
			require.Len(t, w.Probes, maxProbesPerDirection)
			for _, p := range w.Probes {
				require.Equal(t, statusRefused, probeStatus(p.Status.Uint64()))
				require.Equal(t, notTaker, p.Reason)
			}
		}
	}
}

func TestSamplerNeverRevertsOnAHostilePool(t *testing.T) {
	n := newEVMNode(t)
	base := m2Base(8, 6, 75_600)
	asksOf := func(s m2Slots) []sampleAsk {
		return []sampleAsk{m2Ask(s, true, firstSeeds(8)), m2Ask(s, false, firstSeeds(6))}
	}
	for _, tc := range []struct {
		name      string
		slots     m2Slots
		status    probeStatus
		reason    [4]byte
		probes    int // per walk
		truncated bool
	}{
		{"a gas burner", base.with(m2Faults, swBurn), statusStarved, [4]byte{}, 1, false},
		{"a burner leaving 680k", base.with(m2Faults, swBurn, m2BurnLeft, 680_000), statusPriced, [4]byte{}, 5, true},
		{"a return bomb of 1 MiB", base.with(m2Faults, swReturnBomb, m2BombSize, 1<<20), statusMalformed, [4]byte{}, 1,
			false},
		{"a revert bomb of 1 MiB", base.with(m2Faults, swRevertBomb, m2BombSize, 1<<20), statusRefused, [4]byte{}, 9,
			true},
		{"a 64-byte answer", base.with(m2Faults, swShort), statusMalformed, [4]byte{}, 1, false},
		{"ReadStarved", base.with(m2Faults, swReason, m2Reason, reasonOf(readStarved)), statusStarved, readStarved, 1,
			false},
		{"GasShort", base.with(m2Faults, swReason, m2Reason, reasonOf(gasShort)), statusStarved, gasShort, 1, false},
		{"a revert carrying the sampler's tag", base.with(m2Faults, swReason, m2Reason, reasonOf(samplerTag)),
			statusRefused, samplerTag, maxProbesPerDirection, false},
		{"a bare revert", base.with(m2Faults, swBare), statusRefused, [4]byte{}, maxProbesPerDirection, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n.load(t, tc.slots)
			a := n.sample(t, defaultTaker, defaultSampleGas, samplerLimits(testProbeGas), asksOf(tc.slots)...)
			_, gas, size := n.last()
			for _, w := range a.Walks {
				require.Len(t, w.Probes, tc.probes)
				require.Equal(t, tc.truncated, w.Truncated)
				for _, p := range w.Probes {
					require.Equal(t, tc.status, probeStatus(p.Status.Uint64()))
					require.Equal(t, tc.reason, p.Reason)
					// the callee is held to the cap; the probe's own count adds its call's access cost
					require.LessOrEqual(t, p.GasUsed.Uint64(), testProbeGas+2_611)
				}
			}
			require.Equal(t, answerSize(len(a.Walks[0].Probes)+len(a.Walks[1].Probes)), size,
				"the answer's size follows the probes, never the pool's bytes")
			t.Logf("%d + %d probes, first gas %s and %s, call %d, answer %d bytes", len(a.Walks[0].Probes),
				len(a.Walks[1].Probes), a.Walks[0].Probes[0].GasUsed, a.Walks[1].Probes[0].GasUsed, gas, size)
		})
	}
	// every other switch, alone: the sampler answers
	for _, sw := range []int{swCaller, swEqualFee, swCoreBand, swUsed1bp, swRequestRevert, swLevRevert, swLevShort} {
		s := base.with(m2Faults, sw)
		n.load(t, s)
		n.sample(t, defaultTaker, defaultSampleGas, samplerLimits(testProbeGas), asksOf(s)...)
	}
}

func TestSamplerGasGate(t *testing.T) {
	n := newEVMNode(t)
	s := m2Base(8, 6, 75_600)
	n.load(t, s)
	asks := []sampleAsk{m2Ask(s, true, firstSeeds(8)), m2Ask(s, false, firstSeeds(6))}
	whole := n.sample(t, defaultTaker, defaultSampleGas, samplerLimits(testProbeGas), asks...)
	for _, w := range whole.Walks {
		require.False(t, w.Truncated, "at the defaults the gate stops no walk")
	}
	for _, tc := range []struct {
		gas    uint64
		probes int // per walk, -1: the whole walk
	}{{12_000_000, -1}, {6_000_000, 28}, {5_400_000, 0}, {3_000_000, 0}} {
		a := n.sample(t, defaultTaker, tc.gas, samplerLimits(testProbeGas), asks...)
		for k, w := range a.Walks {
			if tc.probes < 0 {
				require.Equal(t, whole.Walks[k], w, "at %d gas the walk is whole", tc.gas)
				continue
			}
			require.True(t, w.Truncated, "at %d gas the gate stops the walk", tc.gas)
			require.Len(t, w.Probes, tc.probes, "at %d gas", tc.gas)
			require.Zero(t, w.BandWad, "at %d gas the request is not read", tc.gas)
			for _, p := range w.Probes { // a prefix of the whole walk, each answer the same
				i := slices.IndexFunc(whole.Walks[k].Probes, func(q sampleProbe) bool { return q.Ask.Cmp(p.Ask) == 0 })
				require.GreaterOrEqual(t, i, 0)
				q := whole.Walks[k].Probes[i]
				require.Equal(t, []*big.Int{q.Used, q.Out, q.FeeWad, q.Status},
					[]*big.Int{p.Used, p.Out, p.FeeWad, p.Status})
			}
		}
	}

	// the gate itself: a probe needs the cap, its 64th and the reserve left, and a call has its intrinsic and entry
	// costs to pay before its first probe, which the floor's 100,000 covers at the largest cap
	floor := maxProbeGas*64/63 + samplerReserveGas + 100_000
	require.Equal(t, uint64(20_717_460), floor)
	for _, probeGas := range []uint64{testProbeGas, maxProbeGas} {
		gate := probeGas*64/63 + samplerReserveGas
		first := func(gas uint64) int {
			return len(n.sample(t, defaultTaker, gas, samplerLimits(probeGas), asks...).Walks[0].Probes)
		}
		require.Zero(t, first(gate), "at the plain gate the call's own costs come first")
		require.Positive(t, first(gate+100_000), "the floor's margin covers them")
		lo, hi := gate, gate+100_000
		for hi-lo > 1 {
			if mid := (lo + hi) / 2; first(mid) > 0 {
				hi = mid
			} else {
				lo = mid
			}
		}
		t.Logf("at a cap of %d the first probe needs %d call gas, %d over the plain gate %d", probeGas, hi, hi-gate,
			gate)
		a := n.sample(t, defaultTaker, defaultSampleGas, samplerLimits(probeGas), asks...)
		for k, w := range a.Walks {
			require.False(t, w.Truncated, "at the default call gas a cap of %d stops no walk", probeGas)
			require.Equal(t, len(whole.Walks[k].Probes), len(w.Probes))
		}
	}

	// a truncated walk is a usable ladder: on a curve alone, where nothing needs the request, it settles and quotes
	// under the pool
	curve := s.with(m2Sale+rtLevR, 0, m2Buy+rtLevR, 0)
	n.load(t, curve)
	a := n.sample(t, defaultTaker, 5_700_000, samplerLimits(testProbeGas), m2Ask(curve, true, firstSeeds(8)),
		m2Ask(curve, false, firstSeeds(6)))
	for k, w := range a.Walks {
		require.True(t, w.Truncated)
		d := [2]int{8, 6}
		sale := k == 0
		l := ladderOf(t, settle(settleOf(curve, sale, d[k], d[1-k], w)))
		res := n.audit(t, sale, l, n.switchesOf(t, sale, w), nil)
		t.Logf("a curve alone, truncated at 5.7M gas, %d probes, direction %d: %s", len(w.Probes), k, res)
		require.Positive(t, res.points)
		require.Zero(t, res.over)
	}
}

func TestSamplerReadsTheRequestOnce(t *testing.T) {
	n := newEVMNode(t)
	base := m2Base(8, 6, 75_600)
	run := func(s m2Slots) (sampleAnswer, []evmFrame) {
		n.load(t, s)
		a := n.sample(t, defaultTaker, defaultSampleGas, samplerLimits(testProbeGas), m2Ask(s, true, firstSeeds(8)),
			m2Ask(s, false, firstSeeds(6)))
		frames, _, _ := n.last()
		return a, frames
	}

	// both walks carry a tier label over dust: one request each, its words kept; levState once, on the buy alone, on
	// the request's 26 words, its word 6 the walk's bid; no curve fee, as each walk shows two labels
	a, frames := run(base)
	for k, w := range a.Walks {
		sale := k == 0
		request := callsOf(frames, pairRequestSelector, sale)
		require.Len(t, request, 1)
		words := request[0].output
		require.Len(t, words, 32*requestWords)
		require.Equal(t, common.LeftPadBytes(defaultTaker[:], 32), words[:32], "the request's word 0 is its caller")
		require.Zero(t, word(words, 3).Cmp(big.NewInt(int64(k))), "word 3 is 1 on a buy")
		require.Equal(t, base.at(m2Band).Uint64(), w.BandWad)
		require.Equal(t, requestOf(words),
			pairWords{band: word(words, 10), usdIn: w.UsdIn, usdOut: w.UsdOut, legs: w.Legs})
		require.Zero(t, w.CurveFeeWad.Sign(), "two labels: no curve fee")
		require.Empty(t, callsOf(frames, quotePairSelector, sale))
		lev := callsOf(frames, levStateSelector, sale)
		if sale {
			require.Empty(t, lev, "a sale reads no bid")
			require.Zero(t, w.BidWad.Sign())
			continue
		}
		require.Len(t, lev, 1)
		require.Equal(t, append(slices.Clone(levStateSelector), words...), lev[0].input)
		require.Zero(t, word(lev[0].output, 6).Cmp(w.BidWad))
		require.Zero(t, base.at(m2Bid).Cmp(w.BidWad))
	}
	for _, f := range frames {
		require.Equal(t, vm.STATICCALL, f.op)
	}

	// a walk of one tier label reads the curve's fee once, on the request's words with word 13, the input, zeroed
	a, frames = run(base.with(m2Sale+rtCurveR, 0))
	request := callsOf(frames, pairRequestSelector, true)
	require.Len(t, request, 1)
	quote := callsOf(frames, quotePairSelector, true)
	require.Len(t, quote, 1)
	zeroed := slices.Clone(request[0].output)
	clear(zeroed[32*13 : 32*14])
	require.NotEqual(t, zeroed, request[0].output)
	require.Equal(t, append(slices.Clone(quotePairSelector), zeroed...), quote[0].input)
	require.Zero(t, base.at(m2CurveFee).Cmp(a.Walks[0].CurveFeeWad))

	// a failed read leaves its words zero
	for _, tc := range []struct {
		name    string
		sw      int
		request bool
	}{{"pairRequest reverts", swRequestRevert, false}, {"levState reverts", swLevRevert, true},
		{"levState answers 7 words", swLevShort, true}} {
		a, _ = run(base.with(m2Faults, tc.sw))
		for k, w := range a.Walks {
			require.Equal(t, tc.request, w.BandWad != 0, tc.name)
			if !tc.request {
				for _, v := range append([]*big.Int{w.CurveFeeWad, w.UsdIn, w.UsdOut}, w.Legs[:]...) {
					require.Zero(t, v.Sign(), tc.name)
				}
			}
			require.Zero(t, w.BidWad.Sign(), "%s: direction %d", tc.name, k)
		}
	}

	// a buy where the curve wins only under dustOut and leverage above: the dust probe on the curve's label does not
	// block the curve fee's read
	a, frames = run(base.with(m2Buy+rtCurveR, 7_000, m2Buy+rtCurveX, 5_000_000, m2Buy+rtLevR, ten(132, 10),
		m2Buy+rtLevX, ten(1, 15)))
	buy := a.Walks[1]
	var dustLabels, labels []string
	for _, p := range buy.Probes {
		if st := probeStatus(p.Status.Uint64()); st == statusPriced || st == statusClipped {
			if p.Out.Cmp(dust) < 0 {
				dustLabels = append(dustLabels, p.FeeWad.String())
			} else {
				labels = append(labels, p.FeeWad.String())
			}
		}
	}
	require.Contains(t, dustLabels, base.at(m2CurveFee).String())
	require.Equal(t, []string{base.at(m2Params + pmSpreadDn).String()},
		slices.Compact(slices.Sorted(slices.Values(labels))))
	require.Zero(t, base.at(m2CurveFee).Cmp(buy.CurveFeeWad))
	require.Len(t, callsOf(frames, quotePairSelector, false), 1)
	require.Len(t, buy.Probes, maxProbesPerDirection+1, "the probe above, past a spent budget")
	last := buy.Probes[len(buy.Probes)-1]
	require.Zero(t, aboveOf(topOf(t, buy, last.Ask)).Cmp(last.Ask), "at used + (used >> 10) + 1 of the top")
	t.Logf("the probe above on the buy: %s", last.Ask)

	// a buy whose leverage wins only on dust reads nothing
	a, frames = run(base.with(m2Buy+rtLevR, 7_000, m2Buy+rtLevX, 5_000_000))
	require.Zero(t, a.Walks[1].BandWad)
	require.Empty(t, callsOf(frames, pairRequestSelector, false))
	require.NotZero(t, a.Walks[0].BandWad)
}

func TestSamplerWalks(t *testing.T) {
	n := newEVMNode(t)
	base := m2Base(8, 6, 75_600)
	sample := func(s m2Slots, sale, buy []*big.Int) (sampleAnswer, []evmFrame) {
		n.load(t, s)
		a := n.sample(t, defaultTaker, defaultSampleGas, samplerLimits(testProbeGas), m2Ask(s, true, sale),
			m2Ask(s, false, buy))
		frames, _, _ := n.last()
		return a, frames
	}

	t.Run("dust windows", func(t *testing.T) {
		// whole-unit outputs under an 8 % band: small buys alternate between a price and a refusal
		a, _ := sample(base.with(m2Faults, swCoreBand, m2Buy+rtLevR, 0), firstSeeds(8), firstSeeds(6))
		head := statusesOf(a.Walks[1])[:4]
		require.Equal(t, []probeStatus{statusRefused, statusPriced, statusRefused, statusPriced}, head)
		out := topOf(t, a.Walks[1], nil).Out
		require.Positive(t, out.Cmp(big.NewInt(1e8)), "the walk reaches past the windows")
		t.Logf("dust windows: the buy walk's top pays %s", out)
	})

	t.Run("the probe above", func(t *testing.T) {
		// leverage alone on the sale: every seed priced on a tier label, the budget spent, and the probe above made
		// last, once
		a, frames := sample(base.with(m2Sale+rtCurveR, 0), firstSeeds(8), firstSeeds(6))
		sale := a.Walks[0]
		require.Len(t, sale.Probes, maxProbesPerDirection+1)
		previews := callsOf(frames, previewSelector, true)
		last := word(previews[len(previews)-1].input[4:], 2)
		top := topOf(t, sale, last)
		require.Zero(t, base.at(m2Params+pmSpread).Cmp(top.FeeWad))
		require.Zero(t, aboveOf(top).Cmp(last), "at used + (used >> 10) + 1 of the top tier price")
		once := slices.DeleteFunc(previews, func(f evmFrame) bool { return word(f.input[4:], 2).Cmp(last) != 0 })
		require.Len(t, once, 1, "the probe above runs once")
		t.Logf("the probe above at %s", last)

		// a curve-labelled top asks no probe above
		buy := a.Walks[1]
		bt := topOf(t, buy, nil)
		require.Zero(t, base.at(m2CurveFee).Cmp(bt.FeeWad))
		above := aboveOf(bt)
		require.False(t, slices.ContainsFunc(buy.Probes, func(p sampleProbe) bool { return p.Ask.Cmp(above) == 0 }))

		// never at or past 2^120: a tier-labelled top at 2^120 - 2^100 asks nothing above it
		topSeed := new(big.Int).Sub(new(big.Int).Lsh(one, 120), new(big.Int).Lsh(one, 100))
		a, _ = sample(base.with(m2Sale+rtCurveR, 0, m2Sale+rtLevR, new(big.Int).Lsh(one, 119), m2Sale+rtLevX,
			new(big.Int).Lsh(one, 120)), []*big.Int{big.NewInt(1e6), big.NewInt(1e12), ten(1, 20), ten(1, 30), topSeed},
			firstSeeds(6))
		sale = a.Walks[0]
		highest := sale.Probes[len(sale.Probes)-1]
		require.Zero(t, highest.Ask.Cmp(topSeed))
		require.Equal(t, statusPriced, probeStatus(highest.Status.Uint64()))
		require.Zero(t, base.at(m2Params+pmSpread).Cmp(highest.FeeWad))
	})

	t.Run("refine splits at the midpoint of the asks", func(t *testing.T) {
		// used is 1 bp under each ask, so the midpoint of the used differs from the midpoint of the asks
		a, _ := sample(base.with(m2Faults, swUsed1bp, m2Sale+rtCurveR, 0),
			[]*big.Int{big.NewInt(1_000_000), big.NewInt(30_000_000), big.NewInt(90_000_000)},
			[]*big.Int{big.NewInt(1e7)})
		asks := map[string]bool{}
		for _, p := range a.Walks[0].Probes {
			asks[p.Ask.String()] = true
			if probeStatus(p.Status.Uint64()) == statusPriced {
				require.Zero(t, new(big.Int).Sub(p.Ask, new(big.Int).Quo(p.Ask, big.NewInt(10_000))).Cmp(p.Used))
			}
		}
		require.True(t, asks["15500000"], "the midpoint of the asks 1e6 and 3e7")
		require.False(t, asks["15498450"], "not the midpoint of their used")
	})
}

// m2Control takes a step out of settle, or forces a word, on the walk the rule settles. over: its quotes must pass
// the pool's somewhere, or the state does not test what it is for; false: the rule's other steps alone keep it safe.
type m2Control struct {
	name string
	over bool
	edit func(in settleIn, noAbove []sampled) settleIn
}

func TestSamplerLadderNeverQuotesMoreThanThePool(t *testing.T) {
	n := newEVMNode(t)
	base := m2Base(8, 6, 75_600)
	keepAll := func(*room, sampled) bool { return true }
	needOnly := func(r *room, p sampled) bool {
		lo, hi := r.limits(p.used)
		return p.out.Cmp(new(big.Int).Add(lo, r.need)) >= 0 && new(big.Int).Add(p.out, r.need).Cmp(hi) <= 0
	}
	roomOff := m2Control{"room test off", false, func(in settleIn, _ []sampled) settleIn {
		in.roomTest = keepAll
		return in
	}}
	aboveOff := func(over bool) m2Control {
		return m2Control{"probe above off", over, func(in settleIn, p []sampled) settleIn {
			in.probes, in.skipAbove = p, true
			return in
		}}
	}
	bothOff := m2Control{"room test and probe above off", true, func(in settleIn, p []sampled) settleIn {
		in.probes, in.skipAbove, in.roomTest = p, true, keepAll
		return in
	}}
	needNoAbove := m2Control{"need only, probe above off", true, func(in settleIn, p []sampled) settleIn {
		in.probes, in.skipAbove, in.roomTest = p, true, needOnly
		return in
	}}
	marginNoAbove := m2Control{"margin, probe above off", false, func(in settleIn, p []sampled) settleIn {
		in.probes, in.skipAbove = p, true
		return in
	}}
	bidForced := m2Control{"bid word forced positive", true, func(in settleIn, _ []sampled) settleIn {
		in.words.bid = ten(736, 15)
		return in
	}}
	trustedForced := m2Control{"trusted forced", true, func(in settleIn, _ []sampled) settleIn {
		in.trusted = true
		return in
	}}

	// lotteryZone finds the far edge's lottery on a sale: the last tier-labelled size of an n-step grid over [lo, hi],
	// bisected to 256 units, then every size within 400 of it. tiered: the tier-labelled sizes there; holes: the
	// sizes between the first and the last of them that the curve fills.
	lotteryZone := func(t *testing.T, lo, hi *big.Int, steps int64, tier *big.Int) (tiered, holes []*big.Int) {
		isTier := func(a *big.Int) bool { p := n.preview(t, true, a); return p != nil && p[2].Cmp(tier) == 0 }
		var grid []*big.Int
		for i := range steps + 1 {
			v := new(big.Int).Sub(hi, lo)
			grid = append(grid, v.Mul(v, big.NewInt(i)).Quo(v, big.NewInt(steps)).Add(v, lo))
		}
		last := -1
		for i, a := range grid {
			if isTier(a) {
				last = i
			}
		}
		require.True(t, last >= 0 && last+1 < len(grid))
		a, b := grid[last], grid[last+1]
		for new(big.Int).Sub(b, a).Cmp(big.NewInt(256)) > 0 {
			if m := new(big.Int).Rsh(new(big.Int).Add(a, b), 1); isTier(m) {
				a = m
			} else {
				b = m
			}
		}
		end := new(big.Int).Add(b, big.NewInt(400))
		var zone []*big.Int
		for v := new(big.Int).Sub(a, big.NewInt(400)); v.Cmp(end) < 0; v = new(big.Int).Add(v, one) {
			if zone = append(zone, v); isTier(v) {
				tiered = append(tiered, v)
			}
		}
		require.NotEmpty(t, tiered)
		for _, v := range zone {
			if p := n.preview(t, true, v); p != nil && p[2].Cmp(tier) != 0 && v.Cmp(tiered[0]) > 0 &&
				v.Cmp(tiered[len(tiered)-1]) < 0 {
				holes = append(holes, v)
			}
		}
		return tiered, holes
	}
	below := func(seeds []*big.Int, lim *big.Int) []*big.Int {
		return slices.DeleteFunc(slices.Clone(seeds), func(a *big.Int) bool { return a.Cmp(lim) >= 0 })
	}
	ascending := func(v []*big.Int) []*big.Int {
		slices.SortFunc(v, func(a, b *big.Int) int { return a.Cmp(b) })
		return slices.CompactFunc(v, func(a, b *big.Int) bool { return a.Cmp(b) == 0 })
	}
	span := func(lo, hi *big.Int) []*big.Int {
		var v []*big.Int
		for a := new(big.Int).Set(lo); a.Cmp(hi) <= 0; a = new(big.Int).Add(a, one) {
			v = append(v, a)
		}
		return v
	}

	const h = 10_000_000 // where the floor flicker and the token ceiling sit on a sale
	near := span(big.NewInt(h-1), big.NewInt(h+4))
	nearSeeds := ascending(append(below(firstSeeds(8), big.NewInt(h-1)), big.NewInt(h-1), big.NewInt(h+1)))
	levAtH := new(big.Int).Mul(base.at(m2Sale+rtLevR), big.NewInt(h)) // the leverage route's output at h
	levAtH.Quo(levAtH, new(big.Int).Add(base.at(m2Sale+rtLevX), big.NewInt(h)))
	const j0 = 50_000_000_000 // where the lever-down's output falls on a buy
	var joinSizes []*big.Int
	for k := int64(-2); k < 40; k++ {
		joinSizes = append(joinSizes, big.NewInt(j0+k))
	}
	for e := 3; e <= 9; e++ {
		joinSizes = append(joinSizes, new(big.Int).Add(big.NewInt(j0), pow10(e)))
	}
	holes := base.with(m2Sale+rtHoleMod, 7, m2Sale+rtHoleRem, 3, m2Buy+rtHoleMod, 7, m2Buy+rtHoleRem, 3)
	rowA := holes.with(m2Params+pmDnCap, ten(1, 15), m2Params+pmBand, ten(1, 15), m2Sale+rtLevR, ten(7556, 12),
		m2Sale+rtLevX, ten(1, 13), m2Buy+rtLevR, ten(13221, 8), m2Buy+rtLevX, ten(1, 15))
	rowB := holes.with(m2Params+pmSpreadDn, 0)
	offLattice := func(d int) []*big.Int { // the seeds stepped off the refused lattice
		v := firstSeeds(d)
		for k, a := range v {
			if new(big.Int).Mod(a, big.NewInt(7)).Int64() == 3 {
				v[k] = new(big.Int).Add(a, one)
			}
		}
		return v
	}
	latticeAbove := func(w sampleWalk) []*big.Int { // the refused sizes just above each priced one
		var v []*big.Int
		for _, p := range w.Probes {
			if probeStatus(p.Status.Uint64()) == statusPriced {
				a := new(big.Int).Sub(p.Ask, new(big.Int).Mod(p.Ask, big.NewInt(7)))
				for k := int64(0); k < 4; k++ {
					v = append(v, new(big.Int).Add(a, big.NewInt(3+7*k)))
				}
			}
		}
		return v
	}

	type m2Case struct {
		name     string
		slots    m2Slots
		sale     bool
		d0, d1   int
		prep     func(t *testing.T) (seeds, extra []*big.Int) // after the state is loaded; nil: a first refresh
		extraOf  func(w sampleWalk) []*big.Int
		gated    bool // the row fails its gate
		dark     bool // the rule publishes nothing
		controls []m2Control
	}
	cases := []m2Case{
		{name: "leverage under the curve, sale", slots: base, sale: true, d0: 8, d1: 6},
		{name: "leverage under the curve, buy", slots: base, d0: 8, d1: 6},
		{name: "a dropped edge", slots: base.with(m2Sale+rtLevEdge, 15_000_000), sale: true, d0: 8, d1: 6},
		{name: "a clip re-asked onto leverage", slots: base.with(m2Sale+rtCurveClip, 15_000_000, m2Sale+rtLevEdge,
			15_000_000), sale: true, d0: 8, d1: 6},
		{name: "dust windows, buy", slots: base.with(m2Faults, swCoreBand, m2Buy+rtLevR, 0), d0: 8, d1: 6},
		{name: "far-edge lottery, 8/6 sale", slots: base.with(m2Sale+rtCurveR, ten(740, 9), m2Sale+rtCurveX, ten(1, 9),
			m2Sale+rtLottery, 2_000), sale: true, d0: 8, d1: 6,
			prep: func(t *testing.T) ([]*big.Int, []*big.Int) {
				tiered, holes := lotteryZone(t, big.NewInt(100_000_000), big.NewInt(103_000_000), 1_500,
					base.at(m2Params+pmSpread))
				t.Logf("lottery zone: %d tier-labelled sizes, %d holes", len(tiered), len(holes))
				seeds := ascending(append(below(firstSeeds(8), tiered[0]), tiered[len(tiered)-8:]...))
				return seeds, append(holes, tiered...)
			},
			controls: []m2Control{roomOff, aboveOff(false), bothOff}},
		{name: "far-edge lottery, 18/18 sale", slots: m2Base(18, 18, 3_000).with(m2Sale+rtCurveR, ten(2900, 23),
			m2Sale+rtCurveX, ten(1, 23), m2Sale+rtLevR, ten(2980*3, 23), m2Sale+rtLevX, ten(3, 23),
			m2Sale+rtLottery, 15_000, m2Legs0, ten(1, 22), m2Legs0+2, ten(3, 21), m2Legs1+3, ten(1, 25),
			m2CV, ten(6, 25)), sale: true, d0: 18, d1: 18,
			prep: func(t *testing.T) ([]*big.Int, []*big.Int) {
				// within margin and outside need: rungs whose room passes need and not the margin, in pairs with a hole
				// between them, so the margin must drop them
				s18 := m2Base(18, 18, 3_000)
				tiered, holes := lotteryZone(t, ten(136, 20), ten(1376, 19), 1_600, s18.at(m2Params+pmSpread))
				data, err := poolABI.Pack("pairRequest", m2Token0, m2Token1, tiered[0])
				require.NoError(t, err)
				ret, err := n.view(t, data)
				require.NoError(t, err)
				r := newRoom(settleIn{words: requestOf(ret), sale: true, decIn: 18, decOut: 18,
					rowBand: m2Row(s18).band})
				require.NotNil(t, r)
				var mid []*big.Int
				for _, a := range tiered {
					p := n.preview(t, true, a)
					lo, hi := r.limits(p[0])
					room := minBig(new(big.Int).Sub(p[1], lo), new(big.Int).Sub(hi, p[1]))
					if room.Cmp(r.need) > 0 && room.Cmp(r.margin(p[0], p[1])) < 0 {
						mid = append(mid, a)
					}
				}
				var pick, extra []*big.Int
				for k := 0; k+1 < len(mid) && len(pick) < 6; k++ {
					between := func(v *big.Int) bool { return v.Cmp(mid[k]) > 0 && v.Cmp(mid[k+1]) < 0 }
					if slices.ContainsFunc(holes, between) {
						pick = append(pick, mid[k], mid[k+1])
						extra = append(extra, span(mid[k], mid[k+1])...)
					}
				}
				pick = ascending(pick)
				t.Logf("lottery zone: %d tier-labelled sizes, %d holes, need %s, %d rungs between need and margin, "+
					"rungs forced %v", len(tiered), len(holes), r.need, len(mid), pick)
				require.NotEmpty(t, pick)
				seeds := ascending(append(below(firstSeeds(18), tiered[0]), pick...))
				return seeds, slices.Concat(holes, tiered, extra)
			},
			controls: []m2Control{needNoAbove, marginNoAbove}},
		{name: "floor flicker, sale", slots: base.with(m2Sale+rtFlickerAt, h, m2Sale+rtFlickerLen, 1,
			m2Sale+rtLevEdge, h+3), sale: true, d0: 8, d1: 6,
			prep:     func(*testing.T) ([]*big.Int, []*big.Int) { return nearSeeds, near },
			controls: []m2Control{aboveOff(true)}},
		{name: "token ceiling, sale", slots: base.with(m2Sale+rtFallAt, h, m2Sale+rtFallBy, 1_100, m2Sale+rtTokenCap,
			new(big.Int).Sub(levAtH, one)), sale: true, d0: 8, d1: 6,
			prep:     func(*testing.T) ([]*big.Int, []*big.Int) { return nearSeeds, near },
			controls: []m2Control{aboveOff(true)}},
		{name: "join, buy, bid word zero", slots: base.with(m2Buy+rtFallAt, j0, m2Buy+rtFallBy, 500_000, m2Bid, 0),
			d0: 8, d1: 6, prep: func(*testing.T) ([]*big.Int, []*big.Int) { return firstSeeds(6), joinSizes },
			dark: true, controls: []m2Control{bidForced}},
	}
	for _, row := range []struct {
		name  string
		slots m2Slots
	}{{"hole row A, dnCap = band = 0.1 %", rowA}, {"hole row B, spreadDn 0", rowB}} {
		for _, sale := range []bool{true, false} {
			d := map[bool]int{true: 8, false: 6}[sale]
			cases = append(cases, m2Case{name: row.name + map[bool]string{true: ", sale", false: ", buy"}[sale],
				slots: row.slots, sale: sale, d0: 8, d1: 6,
				prep:    func(*testing.T) ([]*big.Int, []*big.Int) { return offLattice(d), nil },
				extraOf: latticeAbove, gated: true, dark: true, controls: []m2Control{trustedForced}})
		}
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			n.load(t, tc.slots)
			decIn, decOut := tc.d1, tc.d0
			if tc.sale {
				decIn, decOut = tc.d0, tc.d1
			}
			seeds, extra := firstSeeds(decIn), []*big.Int(nil)
			if tc.prep != nil {
				seeds, extra = tc.prep(t)
			}
			a := n.sample(t, defaultTaker, defaultSampleGas, samplerLimits(testProbeGas), m2Ask(tc.slots, tc.sale, seeds))
			frames, _, _ := n.last()
			w := a.Walks[0]
			if tc.extraOf != nil {
				extra = append(slices.Clone(extra), tc.extraOf(w)...)
			}
			in := settleOf(tc.slots, tc.sale, decIn, decOut, w)
			require.Equal(t, tc.gated, !rowOk(m2Row(tc.slots), tc.slots.at(m2Band)), "the row's gate")
			l := ladderOf(t, settle(in))
			switches := n.switchesOf(t, tc.sale, w)
			rule := n.audit(t, tc.sale, l, switches, extra)
			t.Logf("the rule: %d probes, read %v, capped %v, %d route switches, %s", len(w.Probes), w.BandWad != 0,
				l.Capped, len(switches), rule)
			require.Zero(t, rule.over, "the rule's ladder never quotes more than the pool")
			if tc.dark {
				require.Zero(t, rule.points)
			} else {
				require.Greater(t, rule.sizes, 2_000)
			}
			noAbove, above := withoutAbove(in, frames, tc.sale)
			if above != nil {
				t.Logf("the probe above at %s", above)
			}
			for _, c := range tc.controls {
				if c.edit(settleIn{}, nil).skipAbove {
					require.NotNil(t, above, "%s: the walk made the probe above", c.name)
				}
				res := n.audit(t, tc.sale, ladderOf(t, settle(c.edit(in, noAbove))), switches, extra)
				t.Logf("control, %s: %s", c.name, res)
				if !c.over {
					require.Zero(t, res.over, "%s: the rule's other steps keep the state safe", c.name)
					continue
				}
				require.Positive(t, res.over, "%s: the state is armed", c.name)
				require.Greater(t, res.worstBp, 1.0, "%s: by more than a rounding", c.name)
			}
		})
	}
}
