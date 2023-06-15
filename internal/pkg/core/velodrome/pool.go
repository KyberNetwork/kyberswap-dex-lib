package velodrome

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
	Decimals []*big.Int
	stable   bool
	gas      Gas
}

func NewPool(entityPool entity.Pool) (*Pool, error) {
	var swapFeeFl = new(big.Float).Mul(big.NewFloat(entityPool.SwapFee), constant.BoneFloat)
	var swapFee, _ = swapFeeFl.Int(nil)

	var tokens = make([]string, 2)
	tokens[0] = entityPool.Tokens[0].Address
	tokens[1] = entityPool.Tokens[1].Address

	var reserves = make([]*big.Int, 2)
	reserves[0] = utils.NewBig10(entityPool.Reserves[0])
	reserves[1] = utils.NewBig10(entityPool.Reserves[1])

	var decimals = make([]*big.Int, 2)
	decimals[0] = constant.TenPowInt(entityPool.Tokens[0].Decimals)
	decimals[1] = constant.TenPowInt(entityPool.Tokens[1].Decimals)

	var info = pool.PoolInfo{
		Address:    strings.ToLower(entityPool.Address),
		ReserveUsd: entityPool.ReserveUsd,
		SwapFee:    swapFee,
		Exchange:   entityPool.Exchange,
		Type:       entityPool.Type,
		Tokens:     tokens,
		Reserves:   reserves,
		Checked:    false,
	}

	staticExtra, err := extractStaticExtra(entityPool)
	if err != nil {
		return nil, err
	}

	return &Pool{
		Pool:     pool.Pool{Info: info},
		Decimals: decimals,
		stable:   staticExtra.Stable,
		gas:      DefaultGas,
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
		p.Decimals[tokenInIndex],
		p.Decimals[tokenOutIndex],
		p.Info.SwapFee,
		p.stable,
	)

	if amountOut.Cmp(constant.Zero) <= 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("amountOut is %d", amountOut.Int64())
	}

	if amountOut.Cmp(p.Info.Reserves[tokenOutIndex]) > 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("amountOut is %d bigger than reserve %d", amountOut.Int64(), p.Info.Reserves[tokenOutIndex])
	}

	tokenAmountOut := &pool.TokenAmount{
		Token:  tokenOut,
		Amount: amountOut,
	}

	fee := &pool.TokenAmount{
		Token:  tokenAmountIn.Token,
		Amount: nil,
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: tokenAmountOut,
		Fee:            fee,
		Gas:            p.gas.Swap,
	}, nil
}

func (p *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var inputAmount = calAmountAfterFee(input.Amount, p.Info.SwapFee)
	var outputAmount = output.Amount

	for i := range p.Info.Tokens {
		if p.Info.Tokens[i] == input.Token {
			p.Info.Reserves[i] = new(big.Int).Add(p.Info.Reserves[i], inputAmount)
		}
		if p.Info.Tokens[i] == output.Token {
			p.Info.Reserves[i] = new(big.Int).Sub(p.Info.Reserves[i], outputAmount)
		}
	}
}

func (p *Pool) GetLpToken() string {
	return p.GetAddress()
}

func (p *Pool) GetMidPrice(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	return constant.Zero
}

func (p *Pool) CalcExactQuote(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	var tokenInIndex = p.GetTokenIndex(tokenIn)
	var tokenOutIndex = p.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return constant.Zero
	}

	exactQuote := getExactQuote(
		base,
		p.Info.Reserves[tokenInIndex],
		p.Info.Reserves[tokenOutIndex],
		p.Decimals[tokenInIndex],
		p.Decimals[tokenOutIndex],
		p.stable,
	)

	return exactQuote
}

func (p *Pool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return StaticExtra{
		Stable: p.stable,
	}
}

func extractStaticExtra(pool entity.Pool) (StaticExtra, error) {
	var staticExtra StaticExtra
	err := json.Unmarshal([]byte(pool.StaticExtra), &staticExtra)
	if err != nil {
		return StaticExtra{}, err
	}

	return staticExtra, nil
}
