package lotflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// LOTFLOW v1, the dex-lib adapter (A8): NavJitHookV17 venues as uniswap-v4 hook pools. Unlike PR 1740's `navjit`
// (a sampled V4Quoter ladder), this quotes exactly: Track loads the venue's whole dependency state
// (basket, constituents' candidate pools, NavGuard / MQ pricing inputs, hook Params, standing position,
// PoolManager float) and BeforeSwap runs the hook's swap on it (Quote). The quote is the CONSERVATIVE
// one (QuoteConservative): a probe within the gas envelope of PROBE_GAS is taken at its worse outcome,
// so the adapter never promises more than the chain delivers. Exact-input only (Kyber's v4 legs are
// exact-input); CalcAmountIn is refused. Swaps are priced prefunded (Kyber's executor settles before
// the swap, MEASURED P1-KX).

// Exchange is the proposed dex-lib exchange id (uniswap-v4-lotflow). It must be registered in pkg/valueobject/exchange.go
// (with the uniswap-v4 hook sources) for pool-service; PR 1740's `uniswap-v4-navjit` stays untouched.
const Exchange = valueobject.ExchangeUniswapV4LotFlow

// HookAddresses: the live NavJitHookV17 deployments on Robinhood Chain (4663), PoolManager
// 0x8366a39cc670b4001a1121b8f6a443a643e40951 (Lots review deployment, 2026-10-09).
var HookAddresses = []common.Address{
	common.HexToAddress("0xb672a5d825F82dEA8574711d1dA696FcC48d2AE0"), // NavJitHookV17 (BasketFactory V15 baskets)
	common.HexToAddress("0x447184450e4615c2ec2545540B53F213fB9c2aE0"), // NavJitHookV17 #2 (MQ: WETH-quoted baskets)
}

// MaxAgeSec bounds a tracked state's age under StaleCheck (route finding).
const MaxAgeSec = 60

var (
	ErrNotTracked    = errors.New("lotflow: venue not tracked")
	ErrStale         = errors.New("lotflow: tracked state too old")
	ErrExactOut      = errors.New("lotflow: exact-output quotes are not supported (exact-input venue)")
	ErrBadAmount     = errors.New("lotflow: invalid amount")
	ErrNotAVenue     = errors.New("lotflow: pool is not this hook's LOT/USDG venue")
	ErrZeroOutput    = errors.New("lotflow: zero output")
	ErrPriceLimit    = errors.New("lotflow: invalid NAV band price limit")
	ErrPartialFill   = errors.New("lotflow: price limit prevents a full input fill")
	ErrOverrides     = errors.New("lotflow: RPC state overrides are not supported")
	ErrCurrencyOrder = errors.New("lotflow: LOT-as-currency0 venues are not supported")
	// ErrUnsupportedChain: the hooks are deployed on Robinhood Chain (4663) only.
	ErrUnsupportedChain = errors.New("lotflow: chain not supported (Robinhood Chain 4663 only)")
	// ErrSpreadOutsideLimit: the direction's spread reaches the buffered NAV band (spread >=
	// bandBps-25), so no fill completes inside the executor's price limit.
	ErrSpreadOutsideLimit = errors.New("lotflow: spread outside the NAV band price limit")
)

// Extra is the hook state Track stores in the pool's extra.
type Extra struct {
	Tracked   bool    `json:"tracked"`
	TrackedAt int64   `json:"trackedAt"`
	Block     uint64  `json:"block"`
	Lot       string  `json:"lot"`
	Usdg      string  `json:"usdg"`
	V3Factory string  `json:"v3Factory"`
	Market    *Market `json:"market"`
}

type Hook struct {
	*uniswapv4.BaseHook
	Extra
	staleCheck bool
}

// SwapInfo is what BeforeSwap hands to UpdateBalance.
type SwapInfo struct {
	res *SwapResult
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	h := &Hook{BaseHook: &uniswapv4.BaseHook{Exchange: Exchange}, staleCheck: param.StaleCheck}
	// An extra that does not decode, or decodes into a state Track cannot have produced, is untracked.
	// So is an extra on another chain (a nil Cfg is a caller without chain context, as in evplusai).
	if err := decodeExtra(param.HookExtra, &h.Extra); err != nil || validateExtra(&h.Extra) != nil ||
		(param.Cfg != nil && param.Cfg.ChainID != valueobject.ChainIDRobinhood) {
		h.Extra = Extra{}
	}
	return h
}, HookAddresses...)

// decodeExtra is HookExtra.Unmarshal behind a recover: goccy/go-json v0.10.5 can panic (index out of
// range, internal/decoder/context.go:49) on a short truncated extra with an escape in a key (FuzzHookExtra
// corpus 8e6fd1838d839f3e) when it follows a longer extra; not in isolation, as the decoder reads past
// the input into its pooled buffer (TestDecodeExtraRecoversDecoderPanic).
func decodeExtra(raw uniswapv4.HookExtra, x *Extra) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: decoder panic: %v", errInvalid, r)
		}
	}()
	return raw.Unmarshal(x)
}

func (h *Hook) cfg() *Config { return DefaultConfig(h.Usdg, h.V3Factory, FinalisedRules) }

// AllowEmptyTicks: BeforeSwap takes the whole amount, so the pool's own CL math always runs on a zero
// remainder (a venue without standing liquidity has no ticks).
func (h *Hook) AllowEmptyTicks() bool { return true }

const priceLimitBufferBps = 25

var _ uniswapv4.HookPriceLimitProvider = (*Hook)(nil)

// SqrtPriceLimit follows the NAV-band limit introduced for V15 in #1740. Standing ticks do not
// bound JIT liquidity. BeforeSwap uses this same limit and refuses any partial fill.
func (h *Hook) SqrtPriceLimit(zeroForOne bool) *uint256.Int {
	limit, err := h.priceLimit(zeroForOne)
	if err != nil {
		return nil
	}
	return limit
}

func (h *Hook) priceLimit(zeroForOne bool) (_ *uint256.Int, err error) {
	defer recoverNumeric(&err)
	if !h.Tracked || h.Market == nil || h.Market.Venue == nil {
		return nil, ErrNotTracked
	}
	v := h.Market.Venue
	// The deployed LOT-as-currency0 branch has not been validated. Do not advertise it through
	// a limit derived from the inverse branch while the on-chain band calculation differs.
	if !v.UsdgIs0 {
		return nil, ErrCurrencyOrder
	}
	if v.UsdgDecimals < 0 || v.UsdgDecimals > 18 || v.Params.BandBps <= priceLimitBufferBps || v.Params.BandBps >= 10_000 {
		return nil, ErrPriceLimit
	}
	// The edge is placed spread bps from NAV (sells) or from the executed cost (buys); at or past the
	// buffered band nothing fills inside the limit, so refuse by name rather than by the limit itself.
	spread := v.Params.SellSpreadBps
	if zeroForOne == v.UsdgIs0 { // buy
		spread = v.Params.BuySpreadBps
	}
	if spread >= v.Params.BandBps-priceLimitBufferBps {
		return nil, ErrSpreadOutsideLimit
	}
	nav, err := newMarketState(h.Market).checkedNav()
	if err != nil {
		return nil, err
	}
	offset := v.Params.BandBps - priceLimitBufferBps
	price := mulDiv(nav, new(big.Int).SetUint64(10_000+offset), bigBps)
	if !zeroForOne {
		price = mulDivUp(nav, new(big.Int).SetUint64(10_000-offset), bigBps)
	}
	if price.Sign() <= 0 {
		return nil, ErrPriceLimit
	}
	numerator := new(big.Int).Mul(pow10(36-v.UsdgDecimals), bigQ192)
	ratio := new(big.Int).Quo(numerator, price)
	limit := new(big.Int).Sqrt(ratio)
	if zeroForOne && new(big.Int).Mul(new(big.Int).Mul(limit, limit), price).Cmp(numerator) < 0 {
		limit.Add(limit, big.NewInt(1)) // lower bound: round up, toward NAV
	}
	if limit.Cmp(MinSqrtPrice) <= 0 || limit.Cmp(MaxSqrtPrice) >= 0 {
		return nil, ErrPriceLimit
	}
	u, overflow := uint256.FromBig(limit)
	if overflow {
		return nil, ErrPriceLimit
	}
	return u, nil
}

func (h *Hook) quote(buy bool, amount *big.Int) (_ *SwapResult, err error) {
	defer recoverNumeric(&err)
	limit, err := h.priceLimit(h.Market.Venue.ZeroForOne(buy))
	if err != nil {
		return nil, err
	}
	r, err := QuoteConservative(h.cfg(), h.Market, SwapInput{BuyLot: buy, AmountIn: new(big.Int).Set(amount),
		Prefund: true, SqrtPriceLimitX96: limit.ToBig()})
	if err != nil {
		return nil, err
	}
	if r.Venue.AmountIn.Cmp(amount) != 0 {
		return nil, ErrPartialFill
	}
	return r, nil
}

// ---------------------------------------------------------------- gas

const (
	gasRound = 100_000
	// MqSwapGas: MQ venues (hook #2), from our measured minimum passing limits 3.44-3.77M (Kyber executor on a fork).
	MqSwapGas = 4_200_000
)

// SwapGas is the gas reported for one swap on a venue of n constituents. INFERRED, conservative:
// KYBER-PACK §5's fit (1.6M + 0.45M x constituents + 0.15M x probed candidates) with the probe count
// bounded by MAX_CANDIDATES (3) per constituent, times 1.10 plus 100k (the margin KYBER-PACK's own
// constants use), rounded up to 0.1M. MQ venues report MqSwapGas. TestSwapGasCoversMeasured pins it
// above every minimum passing limit we measured with Kyber's real executor on a fork.
func SwapGas(constituents int, mq bool) int64 {
	if mq {
		return MqSwapGas
	}
	n := int64(constituents)
	fit := 1_600_000 + 450_000*n + 150_000*3*n
	g := fit*110/100 + 100_000
	return (g + gasRound - 1) / gasRound * gasRound
}

// GasFeatures are the parts of a simulated swap that drive its gas (SwapGasFor): the venue's shape, the
// direction, and from the swap's legs the candidates probed, the initialized ticks the executed legs
// cross, and the leg-gas upper bounds (probeGas.go) of the probes and of the executed legs.
type GasFeatures struct {
	Constituents int    `json:"constituents"`
	Mq           bool   `json:"mq"`
	Buy          bool   `json:"buy"`
	Legs         int    `json:"legs"`
	Probes       int    `json:"probes"`
	Ticks        int    `json:"ticks"`
	ProbeGasHi   uint64 `json:"probeGasHi"`
	LegGasHi     uint64 `json:"legGasHi"`
	// StipendProbes: probes that may exhaust PROBE_GAS (ExecutedLeg.StipendProbes).
	StipendProbes int `json:"stipendProbes"`
}

func gasFeatures(m *Market, buy bool, r *SwapResult) GasFeatures {
	f := GasFeatures{Constituents: len(m.Constituents), Mq: m.Mq != nil, Buy: buy, Legs: len(r.Legs)}
	for _, l := range r.Legs {
		f.Probes += l.Probes
		f.Ticks += l.CrossedTicks
		f.ProbeGasHi += l.ProbeGasHi
		f.LegGasHi += l.GasHi
		f.StipendProbes += l.StipendProbes
	}
	return f
}

// Swap gas model (SwapGasFor). Fitted to 30 minimum passing gas limits MEASURED with Kyber's router and
// executor on anvil forks of chain 4663 (testdata/gas/measured.json: both deployed venues, both directions,
// $1 to the largest accepted fill), with every case covered at a 12% margin. What drives the limit is the
// probes, not gasUsed: each probe needs its PROBE_GAS stipend available (63/64 per call frame), so the
// limit sits 0.7M-1.9M above gasUsed and grows with the gas the probes and legs before the last one use.
const (
	gasPerProbeGasHiPct = 45     // % of the probes' summed leg-gas upper bounds
	gasPerTick          = 50_000 // per initialized tick the executed legs cross
	// probeCallerGas is NavJitLegs.PROBE_CALLER_GAS (NavJitLegs.sol:99): what the caller must hold before a
	// probe, the PROBE_GAS stipend plus the frames the contract wraps it in.
	probeCallerGas = (probeStipend*64/63+probeMargin)*4096/3969 + 60_000
	probeStipend   = 1_200_000 // NavJitLegs.PROBE_GAS
	probeMargin    = 50_000    // NavJitLegs.PROBE_MARGIN
	// gasPerStipendProbe (D4-2): a probe that may exhaust its stipend burns all of PROBE_GAS on chain
	// (NavJitLegs.sol:87), not the 45% share the fit gives a filling probe. Its ProbeGasHi share is at
	// least 45% of PROBE_GAS - InexactGasMargin (it is counted only when its bound reaches that), so
	// this tops the probe up to PROBE_CALLER_GAS in full. Not fitted: no measured fork case has one.
	gasPerStipendProbe = probeCallerGas - (probeStipend-100_000)*gasPerProbeGasHiPct/100
)

// gasBase: the intercept per venue kind and direction ([mq][buy]).
var gasBase = [2][2]int64{{3_970_000, 3_380_000}, {4_000_000, 3_620_000}}

// SwapGasFor is the gas reported for one swap with features f: the fitted model plus the full cost of
// every probe that may exhaust its stipend, never below the flat SwapGas (which covers lane B's
// 5-constituent fork measurements), rounded up to 0.1M.
func SwapGasFor(f GasFeatures) int64 {
	mq, buy := 0, 0
	if f.Mq {
		mq = 1
	}
	if f.Buy {
		buy = 1
	}
	g := gasBase[mq][buy] + int64(f.ProbeGasHi)*gasPerProbeGasHiPct/100 + int64(f.Ticks)*gasPerTick +
		int64(f.StipendProbes)*gasPerStipendProbe
	return max(SwapGas(f.Constituents, f.Mq), (g+gasRound-1)/gasRound*gasRound)
}

// ---------------------------------------------------------------- quoting

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if !h.Tracked || h.Market == nil || h.Market.Venue == nil {
		return nil, ErrNotTracked
	}
	if h.staleCheck && time.Now().Unix()-h.TrackedAt > MaxAgeSec {
		return nil, ErrStale
	}
	if !params.CalcOut {
		return nil, ErrExactOut
	}
	if params.AmountSpecified == nil || params.AmountSpecified.Sign() <= 0 {
		return nil, ErrBadAmount
	}
	buy := params.ZeroForOne == h.Market.Venue.UsdgIs0
	res, err := h.quote(buy, params.AmountSpecified)
	if err != nil {
		return nil, err
	}
	if res.AmountOut.Sign() <= 0 {
		return nil, ErrZeroOutput
	}
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   new(big.Int).Set(params.AmountSpecified),
		DeltaUnspecified: new(big.Int).Neg(res.AmountOut),
		// The outer v4 simulator adds its base gas even when the hook consumes the whole input.
		Gas:      max(SwapGasFor(gasFeatures(h.Market, buy, res))-uniswapv4.DefaultGas.BaseGas, 0),
		SwapInfo: &SwapInfo{res: res},
	}, nil
}

// CloneState copies the hook with its own Market (constituent pool maps are copied; pool states are
// never mutated in place, so sharing them is safe).
func (h *Hook) CloneState() uniswapv4.Hook {
	c := *h
	c.Market = cloneMarket(h.Market)
	return &c
}

// UpdateBalance moves the tracked state past a swap the route executed: constituent pools, the
// venue's slot0, standing position and PoolManager float (the same threading A5 proves exact).
func (h *Hook) UpdateBalance(swapInfo any) {
	info, ok := swapInfo.(*SwapInfo)
	if !ok || info == nil || info.res == nil || info.res.PostSqrtPriceX96 == nil || info.res.PmLotAfter == nil ||
		h.Market == nil || h.Market.Venue == nil {
		return // only a BeforeSwap result on a tracked market moves the state
	}
	w := NewWorld()
	w.Absorb(h.Market, info.res)
	m := cloneMarket(h.Market)
	w.Overlay(m)
	h.Market = m
}

func cloneMarket(m *Market) *Market {
	if m == nil {
		return nil
	}
	c := *m
	if m.Venue != nil {
		v := *m.Venue
		c.Venue = &v
	}
	c.Constituents = make([]MarketConstituent, len(m.Constituents))
	for i, mc := range m.Constituents {
		mc.Pools = make(map[string]*LegPool, len(m.Constituents[i].Pools))
		for id, p := range m.Constituents[i].Pools {
			mc.Pools[id] = p
		}
		c.Constituents[i] = mc
	}
	return &c
}

// GetReserves reports what the venue can deliver: the LOT out of a $10k buy and the USDG out of a
// 10k-LOT sell at the tracked state (smaller sizes if those refuse), each in its currency's slot.
func (h *Hook) GetReserves(context.Context, *uniswapv4.HookParam) (_ entity.PoolReserves, err error) {
	defer recoverNumeric(&err)
	if !h.Tracked {
		return nil, nil
	}
	if h.Market == nil || h.Market.Venue == nil {
		return entity.PoolReserves{"0", "0"}, nil
	}
	out := func(buy bool, unit *big.Int) string {
		for _, k := range []int64{10_000, 1_000, 100, 1} {
			r, err := h.quote(buy, new(big.Int).Mul(big.NewInt(k), unit))
			if err == nil {
				return r.AmountOut.String()
			}
		}
		return "0"
	}
	lotRes := out(true, pow10(h.Market.Venue.UsdgDecimals))
	usdgRes := out(false, bigE18)
	if h.Market.Venue.UsdgIs0 {
		return entity.PoolReserves{usdgRes, lotRes}, nil
	}
	return entity.PoolReserves{lotRes, usdgRes}, nil
}

// ---------------------------------------------------------------- tracking

// Track loads the venue's state through RPCLoader on the pool-service RPC client, at param.BlockNumber
// (resolved once if nil, so every read is one snapshot). State overrides are rejected, not ignored.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	started := time.Now() // TrackedAt: the StaleCheck age counts from before the reads
	if len(param.Overrides) != 0 {
		return nil, ErrOverrides
	}
	if param.Cfg == nil || param.Cfg.ChainID != valueobject.ChainIDRobinhood {
		return nil, ErrUnsupportedChain
	}
	p := param.Pool
	if p == nil || len(p.Tokens) < 2 || param.RpcClient == nil {
		return nil, ErrNotAVenue
	}
	ec := param.RpcClient.GetETHClient()
	if ec == nil {
		return nil, ErrNotAVenue
	}
	callerPinned := param.BlockNumber != nil
	if !callerPinned {
		bn, err := ec.BlockNumber(ctx)
		if err != nil {
			return nil, &TransportError{Err: err}
		}
		pinned := *param
		pinned.BlockNumber = new(big.Int).SetUint64(bn)
		param = &pinned
	}
	block := hexutil.EncodeBig(param.BlockNumber)
	// Before any state read: a caller's pin must be recent (checkPinAge). The header is read once.
	pinnedTs, err := blockTimestamp(ctx, ec.Client(), block)
	if err != nil {
		return nil, err
	}
	if callerPinned {
		if err := checkPinAge(ctx, ec.Client(), block, pinnedTs); err != nil {
			return nil, err
		}
	}
	hook := hexutil.Encode(param.HookAddress[:])
	mc := newMulticaller(param.RpcClient, param.BlockNumber)
	// Round 0: the hook's wiring and both tokens' venues.
	wiring := []*callReq{}
	for _, sig := range []string{"poolManager()", "registry()", "venueBook()", "navGuard()", "usdg()",
		"usdgDecimals()", "v3Factory()"} {
		wiring = append(wiring, &callReq{to: hook, data: sel(sig)})
	}
	for _, t := range p.Tokens {
		wiring = append(wiring, &callReq{to: hook, data: append(sel("venueOf(address)"), encAddr(t.Address)...)})
	}
	if err := mc.fetch(ctx, wiring); err != nil {
		return nil, err
	}
	pmW, err := (&RPCLoader{c: ec.Client(), Block: block, mc: mc}).word(ctx, hook, "poolManager()")
	if err != nil {
		return nil, err
	}
	cfg := DefaultConfig("", "", FinalisedRules)
	l, err := newRPCLoader(ctx, ec.Client(), mc, block, hook, wordAddr(pmW), cfg)
	if err != nil {
		return nil, err
	}
	l.ts = pinnedTs
	var lot string
	for _, t := range p.Tokens {
		if !strings.EqualFold(t.Address, l.Usdg) {
			lot = t.Address
		}
	}
	id, err := l.word(ctx, hook, "venueOf(address)", encAddr(lot))
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(common.BigToHash(id).Hex(), p.Address) {
		return nil, ErrNotAVenue
	}
	tr := &Tracker{Cfg: cfg, Hook: hook, PoolManager: l.PM, NavGuard: l.NavGuard, Registry: l.Registry,
		VenueBook: l.VenueBook}
	if err := l.prefetch(ctx, tr, lot); err != nil {
		return nil, err
	}
	raw, err := h.trackMarket(ctx, param, tr, l, lot, started)
	st := mc.Stats()
	if w := statsWarning(st); w != "" {
		logger.Warnf("lotflow: Track %s at block %s: %s", p.Address, block, w)
	} else {
		logger.Debugf("lotflow: Track %s at block %s: multicall %+v", p.Address, block, st)
	}
	if trackStats != nil {
		trackStats(st)
	}
	return raw, err
}

// statsWarning is what a Track's multicall counts say worth a warning: reads the collect passes did
// not predict are fetched one request each, so a loader change that defeats prediction shows here
// before it shows as cost.
func statsWarning(s MulticallStats) string {
	if s.Misses == 0 {
		return ""
	}
	return fmt.Sprintf("%d reads not predicted by the collect passes (one request each); %+v", s.Misses, s)
}

// recoverNumeric is the boundary that keeps a numeric panic in the simulator (a state the structural
// validation accepts but the math cannot price) from escaping to pool-service: the call fails with
// ErrNumeric instead, which refuses the quote or, in Track, publishes the empty market.
func recoverNumeric(err *error) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("%w: %v", ErrNumeric, r)
	}
}

func checkedNavOf(m *Market) (_ *big.Int, err error) {
	defer recoverNumeric(&err)
	return newMarketState(m).checkedNav()
}

// checkPinAge refuses a caller-supplied block more than MaxAgeSec behind the chain head.
// pool-service pins Track at the block number its multicall returns (uniswap/v4 pool_tracker.go:117).
// On Robinhood Chain a contract's block.number is the L1 block number, so a multicall that reports it
// (canonical Multicall3) pins an old L2 block; ArbMulticall2 reports the L2 number. Without this check
// an archive node would serve that old state and Track would publish it with a current TrackedAt.
func checkPinAge(ctx context.Context, c *rpc.Client, block string, pinned uint64) error {
	head, err := blockTimestamp(ctx, c, "latest")
	if err != nil {
		return err
	}
	if head > pinned && head-pinned > MaxAgeSec {
		return fmt.Errorf("%w: block %s is %d s behind the head", ErrStalePin, block, head-pinned)
	}
	return nil
}

// trackStats, when set (tests), receives each Track's multicall counts.
var trackStats func(MulticallStats)

// twapPrefetcher is the optional Loader extension that warms MQ oracle rings before the checked NAV.
type twapPrefetcher interface {
	PrefetchTwap(ctx context.Context, m *Market) error
}

// trackMarket loads the venue's market through tr and l and returns the hook extra, stamped with the
// time the tracking started.
func (h *Hook) trackMarket(ctx context.Context, param *uniswapv4.HookParam, tr *Tracker, l Loader,
	lot string, started time.Time) (json.RawMessage, error) {
	// A transport failure publishes nothing (the error is returned). Anything else that stops the
	// market (a read that reverted on chain, a failed NAV check) publishes an empty market, as V15
	// publishes empty ladders, so a previously valid quote cannot survive it.
	var market *Market
	st, err := tr.Load(ctx, l, lot)
	if isTransport(err) {
		return nil, err
	}
	if err != nil && ctx.Err() != nil { // reads cut short by the caller are not the chain's answer
		return nil, &TransportError{Err: err}
	}
	if err == nil {
		market = st.Market
		if tp, ok := l.(twapPrefetcher); ok {
			if err := tp.PrefetchTwap(ctx, market); err != nil {
				return nil, err
			}
		}
		// This also completes the MQ observation reads before serialization.
		if _, err := checkedNavOf(market); isTransport(err) {
			return nil, err
		} else if err != nil && ctx.Err() != nil {
			return nil, &TransportError{Err: err}
		} else if err != nil {
			market = nil
		}
	}
	// What Track publishes is what the factory accepts: a market that fails validation (a constituent
	// left with no pools after the keeper removed its venues; the chain reverts NoExecutableSource)
	// is published empty, so this instance and a factory-built one both refuse.
	if market != nil && validateMarket(market) != nil {
		market = nil
	}
	var bn uint64
	if param.BlockNumber != nil {
		bn = param.BlockNumber.Uint64()
	}
	next := Extra{Tracked: true, TrackedAt: started.Unix(), Block: bn, Lot: lot, Usdg: tr.Cfg.Usdg,
		V3Factory: tr.Cfg.V3Factory, Market: market}
	raw, err := json.Marshal(next)
	if err == nil {
		h.Extra = next // GetReserves runs on this instance in the outer v4 tracker.
	}
	return raw, err
}
