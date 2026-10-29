package thogprop

import (
	"math/big"
	"slices"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	pool.Pool
	staticExtra      StaticExtra
	state            State
	capturedAtMs     int64
	blockTimeMs      int64
	maxSnapshotAgeMs int64
	noStaleCheck     bool
}

var _ = pool.RegisterFactory(DexType, NewPoolSimulatorFromParams)

func NewPoolSimulatorFromParams(params pool.FactoryParams) (*PoolSimulator, error) {
	s, err := NewPoolSimulator(params.EntityPool)
	if err == nil {
		s.noStaleCheck = !params.Opts.StaleCheck
	}
	return s, err
}

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var static StaticExtra
	var extra Extra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &static); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}
	if err := extra.State.validate(); err != nil {
		return nil, err
	}
	if extra.CapturedAtMs < 0 || extra.BlockTimeMs < 0 || extra.MaxSnapshotAgeMs < 0 {
		return nil, ErrInvalidState
	}
	if extra.CapturedAtMs == 0 && ep.Timestamp > 0 && ep.Timestamp <= int64(^uint64(0)>>1)/1000 {
		extra.CapturedAtMs = ep.Timestamp * 1000
	}
	// Block timestamps have only second precision. The end of that second is
	// the latest possible capture time: a lagging provider cannot refresh it.
	if extra.BlockTimestamp > uint64((int64(^uint64(0)>>1)-999)/1000) {
		return nil, ErrInvalidState
	}
	if extra.BlockTimestamp > 0 && extra.CapturedAtMs > int64(extra.BlockTimestamp)*1000+999 {
		extra.CapturedAtMs = int64(extra.BlockTimestamp)*1000 + 999
	}
	state := extra.State
	n := len(state.Tokens)
	if len(ep.Tokens) != n || len(ep.Reserves) != n || ep.BlockNumber != state.BlockNumber {
		return nil, ErrInvalidState
	}
	tokens := make([]string, n)
	reserves := make([]*big.Int, n)
	for i, t := range state.Tokens {
		if ep.Tokens[i] == nil || ep.Tokens[i].Address != t.Address {
			return nil, ErrInvalidState
		}
		value, err := uint256.FromDecimal(ep.Reserves[i])
		if err != nil || *value != state.Balances[i] {
			return nil, ErrInvalidState
		}
		tokens[i] = t.Address
		reserves[i] = value.ToBig()
	}
	return &PoolSimulator{Pool: pool.Pool{Info: pool.PoolInfo{Address: ep.Address, Exchange: ep.Exchange, Type: ep.Type, Tokens: tokens, Reserves: reserves, BlockNumber: ep.BlockNumber}}, staticExtra: static, state: state,
		capturedAtMs: extra.CapturedAtMs, blockTimeMs: defaultMs(extra.BlockTimeMs, defaultBlockTimeMs), maxSnapshotAgeMs: defaultMs(extra.MaxSnapshotAgeMs, defaultMaxSnapshotAgeMs)}, nil
}

// CalcAmountOut advances the execution block with elapsed snapshot time and one
// block of inclusion delay. The stored contract policy supplies all age widening.
// Gas-price/account-warmth penalties require an actual execution context and are
// not assumed by this router interface. State.QuoteExecution exposes them.
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	return s.calcAmountOut(params, time.Now().UnixMilli())
}

func (s *PoolSimulator) calcAmountOut(params pool.CalcAmountOutParams, nowMs int64) (*pool.CalcAmountOutResult, error) {
	i, o := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() < 0 {
		return nil, ErrOverflow
	}
	amount, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}
	quoteBlock, snapshotAge, err := s.quoteContext(nowMs)
	if err != nil {
		return nil, err
	}
	c, err := s.state.prepareWithAgeCheck(i, o, quoteBlock, !s.noStaleCheck)
	if err != nil {
		return nil, err
	}
	q, err := s.state.quoteReadOnly(&c, *amount)
	if err != nil {
		return nil, err
	}
	swap, err := s.state.settlement(&c, q.AmountIn, q.AmountOut)
	if err != nil {
		return nil, err
	}
	swap.QuoteBlock, swap.SnapshotAgeMs = quoteBlock, snapshotAge
	return &pool.CalcAmountOutResult{TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: q.AmountOut.ToBig()}, Fee: &pool.TokenAmount{Token: params.TokenOut, Amount: new(big.Int)}, Gas: s.estimateGas(i, o), SwapInfo: swap}, nil
}
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swap, ok := params.SwapInfo.(SwapInfo)
	if !ok || swap.InputIndex < 0 || swap.OutputIndex < 0 || swap.InputIndex >= len(s.state.Tokens) || swap.OutputIndex >= len(s.state.Tokens) {
		return
	}
	s.state.Balances[swap.InputIndex] = swap.InputBalance
	s.state.Balances[swap.OutputIndex] = swap.OutputBalance
	for _, endpoint := range []struct {
		index int
		units uint256.Int
	}{{swap.InputIndex, swap.InputUnits}, {swap.OutputIndex, swap.OutputUnits}} {
		s.state.setAccountedUnits(endpoint.index, endpoint.units)
		if isXStock(s.state.Tokens[endpoint.index].Address) {
			s.state.Shares[endpoint.index].Shares = endpoint.units
		}
	}
	s.Info.Reserves[swap.InputIndex] = swap.InputBalance.ToBig()
	s.Info.Reserves[swap.OutputIndex] = swap.OutputBalance.ToBig()
}
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = slices.Clone(s.Info.Reserves)
	// Integer pointers are never mutated: updates replace them. State balances
	// and shares/words are fixed arrays copied by value; token metadata is immutable.
	return &cloned
}
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	block, age, _ := s.quoteContext(time.Now().UnixMilli())
	return MetaInfo{ApprovalAddress: s.Info.Address, BlockNumber: s.Info.BlockNumber, QuoteBlock: block, SnapshotAgeMs: age}
}

func (s *PoolSimulator) quoteContext(nowMs int64) (block uint64, ageMs int64, err error) {
	ageMs = -1
	if s.capturedAtMs > 0 && nowMs >= s.capturedAtMs {
		ageMs = nowMs - s.capturedAtMs
	}
	var elapsed uint64
	if !s.noStaleCheck {
		if ageMs < 0 {
			return 0, ageMs, ErrStaleSnapshot
		}
		elapsed = uint64(ageMs / s.blockTimeMs)
		if ageMs%s.blockTimeMs != 0 {
			elapsed++
		}
	}
	if elapsed == ^uint64(0) || s.state.BlockNumber > ^uint64(0)-elapsed-executionAgeLookahead {
		return 0, ageMs, ErrOverflow
	}
	block = s.state.BlockNumber + elapsed + executionAgeLookahead
	if !s.noStaleCheck {
		posted, maxAge := s.state.small(2, 36, 32), s.state.small(2, 138, 8)
		if ageMs > s.maxSnapshotAgeMs || (posted > 0 && posted <= block && maxAge > 0 && block-posted > maxAge) {
			err = ErrStaleSnapshot
		}
	}
	return block, ageMs, err
}

func (s *PoolSimulator) estimateGas(input, output int) int64 {
	if s.state.Tokens[input].Category == s.state.Tokens[output].Category {
		return sameCategoryGas
	}
	return crossCategoryGas
}
