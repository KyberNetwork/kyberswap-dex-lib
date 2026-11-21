package mhocdesk

import (
	"errors"
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// Tokens[0] is BASE and Tokens[1] is QUOTE (set by the list updater from BASE()/QUOTE()).
const (
	baseIdx  = 0
	quoteIdx = 1
)

type PoolSimulator struct {
	pool.Pool

	extra     Extra
	d         *uint256.Int // 10^baseDecimals * PRICE_SCALE
	quoteUnit *uint256.Int // 10^quoteDecimals
	kScale    uint64       // K_SCALE

	// quoteAt is the clock for block.timestamp: the snapshot time when indexing, 0 = the wall clock.
	quoteAt int64
}

var _ = pool.RegisterFactory(DexType, func(params pool.FactoryParams) (*PoolSimulator, error) {
	sim, err := NewPoolSimulator(params.EntityPool)
	if err != nil {
		return nil, err
	}
	if t := pool.QuoteTime(params.Opts.StaleCheck, params.EntityPool.Timestamp); !t.IsZero() {
		sim.quoteAt = t.Unix()
	}
	return sim, nil
})

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	if len(ep.Tokens) != 2 || len(ep.Reserves) != 2 {
		return nil, errors.New("mhoc-desk: pool must have exactly 2 tokens and 2 reserves")
	}
	var se StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &se); err != nil {
		return nil, err
	}
	if len(ep.Extra) == 0 {
		return nil, errors.New("mhoc-desk: pool extra is empty")
	}
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}
	c := extra.Counters
	if extra.StateWord == nil || extra.FloorBase == nil || extra.FloorQuote == nil || c.CardFilled == nil ||
		c.BlockFilled == nil || c.WindowFilled == nil {
		return nil, errors.New("mhoc-desk: pool extra missing required fields")
	}
	ps, err := uint256.FromDecimal(se.PriceScale)
	if err != nil || ps.IsZero() || se.BaseDecimals > 18 || se.QuoteDecimals > 18 || se.KScale == 0 ||
		se.KScale > 256 {
		return nil, errors.New("mhoc-desk: invalid static extra")
	}
	d := new(uint256.Int).Mul(pow10(se.BaseDecimals), ps)

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      []string{ep.Tokens[baseIdx].Address, ep.Tokens[quoteIdx].Address},
			Reserves:    lo.Map(ep.Reserves, func(s string, _ int) *big.Int { return bignumber.NewBig(s) }),
			BlockNumber: ep.BlockNumber,
		}},
		extra:     extra,
		d:         d,
		quoteUnit: pow10(se.QuoteDecimals),
		kScale:    uint64(se.KScale),
	}, nil
}

// now is block.timestamp for the simulated swap: never earlier than the snapshot's own block.
func (s *PoolSimulator) now() uint64 {
	t := uint64(time.Now().Unix())
	if s.quoteAt > 0 {
		t = uint64(s.quoteAt)
	}
	return max(t, s.extra.ReadTimestamp)
}

// windowNow is block.timestamp for the per-minute budget: the earliest time the swap can land (the snapshot's own
// block.timestamp + 1), capped by now. A wall clock that has crossed a minute the chain has not yet reached must not
// free the minute's budget, or the router would quote sizes that revert with ExceedsWindowCap. Counting the
// snapshot's minute when the swap may land in a later one only makes the quote stricter than the pool.
func (s *PoolSimulator) windowNow(now uint64) uint64 {
	return min(now, s.extra.ReadTimestamp+1)
}

// nextBlock is block.number for the simulated swap: the block after the tracked one, so the per-block counter of
// the snapshot no longer counts (the per-card counter does, if the card is the same), and a card whose
// validUntilBlock is the tracked block itself is no longer tradable (ErrStalePrice). Every swap of one route runs
// in that same block, which UpdateBalance replays.
func (s *PoolSimulator) nextBlock() uint64 {
	return s.Info.BlockNumber + 1
}

// CalcAmountOut is PropAMM.quote(tokenIn, amountIn) evaluated on the tracked state in the next block.
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	now := s.now()
	return s.calcAt(params, s.nextBlock(), now, s.windowNow(now))
}

// calcAt evaluates the swap in a given block at a given block.timestamp (now) for the card's expiry, and at windowNow
// for the per-minute budget. With an exact timestamp (a replay) both are the same.
func (s *PoolSimulator) calcAt(params pool.CalcAmountOutParams, blockNumber, now, windowNow uint64) (
	*pool.CalcAmountOutResult, error) {
	indexIn := s.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := s.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrUnknownToken
	}
	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow || params.TokenAmountIn.Amount.Sign() < 0 {
		return nil, ErrExceedsMaxSize
	}
	balOut, overflow := uint256.FromBig(s.Info.Reserves[indexOut])
	if overflow {
		return nil, ErrOverflow
	}
	res, err := Quote(QuoteInput{
		D:           s.d,
		QuoteUnit:   s.quoteUnit,
		KScale:      s.kScale,
		Extra:       &s.extra,
		SellBase:    indexIn == baseIdx,
		AmountIn:    amountIn,
		BalanceOut:  balOut,
		BlockNumber: blockNumber,
		Now:         now,
		WindowNow:   windowNow,
	})
	if err != nil {
		return nil, err
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: res.AmountOut.ToBig()},
		// The spread is inside the price; the pool charges no separate fee.
		Fee:      &pool.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		Gas:      defaultGas,
		SwapInfo: SwapInfo{Notional: res.Notional, BlockNumber: blockNumber, Timestamp: windowNow},
	}, nil
}

// UpdateBalance replays the swap's state changes: both token balances and the three counters, each restarted first
// if the swap's card, block or minute differs from its tag (afterFill). Every write assigns a fresh value, so the
// shallow copy in CloneState isolates clones.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn := s.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return
	}
	s.Info.Reserves = []*big.Int{s.Info.Reserves[0], s.Info.Reserves[1]}
	s.Info.Reserves[indexIn] = new(big.Int).Add(s.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	s.Info.Reserves[indexOut] = new(big.Int).Sub(s.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
	if s.Info.Reserves[indexOut].Sign() < 0 {
		s.Info.Reserves[indexOut] = new(big.Int)
	}

	info, ok := params.SwapInfo.(SwapInfo)
	if !ok || info.Notional == nil {
		return
	}
	s.extra.Counters = afterFill(&s.extra.Counters, decodeCard(s.extra.StateWord).seq, info.Notional,
		info.BlockNumber, info.Timestamp)
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = lo.Map(s.Info.Reserves, func(r *big.Int, _ int) *big.Int { return new(big.Int).Set(r) })
	return &cloned
}

// GetMetaInfo: the executor approves the pool for tokenIn and calls swap(tokenIn, amountIn, minOut, recipient).
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.MetaInfo{ApprovalAddress: s.Info.Address, BlockNumber: s.Info.BlockNumber}
}

func pow10(n uint8) *uint256.Int {
	return new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(uint64(n)))
}

// GetApprovalAddress: the pool itself pulls tokenIn with transferFrom.
func (s *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return s.Info.Address
}
