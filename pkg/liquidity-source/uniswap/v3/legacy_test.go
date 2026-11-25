package uniswapv3

import (
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// Legacy big.Int CalcAmountOut, kept for parity tests until the golden data is removed.

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	return p.CalcAmountOutWithPriceLimit(param, uint256.Int{})
}

// CalcAmountOutWithPriceLimit is CalcAmountOut with an explicit sqrtPriceLimitX96. A zero limit means
// "no limit" and reproduces plain CalcAmountOut. When the limit is hit the swap only partially fills
// and the unspent input is reported in RemainingTokenAmountIn.
func (p *PoolSimulator) CalcAmountOutWithPriceLimit(param pool.CalcAmountOutParams,
	sqrtPriceLimitX96 uint256.Int) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	tokenIn := tokenAmountIn.Token
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, ErrInvalidToken
	} else if p.isBuyRestricted(tokenOut) {
		return nil, ErrBuyRestricted
	} else if p.exceedsMaxTx(tokenIn, tokenAmountIn.Amount) {
		// tokenIn is the launched token here (a sell): it moves user -> pool, the same leg
		// OdysToken._move caps by maxTx.
		return nil, ErrMaxTxExceeded
	}

	var amountIn uint256.Int
	if overflow := amountIn.SetFromBig(tokenAmountIn.Amount); overflow {
		return nil, ErrOverflow
	}
	zeroForOne := tokenInIndex == 0
	result, err := p.V3Pool.GetOutputAmountV2(zeroForOne, amountIn, sqrtPriceLimitX96)
	if err != nil {
		return nil, err
	} else if !p.allowEmptyTicks && result.AmountCalculated.Sign() <= 0 {
		return nil, ErrZeroAmount
	}
	b := &calcAmountOutBacking{remainingIn: result.RemainingAmountIn, sqrtPrice: result.SqrtRatioX96}
	amountOutBI := big256.ToBigBacked(&b.outBI, &b.outWords, &result.AmountCalculated)
	if amountOutBI.Cmp(p.Info.Reserves[tokenOutIndex]) > 0 {
		return nil, ErrInsufficientBalance
	} else if p.exceedsMaxTx(tokenOut, amountOutBI) {
		// tokenOut is the launched token here (a buy): it moves pool -> user, the same leg
		// OdysToken._move caps by maxTx.
		return nil, ErrMaxTxExceeded
	}

	b.remaining = pool.TokenAmount{Token: tokenIn, Amount: bignumber.ZeroBI}
	if !result.RemainingAmountIn.IsZero() {
		b.remaining.Amount = big256.ToBigBacked(&b.remainingBI, &b.remainingWords, &b.remainingIn)
	}
	b.out, b.fee = pool.TokenAmount{Token: tokenOut, Amount: amountOutBI}, pool.TokenAmount{Token: tokenIn}
	b.res = pool.CalcAmountOutResult{
		TokenAmountOut:         &b.out,
		RemainingTokenAmountIn: &b.remaining, Fee: &b.fee,
		Gas: p.Gas.BaseGas + p.Gas.CrossInitTickGas*int64(result.CrossInitTickLoops) +
			p.Gas.CrossEmptyWordGas*int64(result.CrossEmptyWordLoops),
		SwapInfo: SwapInfo{
			RemainingAmountIn:     &b.remainingIn,
			NextStateSqrtRatioX96: &b.sqrtPrice,
			NextStateLiquidity:    result.Liquidity,
			NextStateTickCurrent:  result.CurrentTick,
		},
	}
	return &b.res, nil
}

// calcAmountOutBacking holds everything a CalcAmountOut result points to, so a quote costs one
// allocation for it plus the SwapInfo interface box instead of eight.
type calcAmountOutBacking struct {
	res                      pool.CalcAmountOutResult
	out, remaining, fee      pool.TokenAmount
	remainingIn, sqrtPrice   uint256.Int // SwapInfo points here
	outBI, remainingBI       big.Int
	outWords, remainingWords [4]big.Word
}
