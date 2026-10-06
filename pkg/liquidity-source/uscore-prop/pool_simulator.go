package uscoreprop

import (
	"math"
	"math/big"
	"time"

	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	*ladder.PoolSimulator
	consumedIn, consumedOut [2]float64
}

var _ = pool.RegisterFactory(DexType, NewPoolSimulator)

func NewPoolSimulator(params pool.FactoryParams) (*PoolSimulator, error) {
	return NewPoolSimulatorWith(params.EntityPool, lo.Ternary(params.Opts.StaleCheck, MaxAge, math.MaxInt64))
}

func NewPoolSimulatorWith(ep entity.Pool, maxAge time.Duration) (*PoolSimulator, error) {
	base, err := ladder.NewPoolSimulatorWith(ep, maxAge)
	if err != nil {
		return nil, err
	}
	for _, points := range base.Extra.Ladders {
		lastIn, lastOut := 0.0, 0.0
		for _, point := range points {
			if !finite(point.AmountIn()) || !finite(point.AmountOut()) || point.AmountIn() <= lastIn || point.AmountOut() <= 0 || point.AmountOut() < lastOut {
				return nil, ErrInvalidState
			}
			lastIn, lastOut = point.AmountIn(), point.AmountOut()
		}
	}
	base.Gas = defaultGas
	return &PoolSimulator{PoolSimulator: base}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.PoolSimulator = s.PoolSimulator.CloneState().(*ladder.PoolSimulator)
	return &cloned
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	in, out := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if in < 0 || out < 0 || in == out {
		return nil, ladder.ErrInvalidToken
	}
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ladder.ErrZeroAmountIn
	}
	amount, _ := params.TokenAmountIn.Amount.Float64()
	totalIn := s.consumedIn[in] + amount
	if totalIn <= s.consumedIn[in] {
		return nil, ladder.ErrNoQuote
	}
	total, err := interpolate(s.Extra.Ladders[in], totalIn)
	if err != nil {
		return nil, err
	}
	value := total - s.consumedOut[in]
	if !finite(value) || value <= 0 {
		return nil, ladder.ErrNoQuote
	}
	output, _ := big.NewFloat(value).Int(nil)
	if output.Sign() <= 0 {
		return nil, ladder.ErrNoQuote
	}
	reserve := s.Reserve1
	if in == 1 {
		reserve = s.Reserve0
	}
	if output.Cmp(reserve.ToBig()) > 0 {
		return nil, ladder.ErrInsufficientLiquidity
	}
	if params.Limit != nil && output.Cmp(params.Limit.GetLimit(params.TokenOut)) > 0 {
		return nil, pool.ErrNotEnoughInventory
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: params.TokenOut, Amount: output},
		Fee:            &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: big.NewInt(0)},
		Gas:            s.Gas,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	in, out := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if in < 0 || out < 0 || in == out || params.TokenAmountIn.Amount == nil || params.TokenAmountOut.Amount == nil || params.TokenAmountIn.Amount.Sign() <= 0 || params.TokenAmountOut.Amount.Sign() <= 0 {
		return
	}
	iu, io := uint256.FromBig(params.TokenAmountIn.Amount)
	ou, oo := uint256.FromBig(params.TokenAmountOut.Amount)
	if io || oo || iu == nil || ou == nil {
		return
	}
	reserveIn, reserveOut := s.Reserve0, s.Reserve1
	if in == 1 {
		reserveIn, reserveOut = reserveOut, reserveIn
	}
	var next uint256.Int
	if _, overflow := next.AddOverflow(reserveIn, iu); overflow || ou.Gt(reserveOut) {
		return
	}
	amount, _ := params.TokenAmountIn.Amount.Float64()
	output, _ := params.TokenAmountOut.Amount.Float64()
	s.consumedIn[in] += amount
	s.consumedOut[in] += output
	s.PoolSimulator.UpdateBalance(params)
}
