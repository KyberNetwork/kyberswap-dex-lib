package syncswapstable

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

type Pool struct {
	pool.Pool
	swapFees                  []*big.Int
	tokenPrecisionMultipliers []*big.Int
	gas                       Gas
}

func NewPool(entityPool entity.Pool) (*Pool, error) {
	var extra ExtraStablePool
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var tokens = make([]string, 2)
	tokens[0] = entityPool.Tokens[0].Address
	tokens[1] = entityPool.Tokens[1].Address

	var reserves = make([]*big.Int, 2)
	reserves[0] = utils.NewBig10(entityPool.Reserves[0])
	reserves[1] = utils.NewBig10(entityPool.Reserves[1])

	var swapFees = make([]*big.Int, 2)
	swapFees[0] = extra.SwapFee0To1
	swapFees[1] = extra.SwapFee1To0

	var tokenPrecisionMultipliers = make([]*big.Int, 2)
	tokenPrecisionMultipliers[0] = extra.Token0PrecisionMultiplier
	tokenPrecisionMultipliers[1] = extra.Token1PrecisionMultiplier

	var info = pool.PoolInfo{
		Address:  strings.ToLower(entityPool.Address),
		Exchange: entityPool.Exchange,
		Type:     entityPool.Type,
		Tokens:   tokens,
		Reserves: reserves,
	}

	return &Pool{
		Pool:                      pool.Pool{Info: info},
		swapFees:                  swapFees,
		tokenPrecisionMultipliers: tokenPrecisionMultipliers,
		gas:                       DefaultGas,
	}, nil
}

func (p *Pool) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	var tokenInIndex = p.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = p.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	amountOut := getAmountOut(
		tokenAmountIn.Amount,
		p.Info.Reserves[tokenInIndex],
		p.Info.Reserves[tokenOutIndex],
		p.swapFees[tokenInIndex],
		p.tokenPrecisionMultipliers[tokenInIndex],
		p.tokenPrecisionMultipliers[tokenOutIndex],
	)

	if amountOut.Cmp(constant.Zero) <= 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("amountOut is %d", amountOut.Int64())
	}

	if amountOut.Cmp(p.Info.Reserves[tokenOutIndex]) > 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("amountOut is %d bigger then reserve %d", amountOut.Int64(), p.Info.Reserves[tokenOutIndex])
	}

	tokenAmountOut := &pool.TokenAmount{
		Token:  tokenOut,
		Amount: amountOut,
	}

	fee := &pool.TokenAmount{
		Token:  tokenAmountOut.Token,
		Amount: nil,
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: tokenAmountOut,
		Fee:            fee,
		Gas:            p.gas.Swap,
	}, nil
}

func (p *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	var input, output = params.TokenAmountIn, params.TokenAmountOut
	var tokenInIndex = p.GetTokenIndex(input.Token)
	var tokenOutIndex = p.GetTokenIndex(output.Token)

	var inputAmount, _ = calAmountAfterFee(input.Amount, p.swapFees[tokenInIndex])
	var outputAmount = output.Amount

	p.Info.Reserves[tokenInIndex] = new(big.Int).Add(p.Info.Reserves[tokenInIndex], inputAmount)
	p.Info.Reserves[tokenOutIndex] = new(big.Int).Sub(p.Info.Reserves[tokenOutIndex], outputAmount)
}

func (p *Pool) CalcExactQuote(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	var tokenInIndex = p.GetTokenIndex(tokenIn)
	var tokenOutIndex = p.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return constant.Zero
	}

	exactQuote := getAmountOut(
		base,
		p.Info.Reserves[tokenInIndex],
		p.Info.Reserves[tokenOutIndex],
		p.swapFees[tokenInIndex],
		p.tokenPrecisionMultipliers[tokenInIndex],
		p.tokenPrecisionMultipliers[tokenOutIndex],
	)

	return exactQuote
}

func (p *Pool) CanSwapTo(address string) []string {
	var ret = make([]string, 0)
	var tokenInIndex = p.GetTokenIndex(address)
	if tokenInIndex < 0 {
		return ret
	}
	for i := 0; i < len(p.Info.Tokens); i++ {
		if i != tokenInIndex {
			ret = append(ret, p.Info.Tokens[i])
		}
	}

	return ret
}

func (p *Pool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}

func (p *Pool) GetMidPrice(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	return constant.Zero
}

func (p *Pool) GetLpToken() string {
	return p.GetAddress()
}
