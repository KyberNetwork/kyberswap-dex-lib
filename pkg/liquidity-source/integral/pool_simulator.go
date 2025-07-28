package integral

import (
	"fmt"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	Extra
	XDecimals uint8
	YDecimals uint8
	gas       Gas
}

var _ = pool.RegisterFactory0(DexTypeIntegral, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Extra: %v", err)
	}

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)

	for i := 0; i < numTokens; i++ {
		tokenAddr := entityPool.Tokens[i].Address
		tokens[i] = tokenAddr
		reserves[i] = bignumber.NewBig(entityPool.Reserves[i])
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Reserves: reserves,
			},
		},
		Extra:     extra,
		XDecimals: entityPool.Tokens[0].Decimals,
		YDecimals: entityPool.Tokens[1].Decimals,
		gas:       defaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokens := p.GetTokens()
	if len(tokens) < 2 {
		return nil, ErrTokenNotFound
	}

	tokenIn := param.TokenAmountIn.Token
	tokenOut := param.TokenOut

	amountIn := number.SetFromBig(param.TokenAmountIn.Amount)
	reserve0 := number.SetFromBig(p.GetReserves()[0])
	reserve1 := number.SetFromBig(p.GetReserves()[1])

	_amountIn, amountOut, fee, err := p.swapExactIn(tokenIn, tokenOut, amountIn)
	if err != nil {
		return nil, err
	}

	var newReserve0, newReserve1 *uint256.Int

	switch tokenIn {
	case tokens[0]:
		if reserve1.Lt(amountOut) {
			return nil, fmt.Errorf("insufficient liquidity for tokenOut")
		}
		newReserve1 = number.SafeSub(reserve1, amountOut)
		newReserve0 = number.SafeAdd(reserve0, _amountIn)
	case tokens[1]:
		if reserve0.Lt(amountOut) {
			return nil, fmt.Errorf("insufficient liquidity for tokenOut")
		}
		newReserve0 = number.SafeSub(reserve0, amountOut)
		newReserve1 = number.SafeAdd(reserve1, _amountIn)
	default:
		return nil, ErrInvalidTokenIn
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: fee.ToBig(),
		},
		Gas: p.gas.Swap,
		SwapInfo: SwapInfo{
			RelayerAddress: p.RelayerAddress,
			NewReserve0:    newReserve0.ToBig(),
			NewReserve1:    newReserve1.ToBig(),
		},
	}, nil
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	return MetaInfo{ApprovalAddress: t.GetApprovalAddress(tokenIn, tokenOut)}
}

func (t *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	return lo.Ternary(valueobject.IsNative(tokenIn), "", t.GetAddress())
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if params.SwapInfo != nil {
		if s, ok := params.SwapInfo.(SwapInfo); ok {
			newToken0LimitMax := new(big.Int).Div(
				new(big.Int).Mul(
					t.Extra.Token0LimitMax.ToBig(),
					s.NewReserve0,
				),
				t.Info.Reserves[0],
			)

			newToken1LimitMax := new(big.Int).Div(
				new(big.Int).Mul(
					t.Extra.Token1LimitMax.ToBig(),
					s.NewReserve1,
				),
				t.Info.Reserves[1],
			)

			t.Info.Reserves[0] = s.NewReserve0
			t.Info.Reserves[1] = s.NewReserve1

			t.Extra.Token0LimitMax = uint256.MustFromBig(newToken0LimitMax)
			t.Extra.Token1LimitMax = uint256.MustFromBig(newToken1LimitMax)
		}
	}
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/TwapRelayer.sol#L275
func (p *PoolSimulator) swapExactIn(tokenIn, tokenOut string, amountIn *uint256.Int) (*uint256.Int, *uint256.Int, *uint256.Int, error) {
	if !p.Extra.IsEnabled {
		return nil, nil, nil, ErrTR05
	}

	tokens := p.GetTokens()
	fee := number.SafeDiv(number.SafeMul(amountIn, p.Extra.SwapFee), precision)

	inverted := tokens[1] == tokenIn

	amountOut := p.calculateAmountOut(inverted, number.SafeSub(amountIn, fee))

	if err := p.checkLimits(tokenOut, amountOut); err != nil {
		return nil, nil, nil, err
	}

	return amountIn, amountOut, fee, nil
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/TwapRelayer.sol#L520
func (p *PoolSimulator) checkLimits(token string, amount *uint256.Int) error {
	if token == p.GetTokens()[0] {
		if amount.Lt(p.Extra.Token0LimitMin) {
			return ErrTR03
		}

		if amount.Gt(p.Extra.Token0LimitMax) {
			return ErrTR3A
		}
	} else if token == p.GetTokens()[1] {
		if amount.Lt(p.Extra.Token1LimitMin) {
			return ErrTR03
		}

		if amount.Gt(p.Extra.Token1LimitMax) {
			return ErrTR3A
		}
	}

	return nil
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/TwapRelayer.sol#L324
func (p *PoolSimulator) calculateAmountOut(inverted bool, amountIn *uint256.Int) *uint256.Int {
	decimalsConverter := getDecimalsConverter(p.XDecimals, p.YDecimals, inverted)

	if inverted {
		return number.SafeDiv(number.SafeMul(amountIn, p.InvertedPrice), decimalsConverter)
	}

	return number.SafeDiv(number.SafeMul(amountIn, p.Price), decimalsConverter)
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/TwapRelayer.sol#L334
func getDecimalsConverter(xDecimals, yDecimals uint8, inverted bool) *uint256.Int {
	var exponent uint8
	if inverted {
		exponent = 18 + (yDecimals - xDecimals)
	} else {
		exponent = 18 + (xDecimals - yDecimals)
	}

	return new(uint256.Int).Set(u256.TenPow(exponent))
}
