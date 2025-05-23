package uniswapv1

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	ErrInvalidToken             = errors.New("invalid token")
	ErrInvalidReserve           = errors.New("invalid reserve")
	ErrInvalidAmountIn          = errors.New("invalid amount in")
	ErrInsufficientInputAmount  = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInvalidAmountOut         = errors.New("invalid amount out")
	ErrInsufficientOutputAmount = errors.New("INSUFFICIENT_OUTPUT_AMOUNT")
	ErrInsufficientLiquidity    = errors.New("INSUFFICIENT_LIQUIDITY")
)

type (
	PoolSimulator struct {
		pool.Pool
		gas Gas
	}

	Gas struct {
		Swap int64
	}
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return utils.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		gas: defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
	)

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmountIn
	}

	if amountIn.Cmp(number.Zero) <= 0 {
		return nil, ErrInsufficientInputAmount
	}

	reserveIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, ErrInvalidReserve
	}

	reserveOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, ErrInvalidReserve
	}

	amountOut, err := s.getInputPrice(amountIn, reserveIn, reserveOut)
	if err != nil {
		return nil, err
	}

	if amountOut.LtUint64(1) {
		return nil, ErrInsufficientOutputAmount
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas:            s.gas.Swap,
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	var (
		tokenAmountOut = param.TokenAmountOut
		tokenIn        = param.TokenIn
	)
	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmountOut
	}

	if amountOut.Cmp(number.Zero) <= 0 {
		return nil, ErrInsufficientOutputAmount
	}

	reserveIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, ErrInvalidReserve
	}

	reserveOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, ErrInvalidReserve
	}

	amountIn, err := s.getOutputPrice(amountOut, reserveIn, reserveOut)
	if err != nil {
		return nil, err
	}

	if amountIn.Cmp(reserveIn) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas:           s.gas.Swap,
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	s.Pool.Info.Reserves[indexIn] = new(big.Int).Add(s.Pool.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	s.Pool.Info.Reserves[indexOut] = new(big.Int).Sub(s.Pool.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return PoolMeta{
		BlockNumber:     s.Pool.Info.BlockNumber,
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
	if inputReserve.CmpUint64(0) <= 0 || outputReserve.CmpUint64(0) <= 0 {
		return nil, ErrInsufficientLiquidity
	}

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
	if inputReserve.CmpUint64(0) <= 0 || outputReserve.CmpUint64(0) <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	var numerator, denominator uint256.Int

	numerator.Mul(inputReserve, outputAmount)
	numerator.Mul(&numerator, U1000)

	denominator.Sub(outputReserve, outputAmount)
	denominator.Mul(&denominator, U997)

	result := new(uint256.Int).Div(&numerator, &denominator)

	return result.AddUint64(result, 1), nil
}
