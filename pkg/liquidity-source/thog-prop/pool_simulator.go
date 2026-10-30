package thogprop

import (
	"math/big"
	"slices"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	pool.Pool
	staticExtra StaticExtra
	state       State
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

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
	return &PoolSimulator{Pool: pool.Pool{Info: pool.PoolInfo{Address: ep.Address, Exchange: ep.Exchange, Type: ep.Type, Tokens: tokens, Reserves: reserves, BlockNumber: ep.BlockNumber}}, staticExtra: static, state: state}, nil
}

// CalcAmountOut models one block of delay without an intervening price update.
// At the deployed policy this adds 50 hundredths of a bp to the curve spread.
// Gas-price/account-warmth penalties require an actual execution context and are
// not assumed by this router interface. State.QuoteExecution exposes them.
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	i, o := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() < 0 {
		return nil, ErrOverflow
	}
	amount, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}
	if s.state.BlockNumber == ^uint64(0) {
		return nil, ErrOverflow
	}
	quoteBlock := s.state.BlockNumber + executionAgeLookahead
	c, err := s.state.prepare(i, o, quoteBlock)
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
	return pool.MetaInfo{ApprovalAddress: s.Info.Address, BlockNumber: s.Info.BlockNumber}
}

func (s *PoolSimulator) estimateGas(input, output int) int64 {
	if s.state.Tokens[input].Category == s.state.Tokens[output].Category {
		return sameCategoryGas
	}
	return crossCategoryGas
}
