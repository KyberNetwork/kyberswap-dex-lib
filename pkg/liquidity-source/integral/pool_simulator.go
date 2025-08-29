package integral

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/logger"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"

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

	relayerAddress string

	isEnabled     bool
	price         *uint256.Int
	invertedPrice *uint256.Int
	swapFee       *uint256.Int

	token0LimitMin *uint256.Int
	token1LimitMin *uint256.Int

	token0LimitMaxMultiplier *uint256.Int
	token1LimitMaxMultiplier *uint256.Int

	xDecimals uint8
	yDecimals uint8

	gas Gas
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

	// apply safety buffer to limit max multiplier (85% for now)
	extra.Token0LimitMaxMultiplier.
		Mul(extra.Token0LimitMaxMultiplier, safetyBufferPercent).
		Div(extra.Token0LimitMaxMultiplier, u256.U100)

	extra.Token1LimitMaxMultiplier.
		Mul(extra.Token1LimitMaxMultiplier, safetyBufferPercent).
		Div(extra.Token1LimitMaxMultiplier, u256.U100)

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
		isEnabled:                extra.IsEnabled,
		relayerAddress:           extra.RelayerAddress,
		price:                    extra.Price,
		swapFee:                  extra.SwapFee,
		invertedPrice:            extra.InvertedPrice,
		token0LimitMin:           extra.Token0LimitMin,
		token1LimitMin:           extra.Token1LimitMin,
		token0LimitMaxMultiplier: extra.Token0LimitMaxMultiplier,
		token1LimitMaxMultiplier: extra.Token1LimitMaxMultiplier,
		xDecimals:                entityPool.Tokens[0].Decimals,
		yDecimals:                entityPool.Tokens[1].Decimals,
		gas:                      defaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if !p.isEnabled {
		return nil, ErrTR05
	}

	if params.Limit == nil {
		return nil, ErrNoSwapLimit
	}

	tokens := p.GetTokens()
	if len(tokens) < 2 {
		return nil, ErrTokenNotFound
	}

	tokenIn := params.TokenAmountIn.Token
	tokenOut := params.TokenOut

	amountIn := number.SetFromBig(params.TokenAmountIn.Amount)

	maxAmountOut, _ := new(uint256.Int).MulDivOverflow(
		number.SetFromBig(params.Limit.GetLimit(tokenOut)),
		lo.Ternary(tokenOut == tokens[0], p.token0LimitMaxMultiplier, p.token1LimitMaxMultiplier),
		precision,
	)

	amountOut, fee, err := p.swapExactIn(tokenIn, tokenOut, amountIn, maxAmountOut)
	if err != nil {
		return nil, err
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
			RelayerAddress: p.relayerAddress,
		},
	}, nil
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	return MetaInfo{ApprovalAddress: p.GetApprovalAddress(tokenIn, tokenOut)}
}

func (p *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	return lo.Ternary(valueobject.IsNative(tokenIn), "", p.GetAddress())
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenAmountOut.Token
	_, _, err := params.SwapLimit.UpdateLimit(tokenOut, tokenIn, params.TokenAmountOut.Amount, params.TokenAmountIn.Amount)
	if err != nil {
		logger.Errorf("unable to update integral limit, error: %v", err)
	}
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/TwapRelayer.sol#L275
func (p *PoolSimulator) swapExactIn(tokenIn, tokenOut string, amountIn, maxAmountOut *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	tokens := p.GetTokens()
	fee := number.SafeDiv(number.SafeMul(amountIn, p.swapFee), precision)

	inverted := tokens[1] == tokenIn

	amountOut := p.calculateAmountOut(inverted, number.SafeSub(amountIn, fee))

	if err := p.checkLimits(tokenOut, amountOut, maxAmountOut); err != nil {
		return nil, nil, err
	}

	return amountOut, fee, nil
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/TwapRelayer.sol#L520
func (p *PoolSimulator) checkLimits(token string, amount, maxAmount *uint256.Int) error {
	if token == p.GetTokens()[0] {
		if amount.Lt(p.token0LimitMin) {
			return ErrTR03
		}

		if amount.Gt(maxAmount) {
			return ErrTR3A
		}
	} else if token == p.GetTokens()[1] {
		if amount.Lt(p.token1LimitMin) {
			return ErrTR03
		}

		if amount.Gt(maxAmount) {
			return ErrTR3A
		}
	}

	return nil
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/TwapRelayer.sol#L324
func (p *PoolSimulator) calculateAmountOut(inverted bool, amountIn *uint256.Int) *uint256.Int {
	decimalsConverter := getDecimalsConverter(p.xDecimals, p.yDecimals, inverted)

	if inverted {
		return number.SafeDiv(number.SafeMul(amountIn, p.invertedPrice), decimalsConverter)
	}

	return number.SafeDiv(number.SafeMul(amountIn, p.price), decimalsConverter)
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

func (p *PoolSimulator) CalculateLimit() map[string]*big.Int {
	tokens, reserves := p.GetTokens(), p.GetReserves()
	limits := make(map[string]*big.Int, len(tokens))

	for i, token := range tokens {
		limits[token] = new(big.Int).Set(reserves[i])
	}

	return limits
}
