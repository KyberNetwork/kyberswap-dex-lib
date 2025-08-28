package uniswapv1

import (
	"math/big"
	"slices"

	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	reserves []*uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
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
		reserves: reserves,
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

	amountOut, err := s.getInputPrice(amountIn, reserveIn, reserveOut)
	if err != nil {
		return nil, err
	} else if amountOut.Cmp(reserveOut) > 0 {
		return nil, ErrInsufficientLiquidity
	} else if amountOut.Sign() <= 0 {
		return nil, ErrInsufficientOutputAmount
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[indexIn], Amount: bignumber.ZeroBI},
		Gas:            defaultGas,
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

	amountIn, err := s.getOutputPrice(amountOut, reserveIn, reserveOut)
	if err != nil {
		return nil, err
	} else if amountIn.Sign() <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: s.Info.Tokens[indexIn], Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: s.Info.Tokens[indexIn], Amount: bignumber.ZeroBI},
		Gas:           defaultGas,
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

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return PoolMeta{
		BlockNumber:     s.Info.BlockNumber,
		ApprovalAddress: s.GetApprovalAddress(tokenIn, tokenOut),
	}
}

func (s *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	return lo.Ternary(valueobject.IsNative(tokenIn), "", s.GetAddress())
}

// def getInputPrice(input_amount: uint256, input_reserve: uint256, output_reserve: uint256) -> uint256:
//
//	assert input_reserve > 0 and output_reserve > 0
//	input_amount_with_fee: uint256 = input_amount * 997
//	numerator: uint256 = input_amount_with_fee * output_reserve
//	denominator: uint256 = (input_reserve * 1000) + input_amount_with_fee
//	return numerator / denominator
func (s *PoolSimulator) getInputPrice(inputAmount, inputReserve, outputReserve *uint256.Int) (*uint256.Int, error) {
	var inputAmountWithFee, numerator, denominator uint256.Int

	inputAmountWithFee.Mul(inputAmount, U997)

	numerator.Mul(&inputAmountWithFee, outputReserve)

	denominator.Mul(inputReserve, U1000)
	denominator.Add(&denominator, &inputAmountWithFee)

	return numerator.Div(&numerator, &denominator), nil
}

// def getOutputPrice(output_amount: uint256, input_reserve: uint256, output_reserve: uint256) -> uint256:
//
//	assert input_reserve > 0 and output_reserve > 0
//	numerator: uint256 = input_reserve * output_amount * 1000
//	denominator: uint256 = (output_reserve - output_amount) * 997
//	return numerator / denominator + 1
func (s *PoolSimulator) getOutputPrice(outputAmount, inputReserve, outputReserve *uint256.Int) (*uint256.Int, error) {
	var numerator, denominator uint256.Int

	numerator.Mul(inputReserve, outputAmount)
	numerator.Mul(&numerator, U1000)

	denominator.Sub(outputReserve, outputAmount)
	denominator.Mul(&denominator, U997)

	result := numerator.Div(&numerator, &denominator)
	return result.AddUint64(result, 1), nil
}
