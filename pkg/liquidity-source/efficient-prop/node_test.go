package efficientprop

import (
	"bytes"
	"encoding/json"
	"errors"
	"maps"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	abiutil "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
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

// sent is one call the node answered (a multicall's calls each), with its block tag, whether the eth_call carrying
// it stated a gas price, the block time and number it ran at (the head's, or a block override's), the sub-second
// BaseTime stores in it, where a state override set it (-1: none), and the call's state override (nil: none).
type sent struct {
	to       common.Address
	data     []byte
	tag      string
	gasPrice bool
	at       uint64
	number   uint64
	sub      int64
	state    stateDiffs
}

// stateDiffs is an eth_call's state override, each account's stateDiff.
type stateDiffs map[common.Address]map[common.Hash]common.Hash

// node is a chain at one block behind JSON-RPC: single and batched requests, eth_call (Multicall3's aggregates
// answered call by call, at the head's time and number or a block override's), eth_getCode, eth_getStorageAt and
// eth_blockNumber. respond answers a call at a block time and number, with BaseTime's stored sub-second set by the
// call's state override (sub, -1 none; ok false: it reverts), code a code read and storage a storage read.
type node struct {
	*httptest.Server
	t       testing.TB
	block   uint64
	time    uint64
	respond func(to common.Address, data []byte, at, number uint64, sub int64) (bool, []byte)
	code    func(common.Address) []byte
	storage func(common.Address, common.Hash) common.Hash
	// refuseOverrides: the node refuses an eth_call with a block override, as a node without them does
	refuseOverrides bool
	// ignoreOverrides: the node takes an eth_call with a block override and answers at the head's time and number
	ignoreOverrides bool
	// ignoreNumber: the node honours a block override's time but answers at the head's number
	ignoreNumber bool
	// ignoreOverridesIf, where set, makes the node take and ignore the block override of an eth_call whose data it
	// answers true for, as ignoreOverrides does for every call
	ignoreOverridesIf func(data []byte) bool
	// refuseOverridesAfter, where set, refuses an eth_call whose block override's time is after it
	refuseOverridesAfter uint64
	// refuseNumber, where set, refuses an eth_call whose block override names that block number
	refuseNumber uint64
	// refuseSlot, where set, is a storage slot whose read the node answers with an error
	refuseSlot common.Hash
	// ignoreStateOverrides: the node takes an eth_call with a state override and answers on the pinned state
	ignoreStateOverrides bool
	// baseTime is the chain's millisecond-time contract, whose slot 0 a state override sets
	baseTime common.Address
	// respondState, where set, answers every call in place of respond, with the call's state override (nil: none)
	respondState func(to common.Address, data []byte, at, number uint64, sub int64, state stateDiffs) (bool, []byte)

	mu   sync.Mutex
	sent []sent
	// multis is how many calls each multicall the node answered held
	multis []int
	// reorg salts the head's hash: a node whose head at the same height is another block answers another hash
	reorg uint64
	// noBlocks: the node answers no block by number
	noBlocks bool
	// requests is how many JSON-RPC requests of each method the node answered, a batch's each counted
	requests map[string]int
}

// requestCounts is a copy of how many requests of each method the node has answered.
func (n *node) requestCounts() map[string]int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return maps.Clone(n.requests)
}

type rpcMessage struct {
	ID     json.RawMessage   `json:"id"`
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

type callArgs struct {
	To       common.Address  `json:"to"`
	Data     *hexutil.Bytes  `json:"data"`
	Input    *hexutil.Bytes  `json:"input"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	MaxFee   *hexutil.Big    `json:"maxFeePerGas"`
	MaxTip   *hexutil.Big    `json:"maxPriorityFeePerGas"`
	Gas      *hexutil.Uint64 `json:"gas"`
}

func newNode(t testing.TB, block uint64, respond func(common.Address, []byte, uint64, uint64, int64) (bool, []byte),
	code func(common.Address) []byte, storage func(common.Address, common.Hash) common.Hash) *node {
	n := &node{t: t, block: block, time: uint64(time.Now().Unix()), respond: respond, code: code, storage: storage}
	n.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("a request the node cannot read: %v", err)
			return
		}
		if bytes.HasPrefix(bytes.TrimSpace(body), []byte("[")) {
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
		_ = json.NewEncoder(w).Encode(n.answer(m))
	}))
	t.Cleanup(n.Close)
	return n
}

func (n *node) client() *ethrpc.Client {
	return ethrpc.New(n.URL).SetMulticallContract(multicall3)
}

func (n *node) calls() []sent {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]sent(nil), n.sent...)
}

func (n *node) keep(s sent) {
	n.mu.Lock()
	n.sent = append(n.sent, s)
	n.mu.Unlock()
}

func (n *node) answer(m rpcMessage) map[string]any {
	reply := map[string]any{"jsonrpc": "2.0", "id": m.ID}
	n.mu.Lock()
	if n.requests == nil {
		n.requests = map[string]int{}
	}
	n.requests[m.Method]++
	n.mu.Unlock()
	fail := func(err error) map[string]any {
		n.t.Errorf("%s: %v", m.Method, err)
		reply["error"] = map[string]any{"code": -32602, "message": err.Error()}
		return reply
	}
	tag := "latest"
	if len(m.Params) > 1 && m.Method != "eth_getBlockByNumber" {
		if err := json.Unmarshal(m.Params[1], &tag); err != nil {
			return fail(err)
		}
	}
	switch m.Method {
	case "eth_blockNumber":
		reply["result"] = hexutil.EncodeUint64(n.block)
	case "eth_getBlockByNumber":
		n.mu.Lock()
		salt, none := n.reorg, n.noBlocks
		n.mu.Unlock()
		if none {
			reply["error"] = map[string]any{"code": -32000, "message": "no block"}
			return reply
		}
		reply["result"] = map[string]any{"number": hexutil.EncodeUint64(n.block),
			"hash": crypto.Keccak256Hash(common.BigToHash(new(big.Int).SetUint64(n.block)).Bytes(),
				common.BigToHash(new(big.Int).SetUint64(salt)).Bytes())}
	case "eth_getCode":
		var a common.Address
		if err := json.Unmarshal(m.Params[0], &a); err != nil {
			return fail(err)
		}
		n.keep(sent{to: a, tag: tag, sub: -1})
		reply["result"] = hexutil.Encode(n.code(a))
	case "eth_getStorageAt":
		var a common.Address
		var slot common.Hash
		if len(m.Params) < 3 {
			return fail(errors.New("a storage read with no block"))
		}
		if err := json.Unmarshal(m.Params[0], &a); err != nil {
			return fail(err)
		}
		if err := json.Unmarshal(m.Params[1], &slot); err != nil {
			return fail(err)
		}
		if err := json.Unmarshal(m.Params[2], &tag); err != nil {
			return fail(err)
		}
		n.keep(sent{to: a, data: slot[:], tag: tag, sub: -1})
		if n.refuseSlot != (common.Hash{}) && slot == n.refuseSlot {
			reply["error"] = map[string]any{"code": -32000, "message": "missing trie node"}
			return reply
		}
		var v common.Hash
		if n.storage != nil {
			v = n.storage(a, slot)
		}
		reply["result"] = v
	case "eth_call":
		var c callArgs
		if err := json.Unmarshal(m.Params[0], &c); err != nil {
			return fail(err)
		}
		data := c.Data
		if data == nil {
			data = c.Input
		}
		if data == nil {
			return fail(errors.New("an eth_call with no data"))
		}
		priced := c.GasPrice != nil && c.GasPrice.ToInt().Sign() != 0 ||
			c.MaxFee != nil && c.MaxFee.ToInt().Sign() != 0 || c.MaxTip != nil && c.MaxTip.ToInt().Sign() != 0
		at, number, sub := n.time, n.block, int64(-1)
		var state stateDiffs
		if len(m.Params) > 2 && !n.ignoreStateOverrides {
			// a state override of BaseTime's slot 0, the sub-second it adds to the block's second
			var so map[common.Address]struct {
				StateDiff map[common.Hash]common.Hash `json:"stateDiff"`
			}
			if err := json.Unmarshal(m.Params[2], &so); err != nil {
				return fail(err)
			}
			if v, ok := so[n.baseTime].StateDiff[common.Hash{}]; ok {
				sub = v.Big().Int64()
			}
			state = stateDiffs{}
			for a, acct := range so {
				state[a] = acct.StateDiff
			}
		}
		if len(m.Params) > 3 {
			if n.refuseOverrides {
				reply["error"] = map[string]any{"code": -32602, "message": "block overrides are not supported"}
				return reply
			}
			var o struct {
				Time   *hexutil.Uint64 `json:"time"`
				Number *hexutil.Big    `json:"number"`
			}
			if err := json.Unmarshal(m.Params[3], &o); err != nil {
				return fail(err)
			}
			if o.Time != nil && n.refuseOverridesAfter != 0 && uint64(*o.Time) > n.refuseOverridesAfter {
				reply["error"] = map[string]any{"code": -32000, "message": "header not found"}
				return reply
			}
			if o.Number != nil && n.refuseNumber != 0 && o.Number.ToInt().Uint64() == n.refuseNumber {
				reply["error"] = map[string]any{"code": -32000, "message": "header not found"}
				return reply
			}
			ignore := n.ignoreOverrides || n.ignoreOverridesIf != nil && n.ignoreOverridesIf(*data)
			if o.Time != nil && !ignore {
				at = uint64(*o.Time)
			}
			if o.Number != nil && !ignore && !n.ignoreNumber {
				number = o.Number.ToInt().Uint64()
			}
		}
		if c.To != multicall3 {
			n.keep(sent{to: c.To, data: *data, tag: tag, gasPrice: priced, at: at, number: number, sub: sub,
				state: state})
			ok, ret := n.call(c.To, *data, at, number, sub, state)
			if !ok {
				reply["error"] = map[string]any{"code": 3, "message": "execution reverted", "data": hexutil.Encode(ret)}
				return reply
			}
			reply["result"] = hexutil.Encode(ret)
			return reply
		}
		ret, err := n.multicall(*data, tag, priced, at, number, sub, state)
		if err != nil {
			return fail(err)
		}
		reply["result"] = hexutil.Encode(ret)
	default:
		return fail(errors.New("unsupported method"))
	}
	return reply
}

// call answers one call, by respondState where set, else by respond.
func (n *node) call(to common.Address, data []byte, at, number uint64, sub int64, state stateDiffs) (bool, []byte) {
	if n.respondState != nil {
		return n.respondState(to, data, at, number, sub, state)
	}
	return n.respond(to, data, at, number, sub)
}

func (n *node) multicall(data []byte, tag string, priced bool, at, number uint64, sub int64,
	state stateDiffs) ([]byte, error) {
	method, err := multicallABI.MethodById(data[:4])
	if err != nil {
		return nil, err
	}
	args, err := method.Inputs.Unpack(data[4:])
	if err != nil {
		return nil, err
	}
	calls := *abi.ConvertType(args[len(args)-1], new([]innerCall)).(*[]innerCall)
	results := make([]innerResult, len(calls))
	n.mu.Lock()
	n.multis = append(n.multis, len(calls))
	n.mu.Unlock()
	for k, c := range calls {
		n.keep(sent{to: c.Target, data: c.CallData, tag: tag, gasPrice: priced, at: at, number: number, sub: sub,
			state: state})
		ok, ret := n.call(c.Target, c.CallData, at, number, sub, state)
		results[k] = innerResult{Success: ok, ReturnData: ret}
	}
	block := new(big.Int).SetUint64(n.block)
	switch method.Name {
	case "aggregate":
		rets := make([][]byte, len(results))
		for k, r := range results {
			if !r.Success {
				return nil, errors.New("aggregate: a call reverted")
			}
			rets[k] = r.ReturnData
		}
		return method.Outputs.Pack(block, rets)
	case "tryAggregate":
		return method.Outputs.Pack(results)
	default:
		return method.Outputs.Pack(block, common.Hash{}, results)
	}
}

// ---- a fake front, door factory and listing doors behind the node

// fakeRung is one level of a fake listing's side: the input it fills up to (cumulative) and its price, output per
// input as num/den, so the side's output is concave in its input as the front's walk is.
type fakeRung struct {
	upTo     int64
	num, den int64
}

// fakeSide is one direction of a fake listing: its levels, the least output it fills (under it: a revert), and
// whether it is dark (every quote answers NoFill).
type fakeSide struct {
	rungs  []fakeRung
	minOut int64
	dark   bool
}

// quote walks the side like the front: all or nothing past its last level, the output floored once.
func (s fakeSide) quote(amountIn *big.Int) (*big.Int, error) {
	return s.quoteKeeping(amountIn, 1_000_000)
}

// quoteKeeping is quote with every level's price cut to keepPpm of it, the output still floored once, as the front
// prices its levels after its discount.
func (s fakeSide) quoteKeeping(amountIn *big.Int, keepPpm int64) (*big.Int, error) {
	if s.dark || len(s.rungs) == 0 || amountIn.Sign() <= 0 {
		return nil, errNoFill
	}
	if amountIn.Cmp(big.NewInt(s.rungs[len(s.rungs)-1].upTo)) > 0 {
		return nil, errNoFill
	}
	// sum over levels of t * num / den, as one fraction over the product of the dens, floored once
	num := new(big.Rat)
	var prev int64
	left := new(big.Int).Set(amountIn)
	for _, r := range s.rungs {
		if left.Sign() == 0 {
			break
		}
		t := new(big.Int).SetInt64(r.upTo - prev)
		if t.Cmp(left) > 0 {
			t.Set(left)
		}
		num.Add(num, new(big.Rat).SetFrac(new(big.Int).Mul(t, big.NewInt(r.num)), big.NewInt(r.den)))
		left.Sub(left, t)
		prev = r.upTo
	}
	num.Mul(num, big.NewRat(max(keepPpm, 0), 1_000_000))
	out := new(big.Int).Quo(num.Num(), num.Denom())
	if out.Sign() == 0 {
		return nil, errNoFill
	}
	if out.Cmp(big.NewInt(s.minOut)) < 0 {
		return nil, errBelowMinimum
	}
	return out, nil
}

var (
	errNoFill       = errors.New("NoFill")
	errBelowMinimum = errors.New("BelowMinimum")
)

type fakeListing struct {
	view  DoorView
	sides [2]fakeSide // 0: token0 in, 1: token1 in
	open  bool        // its door has been opened
	// wrongFront, where set, is the front the door's immutable names instead
	wrongFront common.Address
	// wrongListing, where set, is the listing id the door's immutable names instead
	wrongListing uint32
	// decayPpmPerS lowers every output by that many ppm a second of the word's age (its maker time to the age clock,
	// ms, the cut rounded up), as a listing whose levels drift or widen with age pays less as its word ages
	decayPpmPerS int64
	// controls is the listing's controls word as the front stores it (record word 7); its premium (bits 156-175, ppm)
	// lowers every output while the clock is under the word's maker time plus its window (bits 176-191, ms)
	controls *big.Int
	// from block time shrinkAt on, where set, the listing fills no input above shrinkTo
	shrinkAt uint64
	shrinkTo int64
	// printMaxAgeMs, where set, is the max age of the price print the listing reads, made at printMs: once the age
	// clock passes printMs + printMaxAgeMs the door refuses every size (NoFill: the front answers 0 to a stale print)
	printMs, printMaxAgeMs uint64
	// gapFromMs and gapToMs, where set, are a stretch of the age clock in which the door refuses every size, filling
	// again after it
	gapFromMs, gapToMs uint64
	// session is the listing's session word as the front stores it (record word 13): where the listing is bound
	// SESSION and its closedAfter (bits 0-15, s) is nonzero, the door fills again once the print is at least that
	// old, the listing's closed phase
	session *big.Int
	// sourceIds is the listing's bound source ids as the front stores them (record word 10): the primary at bits 0-31,
	// the backup at 32-63
	sourceIds *big.Int
	// price, where set, is the listing's record word 2, the live source's copy a fill reads; where unset the front
	// stores the primary's: its address (bits 0-159), window (236-243) and guard (247)
	price *big.Int
	// outCap, where set, is the most a direction pays out: the door refuses a size whose output is above it, as a side
	// bound by its desk's bucket or its pool's room does
	outCap [2]int64
	// fillableUnderCap: the door's quoteFillable answers the largest prefix whose output is within outCap, as the
	// front's does, so the side's top pays its output bound at the block; else it refuses past the bound
	fillableUnderCap bool
	// heldUntilMs and heldCutPpm, where set, are a hold on the listing's guarded source: while the front's clock is
	// under heldUntilMs a direction is priced heldCutPpm ppm lower (the side's worst held print), and once it is not,
	// on the kept print alone
	heldUntilMs uint64
	heldCutPpm  [2]int64
	// carryBit, where set, is the word's carry bit (4-7) a builder chain's doorView sets for a carried update; bit 4
	// where unset
	carryBit int
	// route, where a direction's is set, is that side's credit route (a listing with a credit mode)
	route [2]*fakeRoute
	// boards, where set, are the listing's board words as the front stores them (record words 8 and 9): the fill
	// memory of the side with the base in, and of the side with the quote in
	boards [2]*big.Int
	// settings, where set, is the listing's settings word as the front stores it (record word 1): its minimum fill in
	// bits 160-183
	settings *big.Int
	// minIn, where a direction's is set, is the least input the door fills of it: the minimum fill of the side with
	// the quote in, which the front counts on the input
	minIn [2]int64
}

// fakeRoute is a credit-mode listing's side past its pool's custody: a size above the custody's cut fills through the
// pool's credit route, which the front's plan tags ROUTED with the side's walk, and which the door's quote (the pool's
// preview) pays less feePpm, refusing every size above routeTo where set (the pool's caps). From block number laterAt
// on, where set, the custody fills up to laterTo instead.
type fakeRoute struct {
	custodyTo, routeTo int64
	feePpm             int64
	laterAt            uint64
	laterTo            int64
	// planWrong: the plan answers a custody size's output plus one, untagged, which no front state gives
	planWrong bool
	// planReverts: the front's plan call reverts
	planReverts bool
	// planCapOut, where set, is the output token's cap the plan holds a route to (PLAN_ONLY's credit re-walk): it
	// answers 0 for a routed size whose output is above it, which the door's preview still pays
	planCapOut *big.Int
}

// routes is whether the route fills amountIn in block number: it is above the custody's cut there.
func (r *fakeRoute) routes(amountIn *big.Int, number uint64) bool {
	if r == nil {
		return false
	}
	cut := r.custodyTo
	if r.laterAt != 0 && number >= r.laterAt {
		cut = r.laterTo
	}
	return amountIn.Cmp(big.NewInt(cut)) > 0
}

// routedTagTest is the plan answer's tag of a routed size (ROUTED).
var routedTagTest = new(big.Int).Lsh(big.NewInt(1), 255)

// planAt is the front's tip-taking quoteFrom of listing l with PLAN_ONLY, direction dir, in a block of second at and
// number on the head's state: a routed size's walk with custody lifted (no pool fee) tagged ROUTED, a custody size the
// door's answer untagged, 0 for a refusal of the block's state.
func (c *fakeChain) planAt(l *fakeListing, dir int, amountIn *big.Int, at, number uint64, sub int64) (bool, []byte) {
	r := l.route[dir]
	if r != nil && r.planReverts {
		return false, crypto.Keccak256([]byte("PlanReverts()"))[:4]
	}
	if r.routes(amountIn, number) {
		nowMs, ageNow := c.clockAtSub(at, number, sub)
		out, err := l.sides[dir].quoteKeeping(amountIn, 1_000_000-l.cutAt(dir, nowMs, ageNow))
		if l.stale(ageNow) || err != nil || r.planCapOut != nil && out.Cmp(r.planCapOut) > 0 {
			return true, packOut(frontABI, "quoteFrom", new(big.Int))
		}
		return true, packOut(frontABI, "quoteFrom", new(big.Int).Or(out, routedTagTest))
	}
	out, err := c.quoteAtSub(l, dir, amountIn, at, number, sub)
	switch {
	case errors.Is(err, errBelowMinimum):
		return false, crypto.Keccak256([]byte("BelowMinimum()"))[:4]
	case err != nil:
		return true, packOut(frontABI, "quoteFrom", new(big.Int))
	case r != nil && r.planWrong:
		out = new(big.Int).Add(out, big.NewInt(1))
	}
	return true, packOut(frontABI, "quoteFrom", out)
}

// stale is whether the door refuses every size at the age clock ageNow: the listing's print is past its max age, or
// the clock is in its gap.
func (l *fakeListing) stale(ageNow uint64) bool {
	closed := (l.view.Scopes>>1)&3 == 3 && bits(l.session, 0, 16) != 0 && ageNow >= l.printMs+bits(l.session, 0, 16)*1000
	return l.printMaxAgeMs != 0 && ageNow > l.printMs+l.printMaxAgeMs && !closed ||
		ageNow >= l.gapFromMs && ageNow < l.gapToMs
}

// cutAt is direction dir's discount, ppm, at the clock nowMs and the age clock ageNow: its age's decay and its
// premium, which runs from the latest of the word's maker time, the side's depth nonce (its depth word 0, bits 12-59)
// and the controls' nonce (bits 16-63), as the front starts it.
func (l *fakeListing) cutAt(dir int, nowMs, ageNow uint64) int64 {
	makerMs := bits(l.view.Word, 72, 48)
	cut := int64(0)
	if ageNow > makerMs {
		cut += (l.decayPpmPerS*int64(ageNow-makerMs) + 999) / 1000
	}
	// a fill selling the pair's base takes the bid, depth[0]; one buying it the ask, depth[2]
	depth := l.view.Depth[2]
	if (dir == 0) == (l.view.PairWord != nil && l.view.PairWord.Bit(8) == 1) {
		depth = l.view.Depth[0]
	}
	start := max(makerMs, bits(depth, 12, 48), bits(l.controls, 16, 48))
	if l.controls != nil && nowMs < start+bits(l.controls, 176, 16) {
		cut += int64(bits(l.controls, 156, 20))
	}
	if nowMs < l.heldUntilMs {
		cut += l.heldCutPpm[dir]
	}
	return cut
}

type fakeChain struct {
	front, factory common.Address
	listings       map[uint32]*fakeListing
	balances       map[[2]common.Address]*big.Int // token, holder
	// factoryFront, where set, is what the factory answers as its front
	factoryFront common.Address
	// factoryCodehash, where set, is what the factory answers as its doors' creation code hash
	factoryCodehash common.Hash
	// noClock: the front does not answer clock()
	noClock bool
	// cells is each desk's contract (deskCellOf) where it is not the CREATE2 child of the front under testCellHash
	cells map[uint32]common.Address
	// deskWords is each desk contract's slot 1 at the head, the desk word (none: zero, epoch 0 and not stopped)
	deskWords map[uint32]*big.Int
	// headSlots are slots of accounts at the head the fake holds, as the chain's own storage (a desk contract's quote
	// word)
	headSlots map[common.Address]map[common.Hash]common.Hash
	// doorOfs, where an id is set, is what the factory's doorOf answers for it in place of its CREATE2 door
	doorOfs map[uint32]common.Address
	// noDoorHash: the factory's DOOR_CODEHASH() reverts
	noDoorHash bool
	// cellHash, where set, is what the front answers as CELL_CODEHASH(); testCellHash where unset
	cellHash common.Hash
	// sources is each registered price source's entry as the front stores it, by id: its address at bits 0-159
	sources map[uint32]*big.Int
	// impl is the front proxy's implementation (its ERC-1967 slot); testImpl where unset
	impl common.Address
	// noSloads: the front does not answer extSloads
	noSloads bool
	// baseTime is where the chain's millisecond-time contract (BaseTime) answers; testBaseTime where unset
	baseTime common.Address
	// holdMs is each price lane's HOLD_MS(), by its address; a lane not in it reverts the call
	holdMs map[common.Address]int64
	// the front's clock (clock()): clockMode 2, Base's, reads clockMs0 at block clockBlock0 and 200 ms a block on,
	// where that agrees with the block's second, else BaseTime, the block's second plus the sub-second it stores,
	// subMs on the head's state; mode 0 the block's second, plus ageSlackMs on the age clock. node anchors mode 2 at
	// its head, so the head reads its second plus subMs, a nonzero sub-second unless a test sets it.
	clockMode, clockBlock0, clockMs0, subMs, ageSlackMs uint64
}

// clockAt is the front's clock and age clock in a block of second at and number on the head's state
// (PropAMMClock.read).
func (c *fakeChain) clockAt(at, number uint64) (nowMs, ageNow uint64) {
	return c.clockAtSub(at, number, -1)
}

// clockAtSub is clockAt where BaseTime stores the sub-second sub (-1: the head's, subMs).
func (c *fakeChain) clockAtSub(at, number uint64, sub int64) (nowMs, ageNow uint64) {
	switch c.clockMode {
	case 0:
		return at * 1000, at*1000 + c.ageSlackMs
	case 2:
		if number >= c.clockBlock0 {
			if ms := c.clockMs0 + (number-c.clockBlock0)*200; ms/1000 == at {
				return ms, ms
			}
		}
	}
	return at*1000 + c.baseSub(sub), at*1000 + c.baseSub(sub)
}

// baseTimeAt is where the chain's BaseTime answers.
func (c *fakeChain) baseTimeAt() common.Address {
	if c.baseTime != (common.Address{}) {
		return c.baseTime
	}
	return testBaseTime
}

// baseSub is the sub-second BaseTime stores: sub, or the head's (subMs) where sub is -1.
func (c *fakeChain) baseSub(sub int64) uint64 {
	if sub < 0 {
		return c.subMs
	}
	return uint64(sub)
}

// testDoorCodehash is the doors' creation code hash the test config names.
var testDoorCodehash = common.HexToHash("0x4b9c446404a722489fdd14d3dbb84f6bca1f4f94cd6521fba7208aeb9f1fa0b8")

// testCellHash is the desk contracts' creation code hash the test config names (the front's CELL_CODEHASH()).
var testCellHash = common.HexToHash("0x9d1e3c1c5f0c7b0f6a2e4d7b8c3a5e1f2b4d6c8a0e9f1a3b5c7d9e0f2a4b6c8d")

// cellOf is desk's contract: the CREATE2 child of the test front at salt (1 << 252) | desk under testCellHash.
func cellOf(desk uint32) common.Address { return deskContractOf(testFront, testCellHash, desk) }

// cellFor is desk's contract as the fake front answers deskCellOf.
func (c *fakeChain) cellFor(desk uint32) common.Address {
	if a, ok := c.cells[desk]; ok {
		return a
	}
	return cellOf(desk)
}

// deskWord is the desk word of desk at the head.
func (c *fakeChain) deskWord(desk uint32) *big.Int {
	if w, ok := c.deskWords[desk]; ok {
		return w
	}
	return new(big.Int)
}

// quoteKey is the slot of pair slot i's quote word in a desk contract, and firmKey of its firm word.
func quoteKey(i uint8) common.Hash { return pairSlot(quoteWordBase, i) }
func firmKey(i uint8) common.Hash  { return pairSlot(firmWordBase, i) }

// deskWordOf is a desk word: the pusher bits, desk epoch d and, where stopped, the stop's block.
func deskWordOf(d uint64, stopped bool, stopBlock uint64) *big.Int {
	w := new(big.Int).SetBytes(common.HexToAddress("0x985442e6d7c3da32f2f0c8c65fb8f8c60dcabf03").Bytes())
	w.Or(w, new(big.Int).Lsh(new(big.Int).SetUint64(d&0x7fffffff), 160))
	if stopped {
		w.SetBit(w, 224, 1)
		w.Or(w, new(big.Int).Lsh(new(big.Int).SetUint64(stopBlock&0x7fffffff), 225))
	}
	return w
}

// fakeDoorOf is the factory's doorOf under the default doors' creation code: the CREATE2 address of the listing id.
func fakeDoorOf(factory common.Address, id uint32) common.Address {
	return crypto.CreateAddress2(factory, common.BigToHash(big.NewInt(int64(id))), testDoorCodehash[:])
}

// doorHash is the doors' creation code hash the factory answers.
func (c *fakeChain) doorHash() common.Hash {
	if c.factoryCodehash != (common.Hash{}) {
		return c.factoryCodehash
	}
	return testDoorCodehash
}

// doorAddr is the factory's doorOf(id) under its own door code hash.
func (c *fakeChain) doorAddr(id uint32) common.Address {
	if a, ok := c.doorOfs[id]; ok {
		return a
	}
	return crypto.CreateAddress2(c.factory, common.BigToHash(big.NewInt(int64(id))), c.doorHash().Bytes())
}

func (c *fakeChain) doorListing(a common.Address) (uint32, *fakeListing) {
	for id, l := range c.listings {
		if l.open && c.doorAddr(id) == a {
			return id, l
		}
	}
	return 0, nil
}

func (c *fakeChain) code(a common.Address) []byte {
	if a == c.front || a == c.factory {
		return []byte{0x60}
	}
	if _, l := c.doorListing(a); l != nil {
		return []byte{0x60}
	}
	return nil
}

func packOut(a abi.ABI, method string, v ...any) []byte {
	b, err := a.Methods[method].Outputs.Pack(v...)
	if err != nil {
		panic(err)
	}
	return b
}

func (c *fakeChain) respond(to common.Address, data []byte, at, number uint64, sub int64) (bool, []byte) {
	switch to {
	case multicall3:
		m, err := abiutil.Multicall3ABI.MethodById(data[:4])
		switch {
		case err == nil && m.Name == "getCurrentBlockTimestamp":
			return true, packOut(abiutil.Multicall3ABI, m.Name, new(big.Int).SetUint64(at))
		case err == nil && m.Name == "getBlockNumber":
			return true, packOut(abiutil.Multicall3ABI, m.Name, new(big.Int).SetUint64(number))
		}
		return false, nil
	case c.baseTimeAt():
		// BaseTime's timestampMs(): the block's second plus the sub-second it stores
		return true, packOut(baseTimeABI, "timestampMs", new(big.Int).SetUint64(at*1000+c.baseSub(sub)))
	case c.front:
		if sl, err := slotsABI.MethodById(data[:4]); err == nil && sl.Name == "extSloads" {
			if c.noSloads {
				return false, nil
			}
			// the front's own storage, word by word
			args, _ := sl.Inputs.Unpack(data[4:])
			slots := args[0].([][32]byte)
			words := make([][32]byte, len(slots))
			for k, slot := range slots {
				words[k] = c.storage(c.front, slot)
			}
			return true, packOut(slotsABI, "extSloads", words)
		}
		m, err := frontABI.MethodById(data[:4])

		if err == nil && m.Name == "clock" && !c.noClock {
			// Base's: a 1 s lead, a 30 s cap on a word's life, the clock's age slack, mode and anchor
			u := func(v uint64) *big.Int { return new(big.Int).SetUint64(v) }
			return true, packOut(frontABI, "clock", big.NewInt(1000), big.NewInt(30_000), new(big.Int),
				u(c.ageSlackMs), u(c.clockMode), u(c.clockBlock0), u(c.clockMs0))
		}
		if err == nil && m.Name == "deskCellOf" {
			args, _ := m.Inputs.Unpack(data[4:])
			return true, packOut(frontABI, "deskCellOf", c.cellFor(args[0].(uint32)))
		}
		if err == nil && m.Name == "CELL_CODEHASH" {
			h := testCellHash
			if c.cellHash != (common.Hash{}) {
				h = c.cellHash
			}
			return true, packOut(frontABI, "CELL_CODEHASH", h)
		}
		if err == nil && m.Name == "quoteFrom" {
			args, _ := m.Inputs.Unpack(data[4:])
			l, ok := c.listings[uint32(args[3].(*big.Int).Uint64())]
			dir := -1
			if ok && args[0].(common.Address) == l.view.Token0 && args[1].(common.Address) == l.view.Token1 {
				dir = 0
			} else if ok && args[0].(common.Address) == l.view.Token1 && args[1].(common.Address) == l.view.Token0 {
				dir = 1
			}
			if dir < 0 || args[4].(*big.Int).Bit(253) == 0 {
				return false, nil // the fake answers the plan of a listing's own pair alone
			}
			return c.planAt(l, dir, args[2].(*big.Int), at, number, sub)
		}
		if err != nil || m.Name != "doorView" {
			return false, nil
		}
		args, _ := m.Inputs.Unpack(data[4:])
		id := uint32(args[0].(*big.Int).Uint64())
		v := zeroView()
		if l, ok := c.listings[id]; ok {
			v = l.view
		}
		return true, packOut(frontABI, "doorView", v)
	case c.factory:
		m, err := factoryABI.MethodById(data[:4])
		if err != nil {
			return false, nil
		}
		switch m.Name {
		case "FRONT":
			f := c.front
			if c.factoryFront != (common.Address{}) {
				f = c.factoryFront
			}
			return true, packOut(factoryABI, "FRONT", f)
		case "DOOR_CODEHASH":
			if c.noDoorHash {
				return false, nil
			}
			return true, packOut(factoryABI, "DOOR_CODEHASH", c.doorHash())
		case "doorOf":
			args, _ := m.Inputs.Unpack(data[4:])
			return true, packOut(factoryABI, "doorOf", c.doorAddr(uint32(args[0].(*big.Int).Uint64())))
		}
		return false, nil
	}
	if hold, ok := c.holdMs[to]; ok {
		if m, err := laneABI.MethodById(data[:4]); err == nil && m.Name == "HOLD_MS" {
			return true, packOut(laneABI, "HOLD_MS", big.NewInt(hold))
		}
		return false, nil
	}
	if m, err := abiutil.Erc20ABI.MethodById(data[:4]); err == nil && m.Name == "balanceOf" {
		args, _ := m.Inputs.Unpack(data[4:])
		bal := c.balances[[2]common.Address{to, args[0].(common.Address)}]
		if bal == nil {
			bal = new(big.Int)
		}
		return true, packOut(abiutil.Erc20ABI, "balanceOf", bal)
	}
	id, l := c.doorListing(to)
	if l == nil {
		return true, nil // a call to an account with no code succeeds with no data
	}
	m, err := doorABI.MethodById(data[:4])
	if err != nil {
		return false, nil
	}
	switch m.Name {
	case "FRONT":
		f := c.front
		if l.wrongFront != (common.Address{}) {
			f = l.wrongFront
		}
		return true, packOut(doorABI, "FRONT", f)
	case "LISTING":
		if l.wrongListing != 0 {
			id = l.wrongListing
		}
		return true, packOut(doorABI, "LISTING", big.NewInt(int64(id)))
	case "TOKEN0":
		return true, packOut(doorABI, "TOKEN0", l.view.Token0)
	case "TOKEN1":
		return true, packOut(doorABI, "TOKEN1", l.view.Token1)
	case "quote":
		args, _ := m.Inputs.Unpack(data[4:])
		tokenIn, tokenOut, amountIn := args[0].(common.Address), args[1].(common.Address), args[2].(*big.Int)
		dir := -1
		if tokenIn == l.view.Token0 && tokenOut == l.view.Token1 {
			dir = 0
		} else if tokenIn == l.view.Token1 && tokenOut == l.view.Token0 {
			dir = 1
		}
		if dir < 0 {
			return false, crypto.Keccak256([]byte("UnsupportedPair()"))[:4]
		}
		out, err := c.quoteAtSub(l, dir, amountIn, at, number, sub)
		if err != nil {
			return false, crypto.Keccak256([]byte(err.Error() + "()"))[:4]
		}
		return true, packOut(doorABI, "quote", out)
	case "quoteFillable":
		args, _ := m.Inputs.Unpack(data[4:])
		tokenIn, amountIn := args[0].(common.Address), args[2].(*big.Int)
		dir := 0
		if tokenIn == l.view.Token1 {
			dir = 1
		}
		side := l.sides[dir]
		if side.dark || len(side.rungs) == 0 {
			return false, crypto.Keccak256([]byte("NoFill()"))[:4]
		}
		fill := new(big.Int).Set(amountIn)
		if top := big.NewInt(side.rungs[len(side.rungs)-1].upTo); fill.Cmp(top) > 0 {
			fill = top
		}
		nowMs, ageNow := c.clockAtSub(at, number, sub)
		if l.stale(ageNow) {
			return false, crypto.Keccak256([]byte("NoFill()"))[:4]
		}
		keep := 1_000_000 - l.cutAt(dir, nowMs, ageNow)
		out, err := side.quoteKeeping(fill, keep)
		if err == nil && l.outCap[dir] != 0 && out.Cmp(big.NewInt(l.outCap[dir])) > 0 {
			err = errNoFill // no prefix search: the side's top is not sampled
			if l.fillableUnderCap {
				// the largest prefix whose output is within the bound
				lo, hi := big.NewInt(0), new(big.Int).Set(fill)
				for new(big.Int).Sub(hi, lo).Cmp(big.NewInt(1)) > 0 {
					mid := new(big.Int).Rsh(new(big.Int).Add(lo, hi), 1)
					if o, e := side.quoteKeeping(mid, keep); e == nil && o.Cmp(big.NewInt(l.outCap[dir])) <= 0 {
						lo = mid
					} else {
						hi = mid
					}
				}
				fill = lo
				out, err = side.quoteKeeping(fill, keep)
			}
		}
		if err != nil {
			return false, crypto.Keccak256([]byte(err.Error() + "()"))[:4]
		}
		return true, packOut(doorABI, "quoteFillable", fill, out)
	}
	return false, nil
}

// quoteAt is the door's quote of listing l in direction dir in a block of second at and number, on the head's state.
func (c *fakeChain) quoteAt(l *fakeListing, dir int, amountIn *big.Int, at, number uint64) (*big.Int, error) {
	return c.quoteAtSub(l, dir, amountIn, at, number, -1)
}

// quoteAtSub is quoteAt where BaseTime stores the sub-second sub (-1: the head's).
func (c *fakeChain) quoteAtSub(l *fakeListing, dir int, amountIn *big.Int, at, number uint64,
	sub int64) (*big.Int, error) {
	if l.shrinkAt != 0 && at >= l.shrinkAt && amountIn.Cmp(big.NewInt(l.shrinkTo)) > 0 {
		return nil, errNoFill
	}
	if l.minIn[dir] != 0 && amountIn.Cmp(big.NewInt(l.minIn[dir])) < 0 {
		return nil, errBelowMinimum
	}
	nowMs, ageNow := c.clockAtSub(at, number, sub)
	if l.stale(ageNow) {
		return nil, errNoFill
	}
	keep := 1_000_000 - l.cutAt(dir, nowMs, ageNow)
	if r := l.route[dir]; r.routes(amountIn, number) {
		// the pool's preview: the walk with custody lifted, less its fee, up to its caps
		if r.routeTo != 0 && amountIn.Cmp(big.NewInt(r.routeTo)) > 0 {
			return nil, errNoFill
		}
		return l.sides[dir].quoteKeeping(amountIn, keep-r.feePpm)
	}
	out, err := l.sides[dir].quoteKeeping(amountIn, keep)
	if err == nil && l.outCap[dir] != 0 && out.Cmp(big.NewInt(l.outCap[dir])) > 0 {
		return nil, errNoFill
	}
	return out, err
}

func zeroView() DoorView {
	z := func() *big.Int { return new(big.Int) }
	return DoorView{PairWord: z(), BidFillMs: z(), AskFillMs: z(), Ticket: z(), Word: z(),
		Depth: [4]*big.Int{z(), z(), z(), z()}}
}

// node is the chain behind a node at block, its clock anchored there where no test anchored it.
func (c *fakeChain) node(t testing.TB, block uint64) *node {
	n := newNode(t, block, c.respond, c.code, c.storage)
	n.baseTime = c.baseTimeAt()
	if c.clockBlock0 == 0 {
		c.clockBlock0, c.clockMs0 = block, n.time*1000+c.subMs
	}
	return n
}

// storage answers a storage read: each listing's controls word, source ids, live source word, session word and board
// words at their slots in the front, and each source's entry at its slot, every other slot zero.
func (c *fakeChain) storage(a common.Address, slot common.Hash) common.Hash {
	if v, ok := c.headSlots[a][slot]; ok {
		return v
	}
	if slot == common.BigToHash(big.NewInt(deskWordSlot)) {
		for desk := range c.deskWords {
			if c.cellFor(desk) == a {
				return common.BigToHash(c.deskWords[desk])
			}
		}
	}
	if a != c.front {
		return common.Hash{}
	}
	if slot == testImplSlot {
		if c.impl != (common.Address{}) {
			return common.BytesToHash(c.impl[:])
		}
		return common.BytesToHash(testImpl[:])
	}
	for id, l := range c.listings {
		if l.controls != nil && controlsSlot(id) == slot {
			return common.BigToHash(l.controls)
		}
		if l.session != nil && recordSlot(id, 13) == slot {
			return common.BigToHash(l.session)
		}
		if l.sourceIds != nil && recordSlot(id, 10) == slot {
			return common.BigToHash(l.sourceIds)
		}
		if recordSlot(id, 2) == slot {
			return common.BigToHash(c.liveWord(l))
		}
		for k, board := range l.boards {
			if board != nil && recordSlot(id, int64(boardBidWord+k)) == slot {
				return common.BigToHash(board)
			}
		}
		if l.settings != nil && recordSlot(id, settingsWord) == slot {
			return common.BigToHash(l.settings)
		}
	}
	for id, e := range c.sources {
		if sourceEntrySlot(id) == slot {
			return common.BigToHash(e)
		}
	}
	return common.Hash{}
}

// liveWord is a listing's record word 2: its price where set, else the primary's entry as the front copies it (its
// address, window and guard), zero where it is bound to none.
func (c *fakeChain) liveWord(l *fakeListing) *big.Int {
	if l.price != nil {
		return l.price
	}
	if l.sourceIds == nil {
		return new(big.Int)
	}
	return liveOf(c.sources[uint32(bits(l.sourceIds, 0, 32))], false)
}

// liveOf is the record word 2 the front writes for a listing reading entry e, its backup where onBackup.
func liveOf(e *big.Int, onBackup bool) *big.Int {
	if e == nil {
		return new(big.Int)
	}
	keep := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 160), big.NewInt(1))
	keep.Or(keep, new(big.Int).Lsh(big.NewInt(0xff), 236))
	keep.SetBit(keep, 247, 1)
	w := new(big.Int).And(e, keep)
	if onBackup {
		w.SetBit(w, 244, 1)
	}
	return w
}

// testImplSlot is the ERC-1967 implementation slot, keccak256("eip1967.proxy.implementation") - 1.
var testImplSlot = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")

// sourceEntrySlot is the front's storage slot of a price source's entry: keccak256(abi.encode(id, namespace + 15)).
func sourceEntrySlot(id uint32) common.Hash {
	base := new(big.Int).Add(frontStore.Big(), big.NewInt(15))
	return crypto.Keccak256Hash(common.BigToHash(big.NewInt(int64(id))).Bytes(), common.BigToHash(base).Bytes())
}

// recordSlot is the front's storage slot of word w of a listing's record, counted from its controls word (word 7).
func recordSlot(listing uint32, w int64) common.Hash {
	return common.BigToHash(new(big.Int).Add(controlsSlot(listing).Big(), big.NewInt(w-7)))
}
