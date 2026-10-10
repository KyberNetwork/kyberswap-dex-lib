package lotflow

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/holiman/uint256"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

// The constituent-leg layer of NavJitHookV17 (MemeFiLP contracts/src/libraries/NavJitLegs.sol):
// per constituent, the admitted source set (`sources`), each source's leg simulated exactly on
// dex-lib's uniswap v3 swap engine (the engine dex-lib's v3 AND v4 pool simulators run on), and the
// probe-and-pick rule (`_pick`): buy = min USDG cost, sell = max USDG out, strict comparison (tie ->
// lowest index), a single candidate executes unprobed.

const zeroAddr = "0x0000000000000000000000000000000000000000"

// floatWalkWords mirrors dex-lib uniswap v3 floatWalkAfterWords: after this many empty bitmap words
// walked at non-zero liquidity its swap loop prices further words in float64 (biased against the
// swapper). A leg bounded at LegLimitBps (±5%) spans a few words at most, so this never triggers on
// the fixtures; if it ever does, the result is flagged Inexact.
const floatWalkWords = 16

// Rules are the admission limits, parameters so both contract generations are expressible.
type Rules struct {
	// MaxBookSources caps admitted VenueBook (v3) venues (pre-D-B NavJitLegs.MAX_BOOK_SOURCES; 0 = no cap).
	MaxBookSources int
	// MaxCandidates caps the whole list, registry pools first (0 = no cap).
	MaxCandidates int
	// MinTickSpacing refuses v4 pools and v3 venues with a narrower tick spacing (0 = no rule).
	MinTickSpacing int
	// HookedPricePool admits the registry price pool even when it has a hook; false = hook-free v4 only.
	HookedPricePool bool
}

var (
	// FinalisedRules: the D-B contracts (NavJitLegs MAX_CANDIDATES 3 in total, registry v4 first;
	// MIN_TICK_SPACING 10 on v4 and v3; hook-free v4 only, the hooked price pool too). The default.
	FinalisedRules = Rules{MaxCandidates: 3, MinTickSpacing: 10, HookedPricePool: false}
	// PreFinalisationRules: the build the first P2-FIX set was recorded on (MAX_BOOK_SOURCES 4,
	// hooked price pool admitted). Kept for reference; no current fixture uses it.
	PreFinalisationRules = Rules{MaxBookSources: 4, HookedPricePool: true}
)

// Config is everything the leg layer needs besides pool state.
type Config struct {
	Rules
	Usdg      string
	V3Factory string
	// LegLimitBps is NavJitLegs.LEG_LIMIT_BPS: each leg is bounded by its pool's own pre-trade price
	// moved this far in the swap direction.
	LegLimitBps uint64
	// ProbeGas is NavJitLegs.PROBE_GAS: a probe whose leg needs this much gas is a skip.
	ProbeGas uint64
	// InexactGasMargin: a probe whose estimated leg gas is within this of ProbeGas is flagged
	// Inexact (the gas model, not the chain, decided it).
	InexactGasMargin uint64
	Gas              GasModel
	// NearStipend decides an Inexact (near-PROBE_GAS) probe: fill (default) or skip. The chain's
	// verdict there depends on the transaction's warm/cold access set (F-FIXTURES §1), so the two
	// policies enumerate both possible outcomes; QuoteConservative takes the worse.
	NearStipend NearStipendPolicy
	// DynamicFee returns the swap fee (pips, as the PoolManager Swap event reports it) a
	// dynamic-fee v4 pool charges; its hook sets it per swap, so it is an input. Unknown -> the
	// candidate cannot be simulated and is reported Inexact.
	DynamicFee func(sourceID string) (uint32, bool)
}

type NearStipendPolicy int

const (
	NearStipendFill NearStipendPolicy = iota
	NearStipendSkip
)

// DefaultConfig returns the contract constants of the fixtures' build with the given rules.
func DefaultConfig(usdg, v3Factory string, rules Rules) *Config {
	return &Config{Rules: rules, Usdg: usdg, V3Factory: v3Factory, LegLimitBps: 500, ProbeGas: 1_200_000,
		InexactGasMargin: 100_000, Gas: DefaultGasModel}
}

// ---------------------------------------------------------------- admission

type Admission string

const (
	AdmAdmitted           Admission = "admitted"
	AdmStableNotUsdg      Admission = "stable-not-usdg"
	AdmHookedNotPricePool Admission = "hooked-not-price-pool"
	AdmNoPrice            Admission = "no-price"
	AdmNoLiquidity        Admission = "no-liquidity"
	AdmFamily0            Admission = "family-0"
	AdmFactoryNotV3       Admission = "factory-not-v3Factory"
	AdmPriceOutOfRange    Admission = "price-unreadable-or-out-of-range"
	AdmOverMaxBook        Admission = "over-MAX_BOOK_SOURCES" // pre-D-B only
	// D-B (fixture labels of the finalised set)
	AdmHooked            Admission = "hooked"
	AdmTickSpacing       Admission = "tick-spacing-lt-10"
	AdmOverMaxCandidates Admission = "over-MAX_CANDIDATES"
)

// RegistryEntry is StockRegistry.getStock(stock) with the PoolManager reads admission makes.
type RegistryEntry struct {
	Stable           string
	PriceIndex       int
	StockIsCurrency0 bool
	Pools            []RegistryPoolView
}

type RegistryPoolView struct {
	PoolID       string
	Hooks        string
	TickSpacing  int
	SqrtPriceX96 *big.Int // PoolManager slot0
	Liquidity    *big.Int // PoolManager in-range liquidity
}

// BookVenueView is one VenueBook.venuesOf(stock) entry with the READ_GAS reads admission makes on it.
// A read that failed or returned < 32 bytes has its OK flag false.
type BookVenueView struct {
	Pool          string
	Family        int
	Factory       string
	FactoryOK     bool
	SqrtPriceX96  *big.Int
	SqrtOK        bool
	Liquidity     *big.Int
	LiqOK         bool
	TickSpacing   int // tickSpacing() read under READ_GAS; only consulted under Rules.MinTickSpacing
	TickSpacingOK bool
}

// Source is one admitted executable source (NavJitLegs.Source).
type Source struct {
	Kind     int    // KindV4 | KindV3
	ID       string // v4 PoolId; v3 bytes32(uint160(pool)) — the id LegExecuted.source carries
	StockIs0 bool
	Pool     string // v3 pool address
}

func isZeroAddr(a string) bool {
	v, ok := new(big.Int).SetString(strings.TrimPrefix(strings.ToLower(a), "0x"), 16)
	return ok && v.Sign() == 0
}

func sameAddr(a, b string) bool { return strings.EqualFold(a, b) }

// addrLess compares two addresses numerically (Solidity `stock < usdg`).
func addrLess(a, b string) bool {
	x, _ := new(big.Int).SetString(strings.TrimPrefix(strings.ToLower(a), "0x"), 16)
	y, _ := new(big.Int).SetString(strings.TrimPrefix(strings.ToLower(b), "0x"), 16)
	return x != nil && y != nil && x.Cmp(y) < 0
}

// v3SourceID is bytes32(uint256(uint160(pool))).
func v3SourceID(pool string) string {
	return "0x" + strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(pool), "0x")
}

// Sources is NavJitLegs.sources(stock): the candidate list in execution order, plus the admission
// verdict of every registry pool and every VenueBook venue (in their own order). venuesOK = false
// means venuesOf(stock) reverted (no v3 sources).
func Sources(cfg *Config, stock string, reg RegistryEntry, venuesOK bool, venues []BookVenueView) (
	out []Source, regAdm, bookAdm []Admission) {
	regAdm = make([]Admission, len(reg.Pools))
	if !venuesOK {
		venues = nil
	}
	bookAdm = make([]Admission, len(venues))
	if !sameAddr(reg.Stable, cfg.Usdg) || len(reg.Pools) == 0 {
		for i := range regAdm {
			regAdm[i] = AdmStableNotUsdg
		}
		for i := range bookAdm {
			bookAdm[i] = AdmStableNotUsdg
		}
		return nil, regAdm, bookAdm
	}
	full := func() bool { return cfg.MaxCandidates > 0 && len(out) >= cfg.MaxCandidates }
	for i, p := range reg.Pools {
		hooked := !isZeroAddr(p.Hooks)
		switch {
		case full():
			regAdm[i] = AdmOverMaxCandidates
		case hooked && !cfg.HookedPricePool:
			regAdm[i] = AdmHooked
		case hooked && i != reg.PriceIndex:
			regAdm[i] = AdmHookedNotPricePool
		case cfg.MinTickSpacing > 0 && p.TickSpacing < cfg.MinTickSpacing:
			regAdm[i] = AdmTickSpacing
		case p.SqrtPriceX96 == nil || p.SqrtPriceX96.Sign() == 0:
			regAdm[i] = AdmNoPrice
		case p.Liquidity == nil || p.Liquidity.Sign() == 0:
			regAdm[i] = AdmNoLiquidity
		default:
			regAdm[i] = AdmAdmitted
			out = append(out, Source{Kind: KindV4, ID: strings.ToLower(p.PoolID), StockIs0: reg.StockIsCurrency0})
		}
	}
	v3StockIs0 := addrLess(stock, cfg.Usdg)
	m := 0
	for j, v := range venues {
		switch {
		case full():
			bookAdm[j] = AdmOverMaxCandidates
		case cfg.MaxBookSources > 0 && m >= cfg.MaxBookSources:
			bookAdm[j] = AdmOverMaxBook
		case v.Family == 0:
			bookAdm[j] = AdmFamily0
		case !v.FactoryOK || !sameAddr(v.Factory, cfg.V3Factory):
			bookAdm[j] = AdmFactoryNotV3
		case cfg.MinTickSpacing > 0 && (!v.TickSpacingOK || v.TickSpacing < cfg.MinTickSpacing):
			bookAdm[j] = AdmTickSpacing // D-B order: after factory(), before slot0()
		case !v.SqrtOK || v.SqrtPriceX96.Cmp(MinSqrtPrice) < 0 || v.SqrtPriceX96.Cmp(MaxSqrtPrice) >= 0:
			bookAdm[j] = AdmPriceOutOfRange
		case !v.LiqOK || v.Liquidity.Sign() == 0:
			bookAdm[j] = AdmNoLiquidity
		default:
			bookAdm[j] = AdmAdmitted
			out = append(out, Source{Kind: KindV3, ID: v3SourceID(v.Pool), StockIs0: v3StockIs0,
				Pool: strings.ToLower(v.Pool)})
			m++
		}
	}
	return out, regAdm, bookAdm
}

// ---------------------------------------------------------------- one leg

// LegPool is the state of one source a leg needs: slot0, in-range liquidity and the initialized
// ticks covering the leg's price window (LegLimitBps each side of spot), plus its fee.
type LegPool struct {
	Kind         int
	SqrtPriceX96 *big.Int
	Tick         int
	Liquidity    *big.Int
	TickSpacing  int
	Ticks        []uniswapv3.TickU256 // ascending, initialized only
	Fee          uint32               // v3 fee()
	LpFee        uint32               // v4 slot0 lpFee
	ProtocolFee  uint32               // v4 slot0 protocolFee, packed 12|12 bits
	DynamicFee   bool                 // v4 key.fee == 0x800000
	// kind 3 (MQ): the conversion pool's state at this point of the swap, its WETH side, and the
	// PoolManager's WETH balance (the buy hop takes WETH from it)
	Conv    *LegPool `json:"-"`
	WethIs0 bool     `json:"-"`
	PmWeth  *big.Int `json:"-"`
}

var ErrUnknownDynamicFee = errors.New("lotflow: dynamic-fee pool without a known fee")

// swapFee is the fee in pips the leg's swap charges.
func (p *LegPool) swapFee(cfg *Config, id string, zeroForOne bool) (uint32, error) {
	if p.Kind == KindV3 {
		return p.Fee, nil
	}
	if p.DynamicFee {
		if cfg.DynamicFee != nil {
			if f, ok := cfg.DynamicFee(id); ok {
				return f, nil
			}
		}
		return 0, ErrUnknownDynamicFee
	}
	// ProtocolFeeLibrary.calculateSwapFee
	pf := p.ProtocolFee & 0xfff
	if !zeroForOne {
		pf = p.ProtocolFee >> 12
	}
	if pf == 0 {
		return p.LpFee, nil
	}
	return pf + p.LpFee - uint32(uint64(pf)*uint64(p.LpFee)/1_000_000), nil
}

// LegLimit is NavJitLegs._limit: spot moved LegLimitBps in the swap direction, rounded toward spot.
func (cfg *Config) LegLimit(spot *big.Int, zeroForOne bool) *big.Int {
	if spot.Sign() == 0 {
		if zeroForOne {
			return new(big.Int).Set(MaxSqrtPrice)
		}
		return new(big.Int)
	}
	bps := new(big.Int).SetUint64(cfg.LegLimitBps)
	if zeroForOne {
		f := new(big.Int).Sqrt(new(big.Int).Mul(new(big.Int).Sub(bigBps, bps), big.NewInt(1e14)))
		return mulDivUp(spot, f, big.NewInt(1e9))
	}
	f := new(big.Int).Sqrt(new(big.Int).Mul(new(big.Int).Add(bigBps, bps), big.NewInt(1e14)))
	up := mulDiv(spot, f, big.NewInt(1e9))
	if up.Cmp(MaxSqrtPrice) >= 0 {
		return new(big.Int).Sub(MaxSqrtPrice, big.NewInt(1))
	}
	return up
}

// LegResult is one simulated leg (NavJitLegs._leg), in the leg's own direction: buy = USDG Paid for
// exactly Got stock; sell = Paid stock for Got USDG. Result is what a probe returns (buy: Paid, sell:
// Got).
type LegResult struct {
	Fill         bool // delivered (buy) / consumed (sell) the whole amount inside the leg's limit
	Paid, Got    *big.Int
	Result       *big.Int
	SqrtPriceX96 *big.Int // pool after the leg
	Tick         int
	Liquidity    *big.Int
	CrossedTicks int    // initialized ticks crossed
	CrossedWords int    // empty bitmap words walked
	GasLo, GasHi uint64 // leg gas envelope (probeGas.go)
	GasSkip      bool   // a probe certainly exhausts PROBE_GAS (REV 6 G1): a skip
	Inexact      bool   // the verdict rests on the gas model or an unknown input, not on exact maths
	Err          error
	ConvPost     *PoolPost // kind 3: the conversion pool after the leg
}

// Filled is the probe verdict: an exact fill that the stipend can afford.
func (r *LegResult) Filled() bool { return r.Err == nil && r.Fill && !r.GasSkip }

// SimulateLeg runs one leg of `amount` on source s with state p.
func SimulateLeg(cfg *Config, s Source, p *LegPool, buy bool, amount *big.Int) *LegResult {
	if s.Kind == KindV3W {
		return simulateLegW(cfg, s, p, buy, amount) // single candidate, never probed: no gas verdict
	}
	r := &LegResult{Paid: new(big.Int), Got: new(big.Int), Result: new(big.Int)}
	zfo := buy != s.StockIs0
	if p.SqrtPriceX96 == nil || p.SqrtPriceX96.Sign() == 0 {
		r.Err = errors.New("lotflow: source has no price")
		return r
	}
	fee, err := p.swapFee(cfg, s.ID, zfo)
	if err != nil {
		r.Err, r.Inexact = err, true
		return r
	}
	r = swapOnPool(fee, p, zfo, buy, amount, cfg.LegLimit(p.SqrtPriceX96, zfo))
	if r.Err != nil {
		return r
	}
	if s.Kind == KindV3 && r.Paid.Sign() != 0 && r.Got.Sign() == 0 {
		r.Fill = false // V3Leg ExtraLegNoFill
	}
	if r.CrossedWords >= floatWalkWords {
		r.Inexact = true // dex-lib prices long empty-word runs in float64 past this point
	}
	cfg.applyGas(s, r)
	return r
}

// swapOnPool runs one swap of `amount` on p: exact output when exactOut (a buy leg), else exact
// input, bounded by limit. Paid/Got are in the swap's own direction (in / out).
func swapOnPool(fee uint32, p *LegPool, zfo, exactOut bool, amount, limit *big.Int) *LegResult {
	r := &LegResult{Paid: new(big.Int), Got: new(big.Int), Result: new(big.Int)}
	buy := exactOut
	pool, err := newV3Pool(fee, p.SqrtPriceX96, p.Liquidity, p.Tick, p.Ticks, p.TickSpacing)
	if err != nil {
		r.Err = err
		return r
	}
	amt, overflow := uint256.FromBig(amount)
	if overflow {
		r.Err = errors.New("lotflow: leg amount overflows uint256")
		return r
	}
	specified := *amt // exact input (sell)
	if buy {
		specified.Neg(amt) // exact output
	}
	lim, overflow := uint256.FromBig(limit)
	if overflow {
		r.Err = errors.New("lotflow: bad leg limit")
		return r
	}
	res, err := pool.Swap(zfo, specified, *lim)
	if err != nil {
		r.Err = err // the pool swap reverts on chain too (e.g. limit out of bounds): no fill
		return r
	}
	rem := signed(&res.RemainingAmountIn)
	calc := signed(&res.AmountCalculated)
	if buy {
		r.Got = new(big.Int).Add(amount, rem) // rem <= 0: output still owed
		r.Paid = calc
		r.Fill = r.Got.Cmp(amount) == 0
		r.Result = r.Paid
	} else {
		r.Paid = new(big.Int).Sub(amount, rem)
		r.Got = new(big.Int).Neg(calc)
		r.Fill = r.Paid.Cmp(amount) == 0
		r.Result = r.Got
	}
	r.SqrtPriceX96 = res.SqrtRatioX96.ToBig()
	r.Tick = res.CurrentTick
	r.Liquidity = res.Liquidity.ToBig()
	r.CrossedTicks, r.CrossedWords = res.CrossInitTickLoops, res.CrossEmptyWordLoops
	return r
}

// ---------------------------------------------------------------- pick

// PickResult is NavJitLegs._pick for one constituent leg.
type PickResult struct {
	Index   int
	Probed  bool // false: a single candidate executed unprobed
	Results []*LegResult
	// Inexact: some candidate's verdict rests on the gas model or an unknown input.
	Inexact bool
}

// LegShortError: the only candidate (executed unprobed) cannot deliver/consume the leg's amount.
type LegShortError struct {
	Stock       string
	Wanted, Got *big.Int
	Underlying  error
}

func (e *LegShortError) Error() string {
	return fmt.Sprintf("lotflow: LegShort(%s, %s, %s)", e.Stock, e.Wanted, e.Got)
}
func (e *LegShortError) ErrorName() string { return "LegShort" }
func (e *LegShortError) ErrorArgs() []string {
	return []string{strings.ToLower(e.Stock), e.Wanted.String(), e.Got.String()}
}

// Pick chooses the source a leg of `amount` executes on. One candidate executes unprobed (a failure
// is that leg's own revert); two or more are each probed at the leg's exact amount and the best fill
// wins (buy: min cost, sell: max out; strict, so ties go to the lowest index). No fill at all is
// NoExecutableSource(stock).
func Pick(cfg *Config, stock string, srcs []Source, pools []*LegPool, buy bool, amount *big.Int) (*PickResult,
	error) {
	n := len(srcs)
	if n == 0 {
		return nil, &NoExecutableSourceError{Stock: stock}
	}
	pr := &PickResult{Results: make([]*LegResult, n)}
	if n == 1 {
		r := SimulateLeg(cfg, srcs[0], pools[0], buy, amount)
		// unprobed: no stipend, so only the maths decides and the gas envelope is irrelevant
		r.GasSkip = false
		r.Inexact = errors.Is(r.Err, ErrUnknownDynamicFee)
		pr.Results[0], pr.Inexact = r, r.Inexact
		if ls, ok := r.Err.(*LegShortError); ok {
			if ls.Stock != "weth" {
				ls.Stock = stock
			}
			return pr, ls
		}
		if r.Err != nil || !r.Fill {
			got := r.Got
			if !buy {
				got = r.Paid
			}
			return pr, &LegShortError{Stock: stock, Wanted: new(big.Int).Set(amount), Got: got, Underlying: r.Err}
		}
		return pr, nil
	}
	pr.Probed = true
	found := false
	for i := range srcs {
		r := SimulateLeg(cfg, srcs[i], pools[i], buy, amount)
		pr.Results[i] = r
		if r.Inexact {
			pr.Inexact = true
		}
		if !r.Filled() {
			continue
		}
		if !found || (buy && r.Result.Cmp(pr.Results[pr.Index].Result) < 0) ||
			(!buy && r.Result.Cmp(pr.Results[pr.Index].Result) > 0) {
			pr.Index, found = i, true
		}
	}
	if !found {
		return pr, &NoExecutableSourceError{Stock: stock}
	}
	return pr, nil
}
