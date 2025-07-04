package uniswapv2

import (
	"fmt"
	"math/big"
	"slices"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	reserves     []*uint256.Int
	fee          *uint256.Int
	feePrecision *uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	reserves := make([]*uint256.Int, len(entityPool.Reserves))
	for i, reserveStr := range entityPool.Reserves {
		reserve, err := uint256.FromDecimal(reserveStr)
		if err != nil {
			return nil, errors.WithMessage(err, "invalid reserve")
		}
		reserves[i] = reserve
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens: lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves: lo.Map(entityPool.Reserves,
				func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		reserves:     reserves,
		fee:          uint256.NewInt(extra.Fee),
		feePrecision: uint256.NewInt(extra.FeePrecision),
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	reserveIn := s.reserves[indexIn]
	reserveOut := s.reserves[indexOut]

	amountOut := s.getAmountOut(amountIn, reserveIn, reserveOut)
	if amountOut.Cmp(reserveOut) > 0 {
		return nil, ErrInsufficientLiquidity
	} else if amountOut.Sign() <= 0 {
		return nil, ErrInsufficientOutputAmount
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: bignumber.ZeroBI},
		Gas:            defaultGas + extraGasByExchange[s.GetExchange()],
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenAmountOut, tokenIn := param.TokenAmountOut, param.TokenIn
	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}
	reserveIn, reserveOut := s.reserves[indexIn], s.reserves[indexOut]

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow || amountOut.Sign() <= 0 {
		return nil, ErrInvalidAmountOut
	} else if amountOut.Cmp(reserveOut) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	amountIn, err := s.getAmountIn(amountOut, reserveIn, reserveOut)
	if err != nil {
		return nil, err
	} else if amountIn.Sign() <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: bignumber.ZeroBI},
		Gas:           defaultGas + extraGasByExchange[s.GetExchange()],
	}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.reserves = slices.Clone(s.reserves)
	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	s.reserves[indexIn] = new(uint256.Int).Add(s.reserves[indexIn], uint256.MustFromBig(params.TokenAmountIn.Amount))
	s.reserves[indexOut] = new(uint256.Int).Sub(s.reserves[indexOut], uint256.MustFromBig(params.TokenAmountOut.Amount))
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	exchange := s.GetExchange()
	return PoolMeta{
		Extra: Extra{
			Fee:          s.fee.Uint64(),
			FeePrecision: s.feePrecision.Uint64(),
		},
		PoolMetaGeneric: PoolMetaGeneric{
			ApprovalAddress: routerAddressByExchange[exchange],
			NoFOT:           noFOTByExchange[exchange],
		},
	}
}

func (s *PoolSimulator) getAmountOut(amountIn, reserveIn, reserveOut *uint256.Int) *uint256.Int {
	var numerator, denominator uint256.Int
	amountInWithFee := numerator.Mul(amountIn, numerator.Sub(s.feePrecision, s.fee))
	denominator.Add(denominator.Mul(reserveIn, s.feePrecision), amountInWithFee)
	numerator.Mul(amountInWithFee, reserveOut)
	return numerator.Div(&numerator, &denominator)
}

func (s *PoolSimulator) getAmountIn(amountOut, reserveIn, reserveOut *uint256.Int) (amountIn *uint256.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			if recoveredError, ok := r.(error); ok {
				err = recoveredError
			} else {
				err = fmt.Errorf("unexpected panic: %v", r)
			}
		}
	}()

	numerator := SafeMul(
		SafeMul(reserveIn, amountOut),
		s.feePrecision,
	)
	denominator := SafeMul(
		SafeSub(reserveOut, amountOut),
		SafeSub(s.feePrecision, s.fee),
	)

	return SafeAdd(numerator.Div(numerator, denominator), number.Number_1), nil
}
