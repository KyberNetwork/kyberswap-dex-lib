package stablemetang

import (
	"fmt"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	stableng "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/stable-ng"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"github.com/holiman/uint256"
)

// ICurveBasePool is the interface for curve base pool inside a meta pool
// this is slightly different to the one in SC (return fee)
type ICurveBasePool interface {
	GetInfo() pool.PoolInfo
	GetTokenIndex(address string) int

	GetVirtualPriceU256(vPrice *uint256.Int, D *uint256.Int) error

	CalculateTokenAmountU256(amounts []uint256.Int, deposit bool, mintAmount *uint256.Int, feeAmounts []uint256.Int) error
	CalculateWithdrawOneCoinU256(tokenAmount *uint256.Int, i int, dy *uint256.Int, dyFee *uint256.Int) error

	// similar to RemoveLiquidityOneCoinU256, but pass in result from CalculateWithdrawOneCoinU256
	ApplyRemoveLiquidityOneCoinU256(i int, tokenAmount, dy, dyFee *uint256.Int) error

	// similar to AddLiquidity, but pass in result from CalculateTokenAmountU256
	ApplyAddLiquidity(amounts, feeAmounts []uint256.Int, mintAmount *uint256.Int) error
}

// the normal swap at meta pool is identical to stable-ng,
// so we'll inherit from stableng.PoolSimulator to reuse its methods
type PoolSimulator struct {
	stableng.PoolSimulator
	basePool ICurveBasePool
}

func NewPoolSimulator(entityPool entity.Pool, basePool ICurveBasePool) (*PoolSimulator, error) {
	sim, err := stableng.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, err
	}

	return &PoolSimulator{*sim, basePool}, err
}

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut

	var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)

	// cannot swap between the last meta coin and base pool's coins (because the last coin is LPtoken of base pool)
	if (tokenIndexFrom == t.NumTokens-1 && tokenIndexTo < 0) || (tokenIndexTo == t.NumTokens-1 && tokenIndexFrom < 0) {
		return &pool.CalcAmountOutResult{}, ErrTokenToUnderLyingNotSupported
	}

	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		// this is normal swap at meta pool, reuse the method from stable-ng
		return t.PoolSimulator.CalcAmountOut(param)
	}

	// swap between meta coins and base pool's coins
	var baseInputIndex = t.basePool.GetTokenIndex(tokenAmountIn.Token)
	var baseOutputIndex = t.basePool.GetTokenIndex(tokenOut)
	if baseInputIndex >= 0 && baseOutputIndex >= 0 {
		// if both coins are from base pool, it's better to swap at the base pool directly to save gas
		return &pool.CalcAmountOutResult{}, ErrAllBasePoolTokens
	}

	var maxCoin = t.NumTokens - 1
	if tokenIndexFrom < 0 && baseInputIndex >= 0 {
		tokenIndexFrom = maxCoin + baseInputIndex
	}
	if tokenIndexTo < 0 && baseOutputIndex >= 0 {
		tokenIndexTo = maxCoin + baseOutputIndex
	}
	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		// get_dy_underlying
		var amountIn, amountOut, adminFee uint256.Int
		var addLiquidityInfo BasePoolAddLiquidityInfo
		var metaswapInfo MetaPoolSwapInfo
		var withdrawInfo BasePoolWithdrawInfo
		amountIn.SetFromBig(tokenAmountIn.Amount)
		err := t.GetDyUnderlying(
			tokenIndexFrom,
			tokenIndexTo,
			&amountIn,
			&amountOut,
			&addLiquidityInfo, &metaswapInfo, &withdrawInfo,
		)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		if !amountOut.IsZero() {
			swapInfo := SwapInfo{
				Meta: &metaswapInfo,
			}
			if !addLiquidityInfo.MintAmount.IsZero() {
				swapInfo.AddLiquidity = &addLiquidityInfo
			}
			if !withdrawInfo.TokenAmount.IsZero() {
				swapInfo.Withdraw = &withdrawInfo
			}

			return &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: amountOut.ToBig(),
				},
				Fee: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: adminFee.ToBig(),
				},
				Gas:      DefaultGasUnderlying,
				SwapInfo: swapInfo,
			}, nil
		}
	}
	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenIndexFrom %v or tokenIndexTo %v is not correct", tokenIndexFrom, tokenIndexTo)
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var inputIndex = t.GetTokenIndex(input.Token)
	var outputIndex = t.GetTokenIndex(output.Token)
	if inputIndex >= 0 && outputIndex >= 0 {
		// this is normal swap at meta pool, reuse the method from stable-ng
		t.PoolSimulator.UpdateBalance(params)
		return
	}

	// meta <-> base swap
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warnf("failed to UpdateBalance for curve-stable-meta-ng %v %v pool, wrong swapInfo type", t.Info.Address, t.Info.Exchange)
		return
	}

	// if input coin is from base pool
	addLiq := swapInfo.AddLiquidity
	if addLiq != nil {
		baseNTokens := len(t.basePool.GetInfo().Tokens)
		t.basePool.ApplyAddLiquidity(addLiq.Amounts[:baseNTokens], addLiq.FeeAmounts[:baseNTokens], &addLiq.MintAmount)
	}

	// update balance from the meta swap component
	metaInfo := swapInfo.Meta
	t.Reserves[metaInfo.TokenInIndex].Add(&t.Reserves[metaInfo.TokenInIndex], &metaInfo.AmountIn)
	number.FillBig(&t.Reserves[metaInfo.TokenInIndex], t.Info.Reserves[metaInfo.TokenInIndex])

	t.Reserves[metaInfo.TokenOutIndex].Sub(&t.Reserves[metaInfo.TokenOutIndex], number.Add(&metaInfo.AmountOut, &metaInfo.AdminFee))
	number.FillBig(&t.Reserves[metaInfo.TokenOutIndex], t.Info.Reserves[metaInfo.TokenOutIndex])

	// if output coin is from base pool
	withDraw := swapInfo.Withdraw
	if withDraw != nil {
		t.basePool.ApplyRemoveLiquidityOneCoinU256(
			withDraw.TokenIndex,
			&withDraw.TokenAmount,
			&withDraw.Dy,
			&withDraw.DyFee,
		)
	}

	// the base pool has been updated, so we need to recalculate its vPrice (last component in stored_rates)
	var dummyD uint256.Int
	t.basePool.GetVirtualPriceU256(&t.Extra.RateMultipliers[t.NumTokens-1], &dummyD)
}

func (t *PoolSimulator) CanSwapFrom(address string) []string { return t.CanSwapTo(address) }

func (t *PoolSimulator) CanSwapTo(address string) []string {
	var ret = make([]string, 0)
	var tokenIndex = t.GetTokenIndex(address)
	if tokenIndex < 0 {
		// check from underlying
		tokenIndex = t.basePool.GetTokenIndex(address)
		if tokenIndex >= 0 {
			// base token can be swapped to anything other than the last meta token
			for i := 0; i < t.NumTokens-1; i += 1 {
				ret = append(ret, t.Info.Tokens[i])
			}
			underlyingTokens := t.basePool.GetInfo().Tokens
			for i := 0; i < len(underlyingTokens); i += 1 {
				if i != tokenIndex {
					ret = append(ret, underlyingTokens[i])
				}
			}
		}
		return ret
	}
	// exchange
	for i := 0; i < t.NumTokens; i += 1 {
		if i != tokenIndex {
			ret = append(ret, t.Info.Tokens[i])
		}
	}
	// exchange_underlying
	// last meta token can't be swapped with underlying tokens
	if tokenIndex != t.NumTokens-1 {
		ret = append(ret, t.basePool.GetInfo().Tokens...)
	}
	return ret
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	if fromId >= 0 && toId >= 0 {
		return curve.Meta{
			TokenInIndex:  fromId,
			TokenOutIndex: toId,
			Underlying:    false,
		}
	}
	var baseFromId = t.getUnderlyingIndex(tokenIn)
	var baseToId = t.getUnderlyingIndex(tokenOut)
	return curve.Meta{
		TokenInIndex:  baseFromId,
		TokenOutIndex: baseToId,
		Underlying:    true,
	}
}

func (t *PoolSimulator) GetTokens() []string {
	var result []string
	result = append(result, t.GetInfo().Tokens...)
	result = append(result, t.basePool.GetInfo().Tokens...)
	return result
}

func (t *PoolSimulator) GetBasePoolTokens() []string {
	return t.basePool.GetInfo().Tokens
}

func (t *PoolSimulator) getUnderlyingIndex(token string) int {
	var tokenIndex = t.GetTokenIndex(token)
	if tokenIndex >= 0 {
		return tokenIndex
	}
	var baseIndex = t.basePool.GetTokenIndex(token)
	var maxCoin = t.NumTokens - 1
	if tokenIndex < 0 && baseIndex >= 0 {
		tokenIndex = maxCoin + baseIndex
	}
	return tokenIndex
}
