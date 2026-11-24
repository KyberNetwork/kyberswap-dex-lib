package efficientprop

import (
	"math/big"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// PoolSimulator quotes one listing door from the ladders its tracker sampled at one block (curve).
type PoolSimulator struct {
	pool.Pool

	curves [2]curve
	// custody is what the listing's pool holds of each token: it pays every listing of that pool.
	custody [2]uint256.Int
	// filled marks a direction that took a fill in this route: the fill moved the side's fill memory, rounded up to
	// the listing's board unit, so a second fill walks from further up than the ladder can say.
	filled [2]bool

	staticExtra StaticExtra
	tipCapWei   uint64
	// gas is each direction's (0: token0 in, 1: token1 in), by the fill mode, by whether the listing is past its pool's
	// hook pair, by whether its fill reads a price source or an asset module, and by whether its input is the base or
	// the quote token (gasOf); on a listing with a credit mode, its fills' inside the custody (Credit)
	gas [2]int64
	// credit is whether the listing has a credit mode (Extra.Credit): a fill at or past its ladder's first routed point
	// is a route, charged routeGas and paid past the custody, and a fill in either direction ends both, since a fill
	// moves the custody and a route the pool's debt, which every later quote of either side reads (poolKey)
	credit   bool
	routeGas [2]int64
	// routeLimit is, per direction, the gas limit its config states for a route fill, 0 where it states none
	routeLimit [2]int64
	// noneHeld is, per token, whether the pool held none of it at the state's block (Extra.NoneHeld): a fill with it
	// in is charged gasFirstOfInput more, until a fill of the route pays the pool some
	noneHeld [2]bool
	// validUntil is the last unix second a fill quoted from the state may land, the end of the sampled window
	// (Extra.ValidUntil); route finding (staleCheck) quotes no fill that lands after it, a fill landing allowance
	// seconds after its quote.
	validUntil uint64
	allowance  uint64
	staleCheck bool
}

var (
	_ = pool.RegisterFactory(DexType, NewPoolSimulator)
	// Listings of one pool pay out of the same custody: share the limit across them, keyed by pool and token.
	_ = pool.RegisterUseSwapLimit(DexType)
)

// poolOpen is every listing's value of its pool's route key (poolKey): a fill of the pool's in a route takes 1 off it,
// and a credit route all of it. A route is quoted only on a pool no fill of the route has touched (the key still
// poolOpen), and once one is taken, no listing of the pool is quoted again in that route (the key 0): a route borrows
// or repays, which moves what every listing of the pool fills (a cheap fill of a token the pool owes is refused, and
// a mode-2 bid routes, until a flush).
var poolOpen = new(big.Int).Lsh(big.NewInt(1), 62)

// SwapInfo is a fill's class: whether it is a credit route, and where its listing's config states one, the gas limit
// a route fill needs. The limit is not the charge: Gas is the least route fill's, which a route through the pool's
// dearer lending venue spends more than, and GasLimit covers that one.
type SwapInfo struct {
	Routed   bool  `json:"routed,omitempty"`
	GasLimit int64 `json:"gasLimit,omitempty"`
}

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	ep := params.EntityPool
	if params.Opts.StaleCheck && time.Since(time.Unix(ep.Timestamp, 0)) > maxAge {
		return nil, ladder.ErrStale
	}
	if len(ep.Tokens) != 2 || len(ep.Reserves) != 2 {
		return nil, ErrInvalidState
	}
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	if staticExtra.Mode != modeDoor && staticExtra.Mode != modeFront {
		return nil, ErrInvalidState
	}
	// a listing's gas: its own figures where its front's config names them (Extra.Gas), its credit figures where it
	// has a credit mode (below), else its class's
	baseIn, quoteIn, ok := gasOf(staticExtra.Mode, extra.Print, extra.Guarded, extra.Past)
	if extra.Builder {
		baseIn, quoteIn, ok = gasOfBuilder(staticExtra.Mode, extra.Print, extra.Guarded, extra.Past, extra.NoAsset,
			extra.AssetQuote)
	}
	if g := extra.Gas; g != nil {
		if g[0] <= 0 || g[1] <= 0 {
			return nil, ErrInvalidState
		}
		baseIn, quoteIn, ok = g[0], g[1], true
	}
	// a class with no figure in the listing's mode: the tracker quotes no such listing, so a state with a ladder of
	// one is not the tracker's (a credit listing's ladders ride in Credit, at its configured figures)
	if !ok && (len(extra.Ladders[0]) > 0 || len(extra.Ladders[1]) > 0) {
		return nil, ErrInvalidState
	}

	ladders := extra.ladders()
	if params.Opts.StaleCheck && (len(ladders[0]) > 0 || len(ladders[1]) > 0) &&
		uint64(time.Now().Unix())+extra.Allowance > extra.ValidUntil {
		return nil, ladder.ErrStale
	}

	s := &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    make([]*big.Int, 2),
			BlockNumber: ep.BlockNumber,
		}},
		staticExtra: staticExtra,
		tipCapWei:   extra.TipCapWei,
		gas:         [2]int64{baseIn, quoteIn},
		validUntil:  extra.ValidUntil,
		allowance:   extra.Allowance,
		staleCheck:  params.Opts.StaleCheck,
		noneHeld:    [2]bool{extra.NoneHeld&1 != 0, extra.NoneHeld&2 != 0},
	}
	if extra.BaseToken1 {
		s.gas = [2]int64{quoteIn, baseIn}
	}
	var from [2]int
	for i := range 2 {
		from[i] = len(ladders[i])
	}
	if c := extra.Credit; c != nil {
		// only a front read through /state quotes a listing with a credit mode, each class at its configured figure,
		// its ladders in Credit alone
		if !extra.Builder || extra.CreditMode == 0 || len(extra.Ladders[0]) > 0 || len(extra.Ladders[1]) > 0 {
			return nil, ErrInvalidState
		}
		for i := range 2 {
			// a side with no route figure holds no routed point
			routes := c.From[i] >= 0 && c.From[i] < len(c.Ladders[i])
			if c.Gas[i][0] <= 0 || c.Gas[i][1] < 0 || c.Gas[i][1] == 0 && routes {
				return nil, ErrInvalidState
			}
		}
		s.credit, from = true, c.From
		s.gas, s.routeGas = [2]int64{c.Gas[0][0], c.Gas[1][0]}, [2]int64{c.Gas[0][1], c.Gas[1][1]}
		if c.Limit != nil {
			// a limit covers the gas it is for
			for i := range 2 {
				if c.Limit[i] != 0 && c.Limit[i] < c.Gas[i][1] {
					return nil, ErrInvalidState
				}
			}
			s.routeLimit = *c.Limit
		}
	} else if extra.CreditMode != 0 && (len(extra.Ladders[0]) > 0 || len(extra.Ladders[1]) > 0) {
		return nil, ErrInvalidState
	}
	for i := range 2 {
		c, err := newCurve(ladders[i], from[i])
		if err != nil {
			return nil, err
		}
		s.curves[i] = c
		if err := s.custody[i].SetFromDecimal(ep.Reserves[i]); err != nil {
			return nil, ErrInvalidState
		}
		s.Info.Reserves[i] = s.custody[i].ToBig()
	}
	return s, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	in, out := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if in < 0 || out < 0 || in == out {
		return nil, ladder.ErrInvalidToken
	}
	if s.staleCheck && !s.validAt(uint64(time.Now().Unix())) {
		return nil, ladder.ErrStale
	}
	if s.filled[in] {
		return nil, ErrSecondFill
	}
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ladder.ErrZeroAmountIn
	}
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ladder.ErrAmountInTooLarge
	}

	var amountOut uint256.Int
	if err := s.curves[in].quote(amountIn, &amountOut); err != nil {
		return nil, err
	}
	if amountOut.IsZero() {
		return nil, ladder.ErrNoQuote
	}
	// a route pays past the custody (it may borrow the token it pays): its reach is its ladder's, sampled on the
	// custody as the refresh found it, so it is quoted only on a pool no fill of the route has touched
	routed := s.credit && s.curves[in].routes(amountIn)
	if !routed && amountOut.Gt(&s.custody[out]) {
		return nil, ladder.ErrInsufficientLiquidity
	}
	outBig := amountOut.ToBig()
	if limit := params.Limit; limit != nil {
		open := limit.GetLimit(s.poolKey())
		if routed && (open == nil || open.Cmp(poolOpen) != 0) || open != nil && open.Sign() <= 0 {
			return nil, pool.ErrNotEnoughInventory
		}
		if l := limit.GetLimit(s.limitKey(out)); !routed && (l == nil || outBig.Cmp(l) > 0) {
			return nil, pool.ErrNotEnoughInventory
		}
	}

	gas, info := s.gas[in], SwapInfo{Routed: routed}
	if routed {
		gas, info.GasLimit = s.routeGas[in], s.routeLimit[in]
	}
	if s.noneHeld[in] {
		gas += gasFirstOfInput
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: outBig},
		Fee:            &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: new(big.Int)},
		Gas:            gas,
		SwapInfo:       info,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	in, out := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if in < 0 || out < 0 || in == out {
		return
	}
	amountIn, inOverflow := uint256.FromBig(params.TokenAmountIn.Amount)
	amountOut, outOverflow := uint256.FromBig(params.TokenAmountOut.Amount)
	if inOverflow || outOverflow {
		return
	}
	s.filled[in], s.noneHeld[in] = true, false
	if s.credit {
		s.filled[out] = true
	}
	if _, overflow := s.custody[in].AddOverflow(&s.custody[in], amountIn); overflow {
		s.custody[in].SetAllOne()
	}
	if amountOut.Gt(&s.custody[out]) {
		s.custody[out].Clear()
	} else {
		s.custody[out].Sub(&s.custody[out], amountOut)
	}
	s.Info.Reserves = []*big.Int{s.custody[0].ToBig(), s.custody[1].ToBig()}

	if limit := params.SwapLimit; limit != nil {
		_, _, _ = limit.UpdateLimit(s.limitKey(out), s.limitKey(in),
			params.TokenAmountOut.Amount, params.TokenAmountIn.Amount)
		// the fill touched the pool: no route of it is quoted after it, and after a route nothing of it is
		if open := limit.GetLimit(s.poolKey()); open != nil && open.Sign() > 0 {
			taken := big.NewInt(1)
			if s.routedFill(params.SwapInfo, in, amountIn) {
				taken = new(big.Int).Set(open)
			}
			_, _, _ = limit.UpdateLimit(s.poolKey(), s.poolKey(), taken, new(big.Int))
		}
	}
}

// routedFill is whether a fill of amountIn in direction in, which UpdateBalance applies, was a route: its SwapInfo's
// class, or where it carries none, on a listing with a credit mode, the class of its size on the ladder.
func (s *PoolSimulator) routedFill(info any, in int, amountIn *uint256.Int) bool {
	switch si := info.(type) {
	case SwapInfo:
		return si.Routed
	case *SwapInfo:
		if si != nil {
			return si.Routed
		}
	}
	return s.credit && s.curves[in].routes(amountIn)
}

// validAt reports whether a fill quoted in unix second now, landing within the allowance, lands within the sampled
// window, where the sampled word and depth certainly stand.
func (s *PoolSimulator) validAt(now uint64) bool {
	return now+s.allowance <= s.validUntil
}

// CloneState copies what UpdateBalance writes: the custody and filled arrays are values, and Info.Reserves is
// replaced whole. The curves never change after construction, so clones share them.
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = []*big.Int{new(big.Int).Set(s.Info.Reserves[0]), new(big.Int).Set(s.Info.Reserves[1])}
	return &cloned
}

// CalculateLimit is, for each token, what the ladder paying it out proves its pool pays at the refresh from its
// custody: its last point's output inside the custody (a credit listing's last custody point), no more than the pool
// holds (none where no ladder pays it). It is keyed by pool and token, so the listings of one pool share it, the larger
// of theirs standing (the default merge): a pool's room for a token (its balance less any fee owed, its outflow
// bucket, its cap) is shared by its listings and may be under its balance, and each ladder's top is a fill the room
// took at the refresh. A route's reach past the custody is its own listing's (CalcAmountOut). Beside them, the pool's
// route key at poolOpen, which every listing of the pool reports alike.
func (s *PoolSimulator) CalculateLimit() map[string]*big.Int {
	limits := make(map[string]*big.Int, 3)
	for out := range 2 {
		var l uint256.Int
		if c := &s.curves[1-out]; c.from > 0 {
			l.Set(&c.y[c.from-1])
		}
		if l.Gt(&s.custody[out]) {
			l.Set(&s.custody[out])
		}
		limits[s.limitKey(out)] = l.ToBig()
	}
	limits[s.poolKey()] = new(big.Int).Set(poolOpen)
	return limits
}

func (s *PoolSimulator) limitKey(i int) string {
	return strings.ToLower(s.staticExtra.Custody) + ":" + s.Info.Tokens[i]
}

// poolKey is the pool's route key, shared by its listings: poolOpen until a fill of the route touches the pool.
func (s *PoolSimulator) poolKey() string {
	return strings.ToLower(s.staticExtra.Custody) + ":route"
}

// GetMetaInfo names what the executor data is built from, abi.encode(door, front, listing, mode), and the contract
// the adapter may approve: the door in the door's mode, the front in the front's.
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	approval := s.Info.Address
	if s.staticExtra.Mode == modeFront {
		approval = strings.ToLower(s.staticExtra.Front)
	}
	return PoolMeta{
		BlockNumber:     s.Info.BlockNumber,
		Door:            s.Info.Address,
		Front:           strings.ToLower(s.staticExtra.Front),
		Listing:         s.staticExtra.Listing,
		Mode:            s.staticExtra.Mode,
		ApprovalAddress: approval,
		TipCapWei:       s.tipCapWei,
	}
}

// gasOf is a fill's gas with the base in and with the quote in, by the pool's fill mode, whether the listing's fill
// reads a price source or an asset module (print), whether a source it reads is guarded (guarded), and whether its
// pair is past its pool's hook pair (past); ok is false for a mode the adapter does not take.
func gasOf(mode uint8, print, guarded, past bool) (baseIn, quoteIn int64, ok bool) {
	switch {
	case mode == modeFront && !past:
		baseIn, quoteIn = gasWordFrontBaseIn, gasWordFrontQuoteIn
	case mode == modeFront:
		baseIn, quoteIn = gasWordFrontBaseInPast, gasWordFrontQuoteInPast
	case mode == modeDoor && !past:
		baseIn, quoteIn = gasWordDoorBaseIn, gasWordDoorQuoteIn
	case mode == modeDoor:
		baseIn, quoteIn = gasWordDoorBaseInPast, gasWordDoorQuoteInPast
	default:
		return 0, 0, false
	}
	switch {
	case guarded && !past && mode == modeFront:
		baseIn, quoteIn = gasGuardedFrontBaseIn, gasGuardedFrontQuoteIn
	case guarded && !past:
		baseIn, quoteIn = gasGuardedDoorBaseIn, gasGuardedDoorQuoteIn
	case guarded && mode == modeFront:
		baseIn, quoteIn = baseIn+gasGuardedFrontBaseInDiff, quoteIn+gasGuardedFrontQuoteInDiff
	case guarded:
		baseIn, quoteIn = baseIn+gasGuardedDoorBaseInDiff, quoteIn+gasGuardedDoorQuoteInDiff
	case print:
		baseIn, quoteIn = baseIn+gasPrintBaseIn, quoteIn+gasPrintQuoteIn
	}
	return baseIn, quoteIn, true
}

// gasOfBuilder is gasOf for a listing read through /state, on Ethereum, by its fill mode, whether its fill reads a
// price source (print), whether its pair is past its pool's hook pair (past) and, past it, whether neither of its
// tokens is the pool's asset (noAsset). In the front's mode only the classes measured in it have a figure, a listing
// that reads a price source: ok is false for any other, which is quoted there only with figures of its own
// (Front.ListingGas). Each side's figure follows its input token, not
// its place as base or quote: with the pool's asset one of the pair's tokens, the asset-in figure goes to the side
// whose input is the asset, the quote's where the quote is the asset (assetQuote), else the base's. Only pairs whose
// base is the asset were measured: a pair whose quote is the asset is charged the swapped figures without a measurement
// of its own, and that shape is measured before such a pair is listed. On the hook pair the base is the asset (its
// index 0). With neither the asset, the figures stay by base and quote, as measured. A listing bound to a guarded
// entry, which is never quoted there (its sources are price contracts, never guarded), has no figure (ok false).
func gasOfBuilder(mode uint8, print, guarded, past, noAsset, assetQuote bool) (baseIn, quoteIn int64, ok bool) {
	if guarded {
		return 0, 0, false
	}
	if mode == modeFront {
		switch {
		case !print:
			return 0, 0, false
		case !past:
			return gasEthFrontPrintBaseIn, gasEthFrontPrintQuoteIn, true
		case noAsset:
			return gasEthFrontPrintBaseInPastNoAsset, gasEthFrontPrintQuoteInPastNoAsset, true
		case assetQuote:
			return gasEthFrontPrintOtherInPast, gasEthFrontPrintAssetInPast, true
		}
		return gasEthFrontPrintAssetInPast, gasEthFrontPrintOtherInPast, true
	}
	if mode != modeDoor {
		return 0, 0, false
	}
	switch {
	case past && noAsset:
		baseIn, quoteIn = gasEthWordBaseInPastNoAsset, gasEthWordQuoteInPastNoAsset
		if print {
			baseIn, quoteIn = gasEthPrintBaseInPastNoAsset, gasEthPrintQuoteInPastNoAsset
		}
		return baseIn, quoteIn, true
	case past:
		baseIn, quoteIn = gasEthWordAssetInPast, gasEthWordOtherInPast
		if print {
			baseIn, quoteIn = gasEthPrintAssetInPast, gasEthPrintOtherInPast
		}
	default:
		baseIn, quoteIn = gasEthWordBaseIn, gasEthWordQuoteIn
		if print {
			baseIn, quoteIn = gasEthPrintBaseIn, gasEthPrintQuoteIn
		}
	}
	if past && assetQuote {
		baseIn, quoteIn = quoteIn, baseIn
	}
	return baseIn, quoteIn, true
}
