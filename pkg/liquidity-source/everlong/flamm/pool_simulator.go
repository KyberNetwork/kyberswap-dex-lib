package everlongflamm

import (
	"math"
	"math/big"
	"slices"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// PoolSimulator quotes every pair of one pool from the ladders its tracker probed (curve). The pool's pairs share
// its custody and its venue book, so the simulator keeps one payout room per token for all of them: a fill paying
// token j lowers what every pair paying j may still pay, and each direction takes one fill per route: the chain
// charges a second fill its own fee on the book the first moved, which the ladder cannot say. No inventory spans pools:
// each pool has its own custody and its own venue accounts. What the pools of the source do share is the cost of
// touching the same contracts and storage in one transaction, so the route's swap limit, where the router passes
// one, counts the fills applied on them (limitFilled): a pool's first fill after another pool's is charged its gas
// measured after another pool (gas.go). The limit never refuses or resizes a fill.
type PoolSimulator struct {
	pool.Pool
	curves     []curve       // by direction, i*n + j; empty where the pool does not route i for j
	consumedIn []uint256.Int // by direction: the input of the route's fill there, zero where it has none
	room       []uint256.Int // by token: what the pool may still pay of it
	haircutBps uint64
}

var (
	_ = pool.RegisterFactory(DexType, NewPoolSimulator)
	_ = pool.RegisterUseSwapLimit(DexType)
)

// limitFilled is the swap limit's one key: how many fills the route has applied on pools of this source. Every
// pool reports it at zero (CalculateLimit), and each fill applied with the limit raises it by one (UpdateBalance).
// One count serves every pool of the source in the request, so on its chain: what a fill after another pool saves
// was measured with the other pool on another venue kind, and does not turn on the measured pool's kind.
const limitFilled = "filled"

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	return newPoolSimulator(params.EntityPool, lo.Ternary[time.Duration](params.Opts.StaleCheck, MaxAge, math.MaxInt64))
}

func newPoolSimulator(ep entity.Pool, maxAge time.Duration) (*PoolSimulator, error) {
	if time.Since(time.Unix(ep.Timestamp, 0)) > maxAge {
		return nil, ladder.ErrStale
	}
	n := len(ep.Tokens)
	if n < 2 || n > maxTokens || len(ep.Reserves) != n {
		return nil, ErrInvalidState
	}
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}
	if extra.HaircutBps > 10_000 {
		return nil, ErrInvalidState
	}
	s := &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    make([]*big.Int, n),
			BlockNumber: ep.BlockNumber,
		}},
		curves:     make([]curve, n*n),
		consumedIn: make([]uint256.Int, n*n),
		room:       make([]uint256.Int, n),
		haircutBps: extra.HaircutBps,
	}
	for t, r := range ep.Reserves {
		if err := s.room[t].SetFromDecimal(r); err != nil {
			return nil, err
		}
		s.Info.Reserves[t] = s.room[t].ToBig()
	}
	for _, l := range extra.Ladders {
		i, j := int(l.In), int(l.Out)
		if i >= n || j >= n || i == j || len(s.curves[i*n+j].x) != 0 {
			return nil, ErrInvalidState
		}
		c, err := newCurve(l)
		if err != nil {
			return nil, err
		}
		s.curves[i*n+j] = c
	}
	return s, nil
}

func (s *PoolSimulator) direction(tokenIn, tokenOut string) (int, int, *curve) {
	i, j := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenOut)
	if i < 0 || j < 0 || i == j {
		return i, j, nil
	}
	return i, j, &s.curves[i*len(s.Info.Tokens)+j]
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	i, j, c := s.direction(params.TokenAmountIn.Token, params.TokenOut)
	if c == nil {
		return nil, ErrInvalidToken
	}
	if len(c.x) == 0 {
		return nil, ladder.ErrNoQuote
	}
	a := params.TokenAmountIn.Amount
	if a == nil || a.Sign() <= 0 {
		return nil, ladder.ErrZeroAmountIn
	}
	var amountIn uint256.Int
	if amountIn.SetFromBig(a) {
		return nil, ladder.ErrAmountInTooLarge
	}
	d := i*len(s.Info.Tokens) + j
	if !s.consumedIn[d].IsZero() {
		// the chain charges a second fill its own fee on the book the first one moved, which the ladder cannot say, so
		// a direction takes one fill per route; a fill the other way only moves the pool in this one's favour
		return nil, ladder.ErrNoQuote
	}
	var used, out uint256.Int
	if _, err := c.quote(&amountIn, s.haircutBps, &used, &out); err != nil {
		return nil, err
	}
	if used.IsZero() || out.IsZero() {
		return nil, ladder.ErrNoQuote
	}
	moved := s.moved()
	gas := c.charge(&out, moved, !moved && filledBefore(params.Limit))
	if out.Gt(&s.room[j]) {
		return nil, ladder.ErrInsufficientLiquidity
	}
	var remaining uint256.Int
	remaining.Sub(&amountIn, &used)
	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: params.TokenOut, Amount: out.ToBig()},
		Fee:                    &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: new(big.Int)},
		RemainingTokenAmountIn: &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: remaining.ToBig()},
		Gas:                    gas,
		SwapInfo: SwapInfo{
			Venue:        venueSwap,
			PoolAssetIn:  i == 0,
			AmountInUsed: used.Dec(),
			AmountOut:    out.Dec(),
		},
	}, nil
}

// moved: a fill was taken on the pool since the refresh, by another fill of the same route, so in the same
// transaction: it moved the custody, supply and debts the charge's reads saw and warmed what the next fill touches,
// so every direction is charged its second-fill charge (gas.go).
func (s *PoolSimulator) moved() bool {
	for k := range s.consumedIn {
		if !s.consumedIn[k].IsZero() {
			return true
		}
	}
	return false
}

// filledBefore: the route's swap limit says a fill was applied on a pool of this source. Where this pool has not
// moved, that fill was another pool's, earlier in the route, so in the same transaction. No limit, a limit without
// the key or one at zero says none was.
func filledBefore(limit pool.SwapLimit) bool {
	if limit == nil {
		return false
	}
	n := limit.GetLimit(limitFilled)
	return n != nil && n.Sign() > 0
}

// CalculateLimit starts the count of the source's fills in a route at zero.
func (s *PoolSimulator) CalculateLimit() map[string]*big.Int {
	return map[string]*big.Int{limitFilled: new(big.Int)}
}

// UpdateBalance marks the direction as filled by the fill CalcAmountOut quoted, and lowers the paid token's room
// by its output, for every pair paying it, and counts the fill in the route's swap limit, where one is passed.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	i, j, c := s.direction(params.TokenAmountIn.Token, params.TokenAmountOut.Token)
	if c == nil {
		return
	}
	d := i*len(s.Info.Tokens) + j
	// the fill happened whatever its SwapInfo says, so the direction is marked before anything can return; a fill
	// with no readable amount marks it at one
	if s.consumedIn[d].IsZero() {
		s.consumedIn[d].SetOne()
	}
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	var used, out uint256.Int
	if used.SetFromDecimal(si.AmountInUsed) != nil || out.SetFromDecimal(si.AmountOut) != nil ||
		used.BitLen() > maxPointBits || out.BitLen() > maxPointBits {
		return
	}
	if !used.IsZero() {
		s.consumedIn[d].Set(&used) // the input of the fill; a fill is below 2^maxPointBits
	}
	if out.Gt(&s.room[j]) {
		s.room[j].Clear()
	} else {
		s.room[j].Sub(&s.room[j], &out)
	}
	s.Info.Reserves[j] = s.room[j].ToBig()
	if params.SwapLimit != nil {
		// the count rises by one and nothing falls; a limit that lacks the key keeps none, and the charges stay the
		// cells'
		_, _, _ = params.SwapLimit.UpdateLimit(limitFilled, limitFilled, bignumber.ZeroBI, bignumber.One)
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.consumedIn = slices.Clone(s.consumedIn)
	cloned.room = slices.Clone(s.room)
	cloned.Info.Reserves = slices.Clone(s.Info.Reserves)
	return &cloned
}

// CanSwapTo lists the tokens the pool routes for `address`.
func (s *PoolSimulator) CanSwapTo(address string) []string {
	j := s.GetTokenIndex(address)
	if j < 0 {
		return nil
	}
	n := len(s.Info.Tokens)
	return lo.Filter(s.Info.Tokens, func(_ string, i int) bool { return i != j && len(s.curves[i*n+j].x) != 0 })
}

// CanSwapFrom lists the tokens the pool routes `address` for.
func (s *PoolSimulator) CanSwapFrom(address string) []string {
	i := s.GetTokenIndex(address)
	if i < 0 {
		return nil
	}
	n := len(s.Info.Tokens)
	return lo.Filter(s.Info.Tokens, func(_ string, j int) bool { return i != j && len(s.curves[i*n+j].x) != 0 })
}

// GetApprovalAddress is the pool itself: it pulls the input with transferFrom.
func (s *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return s.Info.Address
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{ApprovalAddress: s.Info.Address, BlockNumber: s.Info.BlockNumber}
}
