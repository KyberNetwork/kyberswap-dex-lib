package syncswapv2stable

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/syncswap/syncswapstable"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	vaultAddress              string
	swapFees                  []*uint256.Int
	tokenPrecisionMultipliers []*uint256.Int
	A                         *uint256.Int
	gas                       syncswap.Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra ExtraStablePool
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

	var swapFees = make([]*uint256.Int, 2)
	swapFees[0] = extra.SwapFee0To1
	swapFees[1] = extra.SwapFee1To0

	var tokenPrecisionMultipliers = make([]*uint256.Int, 2)
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
		A:                         extra.A,
		gas:                       syncswapstable.DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	var tokenInIndex = p.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = p.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	amountOut := getAmountOut(
		uint256.MustFromBig(tokenAmountIn.Amount),
		uint256.MustFromBig(p.Info.Reserves[tokenInIndex]),
		uint256.MustFromBig(p.Info.Reserves[tokenOutIndex]),
		p.swapFees[tokenInIndex],
		p.tokenPrecisionMultipliers[tokenInIndex],
		p.tokenPrecisionMultipliers[tokenOutIndex],
		p.A,
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

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenAmountOut := param.TokenAmountOut
	tokenIn := param.TokenIn
	var tokenInIndex = p.GetTokenIndex(tokenIn)
	var tokenOutIndex = p.GetTokenIndex(tokenAmountOut.Token)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return &pool.CalcAmountInResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	if tokenAmountOut.Amount.Cmp(p.Info.Reserves[tokenOutIndex]) > 0 {
		return &pool.CalcAmountInResult{}, fmt.Errorf("expected amountOut is %v bigger than reserve %v", tokenAmountOut.Amount.String(), p.Info.Reserves[tokenOutIndex])
	}

	amountIn := _getAmountIn(
		p.swapFees[tokenInIndex],
		uint256.MustFromBig(tokenAmountOut.Amount),
		uint256.MustFromBig(p.Info.Reserves[tokenInIndex]),
		uint256.MustFromBig(p.Info.Reserves[tokenOutIndex]),
		p.tokenPrecisionMultipliers[tokenInIndex],
		p.tokenPrecisionMultipliers[tokenOutIndex],
		p.A,
	)

	if amountIn.Cmp(integer.Zero()) <= 0 {
		return &pool.CalcAmountInResult{}, fmt.Errorf("amountIn is %v", amountIn.String())
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: nil,
		},
		Gas: p.gas.Swap,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	var input, output = params.TokenAmountIn, params.TokenAmountOut
	var tokenInIndex = p.GetTokenIndex(input.Token)
	var tokenOutIndex = p.GetTokenIndex(output.Token)

	var inputAmount, _ = calAmountAfterFee(uint256.MustFromBig(input.Amount), p.swapFees[tokenInIndex])
	var outputAmount = output.Amount

	p.Info.Reserves[tokenInIndex] = new(big.Int).Add(p.Info.Reserves[tokenInIndex], inputAmount.ToBig())
	p.Info.Reserves[tokenOutIndex] = new(big.Int).Sub(p.Info.Reserves[tokenOutIndex], outputAmount)
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return syncswap.Meta{
		VaultAddress: common.Address{}.Hex(),
	}
}
