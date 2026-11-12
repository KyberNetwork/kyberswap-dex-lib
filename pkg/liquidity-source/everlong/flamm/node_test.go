package everlongflamm

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var multicall3 = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")

// multicallABI holds the Multicall3 methods ethrpc sends.
var multicallABI = func() abi.ABI {
	const calls = `{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},` +
		`{"name":"callData","type":"bytes"}]}`
	const results = `{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},` +
		`{"name":"returnData","type":"bytes"}]}`
	a, err := abi.JSON(strings.NewReader(`[
{"name":"aggregate","type":"function","inputs":[` + calls + `],
 "outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}]},
{"name":"tryAggregate","type":"function","inputs":[{"name":"requireSuccess","type":"bool"},` + calls + `],
 "outputs":[` + results + `]},
{"name":"tryBlockAndAggregate","type":"function","inputs":[{"name":"requireSuccess","type":"bool"},` + calls + `],
 "outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},` + results + `]}]`))
	if err != nil {
		panic(err)
	}
	return a
}()

type innerCall struct {
	Target   common.Address
	CallData []byte
}

type innerResult struct {
	Success    bool
	ReturnData []byte
}

// sent is one call the node answered, a multicall's calls each (sent from Multicall3), with its block tag, the gas
// it stated and its state overrides. The sampler's own call is marked, and every call the sampler made inside it is
// kept as inner, from the sampler's address.
type sent struct {
	from, to common.Address
	data     []byte
	tag      string
	gas      uint64
	ov       overrideArgs
	sampler  bool
	inner    bool
}

// node is a chain at one block behind JSON-RPC, for the lister's and the tracker's reads: single and batched
// eth_calls, Multicall3's aggregates (each call answered alone), eth_getCode, eth_getStorageAt and
// eth_blockNumber. respond answers a call (ok false: it reverts, its answer the revert data), code a code read and
// storage a slot read (nil: zero), and callError, where set, gives a plain eth_call the JSON-RPC error it returns
// instead (nil: none); every call is kept with the block tag it was sent at.
//
// A call with FlammSampler's code set at its `to` by a state override is the sampler's. The node answers it with
// walks it computes from respond (sample), unless it replays a recording (recorded), where respond answers it as
// any call, or it drops overrides (dropOverride), where the code already at `to` runs. fault answers the k-th
// sampler call, from 0, in the sampler's place.
type node struct {
	*httptest.Server
	t         testing.TB
	block     uint64
	respond   func(from, to common.Address, data []byte) (bool, []byte)
	code      func(common.Address) []byte
	storage   func(common.Address, common.Hash) common.Hash
	callError func(to common.Address, data []byte) map[string]any

	recorded      bool
	dropOverride  bool
	refuseBatches bool
	gasCap        uint64                                      // the most gas the node gives a call; zero: what it states
	gasOf         func(to common.Address, data []byte) uint64 // what a call inside the sampler spends; nil: fakeCallGas
	walkBlock     uint64                                      // the block the sampler's answer reports; zero: the node's
	walkTag       [4]byte                                     // the tag it reports; zero: the sampler's
	fault         func(k int) *fault

	mu       sync.Mutex
	sent     []sent
	down     bool
	requests int      // the HTTP requests it served, a batch as one
	sampled  int      // the sampler calls it saw
	answers  [][]byte // the sampler answers it computed (sample)
}

// fault is how the node answers a sampler call in the sampler's place: a JSON-RPC error, an HTTP status, or a
// result (null, or bytes that are not the sampler's answer), each after a wait; with none of them, the wait and then
// the call answered as usual.
type fault struct {
	err    map[string]any
	status int
	result json.RawMessage
	delay  time.Duration
}

type rpcMessage struct {
	ID     json.RawMessage   `json:"id"`
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

// callArgs is an eth_call's call object, in either field name a client gives its input.
type callArgs struct {
	From  common.Address  `json:"from"`
	To    common.Address  `json:"to"`
	Gas   *hexutil.Uint64 `json:"gas"`
	Data  *hexutil.Bytes  `json:"data"`
	Input *hexutil.Bytes  `json:"input"`
}

// overrideArgs are an eth_call's state overrides, of which the node reads the code.
type overrideArgs map[common.Address]struct {
	Code *hexutil.Bytes `json:"code"`
}

func callOf(m rpcMessage) (callArgs, overrideArgs, error) {
	var c callArgs
	if len(m.Params) == 0 {
		return c, nil, errors.New("an eth_call with no call")
	}
	if err := json.Unmarshal(m.Params[0], &c); err != nil {
		return c, nil, err
	}
	var ov overrideArgs
	if len(m.Params) > 2 {
		if err := json.Unmarshal(m.Params[2], &ov); err != nil {
			return c, nil, err
		}
	}
	return c, ov, nil
}

// isSampler reports a call with the sampler's code set at its `to`.
func isSampler(to common.Address, ov overrideArgs) bool {
	o, ok := ov[to]
	return ok && o.Code != nil && bytes.Equal(*o.Code, samplerRuntime)
}

func newNode(t testing.TB, block uint64, respond func(from, to common.Address, data []byte) (bool, []byte),
	code func(common.Address) []byte) *node {
	n := &node{t: t, block: block, respond: respond, code: code}
	n.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("a request the node cannot read: %v", err)
			return
		}
		n.mu.Lock()
		n.requests++
		down := n.down
		n.mu.Unlock()
		if down {
			http.Error(w, "node down", http.StatusBadGateway)
			return
		}
		if bytes.HasPrefix(bytes.TrimSpace(body), []byte("[")) {
			if n.refuseBatches {
				http.Error(w, "batch requests are not supported", http.StatusBadRequest)
				return
			}
			var msgs []rpcMessage
			if err := json.Unmarshal(body, &msgs); err != nil {
				t.Errorf("a batch the node cannot read: %v", err)
				return
			}
			replies := make([]map[string]any, len(msgs))
			for k, m := range msgs {
				replies[k] = n.answer(m)
			}
			_ = json.NewEncoder(w).Encode(replies)
			return
		}
		var m rpcMessage
		if err := json.Unmarshal(body, &m); err != nil {
			t.Errorf("a message the node cannot read: %v", err)
			return
		}
		if f := n.faultOf(m); f != nil {
			select {
			case <-time.After(f.delay):
			case <-r.Context().Done():
				return
			}
			reply := map[string]any{"jsonrpc": "2.0", "id": m.ID}
			switch {
			case f.status != 0:
				http.Error(w, http.StatusText(f.status), f.status)
				return
			case f.err != nil:
				reply["error"] = f.err
				_ = json.NewEncoder(w).Encode(reply)
				return
			case f.result != nil:
				reply["result"] = f.result
				_ = json.NewEncoder(w).Encode(reply)
				return
			}
		}
		_ = json.NewEncoder(w).Encode(n.answer(m))
	}))
	t.Cleanup(n.Close)
	return n
}

// client is an ethrpc client of the node, Multicall3 its multicall.
func (n *node) client() *ethrpc.Client {
	return ethrpc.New(n.URL).SetMulticallContract(multicall3)
}

// calls are the calls sent to the node, a multicall's each and the sampler's own, but none the sampler made.
func (n *node) calls() []sent {
	return lo.Filter(n.seen(), func(c sent, _ int) bool { return !c.inner })
}

// seen is every call the node answered, the sampler's inner calls among them.
func (n *node) seen() []sent {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]sent(nil), n.sent...)
}

// counts are the HTTP requests the node served and the sampler calls it saw, refused ones among them.
func (n *node) counts() (requests, sampled int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.requests, n.sampled
}

func (n *node) setDown(down bool) {
	n.mu.Lock()
	n.down = down
	n.mu.Unlock()
}

func (n *node) keep(s sent) {
	n.mu.Lock()
	n.sent = append(n.sent, s)
	n.mu.Unlock()
}

// faultOf counts a sampler call and gives its fault; nil: none, or not the sampler's.
func (n *node) faultOf(m rpcMessage) *fault {
	if m.Method != "eth_call" {
		return nil
	}
	c, ov, err := callOf(m)
	if err != nil || !isSampler(c.To, ov) {
		return nil
	}
	n.mu.Lock()
	k := n.sampled
	n.sampled++
	n.mu.Unlock()
	if n.fault == nil {
		return nil
	}
	return n.fault(k)
}

func (n *node) answer(m rpcMessage) map[string]any {
	reply := map[string]any{"jsonrpc": "2.0", "id": m.ID}
	fail := func(err error) map[string]any {
		n.t.Errorf("%s: %v", m.Method, err)
		reply["error"] = map[string]any{"code": -32602, "message": err.Error()}
		return reply
	}
	tag := "latest"
	if k := lo.Ternary(m.Method == "eth_getStorageAt", 2, 1); len(m.Params) > k {
		if err := json.Unmarshal(m.Params[k], &tag); err != nil {
			return fail(err)
		}
	}
	switch m.Method {
	case "eth_getStorageAt":
		var a common.Address
		var slot common.Hash
		if err := json.Unmarshal(m.Params[0], &a); err != nil {
			return fail(err)
		}
		if err := json.Unmarshal(m.Params[1], &slot); err != nil {
			return fail(err)
		}
		n.keep(sent{to: a, data: slot[:], tag: tag})
		var v common.Hash
		if n.storage != nil {
			v = n.storage(a, slot)
		}
		reply["result"] = v.Hex()
	case "eth_blockNumber":
		reply["result"] = hexutil.EncodeUint64(n.block)
	case "eth_getCode":
		var a common.Address
		if err := json.Unmarshal(m.Params[0], &a); err != nil {
			return fail(err)
		}
		n.keep(sent{to: a, tag: tag})
		reply["result"] = hexutil.Encode(n.code(a))
	case "eth_call":
		c, ov, err := callOf(m)
		if err != nil {
			return fail(err)
		}
		var data []byte
		if in := cmp.Or(c.Input, c.Data); in != nil {
			data = *in
		}
		if n.callError != nil {
			if e := n.callError(c.To, data); e != nil {
				reply["error"] = e
				return reply
			}
		}
		s := sent{from: c.From, to: c.To, data: data, tag: tag, gas: uint64(lo.FromPtr(c.Gas)), ov: ov,
			sampler: isSampler(c.To, ov)}
		if s.sampler && !n.recorded && !n.dropOverride {
			n.keep(s)
			ret, err := n.sample(s)
			if err != nil {
				return fail(err)
			}
			n.mu.Lock()
			n.answers = append(n.answers, ret)
			n.mu.Unlock()
			reply["result"] = hexutil.Encode(ret)
			return reply
		}
		ret, ok, err := n.call(s)
		if err != nil {
			return fail(err)
		}
		if ok {
			reply["result"] = hexutil.Encode(ret)
		} else {
			reply["error"] = map[string]any{"code": 3, "message": "execution reverted", "data": hexutil.Encode(ret)}
		}
	default:
		reply["error"] = map[string]any{"code": -32601, "message": "no method " + m.Method}
	}
	return reply
}

func (n *node) call(s sent) ([]byte, bool, error) {
	method, err := multicallABI.MethodById(s.data)
	if s.to != multicall3 || err != nil {
		n.keep(s)
		ok, ret := n.respond(s.from, s.to, s.data)
		return ret, ok, nil
	}
	args, err := method.Inputs.Unpack(s.data[4:])
	if err != nil {
		return nil, false, err
	}
	calls := *abi.ConvertType(args[len(args)-1], new([]innerCall)).(*[]innerCall)
	results := make([]innerResult, len(calls))
	for k, c := range calls {
		n.keep(sent{from: multicall3, to: c.Target, data: c.CallData, tag: s.tag, ov: s.ov})
		results[k].Success, results[k].ReturnData = n.respond(multicall3, c.Target, c.CallData)
	}
	block := new(big.Int).SetUint64(n.block)
	var ret []byte
	switch method.Name {
	case "aggregate":
		rets := make([][]byte, len(results))
		for k, r := range results {
			if !r.Success {
				return nil, false, nil
			}
			rets[k] = r.ReturnData
		}
		ret, err = method.Outputs.Pack(block, rets)
	case "tryAggregate":
		ret, err = method.Outputs.Pack(results)
	default:
		ret, err = method.Outputs.Pack(block, common.Hash{}, results)
	}
	return ret, err == nil, err
}

// The fake sampler's gas: what a call inside it spends where gasOf says nothing, what it spends before its first
// call, and the gas of a sampler call that states none.
const (
	fakeCallGas  uint64 = 100_000
	fakeEntryGas uint64 = 100_000
	fakeNodeGas  uint64 = 50_000_000
)

// fakeSampler is one sampler call the node answers: walks computed as FlammSampler's bytes compute them, with no
// bisection (sampler_evm_test.go runs the bytes themselves). Every call it makes is answered by respond, from the
// sampler's address, at the probe cap, and kept as inner; each spends what gasOf says, up to the cap.
type fakeSampler struct {
	n        *node
	at, pool common.Address
	tag      string
	left     uint64 // the gas left
	cap      uint64
	reserve  uint64
}

// fakeRun is a walk in progress, as FlammSampler's Run.
type fakeRun struct {
	ask              sampleAsk
	w                *sampleWalk
	cap, budget      int
	next             int
	seeding, stopped bool
	read, priced     bool
	lastOut, lastFee *big.Int
	labels           []*big.Int
}

// sample answers the sampler's call as its bytes would, but for bisection: the seeds round robin across the walks
// and ascending, each walk stopping as FlammSampler's `_judge` says; then each walk's request read; then its probe
// above. The answer reports walkBlock and walkTag where set.
func (n *node) sample(c sent) ([]byte, error) {
	m := samplerABI.Methods["sample"]
	if len(c.data) < 4 || !bytes.Equal(c.data[:4], m.ID) {
		return nil, fmt.Errorf("a sampler call that is not sample: %x", c.data)
	}
	args, err := m.Inputs.Unpack(c.data[4:])
	if err != nil {
		return nil, err
	}
	var in struct {
		Pool   common.Address
		Asks   []sampleAsk
		Limits sampleLimits
	}
	if err := m.Inputs.Copy(&in, args); err != nil {
		return nil, err
	}
	gas := lo.Ternary(c.gas != 0, c.gas, fakeNodeGas)
	if n.gasCap != 0 {
		gas = min(gas, n.gasCap)
	}
	s := &fakeSampler{n: n, at: c.to, pool: in.Pool, tag: c.tag, left: gas - min(gas, fakeEntryGas),
		cap: in.Limits.ProbeGas.Uint64(), reserve: in.Limits.ReserveGas.Uint64()}
	walks := make([]sampleWalk, len(in.Asks))
	runs := make([]*fakeRun, len(in.Asks))
	for k, a := range in.Asks {
		walks[k] = sampleWalk{Probes: []sampleProbe{}, CurveFeeWad: new(big.Int), UsdIn: new(big.Int),
			UsdOut: new(big.Int), BidWad: new(big.Int)}
		for j := range walks[k].Legs {
			walks[k].Legs[j] = new(big.Int)
		}
		budget := int(a.MaxProbes.Int64())
		runs[k] = &fakeRun{ask: a, w: &walks[k], cap: budget + 1, budget: budget, seeding: true}
	}
	for again := true; again; {
		again = false
		for _, r := range runs {
			if !r.seeding {
				continue
			}
			if r.stopped || r.budget == 0 || r.next >= len(r.ask.Seeds) {
				r.seeding = false
				continue
			}
			k, _ := s.probe(r, r.ask.Seeds[r.next])
			r.next++
			again = true
			if k < 0 {
				r.seeding = false
			} else {
				r.judge(r.w.Probes[k])
			}
		}
	}
	for _, r := range runs {
		s.tryRead(r)
	}
	for _, r := range runs {
		s.above(r)
	}
	tag := lo.Ternary(n.walkTag != [4]byte{}, n.walkTag, samplerTag)
	block := lo.Ternary(n.walkBlock != 0, n.walkBlock, n.block)
	return m.Outputs.Pack(tag, new(big.Int).SetUint64(block), walks)
}

// open is the gas gate: the next call gets its whole cap, and the reserve stays.
func (s *fakeSampler) open() bool { return s.left >= s.cap*64/63+s.reserve }

func (s *fakeSampler) stop(r *fakeRun) { r.w.Truncated, r.stopped = true, true }

// call is one call inside the sampler: whether it returned, what came back, and the gas it spent. A call that needs
// more than the cap runs out of gas, with nothing back.
func (s *fakeSampler) call(to common.Address, data []byte) (bool, []byte, uint64) {
	s.n.keep(sent{from: s.at, to: to, data: data, tag: s.tag, gas: s.cap, inner: true})
	need := fakeCallGas
	if s.n.gasOf != nil {
		need = s.n.gasOf(to, data)
	}
	spent := min(need, s.cap)
	s.left -= spent
	if need > s.cap {
		return false, nil, spent
	}
	ok, ret := s.n.respond(s.at, to, data)
	return ok, ret, spent
}

// probe is the probe at `ask`: its index, -1 where none is made, and whether it was made now.
func (s *fakeSampler) probe(r *fakeRun, ask *big.Int) (int, bool) {
	k := 0
	for ; k < len(r.w.Probes); k++ {
		if c := r.w.Probes[k].Ask.Cmp(ask); c == 0 {
			return k, false
		} else if c > 0 {
			break
		}
	}
	if r.stopped || len(r.w.Probes) >= r.cap {
		return -1, false
	}
	if !s.open() {
		s.stop(r)
		return -1, false
	}
	data, err := poolABI.Pack("previewSwap", r.ask.TokenIn, r.ask.TokenOut, ask)
	require.NoError(s.n.t, err)
	ok, ret, spent := s.call(s.pool, data)
	p := sampleProbe{Ask: new(big.Int).Set(ask), Used: new(big.Int), Out: new(big.Int), FeeWad: new(big.Int),
		GasUsed: new(big.Int).SetUint64(spent)}
	if ok && len(ret) == 96 {
		p.Used, p.Out, p.FeeWad = word(ret, 0), word(ret, 1), word(ret, 2)
	}
	if !ok && len(ret) > 3 {
		copy(p.Reason[:], ret[:4])
	}
	var st probeStatus
	switch {
	case spent*8 > s.cap*7:
		st = statusStarved
	case !ok:
		st = lo.Ternary(slices.ContainsFunc(starvedReverts, func(b []byte) bool { return bytes.Equal(b, p.Reason[:]) }),
			statusStarved, statusRefused)
	case len(ret) != 96:
		st = statusMalformed
	default:
		st = fillStatus(ask, p.Used, p.Out)
	}
	p.Status = big.NewInt(int64(st))
	r.w.Probes = slices.Insert(r.w.Probes, k, p)
	if r.budget != 0 {
		r.budget--
	}
	return k, true
}

// judge is FlammSampler's `_judge`: whether a seed's answer ends the walk's seeding.
func (r *fakeRun) judge(p sampleProbe) {
	switch probeStatus(p.Status.Uint64()) {
	case statusStarved, statusMalformed, statusClipped:
		r.seeding = false
		return
	case statusRefused:
		if r.priced {
			r.seeding = false
		}
		return
	}
	if p.Out.Cmp(dust) < 0 {
		return
	}
	f := p.FeeWad
	if r.priced && p.Out.Cmp(r.lastOut) < 0 && f.Cmp(r.lastFee) == 0 {
		r.seeding = false
		return
	}
	if !slices.ContainsFunc(r.labels, func(l *big.Int) bool { return l.Cmp(f) == 0 }) {
		if len(r.labels) == 2 {
			r.seeding = false
			return
		}
		r.labels = append(r.labels, f)
	}
	r.priced, r.lastOut, r.lastFee = true, p.Out, f
}

func (r *fakeRun) tier(f *big.Int) bool {
	return r.ask.TierA.Sign() != 0 && (f.Cmp(r.ask.TierA) == 0 || f.Cmp(r.ask.TierB) == 0)
}

// isCounted: priced or clipped, paying at least dustOut, what the request read counts. isGood: priced, paying at
// least dustOut, what the probe above counts.
func isCounted(p sampleProbe) bool {
	st := probeStatus(p.Status.Uint64())
	return (st == statusPriced || st == statusClipped) && p.Out.Cmp(dust) >= 0
}

func isGood(p sampleProbe) bool {
	return probeStatus(p.Status.Uint64()) == statusPriced && p.Out.Cmp(dust) >= 0
}

// tryRead reads the request once, at the lowest probe that counts with a tier label, on a walk with a hook and a
// row; then the curve's fee where the counted probes carry one label and it is a tier, and on a buy the hook's
// levState. They take one probe of the budget.
func (s *fakeSampler) tryRead(r *fakeRun) {
	if r.read || r.stopped || r.ask.Hook == (common.Address{}) || r.ask.TierA.Sign() == 0 {
		return
	}
	k := slices.IndexFunc(r.w.Probes, func(p sampleProbe) bool { return isCounted(p) && r.tier(p.FeeWad) })
	if k < 0 {
		return
	}
	r.read = true
	if r.budget != 0 {
		r.budget--
	}
	if !s.open() {
		s.stop(r)
		return
	}
	data, err := poolABI.Pack("pairRequest", r.ask.TokenIn, r.ask.TokenOut, r.w.Probes[k].Ask)
	require.NoError(s.n.t, err)
	ok, req, _ := s.call(s.pool, data)
	if !ok || len(req) != 32*requestWords || !word(req, 10).IsUint64() {
		return
	}
	w := r.w
	w.BandWad, w.UsdIn, w.UsdOut = word(req, 10).Uint64(), word(req, 11), word(req, 12)
	for j := range w.Legs {
		w.Legs[j] = word(req, 15+j)
	}
	var label *big.Int
	single := true
	for _, p := range w.Probes {
		switch {
		case !isCounted(p):
		case label == nil:
			label = p.FeeWad
		case p.FeeWad.Cmp(label) != 0:
			single = false
		}
	}
	if single && label != nil && r.tier(label) {
		if !s.open() {
			s.stop(r)
			return
		}
		q := slices.Clone(req)
		clear(q[32*13 : 32*14]) // the amount reads zero for quotePair
		if ok, ret, _ := s.call(r.ask.Hook, slices.Concat(quotePairSelector, q)); ok && len(ret) == 32*7 {
			w.CurveFeeWad = word(ret, 3)
		}
	}
	if word(req, 3).Cmp(big.NewInt(1)) == 0 {
		if !s.open() {
			s.stop(r)
			return
		}
		if ok, ret, _ := s.call(r.ask.Hook, slices.Concat(levStateSelector, req)); ok && len(ret) == 32*levStateWords {
			w.BidWad = word(ret, 6)
		}
	}
}

// above probes used + (used >> aboveShift) + 1 once, where the walk's top good probe carries a tier label.
func (s *fakeSampler) above(r *fakeRun) {
	if r.stopped || r.ask.Hook == (common.Address{}) || r.ask.TierA.Sign() == 0 {
		return
	}
	var top *sampleProbe
	for k := range r.w.Probes {
		if isGood(r.w.Probes[k]) {
			top = &r.w.Probes[k]
		}
	}
	if top == nil || !r.tier(top.FeeWad) {
		return
	}
	a := new(big.Int).Rsh(top.Used, aboveShift)
	a.Add(a, top.Used).Add(a, big.NewInt(1))
	if a.BitLen() > maxPointBits {
		return
	}
	if _, made := s.probe(r, a); made {
		s.tryRead(r)
	}
}

// samplerAsk is one sampler call the node answered, decoded.
type samplerAsk struct {
	sent
	pool   common.Address
	asks   []sampleAsk
	limits sampleLimits
}

// asked are the sampler calls the node answered since the from-th call it was sent.
// answered is the walks of the sampler answers the node computed, from its `from`-th on (answerCount).
func (n *node) answered(t testing.TB, from int) []sampleWalk {
	t.Helper()
	n.mu.Lock()
	answers := slices.Clone(n.answers[from:])
	n.mu.Unlock()
	var out []sampleWalk
	for _, ret := range answers {
		var a sampleAnswer
		require.NoError(t, samplerABI.UnpackIntoInterface(&a, "sample", ret))
		out = append(out, a.Walks...)
	}
	return out
}

func (n *node) answerCount() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.answers)
}

func (n *node) asked(t testing.TB, from int) []samplerAsk {
	t.Helper()
	var out []samplerAsk
	for _, c := range n.calls()[from:] {
		if !c.sampler {
			continue
		}
		m := samplerABI.Methods["sample"]
		args, err := m.Inputs.Unpack(c.data[4:])
		require.NoError(t, err)
		var in struct {
			Pool   common.Address
			Asks   []sampleAsk
			Limits sampleLimits
		}
		require.NoError(t, m.Inputs.Copy(&in, args))
		out = append(out, samplerAsk{sent: c, pool: in.Pool, asks: in.Asks, limits: in.Limits})
	}
	return out
}

// plainProbes are the previews sent to the node itself since its from-th call: the plain path's.
func plainProbes(n *node, from int) []sent {
	return lo.Filter(n.calls()[from:], func(c sent, _ int) bool {
		return bytes.HasPrefix(c.data, poolABI.Methods["previewSwap"].ID)
	})
}

func refresh(tr *PoolTracker, p entity.Pool) (entity.Pool, error) {
	return tr.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
}

func rpcError(code int, msg string) map[string]any {
	return map[string]any{"code": code, "message": msg}
}

// What the tracker does with each way a node can answer the sampler's call: the plain path where the node did not
// run it as sent, asked again once first where the answer is a revert or not the sampler's; otherwise the refresh
// fails and the last state stands. One refusal counts one, and only eight in a row set the cooldown.
func TestTrackerTakesThePlainPathOnlyWhereTheNodeDoesNotRunTheSampler(t *testing.T) {
	type want struct {
		plain bool // the refresh took the plain path, and it laddered
		err   bool
		calls int   // the sampler calls it made
		count int32 // the refusals counted after it
	}
	notRun := want{plain: true, calls: 1, count: 1}
	retried := want{plain: true, calls: 2, count: 1}
	failed := want{err: true, calls: 1}
	always := func(f fault) func(int) *fault { return func(int) *fault { return &f } }
	cases := map[string]struct {
		set  func(n *node, w *world)
		want want
	}{
		"the override dropped, the executor's own revert": {set: func(n *node, w *world) {
			n.dropOverride = true
			n.respond = func(from, to common.Address, data []byte) (bool, []byte) {
				if to == defaultTaker { // the KyberSwap executor's answer to a selector it lacks
					return false, common.FromHex("0xf902f96b")
				}
				return w.respond(from, to, data)
			}
		}, want: retried},
		"a bare revert":                               {set: faultBy(always(fault{err: rpcError(3, "execution reverted")})), want: retried},
		"a client's own revert code":                  {set: faultBy(always(fault{err: rpcError(-32000, "execution reverted")})), want: retried},
		"-32601, no such method":                      {set: faultBy(always(fault{err: rpcError(-32601, "the method eth_call does not exist/is not available")})), want: notRun},
		"-32602, no third parameter":                  {set: faultBy(always(fault{err: rpcError(-32602, "too many arguments, want at most 2")})), want: notRun},
		"-32604, not supported":                       {set: faultBy(always(fault{err: rpcError(-32604, "method not supported")})), want: notRun},
		"-32600, an invalid request":                  {set: faultBy(always(fault{err: rpcError(-32600, "invalid request")})), want: notRun},
		"-32600 with a rate limit":                    {set: faultBy(always(fault{err: rpcError(-32600, "request limit reached")})), want: failed},
		"an override refused":                         {set: faultBy(always(fault{err: rpcError(-32000, "state overrides are not supported")})), want: notRun},
		"a gas cap under the call's":                  {set: faultBy(always(fault{err: rpcError(-32000, "gas limit too high")})), want: notRun},
		"a gas allowance":                             {set: faultBy(always(fault{err: rpcError(-32000, "gas required exceeds allowance (25000000)")})), want: notRun},
		"gas limit reached, list 1 first":             {set: faultBy(always(fault{err: rpcError(-32005, "gas limit reached")})), want: notRun},
		"the call itself out of gas":                  {set: faultBy(always(fault{err: rpcError(-32000, "out of gas")})), want: failed},
		"HTTP 413":                                    {set: faultBy(always(fault{status: http.StatusRequestEntityTooLarge})), want: notRun},
		"a wrong tag":                                 {set: func(n *node, _ *world) { n.walkTag = [4]byte{1, 2, 3, 4} }, want: retried},
		"an empty answer":                             {set: faultBy(always(fault{result: json.RawMessage(`"0x"`)})), want: retried},
		"an answer that does not decode":              {set: faultBy(always(fault{result: json.RawMessage(`"0x1234"`)})), want: retried},
		"no probes, the node's gas under one probe's": {set: func(n *node, _ *world) { n.gasCap = 5_000_000 }, want: retried},
		"a null answer":                               {set: faultBy(always(fault{result: json.RawMessage(`null`)})), want: failed},
		"an answer of an older block":                 {set: func(n *node, _ *world) { n.walkBlock = testBlk - 1 }, want: failed},
		"a node error":                                {set: faultBy(always(fault{err: rpcError(-32603, "internal error")})), want: failed},
		"HTTP 502":                                    {set: faultBy(always(fault{status: http.StatusBadGateway})), want: failed},
		"batches refused":                             {set: func(n *node, _ *world) { n.refuseBatches = true }, want: want{calls: 1}},
		"a good answer":                               {want: want{calls: 1}},
	}
	for _, msg := range []string{"header not found", "unknown block", "block not found", "missing trie node 0x12"} {
		cases["the block missed: "+msg] = struct {
			set  func(n *node, w *world)
			want want
		}{set: faultBy(always(fault{err: rpcError(-32000, msg)})), want: failed}
	}
	for _, code := range rateCodes {
		cases[fmt.Sprintf("a rate limit's code %d", code)] = struct {
			set  func(n *node, w *world)
			want want
		}{set: faultBy(always(fault{err: rpcError(code, "busy")})), want: failed}
	}
	for _, words := range rateWords {
		cases["a rate limit's words: "+words] = struct {
			set  func(n *node, w *world)
			want want
		}{set: faultBy(always(fault{err: rpcError(-32000, "upstream: "+words)})), want: failed}
	}
	for _, status := range []int{http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		cases[fmt.Sprintf("HTTP %d", status)] = struct {
			set  func(n *node, w *world)
			want want
		}{set: faultBy(always(fault{status: status})), want: failed}
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			w, cfg := testWorld(), testConfig()
			n, pools := listed(t, w, cfg)
			if tc.set != nil {
				tc.set(n, w)
			}
			tr := NewPoolTracker(cfg, n.client())
			before := len(n.calls())
			_, sampledBefore := n.counts()
			p, err := refresh(tr, pools[0])
			_, sampled := n.counts()
			require.Equal(t, tc.want.calls, sampled-sampledBefore, "the sampler calls")
			require.Equal(t, tc.want.count, tr.samplerFailures.Load(), "the refusals counted")
			require.Zero(t, tr.samplerRefusedUntilMs.Load(), "one refusal sets no cooldown")
			if tc.want.err {
				require.Error(t, err)
				require.Empty(t, plainProbes(n, before), "no plain path")
				return
			}
			require.NoError(t, err)
			require.ElementsMatch(t, [][2]uint8{{0, 1}, {1, 0}}, directionsOf(extraOf(t, p)))
			if !tc.want.plain {
				require.Empty(t, plainProbes(n, before), "the sampler's path")
				return
			}
			probes := plainProbes(n, before)
			require.NotEmpty(t, probes, "the plain path")
			for _, c := range probes {
				require.Equal(t, defaultTaker, c.from)
				require.Equal(t, plainProbeGas, c.gas)
				require.Equal(t, hexutil.EncodeUint64(testBlk), c.tag)
			}
		})
	}
}

func faultBy(f func(int) *fault) func(*node, *world) { return func(n *node, _ *world) { n.fault = f } }

// The sampler's call runs under samplerTimeout, past which the refresh fails as a node's error would.
func TestTrackerGivesTheSamplerCallItsDeadline(t *testing.T) {
	t.Parallel()
	w, cfg := testWorld(), testConfig()
	n, pools := listed(t, w, cfg)
	n.fault = func(int) *fault { return &fault{delay: samplerTimeout + 500*time.Millisecond} }
	tr := NewPoolTracker(cfg, n.client())
	before := len(n.calls())
	start := time.Now()
	_, err := refresh(tr, pools[0])
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(start), samplerTimeout+400*time.Millisecond)
	require.Empty(t, plainProbes(n, before), "a timeout is no refusal")
	require.Zero(t, tr.samplerFailures.Load())
	require.Zero(t, tr.samplerRefusedUntilMs.Load())
}

// Eight refusals in a row set the cooldown: the plain path, and no sampler call, until it ends. A good answer from
// any pool resets the count; an answer with no probes, the node's gas under one probe's, counts like a refusal.
func TestTrackerCoolsOffANodeThatRefusesTheSamplerEightTimesInARow(t *testing.T) {
	w, cfg := testWorld(), testConfig()
	base := *w.pools[common.HexToAddress(poolX)]
	for k := range samplerFailuresToCooldown - 1 { // eight pools in all
		fp := base
		w.add(common.HexToAddress(fmt.Sprintf("0x%040x", 0xa2+k)), &fp)
	}
	n, pools := listed(t, w, cfg)
	require.Len(t, pools, samplerFailuresToCooldown)
	tr := NewPoolTracker(cfg, n.client())

	// seven pools whose answers have no probes, then a good one from the eighth: the count starts again
	n.gasCap = 5_000_000
	for k, p := range pools[:samplerFailuresToCooldown-1] {
		_, err := refresh(tr, p)
		require.NoError(t, err, "the plain path")
		require.Equal(t, int32(k+1), tr.samplerFailures.Load())
		require.Zero(t, tr.samplerRefusedUntilMs.Load())
	}
	n.gasCap = 0
	_, err := refresh(tr, pools[samplerFailuresToCooldown-1])
	require.NoError(t, err)
	require.Zero(t, tr.samplerFailures.Load(), "a good answer from another pool")

	// eight in a row, across the pools
	n.gasCap = 5_000_000
	for k, p := range pools {
		_, err := refresh(tr, p)
		require.NoError(t, err)
		require.Equal(t, k == len(pools)-1, tr.samplerRefusedUntilMs.Load() != 0, "after refusal %d", k+1)
	}
	until := tr.samplerRefusedUntilMs.Load()
	require.InDelta(t, time.Now().Add(samplerRetryAfter).UnixMilli(), until, 5_000)

	// while it lasts: the plain path alone
	n.gasCap = 0
	before := len(n.calls())
	_, sampled := n.counts()
	p, err := refresh(tr, pools[0])
	require.NoError(t, err)
	require.NotEmpty(t, extraOf(t, p).Ladders)
	_, after := n.counts()
	require.Equal(t, sampled, after, "no sampler call in the cooldown")
	require.NotEmpty(t, plainProbes(n, before))

	// past it: the sampler again, and its good answer resets the count
	tr.samplerRefusedUntilMs.Store(time.Now().Add(-time.Millisecond).UnixMilli())
	before = len(n.calls())
	_, err = refresh(tr, pools[0])
	require.NoError(t, err)
	_, sampledNow := n.counts()
	require.Equal(t, after+1, sampledNow)
	require.Empty(t, plainProbes(n, before))
	require.Zero(t, tr.samplerFailures.Load())
}

// A plain path that fails in a cooldown ends it, so a node that refuses batches too gets the sampler again.
func TestTrackerEndsTheCooldownWhereThePlainPathFails(t *testing.T) {
	w, cfg := testWorld(), testConfig()
	n, pools := listed(t, w, cfg)
	tr := NewPoolTracker(cfg, n.client())
	n.fault = func(int) *fault { return &fault{err: rpcError(-32601, "the method eth_call does not exist")} }
	for range samplerFailuresToCooldown {
		_, err := refresh(tr, pools[0])
		require.NoError(t, err)
	}
	require.NotZero(t, tr.samplerRefusedUntilMs.Load())
	n.refuseBatches = true
	_, err := refresh(tr, pools[0])
	require.Error(t, err)
	require.Zero(t, tr.samplerRefusedUntilMs.Load(), "the cooldown ends")

	n.fault, n.refuseBatches = nil, false
	_, sampled := n.counts()
	_, err = refresh(tr, pools[0])
	require.NoError(t, err)
	_, after := n.counts()
	require.Equal(t, sampled+1, after, "the sampler again")
}

// A load balancer that mixes in a backend without overrides, one call in six: each such call is asked again, and the
// sampler stays the main path.
func TestTrackerKeepsTheSamplerThroughOneBadBackendInSix(t *testing.T) {
	w, cfg := testWorld(), testConfig()
	n, pools := listed(t, w, cfg)
	n.fault = func(k int) *fault {
		if k%6 == 5 {
			return &fault{err: rpcError(3, "execution reverted")}
		}
		return nil
	}
	tr := NewPoolTracker(cfg, n.client())
	before := len(n.calls())
	p := pools[0]
	for range 24 {
		var err error
		p, err = refresh(tr, p)
		require.NoError(t, err)
		require.Zero(t, tr.samplerFailures.Load())
	}
	require.Empty(t, plainProbes(n, before), "never the plain path")
	require.Zero(t, tr.samplerRefusedUntilMs.Load())
}

// DisableSampler: every refresh on the plain path, no sampler call.
func TestTrackerTakesThePlainPathWhereTheConfigSaysSo(t *testing.T) {
	w, cfg := testWorld(), testConfig()
	cfg.DisableSampler = true
	n, pools := listed(t, w, cfg)
	tr := NewPoolTracker(cfg, n.client())
	before := len(n.calls())
	p, err := refresh(tr, pools[0])
	require.NoError(t, err)
	require.ElementsMatch(t, [][2]uint8{{0, 1}, {1, 0}}, directionsOf(extraOf(t, p)))
	_, sampled := n.counts()
	require.Zero(t, sampled)
	probes := plainProbes(n, before)
	require.NotEmpty(t, probes)
	byDir := lo.CountValuesBy(probes, func(c sent) string { return hexutil.Encode(c.data[4:68]) })
	for _, count := range byDir {
		require.LessOrEqual(t, count, plainSeedsPerDirection, "at most 16 sizes a direction, no bisection")
	}
}

// The pool's cap on each probe: doubled where a probe starves, up to maxProbeGas, back to the default once an answer
// spends under 7/8 of it on every probe. A starved first probe fails the refresh; at the most cap it is a miss. None
// of it counts as a refusal, and a raised cap the node's gas cannot carry goes back to the default in the same refresh.
func TestTrackerRaisesAPoolsProbeCapAndBringsItBack(t *testing.T) {
	sale := func(min int64, need uint64) func(common.Address, []byte) uint64 {
		return func(_ common.Address, data []byte) uint64 {
			m, args, ok := decode(poolABI, data)
			if ok && m.Name == "previewSwap" && args[0].(common.Address) == common.HexToAddress(cbBTC) &&
				args[2].(*big.Int).Cmp(big.NewInt(min)) >= 0 {
				return need
			}
			return fakeCallGas
		}
	}
	w, cfg := testWorld(), testConfig()
	n, pools := listed(t, w, cfg)
	tr := NewPoolTracker(cfg, n.client())
	at := func() uint64 { return tr.capOf(poolX, tr.defaultCap()) }
	lastCap := func(t *testing.T, from int) uint64 {
		a := n.asked(t, from)
		require.NotEmpty(t, a)
		return a[len(a)-1].limits.ProbeGas.Uint64()
	}
	require.Equal(t, defaultProbeGas, at())

	// a sale past 1 cbBTC needs 6M: it starves at 5M above a price, so the ladder stops under it and the cap doubles
	n.gasOf = sale(100_000_000, 6_000_000)
	p, err := refresh(tr, pools[0])
	require.NoError(t, err)
	l, _ := lo.Find(extraOf(t, p).Ladders, func(l Ladder) bool { return l.In == 0 })
	require.Negative(t, l.Points[len(l.Points)-1][0].ToBig().Cmp(big.NewInt(100_000_000)))
	require.Equal(t, 2*defaultProbeGas, at())

	// at 10M it prices, and 6M spent keeps the cap up
	before := len(n.calls())
	p, err = refresh(tr, p)
	require.NoError(t, err)
	require.Equal(t, 2*defaultProbeGas, lastCap(t, before))
	l, _ = lo.Find(extraOf(t, p).Ladders, func(l Ladder) bool { return l.In == 0 })
	require.Positive(t, l.Points[len(l.Points)-1][0].ToBig().Cmp(big.NewInt(100_000_000)))
	require.Equal(t, 2*defaultProbeGas, at())

	// a node whose gas cannot carry the raised cap: no probes there, the default again in the same refresh
	n.gasCap = 10_000_000
	for range 2 * samplerFailuresToCooldown {
		_, sampled := n.counts()
		_, err = refresh(tr, p)
		require.NoError(t, err)
		_, after := n.counts()
		require.Equal(t, sampled+2, after, "asked again at the default")
	}
	require.Zero(t, tr.samplerFailures.Load())
	require.Zero(t, tr.samplerRefusedUntilMs.Load())
	n.gasCap = 0

	// cheap again: back to the default, and the next call asks at it
	n.gasOf = nil
	p, err = refresh(tr, p)
	require.NoError(t, err)
	require.Equal(t, defaultProbeGas, at())
	before = len(n.calls())
	_, err = refresh(tr, p)
	require.NoError(t, err)
	require.Equal(t, defaultProbeGas, lastCap(t, before))

	// every sale needs 6M: its first probe starves, the refresh fails and the cap doubles; at 10M it prices
	n.gasOf = sale(0, 6_000_000)
	_, err = refresh(tr, p)
	require.ErrorIs(t, err, ErrRead)
	require.Equal(t, 2*defaultProbeGas, at())
	_, err = refresh(tr, p)
	require.NoError(t, err)

	// past the most cap: a miss on every refresh, the cap held at maxProbeGas
	n.gasOf = sale(0, 30_000_000)
	for range 4 {
		_, err = refresh(tr, p)
		require.ErrorIs(t, err, ErrRead)
	}
	require.Equal(t, maxProbeGas, at())
	require.Zero(t, tr.samplerFailures.Load())
	require.Zero(t, tr.samplerRefusedUntilMs.Load())
}

// venusGas is what a Venus Core pool's previews spend, as measured on a BSC fork: the first of each input cold
// (1,060,248), the rest warm (596,748), and a sale from `borrowFrom` up, which borrows, at the most measured
// (1,514,071). A full walk of both directions then nears the sampler call's gas.
func venusGas(borrowFrom int64) func(common.Address, []byte) uint64 {
	cold := map[common.Address]bool{}
	return func(_ common.Address, data []byte) uint64 {
		m, args, ok := decode(poolABI, data)
		if !ok || m.Name != "previewSwap" {
			return fakeCallGas
		}
		in := args[0].(common.Address)
		switch {
		case in == common.HexToAddress(cbBTC) && args[2].(*big.Int).Cmp(big.NewInt(borrowFrom)) >= 0:
			return 1_514_071
		case !cold[in]:
			cold[in] = true
			return 1_060_248
		}
		return 596_748
	}
}

// A Venus Core pool on BSC, its previews at their measured gas. No preview starves on its own (each spends under
// 7/8 of the cap), so the cap stays; the walks reach the gas gate first, which stops them, and the ladders are
// shorter, each point still the pool's own quote, with no strike against the node. A Venus read that starves on its
// own (ReadStarved) past a price ends the ladder under it and raises the cap; on a direction's first probe it fails
// the refresh, and the pool's state ages out.
func TestTrackerWalksAVenusPoolToTheGasGate(t *testing.T) {
	w, cfg := testWorld(), testConfig()
	cfg.ChainID = valueobject.ChainIDBSC
	fp := w.pools[common.HexToAddress(poolX)]
	fp.venues = []venueView{venue(2, 0, 1, 0)}
	sale, buy := oracle(1_000, 1, 5, 1_000_000_000_000), oracle(1, 1_000, 5, 100_000_000_000_000)
	fp.quote = func(from common.Address, i, j int, a *big.Int) (*big.Int, *big.Int, bool) { // deep both ways
		switch {
		case from != defaultTaker:
			return nil, nil, false
		case i == 0 && j == 1:
			return sale(from, i, j, a)
		case i == 1 && j == 0:
			return buy(from, i, j, a)
		}
		return nil, nil, false
	}
	n, pools := listed(t, w, cfg)
	cheap, err := track(t, n, cfg, pools[0])
	require.NoError(t, err)
	points := func(p entity.Pool) map[uint8]int {
		out := map[uint8]int{}
		for _, l := range extraOf(t, p).Ladders {
			out[l.In] = len(l.Points)
		}
		return out
	}

	tr := NewPoolTracker(cfg, n.client())
	n.gasOf = venusGas(100_000_000) // a sale past 1 cbBTC borrows
	before := n.answerCount()
	p, err := refresh(tr, pools[0])
	require.NoError(t, err)
	walks := n.answered(t, before)
	for _, w := range walks {
		var g int64
		for _, pr := range w.Probes {
			g += pr.GasUsed.Int64()
		}
		t.Logf("a walk of %d probes, %d gas, truncated %v", len(w.Probes), g, w.Truncated)
	}
	require.True(t, lo.SomeBy(walks, func(w sampleWalk) bool { return w.Truncated }), "the gate stops a walk")
	require.Equal(t, defaultProbeGas, tr.capOf(poolX, tr.defaultCap()), "no probe starved: the cap stays")
	require.Zero(t, tr.samplerFailures.Load(), "no strike")
	e := extraOf(t, p)
	require.ElementsMatch(t, [][2]uint8{{0, 1}, {1, 0}}, directionsOf(e))
	full, cut := points(cheap), points(p)
	require.Less(t, cut[0]+cut[1], full[0]+full[1], "coarser walks")
	for _, l := range e.Ladders {
		require.Equal(t, testCells(testHookFamily, l.In == 0, 2), l.Cells)
		for _, pt := range l.Points {
			used, out, ok := fp.quote(defaultTaker, int(l.In), int(l.Out), pt[0].ToBig())
			require.True(t, ok)
			require.Equal(t, used, pt[0].ToBig())
			require.Equal(t, out, pt[1].ToBig())
		}
	}
	t.Logf("points by input: %v at the default gas, %v at Venus's", full, cut)

	// a Venus read starving on its own past 1 cbBTC, then at every sale
	starve := func(from int64) {
		n.respond = func(from_, to common.Address, data []byte) (bool, []byte) {
			if m, args, ok := decode(poolABI, data); ok && m.Name == "previewSwap" &&
				args[0].(common.Address) == common.HexToAddress(cbBTC) && args[2].(*big.Int).Cmp(big.NewInt(from)) >= 0 {
				return false, starvedReverts[0]
			}
			return w.respond(from_, to, data)
		}
	}
	n.gasOf = nil
	starve(100_000_000)
	p, err = refresh(tr, p)
	require.NoError(t, err)
	l, ok := lo.Find(extraOf(t, p).Ladders, func(l Ladder) bool { return l.In == 0 })
	require.True(t, ok)
	require.Negative(t, l.Points[len(l.Points)-1][0].ToBig().Cmp(big.NewInt(100_000_000)), "the ladder ends under it")
	require.Equal(t, 2*defaultProbeGas, tr.capOf(poolX, tr.defaultCap()), "the cap raised")
	starve(0)
	_, err = refresh(tr, p)
	require.ErrorIs(t, err, ErrRead, "a first probe starved: no state")
}

// What a pool does comes back as walks with the sampler's tag, so no pool counts towards the cooldown, however it
// fails: every size refused, every answer unreadable, its request reverting, or its probes past the most cap.
func TestTrackerCountsNoPoolErrorAsARefusal(t *testing.T) {
	for name, edit := range map[string]func(w *world, fp *fakePool, n *node){
		"every size refused": func(_ *world, fp *fakePool, _ *node) {
			fp.quote = func(common.Address, int, int, *big.Int) (*big.Int, *big.Int, bool) { return nil, nil, false }
		},
		"every answer unreadable": func(w *world, _ *fakePool, n *node) {
			n.respond = func(from, to common.Address, data []byte) (bool, []byte) {
				if m, _, ok := decode(poolABI, data); ok && m.Name == "previewSwap" {
					return true, []byte{1}
				}
				return w.respond(from, to, data)
			}
		},
		"probes past the most cap": func(_ *world, _ *fakePool, n *node) {
			n.gasOf = func(common.Address, []byte) uint64 { return 2 * maxProbeGas }
		},
	} {
		t.Run(name, func(t *testing.T) {
			w, cfg := testWorld(), testConfig()
			n, pools := listed(t, w, cfg)
			edit(w, w.pools[common.HexToAddress(poolX)], n)
			tr := NewPoolTracker(cfg, n.client())
			for range samplerFailuresToCooldown + 2 {
				_, _ = refresh(tr, pools[0])
			}
			require.Zero(t, tr.samplerFailures.Load())
			require.Zero(t, tr.samplerRefusedUntilMs.Load())
		})
	}
}

// The requests a refresh sends, sequentially: at a pool's first refresh the gate's multicall, the front check, the
// pricer's code and its params(), the Router's venues and the sampler; after it, the code and row are cached.
func TestTrackerCountsItsRequestsPerRefresh(t *testing.T) {
	curve := func() (*world, *fakePool) { w := testWorld(); return w, w.pools[common.HexToAddress(poolX)] }
	for name, tc := range map[string]struct {
		world       func() (*world, *fakePool)
		disable     bool
		first, warm int
		why         string
	}{
		"the sampler": {world: curve, first: 7, warm: 5,
			why: "the gate, the front check, the code and params() at first, the venues, their accounts, the sampler"},
		"the sampler, tier-labelled": {world: tierWorld, first: 7, warm: 5,
			why: "the gate, the front check, the code and params() at first, the venues, their accounts, the sampler"},
		"the plain path": {world: curve, disable: true, first: 7, warm: 5,
			why: "the gate, the front check, the code and params() at first, the venues, their accounts, the probes " +
				"in one batch"},
		"the plain path, tier-labelled": {world: tierWorld, disable: true, first: 9, warm: 7,
			why: "the gate, the front check, the code and params() at first, the venues, their accounts, the probes " +
				"in one batch, the request reads and the probes above in a second, levState on the buy in a third"},
	} {
		t.Run(name, func(t *testing.T) {
			w, fp := tc.world()
			cfg := testConfig()
			cfg.DisableSampler = tc.disable
			router := common.HexToAddress("0x00000000000000000000000000000000000000e9")
			w.routers[router], fp.router, fp.venues = true, router, []venueView{venue(0, 0, 1, 0)}
			n, pools := listed(t, w, cfg)
			tr := NewPoolTracker(cfg, n.client())
			start, _ := n.counts()
			p, err := refresh(tr, pools[0])
			require.NoError(t, err)
			first, sampledFirst := n.counts()
			_, err = refresh(tr, p)
			require.NoError(t, err)
			warm, sampledWarm := n.counts()
			t.Logf("requests: %d at the first refresh, %d warm, %d of them the sampler's", first-start, warm-first,
				sampledWarm)
			require.Equal(t, tc.first, first-start, tc.why)
			require.Equal(t, tc.warm, warm-first, tc.why)
			require.Equal(t, lo.Ternary(tc.disable, 0, 1), sampledFirst, "one sampler call a refresh, none on the plain path")
			require.Equal(t, lo.Ternary(tc.disable, 0, 2), sampledWarm)
		})
	}
}

// The caller's overrides reach every read, and the sampler's code takes the taker's place over any of theirs there.
func TestTrackerPassesTheCallersOverridesToEveryRead(t *testing.T) {
	for name, disable := range map[string]bool{"the sampler": false, "the plain path": true} {
		t.Run(name, func(t *testing.T) {
			w, cfg := testWorld(), testConfig()
			cfg.DisableSampler = disable
			router := common.HexToAddress("0x00000000000000000000000000000000000000e9")
			fp := w.pools[common.HexToAddress(poolX)]
			w.routers[router], fp.router, fp.venues = true, router, []venueView{venue(0, 0, 1, 0)}
			n, pools := listed(t, w, cfg)
			theirs := common.HexToAddress("0x00000000000000000000000000000000000000cc")
			overrides := map[common.Address]gethclient.OverrideAccount{theirs: {Code: []byte{0x60, 0x00}},
				defaultTaker: {Code: []byte{0x60, 0x01}}}
			before := len(n.calls())
			p, err := NewPoolTracker(cfg, n.client()).GetNewPoolStateWithOverrides(context.Background(), pools[0],
				pool.GetNewPoolStateWithOverridesParams{Overrides: overrides})
			require.NoError(t, err)
			require.ElementsMatch(t, [][2]uint8{{0, 1}, {1, 0}}, directionsOf(extraOf(t, p)))
			var calls, samplerCalls int
			for _, c := range n.calls()[before:] {
				switch {
				case len(c.data) == 0: // a code read: cached by the tracker, so the caller's overrides stay out of it
					continue
				case bytes.HasPrefix(c.data, hookABI.Methods["params"].ID):
					require.NotContains(t, c.ov, theirs, "the row is cached the same way")
					continue
				}
				calls++
				require.Contains(t, c.ov, theirs, "a read without the caller's overrides: %x", c.data[:4])
				if c.sampler {
					samplerCalls++
					require.Equal(t, hexutil.Bytes(samplerRuntime), *c.ov[defaultTaker].Code, "ours at the taker")
				}
			}
			require.Positive(t, calls)
			require.Equal(t, lo.Ternary(disable, 0, 1), samplerCalls)
		})
	}
}

func TestConfigFloorsTheSamplerGas(t *testing.T) {
	require.Equal(t, defaultSampleGas, (&Config{}).sampleGas())
	require.Equal(t, uint64(minSampleGas), (&Config{SampleGas: 1}).sampleGas())
	require.Equal(t, uint64(60_000_000), (&Config{SampleGas: 60_000_000}).sampleGas())
	require.Equal(t, uint64(20_717_460), uint64(minSampleGas), "a probe at maxProbeGas passes the gas gate")
	tr := NewPoolTracker(&Config{SampleGas: 1}, nil)
	require.Equal(t, maxProbeGas, tr.maxCap())
	require.Equal(t, defaultProbeGas, tr.defaultCap())
	require.Equal(t, uint64(10_000_000), NewPoolTracker(&Config{ProbeGas: 10_000_000}, nil).defaultCap())
}
