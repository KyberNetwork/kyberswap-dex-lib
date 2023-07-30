package syncswapstable

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	vaultAddress              string
	swapFees                  []*big.Int
	tokenPrecisionMultipliers []*big.Int
	gas                       syncswap.Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra syncswap.ExtraStablePool
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var tokens = make([]string, 2)
	tokens[0] = entityPool.Tokens[0].Address
	tokens[1] = entityPool.Tokens[1].Address

	var reserves = make([]*big.Int, 2)
	reserves[0] = bignumber.NewBig10(entityPool.Reserves[0])
	reserves[1] = bignumber.NewBig10(entityPool.Reserves[1])

	var vaultAddress = extra.VaultAddress

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

	return &PoolSimulator{
		Pool:                      pool.Pool{Info: info},
		vaultAddress:              vaultAddress,
		swapFees:                  swapFees,
		tokenPrecisionMultipliers: tokenPrecisionMultipliers,
		gas:                       DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(
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

	if amountOut.Cmp(bignumber.ZeroBI) <= 0 {
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

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	var input, output = params.TokenAmountIn, params.TokenAmountOut
	var tokenInIndex = p.GetTokenIndex(input.Token)
	var tokenOutIndex = p.GetTokenIndex(output.Token)

	var inputAmount, _ = calAmountAfterFee(input.Amount, p.swapFees[tokenInIndex])
	var outputAmount = output.Amount

	p.Info.Reserves[tokenInIndex] = new(big.Int).Add(p.Info.Reserves[tokenInIndex], inputAmount)
	p.Info.Reserves[tokenOutIndex] = new(big.Int).Sub(p.Info.Reserves[tokenOutIndex], outputAmount)
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return syncswap.Meta{
		VaultAddress: p.vaultAddress,
	}
}
