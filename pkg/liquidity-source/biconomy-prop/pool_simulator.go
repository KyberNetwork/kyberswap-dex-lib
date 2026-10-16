package biconomyprop

import (
	"math/big"
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
	cursors [][2]*uint256.Int

	now uint64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	if len(ep.Tokens) != 2 || len(ep.Reserves) != 2 {
		return nil, ErrInvalidToken
	}

	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	cursors := make([][2]*uint256.Int, len(extra.Members))
	for i := range extra.Members {
		cursors[i][0] = cursorOf(&extra.Members[i].Dir0)
		cursors[i][1] = cursorOf(&extra.Members[i].Dir1)
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
		now:         uint64(time.Now().Unix()),
	}, nil
}

func cursorOf(b *Board) *uint256.Int {
	if b.Filled == nil {
		return new(uint256.Int)
	}
	return b.Filled.Clone()
}

func (s *PoolSimulator) board(member, dirIndex int) *Board {
	if dirIndex == 0 {
		return &s.members[member].Dir0
	}
	return &s.members[member].Dir1
}

// merge is the venue's _merge for one direction: allocate, then drop any maker whose allocation
// pays out zero (the executor would revert) and allocate again over the rest.
func (s *PoolSimulator) merge(dirIndex int, amountIn *uint256.Int) (alloc []*uint256.Int, out, covered *uint256.Int) {
	excluded := make([]bool, len(s.members))
	for i := range s.members {
		excluded[i] = !boardLive(s.board(i, dirIndex), s.now)
	}
	for {
		var outBy []*uint256.Int
		alloc, outBy, out, covered = s.allocate(dirIndex, amountIn, excluded)
		redo := false
		for i := range alloc {
			if !alloc[i].IsZero() && outBy[i].IsZero() {
				excluded[i] = true
				redo = true
			}
		}
		if !redo {
			return alloc, out, covered
		}
	}
}

// allocate is one pass of the venue's _allocate.
func (s *PoolSimulator) allocate(dirIndex int, amountIn *uint256.Int, excluded []bool) (alloc, outBy []*uint256.Int, out, covered *uint256.Int) {
	k := len(s.members)
	alloc = make([]*uint256.Int, k)
	outBy = make([]*uint256.Int, k)
	cursor := make([]*uint256.Int, k)
	left := make([]*uint256.Int, k)
	next := make([]int, k) // -1 once a maker is out
	for i := 0; i < k; i++ {
		alloc[i], outBy[i] = new(uint256.Int), new(uint256.Int)
		next[i] = -1
		if excluded[i] {
			continue
		}
		b := s.board(i, dirIndex)
		cursor[i] = s.cursors[i][dirIndex].Clone()
		limit := fillLimit(b)
		if !cursor[i].Lt(limit) {
			continue
		}
		left[i] = new(uint256.Int).Sub(limit, cursor[i])
		j := 0
		for j < len(b.Sizes) && !b.Sizes[j].Gt(cursor[i]) {
			j++
		}
		if j < len(b.Sizes) {
			next[i] = j
		}
	}

	out, covered = new(uint256.Int), new(uint256.Int)
	remaining := amountIn.Clone()
	var take, segOut uint256.Int
	for !remaining.IsZero() {
		best := -1
		var bestPrice *uint256.Int
		for i := 0; i < k; i++ {
			if next[i] < 0 {
				continue
			}
			if price := s.board(i, dirIndex).Prices[next[i]]; best < 0 || price.Gt(bestPrice) {
				best, bestPrice = i, price
			}
		}
		if best < 0 {
			break
		}
		top := s.board(best, dirIndex).Sizes[next[best]]
		take.Sub(top, cursor[best])
		if take.Gt(remaining) {
			take.Set(remaining)
		}
		if take.Gt(left[best]) {
			take.Set(left[best])
		}
		segmentOut(&segOut, &take, bestPrice)
		alloc[best].Add(alloc[best], &take)
		outBy[best].Add(outBy[best], &segOut)
		out.Add(out, &segOut)
		covered.Add(covered, &take)
		remaining.Sub(remaining, &take)
		cursor[best].Add(cursor[best], &take)
		left[best].Sub(left[best], &take)
		if left[best].IsZero() {
			next[best] = -1
		} else if cursor[best].Eq(top) {
			if next[best]++; next[best] >= len(s.board(best, dirIndex).Sizes) {
				next[best] = -1
			}
		}
	}
	return alloc, outBy, out, covered
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}
	if amountIn.IsZero() {
		return nil, ErrZeroAmountIn
	}

	dirIndex := indexIn
	alloc, gross, covered := s.merge(dirIndex, amountIn)
	// The venue never fills partially: short coverage reverts Inactive().
	if covered.Lt(amountIn) {
		return nil, ErrInsufficientLiquidity
	}
	amountOut, fee := afterProtocolFee(gross, s.feeBps)
	if amountOut.IsZero() {
		return nil, ErrZeroAmountOut
	}

	var takes []memberTake
	for i, a := range alloc {
		if !a.IsZero() {
			takes = append(takes, memberTake{Member: i, AmountIn: a})
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: params.TokenOut, Amount: fee.ToBig()},
		Gas:            defaultGas + int64(len(takes)-1)*perMemberGas,
		SwapInfo:       SwapInfo{DirIndex: dirIndex, Takes: takes, AmountOut: amountOut},
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
		cursor := s.cursors[take.Member][swapInfo.DirIndex]
		cursor.Add(cursor, take.AmountIn)
	}

	outIdx := 1 - swapInfo.DirIndex
	reserveOut := new(big.Int).Sub(s.Info.Reserves[outIdx], swapInfo.AmountOut.ToBig())
	if reserveOut.Sign() < 0 {
		reserveOut.SetInt64(0)
	}
	s.Info.Reserves[outIdx] = reserveOut
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.cursors = make([][2]*uint256.Int, len(s.cursors))
	for i := range s.cursors {
		cloned.cursors[i][0] = s.cursors[i][0].Clone()
		cloned.cursors[i][1] = s.cursors[i][1].Clone()
	}
	cloned.Info.Reserves = lo.Map(s.Info.Reserves, func(r *big.Int, _ int) *big.Int { return new(big.Int).Set(r) })
	return &cloned
}

// GetMetaInfo returns the venue swaps execute against, via
// swap(tokenIn, tokenOut, amountIn, minAmountOut, recipient, deadline) or
// swapWithFee(..., extraFeePpm, feeReceiver) after tokenIn has been transferred to it.
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return MetaInfo{Venue: s.staticExtra.Venue, BlockNumber: s.Info.BlockNumber}
}
