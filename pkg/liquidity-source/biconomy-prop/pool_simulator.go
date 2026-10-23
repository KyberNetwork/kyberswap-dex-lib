package biconomyprop

import (
	"math/big"
	"slices"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// PoolSimulator ports PropAMMVenue's merge: the best next level across all makers is taken first,
// ties go to the earlier registered maker, and each maker is bounded by its remaining depth.
// Prices on offset boards decay with the anchor's age, so a quote is exact for the block the
// tracker read and can run slightly above delivery as the read ages.
type PoolSimulator struct {
	pool.Pool

	staticExtra StaticExtra
	members     []MemberExtra
	feeBps      uint64

	// cursors[i][d] is maker i's filled meter for direction d, seeded from Board.Filled and
	// advanced by UpdateBalance. Boards are never mutated, so clones copy only the cursors.
	cursors [][2]uint256.Int

	// quoteAt is the indexing clock (snapshot time); 0 means the wall clock at quote time.
	quoteAt uint64
}

var _ = pool.RegisterFactory(DexType, func(params pool.FactoryParams) (*PoolSimulator, error) {
	sim, err := NewPoolSimulator(params.EntityPool)
	if err != nil {
		return nil, err
	}
	if t := pool.QuoteTime(params.Opts.StaleCheck, params.EntityPool.Timestamp); !t.IsZero() {
		sim.quoteAt = uint64(t.Unix())
	}
	return sim, nil
})

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	if len(ep.Tokens) != 2 || len(ep.Reserves) != 2 {
		return nil, ErrInvalidToken
	}

	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}
	if len(extra.Members) > maxMakers {
		return nil, ErrTooManyMakers
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	cursors := make([][2]uint256.Int, len(extra.Members))
	for i := range extra.Members {
		for d := range cursors[i] {
			// A dark board has no Sizes, so it can never be filled whatever the meter says.
			if f := extra.Members[i].board(d).Filled; f != nil {
				cursors[i][d].Set(f)
			}
		}
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(ep.Reserves, func(s string, _ int) *big.Int { return bignum.NewBig(s) }),
			BlockNumber: ep.BlockNumber,
		}},
		staticExtra: staticExtra,
		members:     extra.Members,
		feeBps:      extra.FeeBps,
		cursors:     cursors,
	}, nil
}

func (m *MemberExtra) board(dirIndex int) *Board {
	if dirIndex == 0 {
		return &m.Dir0
	}
	return &m.Dir1
}

// now is the quote clock: the indexing snapshot time when set, else the wall clock.
func (s *PoolSimulator) now() uint64 {
	if s.quoteAt > 0 {
		return s.quoteAt
	}
	return uint64(time.Now().Unix())
}

// fill is the result of one allocation pass over the book.
type fill struct {
	alloc, outBy [maxMakers]uint256.Int // tokenIn taken from and tokenOut paid by each maker
	out, covered uint256.Int            // total tokenOut and tokenIn covered
}

// merge is the venue's _merge for one direction. The docs say a size is covered only if every
// maker it is allocated to pays out a non-zero amount, so one dust allocation fails the whole
// size (the venue reverts Inactive). The venue source is unverified, so this takes the
// conservative reading rather than reallocating over the remaining makers.
func (s *PoolSimulator) merge(dirIndex int, amountIn *uint256.Int, f *fill) {
	var excluded [maxMakers]bool
	for i := range s.members {
		excluded[i] = !boardLive(s.members[i].board(dirIndex), s.now())
	}
	s.allocate(dirIndex, amountIn, &excluded, f)
	for i := range s.members {
		if !f.alloc[i].IsZero() && f.outBy[i].IsZero() {
			f.covered.Clear()
			return
		}
	}
}

// allocate is one pass of the venue's _allocate.
func (s *PoolSimulator) allocate(dirIndex int, amountIn *uint256.Int, excluded *[maxMakers]bool, f *fill) {
	*f = fill{}
	var cursor, left [maxMakers]uint256.Int
	var next [maxMakers]int // -1 once a maker is out
	k := len(s.members)
	for i := 0; i < k; i++ {
		next[i] = -1
		if excluded[i] {
			continue
		}
		b := s.members[i].board(dirIndex)
		cursor[i].Set(&s.cursors[i][dirIndex])
		var limit uint256.Int
		fillLimit(&limit, b)
		if !cursor[i].Lt(&limit) {
			continue
		}
		left[i].Sub(&limit, &cursor[i])
		j := 0
		for j < len(b.Sizes) && !b.Sizes[j].Gt(&cursor[i]) {
			j++
		}
		if j < len(b.Sizes) {
			next[i] = j
		}
	}

	var remaining, take, segOut uint256.Int
	remaining.Set(amountIn)
	for !remaining.IsZero() {
		best := -1
		var bestPrice *uint256.Int
		for i := 0; i < k; i++ {
			if next[i] < 0 {
				continue
			}
			if price := s.members[i].board(dirIndex).Prices[next[i]]; best < 0 || price.Gt(bestPrice) {
				best, bestPrice = i, price
			}
		}
		if best < 0 {
			break
		}
		b := s.members[best].board(dirIndex)
		top := b.Sizes[next[best]]
		take.Sub(top, &cursor[best])
		if take.Gt(&remaining) {
			take.Set(&remaining)
		}
		if take.Gt(&left[best]) {
			take.Set(&left[best])
		}
		segmentOut(&segOut, &take, bestPrice)
		f.alloc[best].Add(&f.alloc[best], &take)
		f.outBy[best].Add(&f.outBy[best], &segOut)
		f.out.Add(&f.out, &segOut)
		f.covered.Add(&f.covered, &take)
		remaining.Sub(&remaining, &take)
		cursor[best].Add(&cursor[best], &take)
		left[best].Sub(&left[best], &take)
		if left[best].IsZero() {
			next[best] = -1
		} else if cursor[best].Eq(top) {
			if next[best]++; next[best] >= len(b.Sizes) {
				next[best] = -1
			}
		}
	}
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}
	var amountIn uint256.Int
	if params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrZeroAmountIn
	}
	if amountIn.SetFromBig(params.TokenAmountIn.Amount) {
		return nil, ErrOverflow
	}

	dirIndex := indexIn
	var f fill
	s.merge(dirIndex, &amountIn, &f)
	// The venue never fills partially: short coverage reverts Inactive().
	if f.covered.Lt(&amountIn) {
		return nil, ErrInsufficientLiquidity
	}
	var amountOut, fee uint256.Int
	if err := afterProtocolFee(&amountOut, &fee, &f.out, s.feeBps); err != nil {
		return nil, err
	}
	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}

	takes := make([]memberTake, 0, len(s.members))
	for i := range s.members {
		if !f.alloc[i].IsZero() {
			takes = append(takes, memberTake{Member: i, AmountIn: f.alloc[i].Clone()})
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: fee.ToBig()},
		Gas:            defaultGas + int64(len(takes)-1)*perMemberGas,
		SwapInfo:       SwapInfo{DirIndex: dirIndex, Takes: takes, GrossOut: f.out.Clone()},
	}, nil
}

// UpdateBalance advances each touched maker's meter by its allocation, as the executor's fill does.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	for _, take := range swapInfo.Takes {
		if take.Member < 0 || take.Member >= len(s.cursors) || take.AmountIn == nil {
			continue
		}
		cursor := &s.cursors[take.Member][swapInfo.DirIndex]
		cursor.Add(cursor, take.AmountIn)
	}

	// The reserve is the gross deliverable.
	reserveOut := s.Info.Reserves[1-swapInfo.DirIndex]
	if swapInfo.GrossOut != nil {
		reserveOut.Sub(reserveOut, swapInfo.GrossOut.ToBig())
	}
	if reserveOut.Sign() < 0 {
		reserveOut.SetInt64(0)
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.cursors = slices.Clone(s.cursors)
	cloned.Info.Reserves = lo.Map(s.Info.Reserves, func(r *big.Int, _ int) *big.Int { return new(big.Int).Set(r) })
	return &cloned
}

// GetMetaInfo returns the venue swaps execute against, via
// swap(tokenIn, tokenOut, amountIn, minAmountOut, recipient, deadline) after tokenIn has been
// transferred to it.
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.MetaInfo{ApprovalAddress: s.staticExtra.Venue, BlockNumber: s.Info.BlockNumber}
}
