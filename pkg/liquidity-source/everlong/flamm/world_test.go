package everlongflamm

import (
	"bytes"
	"math/big"
	"slices"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
)

// fakePool is one pool of the fake chain, answering the reads the lister and the tracker send.
type fakePool struct {
	factory    common.Address
	listed     bool // the Factory's isPool
	paused     bool
	version    uint32
	tokens     []tokenConfig
	tradable   *big.Int
	hook       common.Address // the swap hook
	pairHook   common.Address
	pairMask   *big.Int
	laneConfig *big.Int
	noCount    bool // tokenCount reverts
	router     common.Address
	feed       common.Address // the pool's price feed
	noFeed     bool           // priceFeed reverts
	venues     []venueView    // the Router's record of the pool's venues
	badVenue   map[int]bool   // records that revert
	acts       map[int]uint8  // venueActions by record; absent: every action (15)
	badActs    map[int]bool   // venueActions reads that revert
	// quote is previewSwap of token i for token j: what the pool uses and pays, or false where it reverts
	quote func(from common.Address, i, j int, amount *big.Int) (used, out *big.Int, ok bool)
	// fee is the fee rate a fill of `amount` of token i for token j reports, its route's label; nil: curveLabel
	fee func(i, j int, amount *big.Int) *big.Int
	// usd is each token's USD price per whole token (WAD), legs its held, supplied, posted and owed legs, as the
	// pool's request carries them (pairRequest); noRequest: pairRequest reverts
	usd       []*big.Int
	legs      [][4]*big.Int
	noRequest bool
}

// fakeHook is a pricing hook of the fake chain: params() answers its curve and tuning in words 0-26 and its leverage
// row in words 27-34 of 35, quotePair the curve's fee in word 3 of 7, and levState, sent the 26 words of a request,
// its top bid in word 6 of 8. Any other call reverts.
type fakeHook struct {
	tuning   [27]*big.Int // params() words 0-26, the row in force: a parameter record a keeper posts moves them
	row      [8]*big.Int  // spread, spreadDn, upCap, dnCap, ceiling, band, feedGate, upFloor
	words    int          // how many words params() answers; zero: 35
	reverts  int          // params() reverts this many times more before it answers; negative: always
	bid      *big.Int     // levState's word 6; nil: 1e18
	noLev    bool         // levState reverts
	levWords int          // how many words levState answers; zero: 8
	asked    int          // the params() calls it saw
	other    int          // the calls it saw of any other method than params, quotePair and levState
}

// curveLabel is the fee rate the fake pools' curve route reports: on no row's tiers.
var curveLabel = big.NewInt(3_000_000_000_000_000)

var quotePairSelector = []byte{0x2f, 0x4d, 0xe4, 0x91}

// leverageRow is a row like the protocol's EverlongHook's: spreads 1.75 % and 1.3 %, caps .5 % and 2 %, ceiling 2.2,
// band 5 %, no feed gate, a lever-up floor of 1.7.
func leverageRow() [8]*big.Int {
	return [8]*big.Int{big.NewInt(17_500_000_000_000_000), big.NewInt(13_000_000_000_000_000),
		big.NewInt(5_000_000_000_000_000), big.NewInt(20_000_000_000_000_000), big.NewInt(2_200_000_000_000_000_000),
		big.NewInt(50_000_000_000_000_000), new(big.Int), big.NewInt(1_700_000_000_000_000_000)}
}

func (h *fakeHook) respond(data []byte) (bool, []byte) {
	switch {
	case bytes.HasPrefix(data, hookABI.Methods["params"].ID):
		h.asked++
		if h.reverts != 0 {
			h.reverts--
			return false, nil
		}
		var w [paramsWords]*big.Int
		for k := range w {
			w[k] = new(big.Int)
		}
		for k, v := range h.tuning {
			if v != nil {
				w[k].Set(v)
			}
		}
		for k, v := range h.row {
			if v != nil {
				w[27+k].Set(v)
			}
		}
		m := hookABI.Methods["params"]
		ok, ret := pack(&m, w)
		if h.words != 0 {
			ret = ret[:32*h.words]
		}
		return ok, ret
	case len(data) != 4+32*requestWords:
		h.other++
		return false, nil
	case bytes.HasPrefix(data, levStateSelector):
		if h.noLev {
			return false, nil
		}
		ret := make([]byte, 32*levStateWords)
		lo.Ternary(h.bid != nil, h.bid, wad).FillBytes(ret[32*6 : 32*7])
		if h.levWords != 0 {
			ret = ret[:32*h.levWords]
		}
		return true, ret
	case bytes.HasPrefix(data, quotePairSelector):
		ret := make([]byte, 32*7)
		curveLabel.FillBytes(ret[32*3 : 32*4])
		return true, ret
	}
	h.other++
	return false, nil
}

// world is a fake chain of Factories, pools and hooks.
type world struct {
	mu        sync.Mutex
	factories map[common.Address][]common.Address
	pools     map[common.Address]*fakePool
	fronts    map[common.Address]bool // the Factory's isListedFront
	code      map[common.Address][]byte
	impl      map[common.Address]common.Address // an ERC-1967 proxy's implementation slot
	routers   map[common.Address]bool
	hooks     map[common.Address]*fakeHook // a contract that answers as a pricing hook; any other reverts
	// the venues' accounts, one per kind (venueAccount): capabilities() as measured of each kind, with capHeldRepay
	// on Lista Moolah's (3), whose minimumDebt answers minimum (nil: not ok); badCaps and badMinimum revert them
	minimum    *big.Int
	badCaps    map[uint8]bool
	badMinimum bool
}

// venueAccount is the account of the fake venues of a kind.
func venueAccount(kind uint8) common.Address {
	return common.BigToAddress(big.NewInt(0xacc0 + int64(kind)))
}

// kindCaps are the accounts' capabilities() by kind: Lista Moolah's, Aave V3's and Venus Core's as read on BSC, and
// Morpho Blue's as its source states it.
var kindCaps = map[uint8]int64{0: 1, 1: 382, 2: 1534, 3: 560}

func (w *world) account(to common.Address, data []byte) (bool, []byte, bool) {
	for kind, caps := range kindCaps {
		if venueAccount(kind) != to {
			continue
		}
		m, _, ok := decode(accountABI, data)
		switch {
		case !ok:
			return false, nil, true
		case m.Name == "capabilities":
			if w.badCaps[kind] {
				return false, nil, true
			}
			ret, _ := m.Outputs.Pack(big.NewInt(caps))
			return true, ret, true
		case kind != 3 || w.badMinimum:
			return false, nil, true
		}
		ret, _ := m.Outputs.Pack(w.minimum != nil, orZero(w.minimum))
		return true, ret, true
	}
	return false, nil, false
}

func newWorld() *world {
	return &world{factories: map[common.Address][]common.Address{}, pools: map[common.Address]*fakePool{},
		fronts: map[common.Address]bool{}, code: map[common.Address][]byte{}, impl: map[common.Address]common.Address{},
		routers: map[common.Address]bool{}, hooks: map[common.Address]*fakeHook{}, badCaps: map[uint8]bool{}}
}

// hook installs code at `a` that answers as a pricing hook with a leverage row like the protocol's.
func (w *world) hook(a common.Address, code []byte) *fakeHook {
	h := &fakeHook{row: leverageRow()}
	w.code[a], w.hooks[a] = code, h
	return h
}

func (w *world) storageAt(a common.Address, slot common.Hash) common.Hash {
	w.mu.Lock()
	defer w.mu.Unlock()
	if slot != erc1967Slot {
		return common.Hash{}
	}
	impl := w.impl[a]
	return common.BytesToHash(impl[:])
}

func (w *world) add(addr common.Address, p *fakePool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pools[addr] = p
	w.factories[p.factory] = append(w.factories[p.factory], addr)
}

func (w *world) codeAt(a common.Address) []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.code[a]
}

func (w *world) respond(from, to common.Address, data []byte) (bool, []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if list, ok := w.factories[to]; ok {
		m, args, ok := decode(factoryABI, data)
		if !ok {
			return false, nil
		}
		switch m.Name {
		case "pools":
			return pack(m, list)
		case "isPool":
			p := w.pools[args[0].(common.Address)]
			return pack(m, p != nil && p.listed && p.factory == to)
		default:
			return pack(m, w.fronts[args[0].(common.Address)])
		}
	}
	if w.routers[to] {
		m, args, ok := decode(routerABI, data)
		if !ok {
			return false, nil
		}
		p := w.pools[args[0].(common.Address)]
		if p == nil || p.router != to {
			return false, nil
		}
		if m.Name == "venueCount" {
			return pack(m, big.NewInt(int64(len(p.venues))))
		}
		id := int(args[1].(uint16))
		if m.Name == "venueActions" {
			if id >= len(p.venues) || p.badActs[id] {
				return false, nil
			}
			if a, ok := p.acts[id]; ok {
				return pack(m, a)
			}
			return pack(m, uint8(15))
		}
		if id < len(p.venues) && !p.badVenue[id] {
			return pack(m, p.venues[id])
		}
		return false, nil
	}
	if h, ok := w.hooks[to]; ok {
		return h.respond(data)
	}
	if ok, ret, isAccount := w.account(to, data); isAccount {
		return ok, ret
	}
	p, ok := w.pools[to]
	if !ok {
		return false, nil
	}
	m, args, ok := decode(poolABI, data)
	if !ok {
		return false, nil
	}
	zero := new(big.Int)
	switch m.Name {
	case "tokenCount":
		if p.noCount {
			return false, nil
		}
		return pack(m, big.NewInt(int64(len(p.tokens))))
	case "tokenConfig":
		if k := int(args[0].(uint8)); k < len(p.tokens) {
			return pack(m, p.tokens[k])
		}
		return false, nil
	case "paused":
		return pack(m, p.paused)
	case "poolCodeVersion":
		return pack(m, p.version)
	case "tradablePairs":
		return pack(m, p.tradable, zero, zero, zero, zero)
	case "hooks":
		return pack(m, hookSet{Hook: p.hook})
	case "pairHook":
		return pack(m, p.pairHook, p.pairMask)
	case "lane":
		if p.laneConfig == nil {
			return false, nil
		}
		return pack(m, zero, p.laneConfig, zero)
	case "router":
		return pack(m, p.router)
	case "priceFeed":
		if p.noFeed {
			return false, nil
		}
		return pack(m, p.feed)
	case "pairRequest":
		in, out, a := p.index(args[0].(common.Address)), p.index(args[1].(common.Address)), args[2].(*big.Int)
		if p.noRequest || in < 0 || out < 0 || in == out || a.Sign() == 0 {
			return false, nil
		}
		return pack(m, p.request(from, in, out, a))
	default: // previewSwap
		in, out := p.index(args[0].(common.Address)), p.index(args[1].(common.Address))
		if in < 0 || out < 0 || in == out || p.paused || p.tradable.Bit(16*in+out) == 0 {
			return false, nil
		}
		used, paid, ok := p.quote(from, in, out, args[2].(*big.Int))
		if !ok || used.Sign() == 0 || paid.Sign() == 0 { // a fill that uses or pays nothing reverts, as the pool's
			return false, nil
		}
		fee := curveLabel
		if p.fee != nil {
			fee = p.fee(in, out, args[2].(*big.Int))
		}
		return pack(m, used, paid, fee)
	}
}

// request is the pool's pairRequest of token i for token j at `amount`, the words a swap hands its hook: the caller,
// the tokens and their indices, the tighter of the two tokens' bands, their USD words, the amount, a ceiling and both
// tokens' legs.
func (p *fakePool) request(from common.Address, i, j int, amount *big.Int) [requestWords]*big.Int {
	var w [requestWords]*big.Int
	for k := range w {
		w[k] = new(big.Int)
	}
	w[0].SetBytes(from[:])
	w[1].SetBytes(p.tokens[i].Token[:])
	w[2].SetBytes(p.tokens[j].Token[:])
	w[3].SetInt64(int64(i))
	w[4].SetInt64(int64(j))
	for _, t := range []int{i, j} {
		if b := p.tokens[t].SwapPriceBandWad; b != 0 && (w[10].Sign() == 0 || b < w[10].Uint64()) {
			w[10].SetUint64(b)
		}
	}
	if p.usd != nil {
		w[11].Set(p.usd[i])
		w[12].Set(p.usd[j])
	}
	w[13].Set(amount)
	w[14].Lsh(one, 200)
	for k := range 4 {
		if p.legs != nil {
			w[15+k].Set(p.legs[i][k])
			w[19+k].Set(p.legs[j][k])
		}
	}
	return w
}

func (p *fakePool) index(token common.Address) int {
	return slices.IndexFunc(p.tokens, func(c tokenConfig) bool { return c.Token == token })
}

func decode(a abi.ABI, data []byte) (*abi.Method, []any, bool) {
	m, err := a.MethodById(data)
	if err != nil {
		return nil, nil, false
	}
	args, err := m.Inputs.Unpack(data[4:])
	return m, args, err == nil
}

func pack(m *abi.Method, v ...any) (bool, []byte) {
	ret, err := m.Outputs.Pack(v...)
	if err != nil {
		panic(err)
	}
	return true, ret
}

// tierWorld is a two-token pool of factory A, cbBTC and USDC at a feed of $100,000, priced by hookA, whose leverage
// route beats its curve on small fills, the better of the two winning as the hook's does: a sale of up to 0.1 cbBTC
// pays 995 USDC units a sat, labelled with the row's upper spread, and a buy of up to 10,000 USDC pays 995 sats a
// 1,000,000 units, labelled with its lower one. Past them the curve pays 100 bp under the feed and a bp less per
// depth, to 50 cbBTC and to a 1,000,000 USDC ceiling. USDC's band is 8 %, so the row's 5 % is the tighter. Both
// tokens hold idle balances, which the request reports as its legs.
func tierWorld() (*world, *fakePool) { return tierWorldFrom(0, 0) }

// tierWorldFrom is tierWorld with the leverage route taking no sale under saleFrom and no buy under buyFrom, where
// the curve pays.
func tierWorldFrom(saleFrom, buyFrom int64) (*world, *fakePool) {
	w := newWorld()
	w.hook(common.HexToAddress(hookA), hookCode(0xaa))
	w.routers[common.HexToAddress(routerA)] = true
	row := leverageRow()
	curveSale, curveBuy := oracle(1_000, 1, 100, 1_000_000_000), oracle(1, 1_000, 100, 100_000_000_000)
	saleTop, buyTop := big.NewInt(tierSaleTop), big.NewInt(tierBuyTop)
	lever := func(i int, a *big.Int) bool {
		if i == 0 {
			return a.Cmp(big.NewInt(saleFrom)) >= 0 && a.Cmp(saleTop) <= 0
		}
		return a.Cmp(big.NewInt(buyFrom)) >= 0 && a.Cmp(buyTop) <= 0
	}
	fp := &fakePool{
		factory: common.HexToAddress(factoryA), listed: true, version: 1,
		tokens:   []tokenConfig{tokenCfg(cbBTC, 8), tokenCfg(usdc, 6)},
		tradable: bits(16*0+1, 16*1+0),
		hook:     common.HexToAddress(hookA), pairMask: new(big.Int), laneConfig: new(big.Int),
		feed: common.HexToAddress(feedA), router: common.HexToAddress(routerA), venues: []venueView{venue(0, 0, 1, 0)},
		usd: []*big.Int{new(big.Int).Mul(big.NewInt(100_000), wad), wad},
		legs: [][4]*big.Int{
			{big.NewInt(3_000_000_000), new(big.Int), new(big.Int), new(big.Int)},     // 30 cbBTC held
			{big.NewInt(3_000_000_000_000), new(big.Int), new(big.Int), new(big.Int)}, // 3,000,000 USDC held
		},
	}
	fp.tokens[1].SwapPriceBandWad = 80_000_000_000_000_000
	fp.quote = func(from common.Address, i, j int, a *big.Int) (*big.Int, *big.Int, bool) {
		switch {
		case from != defaultTaker:
			return nil, nil, false
		case i == 0 && lever(i, a):
			return a, new(big.Int).Mul(a, big.NewInt(995)), true
		case i == 0 && a.Cmp(big.NewInt(5_000_000_000)) <= 0:
			return curveSale(from, i, j, a)
		case i == 1 && lever(i, a):
			out := new(big.Int).Mul(a, big.NewInt(995))
			return a, out.Div(out, big.NewInt(1_000_000)), true
		case i == 1:
			return curveBuy(from, i, j, minBig(a, big.NewInt(1_000_000_000_000)))
		}
		return nil, nil, false
	}
	fp.fee = func(i, _ int, a *big.Int) *big.Int {
		if lever(i, a) {
			return row[i]
		}
		return curveLabel
	}
	w.add(common.HexToAddress(poolX), fp)
	return w, fp
}

// lateLeverWorld is tierWorld with a deep curve on the sale, 100 bp under the feed and a bp more each 0.01 cbBTC,
// and its leverage route taking sales of 2.8 to 10 cbBTC only, paying the curve's output and 2 USDC units a sat
// more: opening just under a probed size, so a chord from the curve's last point to the route's first would pay
// more than the pool just under the opening, and the steeper test alone does not see it.
func lateLeverWorld() (*world, *fakePool) {
	w, fp := tierWorld()
	curve, curveQuote, curveFee := oracle(1_000, 1, 100, 1_000_000), fp.quote, fp.fee
	lever := func(i int, a *big.Int) bool {
		return i == 0 && a.Cmp(big.NewInt(lateLeverFrom)) >= 0 && a.Cmp(big.NewInt(lateLeverTop)) <= 0
	}
	fp.quote = func(from common.Address, i, j int, a *big.Int) (*big.Int, *big.Int, bool) {
		switch {
		case i != 0 || from != defaultTaker:
			return curveQuote(from, i, j, a)
		case lever(i, a):
			_, out, _ := curve(from, i, j, a)
			return a, out.Add(out, new(big.Int).Lsh(a, 1)), true
		case a.Cmp(big.NewInt(5_000_000_000)) <= 0:
			return curve(from, i, j, a)
		}
		return nil, nil, false
	}
	fp.fee = func(i, j int, a *big.Int) *big.Int {
		switch {
		case i != 0:
			return curveFee(i, j, a)
		case lever(i, a):
			return leverageRow()[0]
		}
		return curveLabel
	}
	return w, fp
}

// lateLeverFrom and lateLeverTop bound the sales lateLeverWorld's leverage route takes.
const (
	lateLeverFrom = 280_000_000
	lateLeverTop  = 1_000_000_000
)

// tierSaleTop and tierBuyTop are the largest fills tierWorld's leverage route takes.
const (
	tierSaleTop = 10_000_000
	tierBuyTop  = 10_000_000_000
)

// oracle is a quote at a price of num/den output units per input unit, less a fee that rises with the size,
// feeBps plus a bp per depth of input: an output concave in the input, as an oracle-priced pool's is, floored once.
func oracle(num, den, feeBps, depth int64) func(common.Address, int, int, *big.Int) (*big.Int, *big.Int, bool) {
	return func(_ common.Address, _, _ int, a *big.Int) (*big.Int, *big.Int, bool) {
		return a, oracleOut(a, num, den, feeBps, depth), true
	}
}

func oracleOut(a *big.Int, num, den, feeBps, depth int64) *big.Int {
	// out = a * num/den * (1 - feeBps/1e4 - a/(depth*1e4))
	d := big.NewInt(depth)
	keep := new(big.Int).Mul(big.NewInt(10_000-feeBps), d)
	keep.Sub(keep, a)
	if keep.Sign() <= 0 {
		return new(big.Int)
	}
	out := new(big.Int).Mul(a, big.NewInt(num))
	out.Mul(out, keep)
	return out.Div(out, new(big.Int).Mul(big.NewInt(den*10_000), d))
}

// bits is a mask with the given bits set.
func bits(b ...int) *big.Int {
	m := new(big.Int)
	for _, k := range b {
		m.SetBit(m, k, 1)
	}
	return m
}

// venue is a Router record of a venue of `kind` on the credit pair (coll, debt), with its status bits.
func venue(kind, coll, debt, status uint8) venueView {
	return venueView{Account: venueAccount(kind), Id: [32]byte{31: kind}, Kind: kind, Coll: coll, Debt: debt,
		Status: status, DebtCap: new(big.Int), SupplyCap: new(big.Int)}
}

func tokenCfg(addr string, decimals uint8) tokenConfig {
	return tokenConfig{Token: common.HexToAddress(addr), Decimals: decimals, MaxSwapNotional: new(big.Int),
		ReserveTarget: new(big.Int), MinSupplyBatch: new(big.Int), Held: new(big.Int)}
}
