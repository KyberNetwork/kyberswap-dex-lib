package lotflow

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// The MQ path (A6): baskets that hold WETH-quoted tokens, on the second NavJitHookV17 whose legs
// library is NavJitLegsMq (MemeFiLP contracts/src/libraries/NavJitLegsMq.sol) and whose NavGuard prices
// through MqFactory.spotUsd18 (contracts/src/mq/MqPricing.sol).
//
//   - Price: usdg18(token) = TWAP_1800(token/WETH) x TWAP_1800(USDG/WETH conversion pool), the
//     conversion TWAP checked against Chainlink ETH/USD / USDG/USD within 300 bps; feeds older than
//     MAX_FEED_AGE (90,000 s) or in the future are refused. TWAPs come from v3 Oracle.observe over the
//     pool's observation ring at the swap's block.timestamp (the price moves with time alone).
//   - Source: kind 3, one per token (the token's token/WETH pool), single candidate, never probed.
//     Admission: registry stable == WETH, both pools pass the v3 reads (factory, price, liquidity),
//     the token/WETH pool tickSpacing >= 10; the conversion pool is exempt from the tickSpacing floor
//     (owner decision 2026-10-08, "Keep exemption").
//   - Leg: two V3Leg hops, each bounded at LEG_LIMIT_BPS of its own pool's slot0. Buy exact-out:
//     WETH->token for `amount` (WETH taken physically from the PoolManager: needs PM WETH >= w), then
//     USDG->WETH exact-out w. Sell exact-in: token->WETH, then WETH->USDG. Every hop must fill
//     exactly. The conversion pool is shared by every WETH-token leg of the swap.

const (
	KindV3W         = 3 // NavJitLegsMq.KIND_V3W
	MqTwapWindow    = 1800
	MqMaxFeedAge    = 90_000
	MqCheckTolBps   = 300
	mqWethDecimals  = 18
	observationsMax = 65535
)

// MqError mirrors one of MqPricing's custom errors (name, signature, decimal/address args).
type MqError struct {
	Name string
	sig  string
	Args []string
}

func (e *MqError) Error() string {
	return fmt.Sprintf("lotflow: %s(%s)", e.Name, strings.Join(e.Args, ", "))
}
func (e *MqError) ErrorName() string      { return e.Name }
func (e *MqError) ErrorArgs() []string    { return e.Args }
func (e *MqError) ErrorSignature() string { return e.sig }

func mqErr(name, sig string, args ...any) *MqError {
	e := &MqError{Name: name, sig: sig}
	for _, a := range args {
		switch v := a.(type) {
		case string:
			e.Args = append(e.Args, strings.ToLower(v))
		default:
			e.Args = append(e.Args, fmt.Sprint(v))
		}
	}
	return e
}

// ---------------------------------------------------------------- v3 Oracle.observe

// Observation is one v3 observation-ring slot (only what tickCumulative needs).
type Observation struct {
	Timestamp      uint64
	TickCumulative *big.Int // int56
	Initialized    bool
}

// TwapState is a v3 pool's oracle state: slot0 tick, observationIndex, observationCardinality and the
// ring slots. A slot the binary search needs but the state lacks is an error (never a guess).
type TwapState struct {
	Tick        int
	Index       int
	Cardinality int
	Obs         map[int]Observation
	// Fetch, when set, loads a ring slot the map lacks (an RPC loader reads observations lazily: the
	// binary search touches ~log2(cardinality) slots).
	Fetch func(i int) (Observation, error) `json:"-" msgpack:"-"`
}

func (s *TwapState) slot(i int) (Observation, error) {
	o, ok := s.Obs[i]
	if !ok {
		if s.Fetch == nil {
			return Observation{}, fmt.Errorf("lotflow: observation slot %d not loaded", i)
		}
		f, err := s.Fetch(i)
		if err != nil {
			return Observation{}, err
		}
		s.Obs[i] = f
		return f, nil
	}
	return o, nil
}

// lte is Oracle.lte: a <= b for 32-bit timestamps relative to `time` (wrap-safe).
func lte(time, a, b uint64) bool {
	const m = 1 << 32
	time, a, b = time%m, a%m, b%m
	if a <= time && b <= time {
		return a <= b
	}
	aAdj, bAdj := a, b
	if a <= time {
		aAdj = a + m
	}
	if b <= time {
		bAdj = b + m
	}
	return aAdj <= bAdj
}

func transform(last Observation, ts uint64, tick int) Observation {
	d := new(big.Int).SetUint64((ts - last.Timestamp) % (1 << 32))
	cum := new(big.Int).Add(last.TickCumulative, d.Mul(d, big.NewInt(int64(tick))))
	return Observation{Timestamp: ts, TickCumulative: cum, Initialized: true}
}

// searchMiss is a binary-search slot read that failed, with the search range it was probing (the
// multicall prefetch walks the rest of the search from there).
type searchMiss struct {
	l, r int
	err  error
}

func (e *searchMiss) Error() string { return e.err.Error() }
func (e *searchMiss) Unwrap() error { return e.err }

// maxSearchSteps bounds binarySearch on inconsistent (corrupted) observations, where Oracle's loop
// would not terminate (on chain: out of gas). A block's gas buys far fewer cold-slot steps than this
// (~15k at 30M gas), so every search the chain completes completes here.
const maxSearchSteps = 1 << 17

var errSearch = errors.New("lotflow: oracle binary search did not converge")

func (s *TwapState) binarySearch(time, target uint64) (Observation, Observation, error) {
	l := (s.Index + 1) % s.Cardinality
	r := l + s.Cardinality - 1
	for step := 0; ; step++ {
		if step == maxSearchSteps {
			return Observation{}, Observation{}, errSearch
		}
		i := (l + r) / 2
		before, err := s.slot(i % s.Cardinality)
		if err != nil {
			return Observation{}, Observation{}, &searchMiss{l, r, err}
		}
		if !before.Initialized {
			l = i + 1
			continue
		}
		after, err := s.slot((i + 1) % s.Cardinality)
		if err != nil {
			return Observation{}, Observation{}, &searchMiss{l, r, err}
		}
		atOrAfter := lte(time, before.Timestamp, target)
		if atOrAfter && lte(time, target, after.Timestamp) {
			return before, after, nil
		}
		if !atOrAfter {
			r = i - 1
		} else {
			l = i + 1
		}
	}
}

// Observe is Oracle.observeSingle: the tickCumulative `secondsAgo` before `time`.
func (s *TwapState) Observe(time, secondsAgo uint64) (*big.Int, error) {
	if s.Cardinality == 0 {
		return nil, fmt.Errorf("lotflow: oracle cardinality 0")
	}
	last, err := s.slot(s.Index)
	if err != nil {
		return nil, err
	}
	if secondsAgo == 0 {
		if last.Timestamp != time {
			last = transform(last, time, s.Tick)
		}
		return last.TickCumulative, nil
	}
	target := time - secondsAgo
	var before, after Observation
	if lte(time, last.Timestamp, target) {
		if last.Timestamp == target {
			return last.TickCumulative, nil
		}
		return transform(last, target, s.Tick).TickCumulative, nil
	}
	oldest, err := s.slot((s.Index + 1) % s.Cardinality)
	if err != nil {
		return nil, err
	}
	if !oldest.Initialized {
		if oldest, err = s.slot(0); err != nil {
			return nil, err
		}
	}
	if !lte(time, oldest.Timestamp, target) {
		return nil, errOld
	}
	if before, after, err = s.binarySearch(time, target); err != nil {
		return nil, err
	}
	switch target {
	case before.Timestamp:
		return before.TickCumulative, nil
	case after.Timestamp:
		return after.TickCumulative, nil
	}
	obsDelta := big.NewInt(int64(after.Timestamp - before.Timestamp))
	targetDelta := big.NewInt(int64(target - before.Timestamp))
	q := new(big.Int).Sub(after.TickCumulative, before.TickCumulative)
	q.Quo(q, obsDelta) // Solidity int56 division truncates toward zero, as big.Int.Quo
	return q.Add(before.TickCumulative, q.Mul(q, targetDelta)), nil
}

var errOld = fmt.Errorf("lotflow: v3 oracle OLD")

// TwapTick is MqPricing.twapTick: the mean tick over `window`, rounded toward -inf.
func (s *TwapState) TwapTick(time, window uint64) (int, error) {
	c0, err := s.Observe(time, window)
	if err != nil {
		return 0, err
	}
	c1, err := s.Observe(time, 0)
	if err != nil {
		return 0, err
	}
	delta := new(big.Int).Sub(c1, c0)
	w := new(big.Int).SetUint64(window)
	q, r := new(big.Int).QuoRem(delta, w, new(big.Int))
	if delta.Sign() < 0 && r.Sign() != 0 {
		q.Sub(q, big.NewInt(1))
	}
	if !q.IsInt64() || q.Int64() < -887272 || q.Int64() > 887272 {
		return 0, fmt.Errorf("lotflow: BadTwapTick")
	}
	return int(q.Int64()), nil
}

// ---------------------------------------------------------------- MqPricing

// MqMarket is the route + pricing state shared by every WETH-token constituent of an MQ venue.
type MqMarket struct {
	Weth         string
	ConvPool     string   // the USDG/WETH conversion pool address
	ConvFactory  string   // its factory() read
	Conv         *LegPool // its leg state (fee 100, tickSpacing 1 on chain 4663)
	ConvTwap     *TwapState
	EthUsdFeed   string
	UsdgUsdFeed  string
	EthUsd       *FeedState
	UsdgUsd      *FeedState
	UsdgDecimals int
	Usdg         string
	PmWeth       *big.Int // the PoolManager's WETH balance (the buy hop takes WETH from it)
}

// MqToken is one WETH-quoted constituent's route.
type MqToken struct {
	Pool          string // token/WETH pool
	Factory       string
	TokenIs0      bool
	TokenDecimals int
	Twap          *TwapState
}

func (q *MqMarket) wethIs0() bool { return addrLess(q.Weth, q.Usdg) }

// feed18 is MqPricing.feed18.
func mqFeed18(addr string, f *FeedState, ts uint64) (*big.Int, error) {
	if f.Decimals > 18 {
		return nil, mqErr("BadFeedDecimals", "BadFeedDecimals(address)", addr)
	}
	if f.Answer.Sign() <= 0 {
		return nil, mqErr("BadFeedAnswer", "BadFeedAnswer(address,int256)", addr, f.Answer)
	}
	if f.UpdatedAt > ts {
		return nil, mqErr("FeedInFuture", "FeedInFuture(address,uint256)", addr, f.UpdatedAt)
	}
	if ts-f.UpdatedAt > MqMaxFeedAge {
		return nil, mqErr("StaleFeed", "StaleFeed(address,uint256)", addr, f.UpdatedAt)
	}
	return new(big.Int).Mul(f.Answer, pow10(18-f.Decimals)), nil
}

func mqPx18(pool string, s *TwapState, baseIs0 bool, baseDec, quoteDec int, ts uint64) (*big.Int, error) {
	tick, err := s.TwapTick(ts, MqTwapWindow)
	if err != nil {
		if err == errOld {
			return nil, mqErr("TwapHistoryTooShort", "TwapHistoryTooShort(address)", pool)
		}
		return nil, err
	}
	sp, err := SqrtPriceAtTick(tick)
	if err != nil {
		return nil, err
	}
	return SpotUsd18(sp, baseIs0, baseDec, quoteDec)
}

// ConversionTwapUsdg18 is the conversion pool's TWAP price of 1 WETH in USDG (1e18), unchecked.
func (q *MqMarket) ConversionTwapUsdg18(ts uint64) (*big.Int, error) {
	return mqPx18(q.ConvPool, q.ConvTwap, q.wethIs0(), mqWethDecimals, q.UsdgDecimals, ts)
}

// WethUsdg18 is MqPricing.wethUsdg18: the conversion TWAP checked against Chainlink.
func (q *MqMarket) WethUsdg18(ts uint64) (*big.Int, error) {
	px, err := q.ConversionTwapUsdg18(ts)
	if err != nil {
		return nil, err
	}
	eth, err := mqFeed18(q.EthUsdFeed, q.EthUsd, ts)
	if err != nil {
		return nil, err
	}
	usd, err := mqFeed18(q.UsdgUsdFeed, q.UsdgUsd, ts)
	if err != nil {
		return nil, err
	}
	ref := mulDiv(eth, bigE18, usd)
	gap := new(big.Int).Sub(px, ref)
	gap.Abs(gap)
	if new(big.Int).Mul(gap, bigBps).Cmp(new(big.Int).Mul(big.NewInt(MqCheckTolBps), ref)) > 0 {
		return nil, mqErr("PriceCheckFailed", "PriceCheckFailed(uint256,uint256)", px, ref)
	}
	return px, nil
}

// MqTokenUsdg18 is MqPricing.tokenUsdg18 (= MqFactory.spotUsd18 for a WETH token).
func MqTokenUsdg18(q *MqMarket, t *MqToken, ts uint64) (*big.Int, error) {
	inWeth, err := mqPx18(t.Pool, t.Twap, t.TokenIs0, t.TokenDecimals, mqWethDecimals, ts)
	if err != nil {
		return nil, err
	}
	weth, err := q.WethUsdg18(ts)
	if err != nil {
		return nil, err
	}
	return mulDiv(inWeth, weth, bigE18), nil
}

// ---------------------------------------------------------------- kind-3 admission and leg

func v3Admitted(cfg *Config, factory string, p *LegPool) bool {
	return p != nil && sameAddr(factory, cfg.V3Factory) && p.SqrtPriceX96 != nil &&
		p.SqrtPriceX96.Cmp(MinSqrtPrice) >= 0 && p.SqrtPriceX96.Cmp(MaxSqrtPrice) < 0 &&
		p.Liquidity != nil && p.Liquidity.Sign() != 0
}

// mqSources is NavJitLegsMq._mqSources: the single kind-3 source of a WETH token, or none.
func mqSources(cfg *Config, stock, stable string, q *MqMarket, t *MqToken, pool, conv *LegPool) []Source {
	if q == nil || t == nil || !sameAddr(stable, q.Weth) {
		return nil
	}
	if !v3Admitted(cfg, q.ConvFactory, conv) || !v3Admitted(cfg, t.Factory, pool) {
		return nil
	}
	if cfg.MinTickSpacing > 0 && pool.TickSpacing < cfg.MinTickSpacing { // the conversion pool is exempt
		return nil
	}
	return []Source{{Kind: KindV3W, ID: v3SourceID(t.Pool), StockIs0: addrLess(stock, q.Weth),
		Pool: strings.ToLower(t.Pool)}}
}

// simulateLegW is NavJitLegsMq._legW. p.Conv is the conversion pool's state at this point of the swap.
func simulateLegW(cfg *Config, s Source, p *LegPool, buy bool, amount *big.Int) *LegResult {
	conv := p.Conv
	fail := func(err error) *LegResult {
		return &LegResult{Paid: new(big.Int), Got: new(big.Int), Result: new(big.Int), Err: err}
	}
	if conv == nil {
		return fail(fmt.Errorf("lotflow: kind-3 leg without conversion pool state"))
	}
	hop := func(pool *LegPool, zfo, exactOut bool, amt *big.Int) *LegResult {
		r := swapOnPool(pool.Fee, pool, zfo, exactOut, amt, cfg.LegLimit(pool.SqrtPriceX96, zfo))
		if r.Err == nil && r.Paid.Sign() != 0 && r.Got.Sign() == 0 {
			r.Fill = false // V3Leg ExtraLegNoFill
		}
		return r
	}
	var tok, cv *LegResult
	out := &LegResult{}
	if buy {
		tok = hop(p, !s.StockIs0, true, amount) // WETH -> token, exact-out amount
		if tok.Err != nil || !tok.Fill {
			return fail(&LegShortError{Stock: s.Pool, Wanted: amount, Got: tok.Got, Underlying: tok.Err})
		}
		w := tok.Paid
		if p.PmWeth != nil && p.PmWeth.Cmp(w) < 0 {
			return fail(fmt.Errorf("lotflow: PoolManager WETH %s < %s needed by the token hop", p.PmWeth, w))
		}
		cv = hop(conv, !p.WethIs0, true, w) // USDG -> WETH, exact-out w
		if cv.Err != nil || !cv.Fill {
			return fail(&LegShortError{Stock: "weth", Wanted: w, Got: cv.Got, Underlying: cv.Err})
		}
		out.Paid, out.Got, out.Result = cv.Paid, new(big.Int).Set(amount), cv.Paid
	} else {
		tok = hop(p, s.StockIs0, false, amount) // token -> WETH, exact-in amount
		if tok.Err != nil || !tok.Fill {
			return fail(&LegShortError{Stock: s.Pool, Wanted: amount, Got: tok.Paid, Underlying: tok.Err})
		}
		w := tok.Got
		cv = hop(conv, p.WethIs0, false, w) // WETH -> USDG, exact-in w
		if cv.Err != nil || !cv.Fill {
			return fail(&LegShortError{Stock: "weth", Wanted: w, Got: cv.Paid, Underlying: cv.Err})
		}
		out.Paid, out.Got, out.Result = new(big.Int).Set(amount), cv.Got, cv.Got
	}
	out.Fill = true
	out.SqrtPriceX96, out.Tick, out.Liquidity = tok.SqrtPriceX96, tok.Tick, tok.Liquidity
	out.CrossedTicks, out.CrossedWords = tok.CrossedTicks+cv.CrossedTicks, tok.CrossedWords+cv.CrossedWords
	out.ConvPost = &PoolPost{SqrtPriceX96: cv.SqrtPriceX96, Tick: cv.Tick, Liquidity: cv.Liquidity}
	return out
}

// PoolPost is a pool's slot0 and in-range liquidity after a hop.
type PoolPost struct {
	SqrtPriceX96 *big.Int
	Tick         int
	Liquidity    *big.Int
}
