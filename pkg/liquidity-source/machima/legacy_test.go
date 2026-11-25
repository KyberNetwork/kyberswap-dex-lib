package machima

import (
	"math/big"

	"github.com/holiman/uint256"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// Legacy big.Int CalcAmountOut, CalcAmountIn and grossUpBps, kept for parity tests until the golden data is removed.

// CalcAmountOut is the legacy big.Int entry point. It converts to uint256, quotes with CalcAmtOut, and
// converts back, so the tax rules live only in CalcAmtOut.
func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if param.TokenAmountIn.Amount == nil || param.TokenAmountIn.Amount.Sign() < 0 {
		return nil, ErrZeroAmount
	}
	var amountIn uint256.Int
	if overflow := amountIn.SetFromBig(param.TokenAmountIn.Amount); overflow {
		return nil, ErrOverflow
	}

	res, err := p.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: param.TokenAmountIn.Token, Amount: amountIn},
		TokenOut:      param.TokenOut,
	})
	if err != nil {
		return nil, err
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token: res.TokenAmountOut.Token, Amount: res.TokenAmountOut.Amount.ToBig(),
		},
		RemainingTokenAmountIn: &pool.TokenAmount{
			Token: res.RemainingTokenAmountIn.Token, Amount: res.RemainingTokenAmountIn.Amount.ToBig(),
		},
		Gas: res.Gas,
		// SwapInfo is the uint256 form that UpdateState reads.
		SwapInfo: res.SwapInfo,
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	if p.isInAntiSniperWindow() {
		return nil, ErrAntiSniperActive
	}

	tokenIn, tokenOut := param.TokenIn, param.TokenAmountOut.Token
	isBuy, err := p.classifyPair(tokenIn, tokenOut)
	if err != nil {
		return nil, err
	}

	// Gross the requested output back up to what the pool has to produce before sell tax.
	poolAmountOut := param.TokenAmountOut.Amount
	if !isBuy {
		poolAmountOut = legacyGrossUpBps(poolAmountOut, p.taxBps(false))
	}
	if poolAmountOut.Sign() <= 0 {
		return nil, ErrZeroAmount
	}

	//nolint:staticcheck // QF1008: explicit embedded selector is intentional, see CalcAmountOut
	res, err := p.PoolSimulator.CalcAmountInWithPriceLimit(pool.CalcAmountInParams{
		TokenIn:        tokenIn,
		TokenAmountOut: pool.TokenAmount{Token: tokenOut, Amount: poolAmountOut},
	}, p.sqrtPriceLimit(tokenIn, isBuy))
	if err != nil {
		return nil, err
	}

	v3SwapInfo, ok := res.SwapInfo.(uniswapv3.SwapInfo)
	if !ok {
		return nil, ErrUnexpectedSwapInfo
	}

	// Gross the pool-side input up to what the user has to send before buy tax.
	poolAmountIn := res.TokenAmountIn.Amount
	amountIn := poolAmountIn
	if isBuy {
		amountIn = legacyGrossUpBps(poolAmountIn, p.taxBps(true))
	}

	var poolInU, poolOutU uint256.Int
	if poolInU.SetFromBig(poolAmountIn) || poolOutU.SetFromBig(poolAmountOut) {
		return nil, ErrOverflow
	}
	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: amountIn},
		Fee:           &pool.TokenAmount{Token: tokenIn, Amount: new(big.Int).Sub(amountIn, poolAmountIn)},
		Gas:           res.Gas,
		SwapInfo: SwapInfo{
			V3:            v3SwapInfo,
			PoolAmountIn:  poolInU,
			PoolAmountOut: poolOutU,
		},
	}, nil
}

var bigBpsDenominator = big.NewInt(bpsDenominator)

// legacyGrossUpBps is the big.Int grossUpBps the legacy CalcAmountIn used.
func legacyGrossUpBps(amount *big.Int, bps uint16) *big.Int {
	if bps == 0 {
		return amount
	}
	gross := new(big.Int).Mul(amount, bigBpsDenominator)
	den := big.NewInt(int64(bpsDenominator - bps))
	gross.Add(gross, new(big.Int).Sub(den, bignumber.One))
	return gross.Div(gross, den)
}
