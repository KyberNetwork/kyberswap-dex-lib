package integral

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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
var _ = pool.RegisterUseSwapLimit(DexTypeIntegral)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Extra: %v", err)
	}

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)

	for i := range numTokens {
		tokenAddr := entityPool.Tokens[i].Address
		tokens[i] = tokenAddr
		reserves[i] = bignumber.NewBig(entityPool.Reserves[i])
	}

	// apply safety buffer to limit max multiplier (85% for now)
	if extra.Token0LimitMaxMultiplier != nil && extra.Token0LimitMaxMultiplier.Sign() > 0 {
		extra.Token0LimitMaxMultiplier.
			Mul(extra.Token0LimitMaxMultiplier, safetyBufferPercent).
			Div(extra.Token0LimitMaxMultiplier, u256.U100)
	}

	if extra.Token1LimitMaxMultiplier != nil && extra.Token1LimitMaxMultiplier.Sign() > 0 {
		extra.Token1LimitMaxMultiplier.
			Mul(extra.Token1LimitMaxMultiplier, safetyBufferPercent).
			Div(extra.Token1LimitMaxMultiplier, u256.U100)
	}

	return &PoolSimulator{
		Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
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
	tokenOut := params.TokenOut
	maxAmountOut := p.maxAmountOut(tokenOut, number.SetFromBig(params.Limit.GetLimit(tokenOut)))

	amountOut, fee, err := p.quote(params.TokenAmountIn.Token, tokenOut, number.SetFromBig(params.TokenAmountIn.Amount), maxAmountOut)
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

// CalcAmtOut is CalcAmountOut with uint256 amounts. It shares quote with the legacy method.
func (p *PoolSimulator) CalcAmtOut(params pool.CalcAmtOutParams) (*pool.CalcAmtOutResult, error) {
	if !p.isEnabled {
		return nil, ErrTR05
	}
	if params.Limit == nil {
		return nil, ErrNoSwapLimit
	}
	tokenOut := params.TokenOut
	maxAmountOut := p.maxAmountOut(tokenOut, params.Limit.GetLimit(tokenOut))

	amountOut, fee, err := p.quote(params.TokenAmountIn.Token, tokenOut, &params.TokenAmountIn.Amount, maxAmountOut)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmtOutResult{
		TokenAmountOut: &pool.TokenAmt{Token: tokenOut, Amount: *amountOut},
		Fee:            &pool.TokenAmt{Token: tokenOut, Amount: *fee},
		Gas:            p.gas.Swap,
		SwapInfo:       SwapInfo{RelayerAddress: p.relayerAddress},
	}, nil
}

// maxAmountOut scales the output inventory by the token's limit multiplier. A nil inventory gives zero.
func (p *PoolSimulator) maxAmountOut(tokenOut string, inventory *uint256.Int) *uint256.Int {
	if inventory == nil {
		inventory = new(uint256.Int)
	}
	maxAmountOut, _ := new(uint256.Int).MulDivOverflow(
		inventory,
		lo.Ternary(tokenOut == p.GetTokens()[0], p.token0LimitMaxMultiplier, p.token1LimitMaxMultiplier),
		precision,
	)
	return maxAmountOut
}

// quote runs the swap without the inventory check. The caller passes maxAmountOut.
func (p *PoolSimulator) quote(tokenIn, tokenOut string, amountIn, maxAmountOut *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	if len(p.GetTokens()) < 2 {
		return nil, nil, ErrTokenNotFound
	}
	return p.swapExactIn(tokenIn, tokenOut, amountIn, maxAmountOut)
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	return MetaInfo{ApprovalAddress: p.GetApprovalAddress(tokenIn, tokenOut), BlockNumber: p.Info.BlockNumber}
}

func (p *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	return lo.Ternary(valueobject.IsNative(tokenIn), "", p.GetAddress())
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenAmountOut.Token
	_, _, err := params.SwapLimitBig.UpdateLimit(tokenOut, tokenIn, params.TokenAmountOut.Amount, params.TokenAmountIn.Amount)
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

func (p *PoolSimulator) CloneState() pool.IPoolSimulator { return p }
