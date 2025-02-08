package nomiswapstable

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/nomiswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	swapFee                   uint32
	tokenPrecisionMultipliers []*uint256.Int
	A                         *uint256.Int
	gas                       nomiswap.Gas
}

var _ = pool.RegisterFactory0(nomiswap.DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra nomiswap.ExtraStablePool
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var tokens = make([]string, 2)
	tokens[0] = entityPool.Tokens[0].Address
	tokens[1] = entityPool.Tokens[1].Address

	var reserves = make([]*big.Int, 2)
	reserves[0] = bignumber.NewBig10(entityPool.Reserves[0])
	reserves[1] = bignumber.NewBig10(entityPool.Reserves[1])

	var info = pool.PoolInfo{
		Address:     strings.ToLower(entityPool.Address),
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      tokens,
		Reserves:    reserves,
		BlockNumber: entityPool.BlockNumber,
	}

	return &PoolSimulator{
		Pool:                      pool.Pool{Info: info},
		swapFee:                   extra.SwapFee,
		tokenPrecisionMultipliers: []*uint256.Int{extra.Token0PrecisionMultiplier, extra.Token1PrecisionMultiplier},
		A:                         extra.A,
		gas:                       defaultGas,
	}, nil
}
func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	var tokenInIndex = p.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = p.GetTokenIndex(tokenOut)
	amountOut := getAmountOut(
		uint256.MustFromBig(tokenAmountIn.Amount),
		uint256.MustFromBig(p.Info.Reserves[tokenInIndex]),
		uint256.MustFromBig(p.Info.Reserves[tokenOutIndex]),
		uint256.NewInt(uint64(p.swapFee)),
		p.tokenPrecisionMultipliers[tokenInIndex],
		p.tokenPrecisionMultipliers[tokenOutIndex],
		p.A,
	)

	if amountOut.Cmp(Zero) <= 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("amountOut is %d", amountOut.Uint64())
	}

	if amountOut.Cmp(uint256.MustFromBig(p.Info.Reserves[tokenOutIndex])) > 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("amountOut is %d bigger then reserve %d", amountOut.Uint64(), p.Info.Reserves[tokenOutIndex])
	}

	tokenAmountOut := &pool.TokenAmount{
		Token:  tokenOut,
		Amount: amountOut.ToBig(),
	}

	fee := &pool.TokenAmount{
		Token:  tokenAmountOut.Token,
		Amount: nil,
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: tokenAmountOut,
		Fee:            fee,
		// Gas:            p.gas.Swap,
	}, nil

}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	var input, output = params.TokenAmountIn, params.TokenAmountOut
	var tokenInIndex = p.GetTokenIndex(input.Token)
	var tokenOutIndex = p.GetTokenIndex(output.Token)

	var inputAmount, _ = calAmountAfterFee(uint256.MustFromBig(input.Amount), uint256.NewInt(uint64(p.swapFee)))
	var outputAmount = output.Amount
	p.Info.Reserves[tokenInIndex] = new(big.Int).Add(p.Info.Reserves[tokenInIndex], inputAmount.ToBig())
	p.Info.Reserves[tokenOutIndex] = new(big.Int).Sub(p.Info.Reserves[tokenOutIndex], outputAmount)
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}
