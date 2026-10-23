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

// CalcAmountOut is exactly the read-only contract quote at the snapshot block.
// Execution widening is exposed separately through State.QuoteExecution.
func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	i, o := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() < 0 {
		return nil, ErrOverflow
	}
	amount, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}
	q, err := s.state.QuoteExactInput(i, o, *amount)
	if err != nil {
		return nil, err
	}
	m := arithmetic{}
	swap := SwapInfo{InputIndex: i, OutputIndex: o, InputBalance: m.add(s.state.Balances[i], q.AmountIn), OutputBalance: m.sub(s.state.Balances[o], q.AmountOut)}
	if m.err != nil {
		return nil, m.err
	}
	return &pool.CalcAmountOutResult{TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: q.AmountOut.ToBig()}, Fee: &pool.TokenAmount{Token: params.TokenOut, Amount: new(big.Int)}, Gas: defaultGas, SwapInfo: swap}, nil
}
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swap, ok := params.SwapInfo.(SwapInfo)
	if !ok || swap.InputIndex < 0 || swap.OutputIndex < 0 || swap.InputIndex >= len(s.state.Tokens) || swap.OutputIndex >= len(s.state.Tokens) {
		return
	}
	s.state.Balances[swap.InputIndex] = swap.InputBalance
	s.state.Balances[swap.OutputIndex] = swap.OutputBalance
	s.Info.Reserves[swap.InputIndex] = swap.InputBalance.ToBig()
	s.Info.Reserves[swap.OutputIndex] = swap.OutputBalance.ToBig()
}
func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = slices.Clone(s.Info.Reserves)
	// Integer pointers are never mutated: updates replace them. State balances
	// are fixed arrays copied by value; token metadata and words are immutable.
	return &cloned
}
func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.MetaInfo{ApprovalAddress: s.Info.Address, BlockNumber: s.Info.BlockNumber}
}
