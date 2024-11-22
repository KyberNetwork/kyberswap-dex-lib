package ringswap

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v2"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrReserveIndexOutOfBounds = errors.New("reserve index out of bounds")
	ErrTokenIndexOutOfBounds   = errors.New("token index out of bounds")
	ErrTokenSwapNotAllowed     = errors.New("cannot swap between original token and wrapped token")
)

type (
	PoolSimulator struct {
		poolpkg.Pool
		fee          *uint256.Int
		feePrecision *uint256.Int

		gas              uniswapv2.Gas
		originalReserves uniswapv2.ReserveData
	}
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var originalReserves uniswapv2.ReserveData
	if err := json.Unmarshal([]byte(entityPool.Extra), &originalReserves); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
			Address:     entityPool.Address,
			ReserveUsd:  entityPool.ReserveUsd,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return utils.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		fee:              uint256.NewInt(staticExtra.Fee),
		feePrecision:     uint256.NewInt(staticExtra.FeePrecision),
		gas:              defaultGas,
		originalReserves: originalReserves,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
	)

	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, uniswapv2.ErrInvalidToken
	}

	if indexIn%2 == indexOut%2 {
		return nil, ErrTokenSwapNotAllowed
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, uniswapv2.ErrInvalidAmountIn
	}

	if amountIn.Cmp(number.Zero) <= 0 {
		return nil, uniswapv2.ErrInsufficientInputAmount
	}

	reserveIndex := indexIn % 2
	if reserveIndex >= len(s.Pool.Info.Reserves) {
		return nil, ErrReserveIndexOutOfBounds
	}
	reserveIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[reserveIndex])
	if overflow {
		return nil, uniswapv2.ErrInvalidReserve
	}

	reserveOutIndex := indexOut % 2
	if reserveOutIndex >= len(s.Pool.Info.Reserves) {
		return nil, ErrReserveIndexOutOfBounds
	}
	reserveOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[reserveOutIndex])
	if overflow {
		return nil, uniswapv2.ErrInvalidReserve
	}

	if reserveIn.Cmp(number.Zero) <= 0 || reserveOut.Cmp(number.Zero) <= 0 {
		return nil, uniswapv2.ErrInsufficientLiquidity
	}

	amountOut := s.getAmountOut(amountIn, reserveIn, reserveOut)

	// Ensure that amountOut does not exceed the fw reserve
	if amountOut.Cmp(reserveOut) > 0 {
		return nil, uniswapv2.ErrInsufficientLiquidity
	}

	originalReserve, overflow := uint256.FromBig(lo.Ternary(reserveOutIndex == 0, s.originalReserves.Reserve0, s.originalReserves.Reserve1))
	if overflow {
		return nil, uniswapv2.ErrInvalidReserve
	}

	// Ensure that amountOut does not exceed the original reserve
	if amountOut.Cmp(originalReserve) > 0 {
		return nil, uniswapv2.ErrInsufficientLiquidity
	}

	wTokenInIndex := indexIn%2 + 2
	if wTokenInIndex >= len(s.Pool.Info.Tokens) {
		return nil, ErrTokenIndexOutOfBounds
	}
	wTokenIn := s.Pool.Info.Tokens[wTokenInIndex]

	wTokenOutIndex := indexOut%2 + 2
	if wTokenOutIndex >= len(s.Pool.Info.Tokens) {
		return nil, ErrTokenIndexOutOfBounds
	}
	wTokenOut := s.Pool.Info.Tokens[wTokenOutIndex]

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		// NOTE: we don't use fee to update balance so that we don't need to calculate it. I put it number.Zero to avoid null pointer exception
		Fee: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas: s.gas.Swap,
		SwapInfo: SwapInfo{
			WTokenIn:    wTokenIn,
			WTokenOut:   wTokenOut,
			IsToken0To1: indexIn%2 == 0,
			IsWrapIn:    indexIn < 2,
			IsUnwrapOut: indexOut < 2,
		},
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param poolpkg.CalcAmountInParams) (*poolpkg.CalcAmountInResult, error) {
	var (
		tokenAmountOut = param.TokenAmountOut
		tokenIn        = param.TokenIn
	)

	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, uniswapv2.ErrInvalidToken
	}

	if indexIn%2 == indexOut%2 {
		return nil, ErrTokenSwapNotAllowed
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, uniswapv2.ErrInvalidAmountOut
	}

	if amountOut.Cmp(number.Zero) <= 0 {
		return nil, uniswapv2.ErrInsufficientOutputAmount
	}

	reserveIndex := indexIn % 2
	if reserveIndex >= len(s.Pool.Info.Reserves) {
		return nil, ErrReserveIndexOutOfBounds
	}
	reserveIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[reserveIndex])
	if overflow {
		return nil, uniswapv2.ErrInvalidReserve
	}

	reserveOutIndex := indexOut % 2
	if reserveOutIndex >= len(s.Pool.Info.Reserves) {
		return nil, ErrReserveIndexOutOfBounds
	}
	reserveOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[reserveOutIndex])
	if overflow {
		return nil, uniswapv2.ErrInvalidReserve
	}

	// Ensure that amountOut does not exceed the fw reserve
	if reserveIn.Cmp(number.Zero) <= 0 || reserveOut.Cmp(number.Zero) <= 0 {
		return nil, uniswapv2.ErrInsufficientLiquidity
	}

	originalReserve, overflow := uint256.FromBig(lo.Ternary(reserveOutIndex == 0, s.originalReserves.Reserve0, s.originalReserves.Reserve1))
	if overflow {
		return nil, uniswapv2.ErrInvalidReserve
	}

	// Ensure that amountOut does not exceed the original reserve
	if amountOut.Cmp(originalReserve) > 0 {
		return nil, uniswapv2.ErrInsufficientLiquidity
	}

	amountIn, err := s.getAmountIn(amountOut, reserveIn, reserveOut)
	if err != nil {
		return nil, err
	}

	if amountIn.Cmp(reserveIn) > 0 {
		return nil, uniswapv2.ErrInsufficientLiquidity
	}

	balanceIn := new(uint256.Int).Add(reserveIn, amountIn)
	balanceOut := new(uint256.Int).Sub(reserveOut, amountOut)

	balanceInAdjusted := new(uint256.Int).Sub(
		new(uint256.Int).Mul(balanceIn, s.feePrecision),
		new(uint256.Int).Mul(amountIn, s.fee),
	)
	balanceOutAdjusted := new(uint256.Int).Mul(balanceOut, s.feePrecision)

	kBefore := new(uint256.Int).Mul(new(uint256.Int).Mul(reserveIn, reserveOut), new(uint256.Int).Mul(s.feePrecision, s.feePrecision))
	kAfter := new(uint256.Int).Mul(balanceInAdjusted, balanceOutAdjusted)

	if kAfter.Cmp(kBefore) < 0 {
		return nil, uniswapv2.ErrInvalidK
	}

	wTokenInIndex := indexIn%2 + 2
	if wTokenInIndex >= len(s.Pool.Info.Tokens) {
		return nil, ErrTokenIndexOutOfBounds
	}
	wTokenIn := s.Pool.Info.Tokens[wTokenInIndex]

	wTokenOutIndex := indexOut%2 + 2
	if wTokenOutIndex >= len(s.Pool.Info.Tokens) {
		return nil, ErrTokenIndexOutOfBounds
	}
	wTokenOut := s.Pool.Info.Tokens[wTokenOutIndex]

	return &poolpkg.CalcAmountInResult{
		TokenAmountIn: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: amountIn.ToBig()},
		// NOTE: we don't use fee to update balance so that we don't need to calculate it. I put it number.Zero to avoid null pointer exception
		Fee: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas: s.gas.Swap,
		SwapInfo: SwapInfo{
			WTokenIn:    wTokenIn,
			WTokenOut:   wTokenOut,
			IsToken0To1: indexIn%2 == 0,
			IsWrapIn:    indexIn < 2,
			IsUnwrapOut: indexOut < 2,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	s.Pool.Info.Reserves[indexIn%2] = new(big.Int).Add(s.Pool.Info.Reserves[indexIn%2], params.TokenAmountIn.Amount)
	s.Pool.Info.Reserves[indexOut%2] = new(big.Int).Sub(s.Pool.Info.Reserves[indexOut%2], params.TokenAmountOut.Amount)

	if indexIn%2 == 0 {
		s.originalReserves.Reserve0 = new(big.Int).Add(s.originalReserves.Reserve0, params.TokenAmountIn.Amount)
		s.originalReserves.Reserve1 = new(big.Int).Sub(s.originalReserves.Reserve1, params.TokenAmountOut.Amount)
	} else {
		s.originalReserves.Reserve1 = new(big.Int).Add(s.originalReserves.Reserve1, params.TokenAmountIn.Amount)
		s.originalReserves.Reserve0 = new(big.Int).Sub(s.originalReserves.Reserve0, params.TokenAmountOut.Amount)
	}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return uniswapv2.PoolMeta{
		Fee:          s.fee.Uint64(),
		FeePrecision: s.feePrecision.Uint64(),
		BlockNumber:  s.Pool.Info.BlockNumber,
	}
}

func (s *PoolSimulator) getAmountOut(amountIn, reserveIn, reserveOut *uint256.Int) *uint256.Int {
	amountInWithFee := new(uint256.Int).Mul(amountIn, new(uint256.Int).Sub(s.feePrecision, s.fee))
	numerator := new(uint256.Int).Mul(amountInWithFee, reserveOut)
	denominator := new(uint256.Int).Add(new(uint256.Int).Mul(reserveIn, s.feePrecision), amountInWithFee)

	return new(uint256.Int).Div(numerator, denominator)
}

func (s *PoolSimulator) getAmountIn(amountOut, reserveIn, reserveOut *uint256.Int) (amountIn *uint256.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	numerator := uniswapv2.SafeMul(
		uniswapv2.SafeMul(reserveIn, amountOut),
		s.feePrecision,
	)
	denominator := uniswapv2.SafeMul(
		uniswapv2.SafeSub(reserveOut, amountOut),
		uniswapv2.SafeSub(s.feePrecision, s.fee),
	)

	return uniswapv2.SafeAdd(new(uint256.Int).Div(numerator, denominator), number.Number_1), nil
}
