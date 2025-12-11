package valantisstex

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	Extra
	StaticExtra
	reserve0, reserve1 *uint256.Int
}

type LiquidityQuote struct {
	amountOut             *uint256.Int
	amountInFilled        *uint256.Int
	feeAmount             *uint256.Int
	sqrtSpotPriceX96New   *uint256.Int
	effectiveAMMLiquidity *uint256.Int

	IsZeroToOne bool `json:"isZeroToOne"`
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(item *entity.PoolToken, _ int) string { return item.Address }),
			Reserves:    lo.Map(ep.Reserves, func(item string, _ int) *big.Int { return bignum.NewBig(item) }),
			BlockNumber: ep.BlockNumber,
		}},
		Extra:       extra,
		StaticExtra: staticExtra,
		reserve0:    uint256.MustFromDecimal(ep.Reserves[0]),
		reserve1:    uint256.MustFromDecimal(ep.Reserves[1]),
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut
	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	isZeroToOne := indexIn == 0

	feeInBips := s.DefaultSwapFeeBips
	if !valueobject.IsZeroAddress(s.SwapFeeModule) {
		feeInBips = lo.Ternary(isZeroToOne, s.SwapFeeInBipsZtoO, s.SwapFeeInBipsOtoZ).Clone()
		if feeInBips.Gt(maxSwapFeeBips) {
			return nil, ErrSovereignPoolSwapExcessiveSwapFee
		}
	}

	amountIn := uint256.MustFromBig(tokenAmountIn.Amount)
	if amountIn.IsZero() {
		return nil, ErrZeroSwap
	}

	amountInWithoutFee, overflow := new(uint256.Int).MulDivOverflow(
		amountIn, maxSwapFeeBips,
		new(uint256.Int).Add(maxSwapFeeBips, feeInBips),
	)
	if overflow {
		return nil, number.ErrOverflow
	}

	liquidityQuote, err := s.getLiquidityQuote(isZeroToOne, amountInWithoutFee)
	if err != nil {
		return nil, err
	}

	amountOut := liquidityQuote.amountOut

	if amountOut.Gt(lo.Ternary(isZeroToOne, s.reserve1, s.reserve0)) {
		return nil, ErrInsufficientReserve
	}

	feeAmount := new(uint256.Int).Sub(amountIn, amountInWithoutFee)
	feeAmount.Add(feeAmount, liquidityQuote.feeAmount)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: feeAmount.ToBig(),
		},
		Gas:      defaultGas,
		SwapInfo: liquidityQuote,
	}, nil
}

func (s *PoolSimulator) getLiquidityQuote(isZeroToOne bool, amountInMinusFee *uint256.Int) (*LiquidityQuote, error) {
	newLiquidity := s.calculateAMMLiquidity()
	effectiveAMMLiquidity := s.EffectiveAMMLiquidity.Clone()

	if newLiquidity.Lt(s.EffectiveAMMLiquidity) {
		effectiveAMMLiquidity.Set(newLiquidity)
	}

	sqrtSpotPriceX96New, amountInFilled, amountOut, feeAmount, err := s.computeSwapStep(
		s.AMMState.SqrtSpotPriceX96,
		lo.Ternary(isZeroToOne, s.AMMState.SqrtPriceLowX96, s.AMMState.SqrtPriceHighX96),
		effectiveAMMLiquidity,
		amountInMinusFee,
		// fees have already been deducted
	)
	if err != nil {
		return nil, err
	}

	if sqrtSpotPriceX96New.Eq(s.AMMState.SqrtPriceLowX96) ||
		sqrtSpotPriceX96New.Eq(s.AMMState.SqrtPriceHighX96) {
		return nil, ErrInvalidSpotPriceAfterSwap
	}

	if amountInFilled.Gt(amountInMinusFee) {
		return nil, ErrAmountInFilledGtAmountInWithoutFee
	}

	return &LiquidityQuote{
		amountOut:             amountOut,
		amountInFilled:        amountInFilled,
		feeAmount:             feeAmount,
		sqrtSpotPriceX96New:   sqrtSpotPriceX96New,
		effectiveAMMLiquidity: effectiveAMMLiquidity,
		IsZeroToOne:           isZeroToOne,
	}, nil
}

func (s *PoolSimulator) computeSwapStep(sqrtRatioCurrentX96, sqrtRatioTargetX96, liquidity,
	amountRemaining *uint256.Int) (sqrtRatioNextX96, amountIn, amountOut, feeAmount *uint256.Int, err error) {
	zeroForOne := !sqrtRatioCurrentX96.Lt(sqrtRatioTargetX96)
	amountRemainingLessFee := amountRemaining.Clone()
	if zeroForOne {
		amountIn = s.getAmount0Delta(sqrtRatioTargetX96, sqrtRatioCurrentX96, liquidity, true)
	} else {
		amountIn = s.getAmount1Delta(sqrtRatioCurrentX96, sqrtRatioTargetX96, liquidity, true)
	}

	if !amountRemainingLessFee.Lt(amountIn) {
		sqrtRatioNextX96 = new(uint256.Int).Set(sqrtRatioTargetX96)
	} else {
		sqrtRatioNextX96, err = s.getNextSqrtPriceFromInput(
			sqrtRatioCurrentX96,
			liquidity,
			amountRemainingLessFee,
			zeroForOne,
		)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	_max := sqrtRatioTargetX96.Eq(sqrtRatioNextX96)
	if zeroForOne {
		if !_max {
			amountIn = s.getAmount0Delta(sqrtRatioNextX96, sqrtRatioCurrentX96, liquidity, true)
		}
		amountOut = s.getAmount1Delta(sqrtRatioNextX96, sqrtRatioCurrentX96, liquidity, false)
	} else {
		if !_max {
			amountIn = s.getAmount1Delta(sqrtRatioCurrentX96, sqrtRatioNextX96, liquidity, true)
		}
		amountOut = s.getAmount0Delta(sqrtRatioCurrentX96, sqrtRatioNextX96, liquidity, false)
	}

	if !sqrtRatioNextX96.Eq(sqrtRatioTargetX96) {
		// We didn't reach the target, so take the remainder of the maximum input as fee
		feeAmount = new(uint256.Int).Sub(amountRemaining, amountIn)
	} else {
		feeAmount = new(uint256.Int)
	}

	return
}

func (s *PoolSimulator) getNextSqrtPriceFromInput(sqrtPX96, liquidity, amountIn *uint256.Int, zeroForOne bool) (*uint256.Int, error) {
	if sqrtPX96.Sign() <= 0 {
		return nil, ErrSqrtPX96MustGtZero
	}

	if liquidity.Sign() <= 0 {
		return nil, ErrLiquidityMustGtZero
	}

	if zeroForOne {
		return s.getNextSqrtPriceFromAmount0RoundingUp(sqrtPX96, liquidity, amountIn, true)
	}

	return s.getNextSqrtPriceFromAmount1RoundingDown(sqrtPX96, liquidity, amountIn, true)
}

func (s *PoolSimulator) getNextSqrtPriceFromAmount0RoundingUp(sqrtPX96, liquidity, amount *uint256.Int, add bool) (*uint256.Int, error) {
	if amount.Sign() == 0 {
		return sqrtPX96.Clone(), nil
	}

	res := new(uint256.Int)
	numerator1 := new(uint256.Int).Lsh(liquidity, resolution)
	if add {
		product := new(uint256.Int).Mul(amount, sqrtPX96)
		temp := new(uint256.Int).Div(product, amount)

		if temp.Eq(sqrtPX96) {
			denominator := new(uint256.Int).Add(numerator1, product)

			if !denominator.Lt(numerator1) {
				u256.MulDivRounding(res, numerator1, sqrtPX96, denominator, true)

				return res, nil
			}
		}

		temp.Div(numerator1, sqrtPX96)
		temp.Add(temp, amount)
		return u256.DivUp(res, numerator1, temp), nil
	}

	product := new(uint256.Int).Mul(amount, sqrtPX96)
	temp := new(uint256.Int).Div(product, amount)

	if !temp.Eq(sqrtPX96) || !numerator1.Gt(product) {
		return nil, ErrGetNextSqrtPriceFromAmount0RoundingUp
	}

	temp.Sub(numerator1, product)
	u256.MulDivRounding(res, numerator1, sqrtPX96, temp, true)

	return res, nil
}

func (s *PoolSimulator) getNextSqrtPriceFromAmount1RoundingDown(sqrtPX96, liquidity, amount *uint256.Int, add bool) (*uint256.Int, error) {
	if add {
		var quotient *uint256.Int
		if !amount.Gt(u256.UMaxU160) {
			shifted := new(uint256.Int).Lsh(amount, resolution)
			quotient = new(uint256.Int).Div(shifted, liquidity)
		} else {
			quotient, _ = new(uint256.Int).MulDivOverflow(amount, Q96, liquidity)
		}

		return quotient.Add(sqrtPX96, quotient), nil
	}

	var quotient *uint256.Int
	if !amount.Gt(u256.UMaxU160) {
		quotient := new(uint256.Int).Lsh(amount, resolution)
		u256.DivUp(quotient, quotient, liquidity)
	} else {
		quotient = u256.MulDivRounding(new(uint256.Int), amount, Q96, liquidity, true)
	}

	if !sqrtPX96.Gt(quotient) {
		return nil, ErrGetNextSqrtPriceFromAmount1RoundingDown
	}

	return quotient.Sub(sqrtPX96, quotient), nil
}

func (s *PoolSimulator) getAmount0Delta(sqrtRatioAX96, sqrtRatioBX96, liquidity *uint256.Int, roundUp bool) *uint256.Int {
	sqrtA := new(uint256.Int).Set(sqrtRatioAX96)
	sqrtB := new(uint256.Int).Set(sqrtRatioBX96)
	if sqrtA.Gt(sqrtB) {
		sqrtA, sqrtB = sqrtB, sqrtA
	}

	numerator1 := new(uint256.Int).Lsh(liquidity, resolution)
	numerator2 := new(uint256.Int).Sub(sqrtB, sqrtA)

	res := new(uint256.Int)
	if roundUp {
		u256.MulDivRounding(res, numerator1, numerator2, sqrtB, true)
		u256.DivUp(res, res, sqrtA)
	} else {
		res.MulDivOverflow(numerator1, numerator2, sqrtB)
		res.Div(res, sqrtA)
	}

	return res
}

func (s *PoolSimulator) getAmount1Delta(sqrtRatioAX96, sqrtRatioBX96, liquidity *uint256.Int, roundUp bool) *uint256.Int {
	sqrtA := new(uint256.Int).Set(sqrtRatioAX96)
	sqrtB := new(uint256.Int).Set(sqrtRatioBX96)
	if sqrtA.Gt(sqrtB) {
		sqrtA, sqrtB = sqrtB, sqrtA
	}

	res := new(uint256.Int)
	temp := new(uint256.Int).Sub(sqrtB, sqrtA)

	if roundUp {
		u256.MulDivRounding(res, liquidity, temp, Q96, true)
	} else {
		res, _ = new(uint256.Int).MulDivOverflow(liquidity, temp, Q96)
	}

	return res
}

func (s *PoolSimulator) calculateAMMLiquidity() (updatedLiquidity *uint256.Int) {
	liquidity0 := s.getLiquidityForAmount0(s.AMMState.SqrtSpotPriceX96, s.AMMState.SqrtPriceHighX96, s.reserve0)
	liquidity1 := s.getLiquidityForAmount1(s.AMMState.SqrtPriceLowX96, s.AMMState.SqrtSpotPriceX96, s.reserve1)

	if liquidity0.Lt(liquidity1) {
		return liquidity0
	}

	return liquidity1
}

func (s *PoolSimulator) getLiquidityForAmount0(sqrtRatioAX96, sqrtRatioBX96, amount0 *uint256.Int) *uint256.Int {
	sqrtA := sqrtRatioAX96.Clone()
	sqrtB := sqrtRatioBX96.Clone()
	if sqrtA.Gt(sqrtB) {
		sqrtA, sqrtB = sqrtB, sqrtA
	}

	liquidity, _ := new(uint256.Int).MulDivOverflow(sqrtA, sqrtB, Q96)
	liquidity.MulDivOverflow(amount0, liquidity, sqrtB.Sub(sqrtB, sqrtA))

	return liquidity
}

func (s *PoolSimulator) getLiquidityForAmount1(sqrtRatioAX96, sqrtRatioBX96, amount1 *uint256.Int) *uint256.Int {
	sqrtA := sqrtRatioAX96.Clone()
	sqrtB := sqrtRatioBX96.Clone()
	if sqrtA.Gt(sqrtB) {
		sqrtA, sqrtB = sqrtB, sqrtA
	}

	liquidity, _ := new(uint256.Int).MulDivOverflow(amount1, Q96, sqrtB.Sub(sqrtB, sqrtA))

	return liquidity
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return MetaInfo{
		BlockNumber: s.Info.BlockNumber,
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.EffectiveAMMLiquidity = s.EffectiveAMMLiquidity.Clone()
	cloned.AMMState.SqrtSpotPriceX96 = s.AMMState.SqrtSpotPriceX96.Clone()

	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo := params.SwapInfo.(*LiquidityQuote)

	s.EffectiveAMMLiquidity.Set(swapInfo.effectiveAMMLiquidity)
	s.AMMState.SqrtSpotPriceX96.Set(swapInfo.sqrtSpotPriceX96New)

}
