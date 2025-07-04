package ringswap

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	fee          *uint256.Int
	feePrecision *uint256.Int

	originalReserves uniswapv2.ReserveData
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

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
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens,
				func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves,
				func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		fee:              uint256.NewInt(staticExtra.Fee),
		feePrecision:     uint256.NewInt(staticExtra.FeePrecision),
		originalReserves: originalReserves,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
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

	isWrapIn := indexIn < 2
	isUnwrapOut := indexOut < 2

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, uniswapv2.ErrInvalidAmountIn
	}

	if amountIn.Cmp(number.Zero) <= 0 {
		return nil, uniswapv2.ErrInsufficientInputAmount
	}

	reserveIn, reserveOut, err := s.getReserves(indexIn, indexOut)
	if err != nil {
		return nil, err
	}

	amountOut := s.getAmountOut(amountIn, reserveIn, reserveOut)
	if amountOut.Cmp(reserveOut) >= 0 {
		return nil, uniswapv2.ErrInsufficientLiquidity
	}

	wTokenIn, wTokenOut, err := s.getWrappedTokens(indexIn, indexOut)
	if err != nil {
		return nil, err
	}

	// Ensure that amountOut does not exceed original reserve
	if isUnwrapOut {
		if param.Limit == nil {
			return nil, ErrNoSwapLimit
		}
		if amountOut.CmpBig(param.Limit.GetLimit(wTokenOut)) >= 0 {
			return nil, uniswapv2.ErrInsufficientLiquidity
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		// NOTE: we don't use fee to update balance so that we don't need to calculate it. I put it number.Zero to avoid null pointer exception
		Fee: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas: defaultGas,
		SwapInfo: SwapInfo{
			WTokenIn:    wTokenIn,
			WTokenOut:   wTokenOut,
			IsToken0To1: indexIn%2 == 0,
			IsWrapIn:    isWrapIn,
			IsUnwrapOut: isUnwrapOut,
		},
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
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

	isWrapIn := indexIn < 2
	isUnwrapOut := indexOut < 2

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, uniswapv2.ErrInvalidAmountOut
	}

	if amountOut.Cmp(number.Zero) <= 0 {
		return nil, uniswapv2.ErrInsufficientOutputAmount
	}

	reserveIn, reserveOut, err := s.getReserves(indexIn, indexOut)
	if err != nil {
		return nil, err
	}

	if amountOut.Cmp(reserveOut) >= 0 {
		return nil, uniswapv2.ErrInsufficientLiquidity
	}

	wTokenIn, wTokenOut, err := s.getWrappedTokens(indexIn, indexOut)
	if err != nil {
		return nil, err
	}

	// Ensure that amountOut does not exceed original reserve
	if isUnwrapOut {
		if param.Limit == nil {
			return nil, ErrNoSwapLimit
		}
		if amountOut.CmpBig(param.Limit.GetLimit(wTokenOut)) >= 0 {
			return nil, uniswapv2.ErrInsufficientLiquidity
		}
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

	kBefore := new(uint256.Int).Mul(new(uint256.Int).Mul(reserveIn, reserveOut),
		new(uint256.Int).Mul(s.feePrecision, s.feePrecision))
	kAfter := new(uint256.Int).Mul(balanceInAdjusted, balanceOutAdjusted)

	if kAfter.Cmp(kBefore) < 0 {
		return nil, uniswapv2.ErrInvalidK
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: amountIn.ToBig()},
		// NOTE: we don't use fee to update balance so that we don't need to calculate it. I put it number.Zero to avoid null pointer exception
		Fee: &pool.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas: defaultGas,
		SwapInfo: SwapInfo{
			WTokenIn:    wTokenIn,
			WTokenOut:   wTokenOut,
			IsToken0To1: indexIn%2 == 0,
			IsWrapIn:    isWrapIn,
			IsUnwrapOut: isUnwrapOut,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	s.Pool.Info.Reserves[indexIn%2] = new(big.Int).Add(s.Pool.Info.Reserves[indexIn%2], params.TokenAmountIn.Amount)
	s.Pool.Info.Reserves[indexOut%2] = new(big.Int).Sub(s.Pool.Info.Reserves[indexOut%2], params.TokenAmountOut.Amount)

	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}

	deltaIn := lo.Ternary(swapInfo.IsWrapIn, params.TokenAmountIn.Amount, bignumber.ZeroBI)
	deltaOut := lo.Ternary(swapInfo.IsUnwrapOut, params.TokenAmountOut.Amount, bignumber.ZeroBI)

	_, _, _ = params.SwapLimit.UpdateLimit(swapInfo.WTokenOut, swapInfo.WTokenIn, deltaOut, deltaIn)
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) interface{} {
	return uniswapv2.PoolMeta{
		Extra: uniswapv2.Extra{
			Fee:          s.fee.Uint64(),
			FeePrecision: s.feePrecision.Uint64(),
		},
		PoolMetaGeneric: uniswapv2.PoolMetaGeneric{
			ApprovalAddress: s.GetApprovalAddress(tokenIn, tokenOut),
		},
	}
}

func (s *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	// If wrap in
	if idx := s.GetTokenIndex(tokenIn); idx >= 0 && idx < 2 {
		return s.Info.Tokens[idx+2]
	}

	return ""
}

func (s *PoolSimulator) getReserves(indexIn, indexOut int) (*uint256.Int, *uint256.Int, error) {
	reserveInIndex, reserveOutIndex := indexIn%2, indexOut%2

	if reserveInIndex >= len(s.Pool.Info.Reserves) || reserveOutIndex >= len(s.Pool.Info.Reserves) {
		return nil, nil, ErrReserveIndexOutOfBounds
	}

	reserveIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[reserveInIndex])
	if overflow {
		return nil, nil, uniswapv2.ErrInvalidReserve
	}

	reserveOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[reserveOutIndex])
	if overflow {
		return nil, nil, uniswapv2.ErrInvalidReserve
	}

	if reserveIn.Cmp(number.Zero) <= 0 || reserveOut.Cmp(number.Zero) <= 0 {
		return nil, nil, uniswapv2.ErrInsufficientLiquidity
	}

	return reserveIn, reserveOut, nil
}

func (s *PoolSimulator) getWrappedTokens(indexIn, indexOut int) (wTokenIn, wTokenOut string, err error) {
	wTokenInIndex := indexIn%2 + 2
	wTokenOutIndex := indexOut%2 + 2

	if wTokenInIndex >= len(s.Pool.Info.Tokens) || wTokenOutIndex >= len(s.Pool.Info.Tokens) {
		return "", "", ErrTokenIndexOutOfBounds
	}

	return s.Pool.Info.Tokens[wTokenInIndex], s.Pool.Info.Tokens[wTokenOutIndex], nil
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
			if recoveredError, ok := r.(error); ok {
				err = recoveredError
			} else {
				err = fmt.Errorf("unexpected panic: %v", r)
			}
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

func (s *PoolSimulator) CalculateLimit() map[string]*big.Int {
	tokens := s.GetTokens()

	limits := make(map[string]*big.Int, len(tokens))

	if len(tokens) == 4 {
		limits[tokens[2]] = s.originalReserves.Reserve0
		limits[tokens[3]] = s.originalReserves.Reserve1
	}

	return limits
}

func (s *PoolSimulator) CanSwapTo(address string) []string {
	result := make([]string, 0, len(s.Info.Tokens))
	var tokenIndex = s.GetTokenIndex(address)
	if tokenIndex < 0 {
		return result
	}

	for i := 0; i < len(s.Info.Tokens); i += 1 {
		// ringswap: indexIn%2 == indexOut%2 is not allowed
		if i != tokenIndex && i%2 != tokenIndex%2 {
			result = append(result, s.Info.Tokens[i])
		}
	}

	return result
}

func (s *PoolSimulator) CanSwapFrom(address string) []string {
	return s.CanSwapTo(address)
}
