package lotflow

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

// RPCLoader is the Loader (and MqLoader) over JSON-RPC eth_call at one block tag. v4 pool state is read
// through PoolManager.extsload (StateLibrary slots), v3 pools through their views; each candidate's
// initialized ticks are loaded over its leg window (LegLimitBps either side of spot) so every leg the
// simulator can run stays inside loaded data. Tick reads are batched.
type RPCLoader struct {
	c         *rpc.Client
	Block     string // "latest" or a hex block number
	Cfg       *Config
	Hook      string
	PM        string
	Registry  string
	VenueBook string
	NavGuard  string
	Usdg      string
	UsdgDec   int

	mu   sync.Mutex
	keys map[string]rpcKey // v4 PoolId -> key (from getStock)

	mc *multicaller // nil: one eth_call per read (tooling and fork tests); Track always sets it

	ts uint64 // the pinned block's timestamp, once read
}

type rpcKey struct {
	fee         uint32
	tickSpacing int
	hooks       string
}

// NewRPCLoader dials url and reads the hook's immutable wiring (registry, venueBook, navGuard, v3Factory,
// usdg, usdgDecimals). cfg.Usdg / cfg.V3Factory are filled from the hook when empty.
func NewRPCLoader(ctx context.Context, url, hook, pm string, cfg *Config) (*RPCLoader, error) {
	c, err := rpc.DialContext(ctx, url)
	if err != nil {
		return nil, err
	}
	return NewRPCLoaderFromClient(ctx, c, "latest", hook, pm, cfg)
}

// NewRPCLoaderFromClient is NewRPCLoader on an existing client (e.g. dex-lib's ethrpc client's
// ethclient.Client()) at a block tag ("latest" or a hex number).
func NewRPCLoaderFromClient(ctx context.Context, c *rpc.Client, block, hook, pm string, cfg *Config) (*RPCLoader,
	error) {
	return newRPCLoader(ctx, c, nil, block, hook, pm, cfg)
}

func newRPCLoader(ctx context.Context, c *rpc.Client, mc *multicaller, block, hook, pm string, cfg *Config) (
	*RPCLoader, error) {
	l := &RPCLoader{c: c, Block: block, Cfg: cfg, Hook: hook, PM: pm, keys: map[string]rpcKey{}, mc: mc}
	for _, f := range []struct {
		sig string
		dst *string
	}{{"registry()", &l.Registry}, {"venueBook()", &l.VenueBook}, {"navGuard()", &l.NavGuard}, {"usdg()", &l.Usdg}} {
		w, err := l.word(ctx, hook, f.sig)
		if err != nil {
			return nil, fmt.Errorf("hook %s: %w", f.sig, err)
		}
		*f.dst = wordAddr(w)
	}
	w, err := l.word(ctx, hook, "usdgDecimals()")
	if err != nil {
		return nil, err
	}
	l.UsdgDec = int(w.Int64())
	if cfg.Usdg == "" {
		cfg.Usdg = l.Usdg
	}
	if cfg.V3Factory == "" {
		f, err := l.word(ctx, hook, "v3Factory()")
		if err != nil {
			return nil, err
		}
		cfg.V3Factory = wordAddr(f)
	}
	return l, nil
}

// ---------------------------------------------------------------- raw calls

func sel(sig string) []byte { return crypto.Keccak256([]byte(sig))[:4] }

func encWord(v *big.Int) []byte {
	b := make([]byte, 32)
	if v.Sign() >= 0 {
		v.FillBytes(b)
		return b
	}
	t := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 256), v)
	t.FillBytes(b)
	return b
}

func encAddr(a string) []byte { return common.LeftPadBytes(common.HexToAddress(a).Bytes(), 32) }

type callReq struct {
	to   string
	data []byte
	out  hexutil.Bytes
	err  error
}

func (l *RPCLoader) callMsg(to string, data []byte) map[string]any {
	return map[string]any{"to": to, "data": hexutil.Bytes(data), "gas": hexutil.Uint64(30_000_000)}
}

func (l *RPCLoader) call(ctx context.Context, to string, data []byte) ([]byte, error) {
	if l.mc != nil {
		return l.mc.get(ctx, to, data)
	}
	var out hexutil.Bytes
	err := retryLimited(ctx, func() error {
		return l.c.CallContext(ctx, &out, "eth_call", l.callMsg(to, data), l.Block)
	})
	if err != nil {
		return nil, classifyRPC(err)
	}
	return out, nil
}

// classifyRPC splits a per-call error: the node's "execution reverted" (JSON-RPC code 3, or the geth
// message without a code) is ErrReverted; anything else is a TransportError.
func classifyRPC(err error) error {
	if err == nil {
		return nil
	}
	var re rpc.Error
	if (errors.As(err, &re) && re.ErrorCode() == 3) || strings.Contains(err.Error(), "execution reverted") {
		return fmt.Errorf("%w: %v", ErrReverted, err)
	}
	return &TransportError{Err: err}
}

// readFailed: an admission read that reverted or returned < 32 bytes is a failed read (the READ_GAS
// semantics); a transport error is not.
func readFailed(err error) bool { return err != nil && !isTransport(err) }

// rateLimited: a forked node passes the upstream RPC's rate limit through (HTTP 429). Not a revert.
func rateLimited(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "Rate Limit"))
}

// retryLimited retries f while the upstream rate-limits, up to ~3 minutes.
func retryLimited(ctx context.Context, f func() error) error {
	var err error
	for i := 0; i < 36; i++ {
		if err = f(); !rateLimited(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return &TransportError{Err: ctx.Err()}
		case <-time.After(5 * time.Second):
		}
	}
	return err
}

// batch runs eth_calls in batches of 25 (one multicall round on the multicall path); each request
// keeps its own error.
func (l *RPCLoader) batch(ctx context.Context, reqs []*callReq) error {
	if l.mc != nil {
		if !l.mc.collecting() {
			if err := l.mc.fetch(ctx, reqs); err != nil {
				return err
			}
		}
		for _, r := range reqs {
			out, err := l.mc.get(ctx, r.to, r.data)
			if err != nil && !errors.Is(err, ErrReverted) {
				return err
			}
			r.out, r.err = out, err
		}
		return nil
	}
	for i := 0; i < len(reqs); i += 25 {
		j := min(i+25, len(reqs))
		elems := make([]rpc.BatchElem, j-i)
		for k := i; k < j; k++ {
			elems[k-i] = rpc.BatchElem{Method: "eth_call", Args: []any{l.callMsg(reqs[k].to, reqs[k].data), l.Block},
				Result: &reqs[k].out}
		}
		if err := retryLimited(ctx, func() error {
			if err := l.c.BatchCallContext(ctx, elems); err != nil {
				return err
			}
			for k := range elems {
				if rateLimited(elems[k].Error) {
					return elems[k].Error
				}
			}
			return nil
		}); err != nil {
			return &TransportError{Err: err}
		}
		for k := i; k < j; k++ {
			if err := classifyRPC(elems[k-i].Error); isTransport(err) {
				return err
			} else {
				reqs[k].err = err
			}
		}
	}
	return nil
}

func (l *RPCLoader) word(ctx context.Context, to, sig string, args ...[]byte) (*big.Int, error) {
	out, err := l.call(ctx, to, append(sel(sig), concat(args)...))
	if err != nil {
		return nil, err
	}
	if len(out) < 32 {
		return nil, fmt.Errorf("%s: short return", sig)
	}
	return new(big.Int).SetBytes(out[:32]), nil
}

func concat(bs [][]byte) []byte {
	var o []byte
	for _, b := range bs {
		o = append(o, b...)
	}
	return o
}

func wordAddr(w *big.Int) string { return common.BigToAddress(w).Hex() }

// signed interprets the low `bits` of w as two's complement.
func signedBits(w *big.Int, bits uint) *big.Int {
	m := new(big.Int).Lsh(big.NewInt(1), bits)
	v := new(big.Int).And(w, new(big.Int).Sub(m, big.NewInt(1)))
	if v.Bit(int(bits-1)) == 1 {
		v.Sub(v, m)
	}
	return v
}

func wordAt(b []byte, i int) *big.Int {
	if len(b) < 32*(i+1) {
		return new(big.Int)
	}
	return new(big.Int).SetBytes(b[32*i : 32*(i+1)])
}

// ---------------------------------------------------------------- ABI for dynamic returns

var (
	rpcABI = func() abi.ABI {
		a, err := abi.JSON(strings.NewReader(`[
{"name":"constituents","type":"function","inputs":[],"outputs":[{"type":"address[]"},{"type":"uint256[]"}]},
{"name":"getStock","type":"function","inputs":[{"type":"address"}],"outputs":[{"type":"tuple","components":[
 {"name":"pools","type":"tuple[]","components":[{"name":"currency0","type":"address"},{"name":"currency1","type":"address"},
  {"name":"fee","type":"uint24"},{"name":"tickSpacing","type":"int24"},{"name":"hooks","type":"address"}]},
 {"name":"priceIndex","type":"uint8"},{"name":"stable","type":"address"},{"name":"stockIsCurrency0","type":"bool"},
 {"name":"stockDecimals","type":"uint8"},{"name":"stableDecimals","type":"uint8"},{"name":"enabled","type":"bool"}]}]},
{"name":"venuesOf","type":"function","inputs":[{"type":"address"}],"outputs":[{"type":"tuple[]","components":[
 {"name":"pool","type":"address"},{"name":"family","type":"uint8"}]}]}
]`))
		if err != nil {
			panic(err)
		}
		return a
	}()
)

// ---------------------------------------------------------------- Loader

func (l *RPCLoader) Constituents(ctx context.Context, lot string) ([]string, []*big.Int, error) {
	out, err := l.call(ctx, lot, sel("constituents()"))
	if err != nil {
		return nil, nil, err
	}
	v, err := rpcABI.Methods["constituents"].Outputs.Unpack(out)
	if err != nil {
		return nil, nil, err
	}
	var stocks []string
	for _, a := range v[0].([]common.Address) {
		stocks = append(stocks, a.Hex())
	}
	return stocks, v[1].([]*big.Int), nil
}

func (l *RPCLoader) HookParams(ctx context.Context) (Params, error) {
	out, err := l.call(ctx, l.Hook, sel("getParams()"))
	if err != nil {
		return Params{}, err
	}
	return Params{BuySpreadBps: wordAt(out, 0).Uint64(), SellSpreadBps: wordAt(out, 1).Uint64(),
		BandBps: wordAt(out, 2).Uint64(), SizeBufferBps: wordAt(out, 3).Uint64(),
		WidthTicks: int(signedBits(wordAt(out, 4), 24).Int64())}, nil
}

// v4 StateLibrary slots
func poolStateSlot(poolID string) common.Hash {
	return crypto.Keccak256Hash(common.HexToHash(poolID).Bytes(), common.LeftPadBytes([]byte{6}, 32))
}

func slotPlus(s common.Hash, n int64) common.Hash {
	return common.BigToHash(new(big.Int).Add(s.Big(), big.NewInt(n)))
}

func (l *RPCLoader) extsload(ctx context.Context, slot common.Hash) (*big.Int, error) {
	return l.word(ctx, l.PM, "extsload(bytes32)", slot.Bytes())
}

// v4Slot0 decodes slot0: sqrtPriceX96 | tick (24) | protocolFee (24) | lpFee (24).
func (l *RPCLoader) v4Slot0(ctx context.Context, poolID string) (sqrtP *big.Int, tick int, protocolFee, lpFee uint32,
	err error) {
	w, err := l.extsload(ctx, poolStateSlot(poolID))
	if err != nil {
		return nil, 0, 0, 0, err
	}
	sqrtP = new(big.Int).And(w, new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 160), big.NewInt(1)))
	tick = int(signedBits(new(big.Int).Rsh(w, 160), 24).Int64())
	protocolFee = uint32(new(big.Int).And(new(big.Int).Rsh(w, 184), big.NewInt(0xffffff)).Uint64())
	lpFee = uint32(new(big.Int).And(new(big.Int).Rsh(w, 208), big.NewInt(0xffffff)).Uint64())
	return sqrtP, tick, protocolFee, lpFee, nil
}

func (l *RPCLoader) Venue(ctx context.Context, lot string) (*VenueSnapshot, error) {
	id, err := l.word(ctx, l.Hook, "venueOf(address)", encAddr(lot))
	if err != nil {
		return nil, err
	}
	pid := common.BigToHash(id).Hex()
	sqrtP, tick, _, _, err := l.v4Slot0(ctx, pid)
	if err != nil {
		return nil, err
	}
	vs := &VenueSnapshot{PoolID: pid, UsdgIs0: addrLess(l.Usdg, lot), UsdgDecimals: l.UsdgDec, SqrtPriceX96: sqrtP,
		Tick: tick}
	out, err := l.call(ctx, l.Hook, append(sel("standingOf(bytes32)"), common.BigToHash(id).Bytes()...))
	if err != nil {
		return nil, err
	}
	if liq := wordAt(out, 0); liq.Sign() != 0 {
		vs.Standing = &StandingPosition{Liquidity: liq, Lower: int(signedBits(wordAt(out, 1), 24).Int64()),
			Upper: int(signedBits(wordAt(out, 2), 24).Int64())}
	}
	return vs, nil
}

func (l *RPCLoader) NavGuardParams(ctx context.Context) (NavGuardParams, error) {
	out, err := l.call(ctx, l.NavGuard, sel("getParams()"))
	if err != nil {
		return NavGuardParams{}, err
	}
	return NavGuardParams{NavTolBps: wordAt(out, 0).Uint64(), StaleTolBps: wordAt(out, 1).Uint64(),
		MaxFeedAge: wordAt(out, 2).Uint64(), MinCoverageBps: wordAt(out, 3).Uint64()}, nil
}

func (l *RPCLoader) feedState(ctx context.Context, feed string) (*FeedState, error) {
	out, err := l.call(ctx, feed, sel("latestRoundData()"))
	if err != nil {
		return nil, err
	}
	dec, err := l.word(ctx, feed, "decimals()")
	if err != nil {
		return nil, err
	}
	return &FeedState{Answer: signedBits(wordAt(out, 1), 256), Decimals: int(dec.Int64()),
		UpdatedAt: wordAt(out, 3).Uint64()}, nil
}

func (l *RPCLoader) Feed(ctx context.Context, stock string) (string, *FeedState, error) {
	w, err := l.word(ctx, l.NavGuard, "feedOf(address)", encAddr(stock))
	if err != nil {
		return "", nil, err
	}
	feed := wordAddr(w)
	if isZeroAddr(feed) {
		return feed, nil, nil
	}
	fs, err := l.feedState(ctx, feed)
	return feed, fs, err
}

func (l *RPCLoader) Stock(ctx context.Context, stock string) (*StockSnapshot, error) {
	out, err := l.call(ctx, l.Registry, append(sel("getStock(address)"), encAddr(stock)...))
	if err != nil {
		return nil, err
	}
	v, err := rpcABI.Methods["getStock"].Outputs.Unpack(out)
	if err != nil {
		return nil, err
	}
	s := v[0].(struct {
		Pools []struct {
			Currency0   common.Address `json:"currency0"`
			Currency1   common.Address `json:"currency1"`
			Fee         *big.Int       `json:"fee"`
			TickSpacing *big.Int       `json:"tickSpacing"`
			Hooks       common.Address `json:"hooks"`
		} `json:"pools"`
		PriceIndex       uint8          `json:"priceIndex"`
		Stable           common.Address `json:"stable"`
		StockIsCurrency0 bool           `json:"stockIsCurrency0"`
		StockDecimals    uint8          `json:"stockDecimals"`
		StableDecimals   uint8          `json:"stableDecimals"`
		Enabled          bool           `json:"enabled"`
	})
	snap := &StockSnapshot{Registry: RegistryEntry{Stable: s.Stable.Hex(), PriceIndex: int(s.PriceIndex),
		StockIsCurrency0: s.StockIsCurrency0}, StockDecimals: int(s.StockDecimals), StableDecimals: int(s.StableDecimals)}
	for _, p := range s.Pools {
		id := crypto.Keccak256Hash(encAddr(p.Currency0.Hex()), encAddr(p.Currency1.Hex()), encWord(p.Fee),
			encWord(p.TickSpacing), encAddr(p.Hooks.Hex())).Hex()
		sqrtP, _, _, _, err := l.v4Slot0(ctx, id)
		if err != nil {
			return nil, err
		}
		liq, err := l.extsload(ctx, slotPlus(poolStateSlot(id), 3))
		if err != nil {
			return nil, err
		}
		liq.And(liq, maxU128)
		l.mu.Lock()
		l.keys[lc(id)] = rpcKey{fee: uint32(p.Fee.Uint64()), tickSpacing: int(p.TickSpacing.Int64()), hooks: p.Hooks.Hex()}
		l.mu.Unlock()
		snap.Registry.Pools = append(snap.Registry.Pools, RegistryPoolView{PoolID: id, Hooks: p.Hooks.Hex(),
			TickSpacing: int(p.TickSpacing.Int64()), SqrtPriceX96: sqrtP, Liquidity: liq})
	}
	if w, err := l.word(ctx, stock, "oraclePaused()"); err == nil {
		snap.OraclePaused = w.Sign() != 0
	} else if !readFailed(err) {
		return nil, err
	}
	if !sameAddr(snap.Registry.Stable, l.Usdg) {
		return snap, nil // WETH token on an MQ venue: no VenueBook sources (NavJitLegsMq branches before)
	}
	out, err = l.call(ctx, l.VenueBook, append(sel("venuesOf(address)"), encAddr(stock)...))
	if readFailed(err) {
		return snap, nil // venuesOf reverted: no v3 sources
	} else if err != nil {
		return nil, err
	}
	vv, err := rpcABI.Methods["venuesOf"].Outputs.Unpack(out)
	if err != nil {
		return nil, err
	}
	snap.VenuesOK = true
	for _, ven := range vv[0].([]struct {
		Pool   common.Address `json:"pool"`
		Family uint8          `json:"family"`
	}) {
		bv := BookVenueView{Pool: ven.Pool.Hex(), Family: int(ven.Family)}
		for _, r := range []struct {
			sig string
			set func(*big.Int)
		}{
			{"factory()", func(w *big.Int) { bv.Factory, bv.FactoryOK = wordAddr(w), true }},
			{"slot0()", func(w *big.Int) { bv.SqrtPriceX96, bv.SqrtOK = w, true }},
			{"liquidity()", func(w *big.Int) { bv.Liquidity, bv.LiqOK = w, true }},
			{"tickSpacing()", func(w *big.Int) { bv.TickSpacing, bv.TickSpacingOK = int(signedBits(w, 256).Int64()), true }},
		} {
			w, err := l.word(ctx, bv.Pool, r.sig)
			if err == nil {
				r.set(w)
			} else if !readFailed(err) {
				return nil, err
			}
		}
		snap.Venues = append(snap.Venues, bv)
	}
	return snap, nil
}

// window returns the compressed-tick word range a leg on a pool at sqrtP/tick can touch.
func (l *RPCLoader) window(sqrtP *big.Int, spacing int) (int, int, error) {
	if spacing <= 0 {
		return 0, 0, fmt.Errorf("tick spacing %d", spacing)
	}
	lo, err := TickAtSqrtPrice(l.Cfg.LegLimit(sqrtP, true))
	if err != nil {
		return 0, 0, err
	}
	hi, err := TickAtSqrtPrice(l.Cfg.LegLimit(sqrtP, false))
	if err != nil {
		return 0, 0, err
	}
	return (floorDivInt(lo, spacing) >> 8) - 1, (floorDivInt(hi, spacing) >> 8) + 1, nil
}

func floorDivInt(a, b int) int {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

func (l *RPCLoader) Pool(ctx context.Context, stock string, src Source) (*LegPool, error) {
	if src.Kind == KindV4 {
		return l.v4Pool(ctx, src.ID)
	}
	return l.v3Pool(ctx, src.Pool)
}

func (l *RPCLoader) v4Pool(ctx context.Context, id string) (*LegPool, error) {
	l.mu.Lock()
	k, ok := l.keys[lc(id)]
	l.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("unknown v4 pool %s", id)
	}
	sqrtP, tick, pf, lp, err := l.v4Slot0(ctx, id)
	if err != nil {
		return nil, err
	}
	st := poolStateSlot(id)
	liq, err := l.extsload(ctx, slotPlus(st, 3))
	if err != nil {
		return nil, err
	}
	liq.And(liq, maxU128)
	p := &LegPool{Kind: KindV4, SqrtPriceX96: sqrtP, Tick: tick, Liquidity: liq, TickSpacing: k.tickSpacing,
		LpFee: lp, ProtocolFee: pf, DynamicFee: k.fee == 0x800000}
	wlo, whi, err := l.window(sqrtP, k.tickSpacing)
	if err != nil {
		return nil, err
	}
	bitmapSlot := func(word int) []byte {
		return append([]byte{}, crypto.Keccak256(encWord(big.NewInt(int64(word))), slotPlus(st, 5).Bytes())...)
	}
	tickSlot := func(t int) []byte {
		return append([]byte{}, crypto.Keccak256(encWord(big.NewInt(int64(t))), slotPlus(st, 4).Bytes())...)
	}
	p.Ticks, err = l.loadTicks(ctx, wlo, whi, k.tickSpacing,
		func(word int) *callReq {
			return &callReq{to: l.PM, data: append(sel("extsload(bytes32)"), bitmapSlot(word)...)}
		},
		func(t int) *callReq {
			return &callReq{to: l.PM, data: append(sel("extsload(bytes32)"), tickSlot(t)...)}
		},
		func(out []byte) (*big.Int, *big.Int) {
			w := wordAt(out, 0)
			return new(big.Int).And(w, maxU128), signedBits(new(big.Int).Rsh(w, 128), 128)
		})
	return p, err
}

func (l *RPCLoader) v3Pool(ctx context.Context, pool string) (*LegPool, error) {
	out, err := l.call(ctx, pool, sel("slot0()"))
	if err != nil {
		return nil, err
	}
	sqrtP, tick := wordAt(out, 0), int(signedBits(wordAt(out, 1), 256).Int64())
	liq, err := l.word(ctx, pool, "liquidity()")
	if err != nil {
		return nil, err
	}
	fee, err := l.word(ctx, pool, "fee()")
	if err != nil {
		return nil, err
	}
	tsw, err := l.word(ctx, pool, "tickSpacing()")
	if err != nil {
		return nil, err
	}
	ts := int(signedBits(tsw, 256).Int64())
	p := &LegPool{Kind: KindV3, SqrtPriceX96: sqrtP, Tick: tick, Liquidity: liq, TickSpacing: ts,
		Fee: uint32(fee.Uint64())}
	wlo, whi, err := l.window(sqrtP, ts)
	if err != nil {
		return nil, err
	}
	p.Ticks, err = l.loadTicks(ctx, wlo, whi, ts,
		func(word int) *callReq {
			return &callReq{to: pool, data: append(sel("tickBitmap(int16)"), encWord(big.NewInt(int64(word)))...)}
		},
		func(t int) *callReq {
			return &callReq{to: pool, data: append(sel("ticks(int24)"), encWord(big.NewInt(int64(t)))...)}
		},
		func(out []byte) (*big.Int, *big.Int) { return wordAt(out, 0), signedBits(wordAt(out, 1), 256) })
	return p, err
}

// loadTicks reads bitmap words [wlo, whi] and every initialized tick in them.
func (l *RPCLoader) loadTicks(ctx context.Context, wlo, whi, spacing int, wordReq func(int) *callReq,
	tickReq func(int) *callReq, decode func([]byte) (*big.Int, *big.Int)) ([]TickU256Alias, error) {
	var reqs []*callReq
	for w := wlo; w <= whi; w++ {
		reqs = append(reqs, wordReq(w))
	}
	if err := l.batch(ctx, reqs); err != nil {
		return nil, err
	}
	var ticks []int
	for i, r := range reqs {
		if r.err != nil {
			return nil, fmt.Errorf("bitmap word %d: %w", wlo+i, r.err)
		}
		bm := wordAt(r.out, 0)
		for b := 0; b < 256; b++ {
			if bm.Bit(b) == 1 {
				ticks = append(ticks, ((wlo+i)*256+b)*spacing)
			}
		}
	}
	treqs := make([]*callReq, len(ticks))
	for i, t := range ticks {
		treqs[i] = tickReq(t)
	}
	if err := l.batch(ctx, treqs); err != nil {
		return nil, err
	}
	out := make([]TickU256Alias, 0, len(ticks))
	for i, r := range treqs {
		if r.err != nil {
			return nil, fmt.Errorf("tick %d: %w", ticks[i], r.err)
		}
		gross, net := decode(r.out)
		out = append(out, newTick(ticks[i], gross, net))
	}
	return out, nil
}

func (l *RPCLoader) PoolManagerLotBalance(ctx context.Context, lot string) (*big.Int, error) {
	return l.word(ctx, lot, "balanceOf(address)", encAddr(l.PM))
}

func (l *RPCLoader) BlockTimestamp(ctx context.Context) (uint64, error) {
	l.mu.Lock()
	ts := l.ts
	l.mu.Unlock()
	if ts != 0 {
		return ts, nil
	}
	ts, err := blockTimestamp(ctx, l.c, l.Block)
	if err == nil {
		l.mu.Lock()
		l.ts = ts
		l.mu.Unlock()
	}
	return ts, err
}

// blockTimestamp reads a block header's timestamp (tag: a hex number or "latest").
func blockTimestamp(ctx context.Context, c *rpc.Client, tag string) (uint64, error) {
	var h *struct {
		Timestamp hexutil.Uint64 `json:"timestamp"`
	}
	if err := c.CallContext(ctx, &h, "eth_getBlockByNumber", tag, false); err != nil {
		return 0, &TransportError{Err: err}
	}
	if h == nil {
		return 0, &TransportError{Err: fmt.Errorf("block %s not found", tag)}
	}
	return uint64(h.Timestamp), nil
}

// ---------------------------------------------------------------- MQ

// MqLoader is the optional Loader extension for an MQ venue (hook #2: its registry is the MqRegistry).
type MqLoader interface {
	MqMarket(ctx context.Context) (*MqMarket, error)
	MqToken(ctx context.Context, stock string, q *MqMarket) (*MqToken, *LegPool, error)
}

func (l *RPCLoader) twap(ctx context.Context, pool string) (*TwapState, error) {
	out, err := l.call(ctx, pool, sel("slot0()"))
	if err != nil {
		return nil, err
	}
	s := &TwapState{Tick: int(signedBits(wordAt(out, 1), 256).Int64()), Index: int(wordAt(out, 2).Int64()),
		Cardinality: int(wordAt(out, 3).Int64()), Obs: map[int]Observation{}}
	s.Fetch = func(i int) (Observation, error) {
		r := observationReq(pool, i)
		o, err := l.call(ctx, r.to, r.data)
		if err != nil {
			return Observation{}, err
		}
		return Observation{Timestamp: wordAt(o, 0).Uint64(), TickCumulative: signedBits(wordAt(o, 1), 256),
			Initialized: wordAt(o, 3).Sign() != 0}, nil
	}
	return s, nil
}

func (l *RPCLoader) MqMarket(ctx context.Context) (*MqMarket, error) {
	out, err := l.call(ctx, l.Registry, sel("mqRoute()"))
	if err != nil {
		return nil, err
	}
	weth, conv := wordAddr(wordAt(out, 0)), wordAddr(wordAt(out, 1))
	q := &MqMarket{Weth: weth, ConvPool: conv, Usdg: l.Usdg, UsdgDecimals: l.UsdgDec}
	if w, err := l.word(ctx, conv, "factory()"); err == nil {
		q.ConvFactory = wordAddr(w)
	} else if !readFailed(err) {
		return nil, err
	}
	if q.Conv, err = l.v3Pool(ctx, conv); err != nil {
		return nil, err
	}
	if q.ConvTwap, err = l.twap(ctx, conv); err != nil {
		return nil, err
	}
	if q.PmWeth, err = l.word(ctx, weth, "balanceOf(address)", encAddr(l.PM)); err != nil {
		return nil, err
	}
	return q, nil
}

// MqToken reads MqRegistry.pricingInput(stock) (pool, tokenIs0, tokenDecimals, conv, wethIs0, usdgDecimals,
// ethUsdFeed, usdgUsdFeed) and the token/WETH pool; it fills the feeds on q.
func (l *RPCLoader) MqToken(ctx context.Context, stock string, q *MqMarket) (*MqToken, *LegPool, error) {
	out, err := l.call(ctx, l.Registry, append(sel("pricingInput(address)"), encAddr(stock)...))
	if err != nil {
		return nil, nil, err
	}
	pool := wordAddr(wordAt(out, 0))
	t := &MqToken{Pool: pool, TokenIs0: wordAt(out, 1).Sign() != 0, TokenDecimals: int(wordAt(out, 2).Int64())}
	q.EthUsdFeed, q.UsdgUsdFeed = wordAddr(wordAt(out, 6)), wordAddr(wordAt(out, 7))
	if q.EthUsd, err = l.feedState(ctx, q.EthUsdFeed); err != nil {
		return nil, nil, err
	}
	if q.UsdgUsd, err = l.feedState(ctx, q.UsdgUsdFeed); err != nil {
		return nil, nil, err
	}
	if w, err := l.word(ctx, pool, "factory()"); err == nil {
		t.Factory = wordAddr(w)
	} else if !readFailed(err) {
		return nil, nil, err
	}
	if t.Twap, err = l.twap(ctx, pool); err != nil {
		return nil, nil, err
	}
	p, err := l.v3Pool(ctx, pool)
	if err != nil {
		return nil, nil, err
	}
	p.Kind = KindV3W
	return t, p, nil
}

// hexOf is a debugging aid for revert data.
func hexOf(b []byte) string { return "0x" + hex.EncodeToString(b) }

// TickU256Alias is dex-lib's v3 tick record (the leg engine's input).
type TickU256Alias = uniswapv3.TickU256

func newTick(t int, gross, net *big.Int) TickU256Alias {
	return TickU256Alias{Index: t, LiquidityGross: uint256.MustFromBig(gross), LiquidityNet: int256.MustFromBig(net)}
}
