package balancerv1

import (
	"github.com/holiman/uint256"
)

var BMath *bMath

type bMath struct{}

func init() {
	BMath = &bMath{}
}

// https://github.com/balancer/balancer-core/blob/f4ed5d65362a8d6cec21662fb6eae233b0babc1f/contracts/BMath.sol#L28
func (l *bMath) CalcSpotPrice(
	tokenBalanceIn *uint256.Int,
	tokenWeightIn *uint256.Int,
	tokenBalanceOut *uint256.Int,
	tokenWeightOut *uint256.Int,
	swapFee *uint256.Int,
) (*uint256.Int, error) {
	numer, err := BNum.BDiv(tokenBalanceIn, tokenWeightIn)
	if err != nil {
		return nil, err
	}

	denom, err := BNum.BDiv(tokenBalanceOut, tokenWeightOut)
	if err != nil {
		return nil, err
	}

	ratio, err := BNum.BDiv(numer, denom)
	if err != nil {
		return nil, err
	}

	bSubBONEAndSwapFee, err := BNum.BSub(BConst.BONE, swapFee)
	if err != nil {
		return nil, err
	}

	scale, err := BNum.BDiv(BConst.BONE, bSubBONEAndSwapFee)
	if err != nil {
		return nil, err
	}

	return BNum.BMul(ratio, scale)
}

// https://github.com/balancer/balancer-core/blob/f4ed5d65362a8d6cec21662fb6eae233b0babc1f/contracts/BMath.sol#L55
func (l *bMath) CalcOutGivenIn(
	tokenBalanceIn *uint256.Int,
	tokenWeightIn *uint256.Int,
	tokenBalanceOut *uint256.Int,
	tokenWeightOut *uint256.Int,
	tokenAmountIn *uint256.Int,
	swapFee *uint256.Int,
) (*uint256.Int, error) {
	weightRatio, err := BNum.BDiv(tokenWeightIn, tokenWeightOut)
	if err != nil {
		return nil, err
	}

	adjustedIn, err := BNum.BSub(BConst.BONE, swapFee)
	if err != nil {
		return nil, err
	}

	adjustedIn, err = BNum.BMul(tokenAmountIn, adjustedIn)
	if err != nil {
		return nil, err
	}

	bAddTokenBalanceInAndAdjustedIn, err := BNum.BAdd(tokenBalanceIn, adjustedIn)
	if err != nil {
		return nil, err
	}

	y, err := BNum.BDiv(tokenBalanceIn, bAddTokenBalanceInAndAdjustedIn)
	if err != nil {
		return nil, err
	}

	foo, err := BNum.BPow(y, weightRatio)
	if err != nil {
		return nil, err
	}

	bar, err := BNum.BSub(BConst.BONE, foo)
	if err != nil {
		return nil, err
	}

	return BNum.BMul(tokenBalanceOut, bar)
}
