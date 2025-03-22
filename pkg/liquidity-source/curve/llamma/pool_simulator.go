package llamma

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/kutils"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	UseDynamicFee bool

	A              *uint256.Int
	Aminus1        *uint256.Int
	MaxOracleDnPow *uint256.Int
	LogARatio      *int256.Int

	BasePrice  *uint256.Int
	Po         *uint256.Int
	Fee        *uint256.Int
	AdminFee   *uint256.Int
	AdminFeesX *uint256.Int
	AdminFeesY *uint256.Int
	ActiveBand int64
	MinBand    int64
	MaxBand    int64
	BandsX     map[int64]*uint256.Int
	BandsY     map[int64]*uint256.Int

	collateralPrecision *uint256.Int
	borrowedPrecision   *uint256.Int

	gas Gas
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var (
		staticExtra StaticExtra
		extra       Extra
	)

	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	Aminus1 := new(uint256.Int).Sub(staticExtra.A, number.Number_1)

	ARatio, overflow := new(uint256.Int).MulDivOverflow(number.Number_1e18, staticExtra.A, Aminus1)
	if overflow {
		return nil, ErrMulDivOverflow
	}

	maxOracleDnPow := number.Number_1e18.Clone()
	for range maxTicks {
		maxOracleDnPow.MulDivOverflow(staticExtra.A, maxOracleDnPow, Aminus1)
	}

	logARatio := lnInt(ARatio)

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(ep.Reserves, func(item string, index int) *big.Int { return bignumber.NewBig(item) }),
			BlockNumber: ep.BlockNumber,
		}},
		UseDynamicFee:  staticExtra.UseDynamicFee,
		A:              staticExtra.A,
		Aminus1:        Aminus1,
		LogARatio:      logARatio,
		MaxOracleDnPow: maxOracleDnPow,

		BasePrice:  extra.BasePrice,
		Po:         extra.PriceOracle,
		Fee:        extra.Fee,
		AdminFee:   extra.AdminFee,
		AdminFeesX: extra.AdminFeesX,
		AdminFeesY: extra.AdminFeesY,
		ActiveBand: extra.ActiveBand,
		MinBand:    extra.MinBand,
		MaxBand:    extra.MaxBand,
		BandsX:     lo.SliceToMap(extra.Bands, func(e Band) (int64, *uint256.Int) { return e.Index, e.BandX }),
		BandsY:     lo.SliceToMap(extra.Bands, func(e Band) (int64, *uint256.Int) { return e.Index, e.BandY }),

		borrowedPrecision:   big256.TenPowInt(18 - ep.Tokens[0].Decimals),
		collateralPrecision: big256.TenPowInt(18 - ep.Tokens[1].Decimals),

		gas: defaultGas,
	}, nil
}

func (t *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenOut

	tokenInIdx, tokenToIdx := t.GetTokenIndex(tokenIn), t.GetTokenIndex(tokenOut)
	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)

	out, err := t.exchange(tokenInIdx, tokenToIdx, amountIn, true)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: out.OutAmount.ToBig()},
		Fee:            &pool.TokenAmount{Token: tokenIn, Amount: out.AdminFee.ToBig()},
		SwapInfo:       out,
		Gas:            t.gas.Exchange,
	}, nil
}

func (t *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn, tokenOut := params.TokenIn, params.TokenAmountOut.Token
	tokenInIdx, tokenOutIdx := t.GetTokenIndex(tokenIn), t.GetTokenIndex(tokenOut)
	amountOut := uint256.MustFromBig(params.TokenAmountOut.Amount)

	out, err := t.exchange(tokenInIdx, tokenOutIdx, amountOut, false)
	if err != nil {
		return nil, err
	}

	if out.OutAmount.Lt(amountOut) {
		return nil, ErrInsufficientBalance
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: out.InAmount.ToBig()},
		Fee:           &pool.TokenAmount{Token: tokenIn, Amount: out.AdminFee.ToBig()},
		SwapInfo:      out,
		Gas:           t.gas.Exchange,
	}, nil
}

func (t *PoolSimulator) exchange(i, j int, amount *uint256.Int, calcAmountOut bool) (*DetailedTrade, error) {
	if i^j != 1 {
		return nil, ErrWrongIndex
	}

	if amount.Sign() == 0 {
		return nil, ErrZeroSwapAmount
	}

	inPrecision, outPrecision := t.collateralPrecision, t.borrowedPrecision
	if i == 0 {
		inPrecision, outPrecision = outPrecision, inPrecision
	}

	out := &DetailedTrade{}
	var err error
	if calcAmountOut {
		out, err = t.calcSwapOut(i, new(uint256.Int).Mul(amount, inPrecision), t.Po, inPrecision, outPrecision)
	} else {
		out, err = t.calcSwapIn(i, new(uint256.Int).Mul(amount, outPrecision), t.Po, inPrecision, outPrecision)
	}
	if err != nil {
		return nil, err
	}

	out.InAmount.Div(&out.InAmount, inPrecision)
	out.OutAmount.Div(&out.OutAmount, outPrecision)

	if out.InAmount.Sign() == 0 || out.OutAmount.Sign() == 0 {
		return nil, ErrZeroSwapAmount
	}

	return out, nil
}

func (t *PoolSimulator) calcSwapOut(
	inIdx int, inAmount, po, inPrecision, outPrecision *uint256.Int,
) (*DetailedTrade, error) {
	pump := inIdx == t.getBorrowedIndex()
	minBand := t.MinBand
	maxBand := t.MaxBand

	out := &DetailedTrade{N2: t.ActiveBand}
	poUp, err := t.pOracleUp(out.N2)
	if err != nil {
		return nil, err
	}

	var x, y, inAmountLeft, antifee, temp uint256.Int
	x.Set(t.getBandX(out.N2))
	y.Set(t.getBandY(out.N2))
	inAmountLeft.Set(inAmount)

	if !t.UseDynamicFee {
		antifee.Div(
			Number_1e36,
			temp.Sub(number.Number_1e18, minUint256(t.Fee, tenPow18Minus1)),
		)
	}

	j := maxTicksUnit
	for i := range maxTicks + maxSkipTicks {
		var y0, f, g, inv, dynamicFee uint256.Int
		dynamicFee.Set(t.Fee)

		if x.Sign() > 0 || y.Sign() > 0 {
			if j == maxTicksUnit {
				out.N1 = out.N2
				j = 0
			}
			y0.Set(t.getY0(&x, &y, po, poUp))
			f.Mul(t.A, &y0).Mul(&f, po).Div(&f, poUp).Mul(&f, po).Div(&f, number.Number_1e18)
			g.Mul(t.Aminus1, &y0).Mul(&g, poUp).Div(&g, po)
			inv.Add(&f, &x).Mul(&inv, temp.Add(&g, &y))
			if t.UseDynamicFee {
				dynamicFee.Set(maxUint256(t.getDynamicFee(po, poUp), t.Fee))
			}
		}

		if t.UseDynamicFee {
			antifee.Div(
				Number_1e36,
				temp.Sub(number.Number_1e18, minUint256(&dynamicFee, tenPow18Minus1)),
			)
		}

		if j != maxTicksUnit {
			var tick uint256.Int
			tick.Set(&y)
			if pump {
				tick.Set(&x)
			}
			out.TicksIn = append(out.TicksIn, tick)
		}

		// Need this to break if price is too far
		var pRatio uint256.Int
		pRatio.Mul(poUp, number.Number_1e18).Div(&pRatio, po)

		if pump {
			if y.Sign() != 0 {
				if g.Sign() != 0 {
					var xDest, dx uint256.Int
					xDest.Div(&inv, &g).Sub(&xDest, &f).Sub(&xDest, &x)
					dx.Mul(&xDest, &antifee).Div(&dx, number.Number_1e18)

					if dx.Cmp(&inAmountLeft) >= 0 {
						// This is the last band
						xDest.Mul(&inAmountLeft, number.Number_1e18).Div(&xDest, &antifee)

						out.LastTickJ.Div(&inv, temp.Add(&x, &xDest).Add(&temp, &f)).
							Sub(&out.LastTickJ, &g).Add(&out.LastTickJ, number.Number_1)
						if out.LastTickJ.Cmp(&y) > 0 {
							out.LastTickJ.Set(&y)
						}

						xDest.Sub(&inAmountLeft, &xDest).Mul(&xDest, t.AdminFee).Div(&xDest, number.Number_1e18)
						x.Add(&x, &inAmountLeft)

						// Round down the output
						out.OutAmount.Add(&out.OutAmount, &y).Sub(&out.OutAmount, &out.LastTickJ)
						out.TicksIn[j].Sub(&x, &xDest)
						out.InAmount.Set(inAmount)
						out.AdminFee.Add(&out.AdminFee, &xDest)
						break
					} else { // We go into the next band
						// Prevents from leaving dust in the band
						dx.Set(maxUint256(&dx, number.Number_1))

						xDest.Sub(&dx, &xDest).Mul(&xDest, t.AdminFee).Div(&xDest, number.Number_1e18)
						inAmountLeft.Sub(&inAmountLeft, &dx)

						out.TicksIn[j].Add(&x, &dx).Sub(&out.TicksIn[j], &xDest)
						out.InAmount.Add(&out.InAmount, &dx)
						out.OutAmount.Add(&out.OutAmount, &y)
						out.AdminFee.Add(&out.AdminFee, &xDest)
					}
				}
			}

			if i != maxTicks+maxSkipTicks-1 {
				if out.N2 == maxBand {
					break
				}
				if j == maxTicksUnit-1 {
					break
				}
				if pRatio.Lt(temp.Div(Number_1e36, t.MaxOracleDnPow)) {
					break
				}
				out.N2 += 1
				poUp.Mul(poUp, t.Aminus1).Div(poUp, t.A)
				x.Set(number.Zero)
				y.Set(t.getBandY(out.N2))
			}
		} else { // dump
			if x.Sign() != 0 {
				if f.Sign() != 0 {
					var yDest, dy uint256.Int
					yDest.Div(&inv, &f).Sub(&yDest, &g).Sub(&yDest, &y)
					dy.Mul(&yDest, &antifee).Div(&dy, number.Number_1e18)

					if dy.Cmp(&inAmountLeft) >= 0 {
						// This is the last band
						yDest.Mul(&inAmountLeft, number.Number_1e18).Div(&yDest, &antifee)

						out.LastTickJ.Div(&inv, temp.Add(&y, &yDest).Add(&temp, &g)).
							Sub(&out.LastTickJ, &f).Add(&out.LastTickJ, number.Number_1)
						if out.LastTickJ.Cmp(&x) > 0 {
							out.LastTickJ.Set(&x)
						}

						yDest.Sub(&inAmountLeft, &yDest).Mul(&yDest, t.AdminFee).Div(&yDest, number.Number_1e18)
						y.Add(&y, &inAmountLeft)

						out.OutAmount.Add(&out.OutAmount, &x).Sub(&out.OutAmount, &out.LastTickJ)
						out.TicksIn[j].Sub(&y, &yDest)
						out.InAmount.Set(inAmount)
						out.AdminFee.Add(&out.AdminFee, &yDest)
						break
					} else { // We go into the next band
						// Prevents from leaving dust in the band
						dy.Set(maxUint256(&dy, number.Number_1))

						yDest.Sub(&dy, &yDest).Mul(&yDest, t.AdminFee).Div(&yDest, number.Number_1e18)
						inAmountLeft.Sub(&inAmountLeft, &dy)

						out.TicksIn[j].Add(&y, &dy).Sub(&out.TicksIn[j], &yDest)
						out.InAmount.Add(&out.InAmount, &dy)
						out.OutAmount.Add(&out.OutAmount, &x)
						out.AdminFee.Add(&out.AdminFee, &yDest)
					}
				}
			}
			if i != maxTicks+maxSkipTicks-1 {
				if out.N2 == minBand {
					break
				}
				if j == maxTicksUnit-1 {
					break
				}
				if pRatio.Gt(t.MaxOracleDnPow) {
					// Don't allow to be away by more than ~50 ticks
					break
				}
				out.N2 -= 1
				poUp.Mul(poUp, t.A).Div(poUp, t.Aminus1)
				x.Set(t.getBandX(out.N2))
				y.Set(number.Zero)
			}
		}

		if j != maxTicksUnit {
			j += 1
		}
	}

	inPrecisionMinus1 := new(uint256.Int).Sub(inPrecision, number.Number_1)
	out.InAmount.Add(&out.InAmount, inPrecisionMinus1).Div(&out.InAmount, inPrecision).Mul(&out.InAmount, inPrecision)
	out.OutAmount.Div(&out.OutAmount, outPrecision).Mul(&out.OutAmount, outPrecision)

	return out, nil
}

func (t *PoolSimulator) calcSwapIn(
	inIdx int, outAmount, po, inPrecision, outPrecision *uint256.Int,
) (*DetailedTrade, error) {
	pump := inIdx == t.getBorrowedIndex()
	minBand := t.MinBand
	maxBand := t.MaxBand

	out := &DetailedTrade{N2: t.ActiveBand}
	poUp, err := t.pOracleUp(out.N2)
	if err != nil {
		return nil, err
	}

	var x, y, outAmountLeft, antifee, temp uint256.Int
	x.Set(t.getBandX(out.N2))
	y.Set(t.getBandY(out.N2))
	outAmountLeft.Set(outAmount)

	if !t.UseDynamicFee {
		antifee.Div(
			Number_1e36,
			temp.Sub(number.Number_1e18, minUint256(t.Fee, tenPow18Minus1)),
		)
	}

	j := maxTicksUnit
	for i := range maxTicks + maxSkipTicks {
		var y0, f, g, inv, dynamicFee uint256.Int
		dynamicFee.Set(t.Fee)

		if x.Sign() > 0 || y.Sign() > 0 {
			if j == maxTicksUnit {
				out.N1 = out.N2
				j = 0
			}

			y0.Set(t.getY0(&x, &y, po, poUp))
			f.Mul(t.A, &y0).Mul(&f, po).Div(&f, poUp).Mul(&f, po).Div(&f, number.Number_1e18)
			g.Mul(t.Aminus1, &y0).Mul(&g, poUp).Div(&g, po)
			inv.Add(&f, &x).Mul(&inv, temp.Add(&g, &y))
			if t.UseDynamicFee {
				dynamicFee.Set(maxUint256(t.getDynamicFee(po, poUp), t.Fee))
			}
		}

		if t.UseDynamicFee {
			antifee.Div(
				Number_1e36,
				temp.Sub(number.Number_1e18, minUint256(&dynamicFee, tenPow18Minus1)),
			)
		}

		if j != maxTicksUnit {
			var tick uint256.Int
			tick.Set(&y)
			if pump {
				tick.Set(&x)
			}
			out.TicksIn = append(out.TicksIn, tick)
		}

		// Need this to break if price is too far
		var pRatio uint256.Int
		pRatio.Mul(poUp, number.Number_1e18).Div(&pRatio, po)

		if pump {
			if y.Sign() != 0 {
				if g.Sign() != 0 {
					if !y.Lt(&outAmountLeft) {
						// This is the last band
						out.LastTickJ.Sub(&y, &outAmountLeft)
						var xDest, dx uint256.Int
						xDest.Add(&g, &out.LastTickJ).Div(&inv, &xDest).Sub(&xDest, &f).Sub(&xDest, &x)
						dx.Mul(&xDest, &antifee).Div(&dx, number.Number_1e18)
						out.OutAmount.Set(outAmount)
						out.InAmount.Add(&out.InAmount, &dx)
						xDest.Sub(&dx, &xDest).Mul(&xDest, t.AdminFee).Div(&xDest, number.Number_1e18)
						out.TicksIn[j].Add(&x, &dx).Sub(&out.TicksIn[j], &xDest)
						out.AdminFee.Add(&out.AdminFee, &xDest)
						break
					} else {
						// We go into the next band
						var xDest, dx uint256.Int
						xDest.Div(&inv, &g).Sub(&xDest, &f).Sub(&xDest, &x)
						dx.Set(maxUint256(dx.Mul(&xDest, &antifee).Div(&dx, number.Number_1e18), number.Number_1))
						outAmountLeft.Sub(&outAmountLeft, &y)
						out.InAmount.Add(&out.InAmount, &dx)
						out.OutAmount.Add(&out.OutAmount, &y)
						xDest.Sub(&dx, &xDest).Mul(&xDest, t.AdminFee).Div(&xDest, number.Number_1e18)
						out.TicksIn[j].Add(&x, &dx).Sub(&out.TicksIn[j], &xDest)
						out.AdminFee.Add(&out.AdminFee, &xDest)
					}
				}
			}

			if i != maxTicks+maxSkipTicks-1 {
				if out.N2 == maxBand {
					break
				}
				if j == maxTicksUnit-1 {
					break
				}
				if pRatio.Lt(temp.Div(Number_1e36, t.MaxOracleDnPow)) {
					break
				}
				out.N2 += 1
				poUp.Mul(poUp, t.Aminus1).Div(poUp, t.A)
				x.Set(number.Zero)
				y.Set(t.getBandY(out.N2))
			}
		} else { // dump
			if x.Sign() != 0 {
				if f.Sign() != 0 {
					if !x.Lt(&outAmountLeft) {
						// This is the last band
						out.LastTickJ.Sub(&x, &outAmountLeft)
						var yDest, dy uint256.Int
						yDest.Add(&f, &out.LastTickJ).Div(&inv, &yDest).Sub(&yDest, &g).Sub(&yDest, &y)
						dy.Mul(&yDest, &antifee).Div(&dy, number.Number_1e18)
						out.OutAmount.Set(outAmount)
						out.InAmount.Add(&out.InAmount, &dy)
						yDest.Sub(&dy, &yDest).Mul(&yDest, t.AdminFee).Div(&yDest, number.Number_1e18)
						out.TicksIn[j].Add(&y, &dy).Sub(&out.TicksIn[j], &yDest)
						out.AdminFee.Add(&out.AdminFee, &yDest)
						break
					} else {
						// We go into the next band
						var yDest, dy uint256.Int
						yDest.Div(&inv, &f).Sub(&yDest, &g).Sub(&yDest, &y)
						dy.Set(maxUint256(dy.Mul(&yDest, &antifee).Div(&dy, number.Number_1e18), number.Number_1))
						outAmountLeft.Sub(&outAmountLeft, &x)
						out.InAmount.Add(&out.InAmount, &dy)
						out.OutAmount.Add(&out.OutAmount, &x)
						yDest.Sub(&dy, &yDest).Mul(&yDest, t.AdminFee).Div(&yDest, number.Number_1e18)
						out.TicksIn[j].Add(&y, &dy).Sub(&out.TicksIn[j], &yDest)
						out.AdminFee.Add(&out.AdminFee, &yDest)
					}
				}
			}
			if i != maxTicks+maxSkipTicks-1 {
				if out.N2 == minBand {
					break
				}
				if j == maxTicksUnit-1 {
					break
				}
				if pRatio.Gt(t.MaxOracleDnPow) {
					// Don't allow to be away by more than ~50 ticks
					break
				}
				out.N2 -= 1
				poUp.Mul(poUp, t.A).Div(poUp, t.Aminus1)
				x.Set(t.getBandX(out.N2))
				y.Set(number.Zero)
			}
		}

		if j != maxTicksUnit {
			j += 1
		}
	}

	inPrecisionMinus1 := new(uint256.Int).Sub(inPrecision, number.Number_1)
	out.InAmount.Add(&out.InAmount, inPrecisionMinus1).Div(&out.InAmount, inPrecision).Mul(&out.InAmount, inPrecision)
	out.OutAmount.Div(&out.OutAmount, outPrecision).Mul(&out.OutAmount, outPrecision)

	return out, nil
}

func (t *PoolSimulator) getY0(x, y, po, poUp *uint256.Int) *uint256.Int {
	var b, temp uint256.Int
	if x.Sign() != 0 {
		b.Mul(poUp, t.Aminus1).Mul(&b, x).Div(&b, po)
	}
	if y.Sign() != 0 {
		temp.Mul(t.A, po).Mul(&temp, po).Div(&temp, poUp).Mul(&temp, y).Div(&temp, number.Number_1e18)
		b.Add(&b, &temp)
	}
	var numerator, denominator uint256.Int
	if x.Sign() > 0 && y.Sign() > 0 {
		var D uint256.Int
		D.Mul(number.Number_4, t.A).Mul(&D, po).Mul(&D, y).Div(&D, number.Number_1e18)
		D.Mul(&D, x)

		D.Add(&D, temp.Mul(&b, &b))

		numerator.Add(&b, temp.Sqrt(&D)).Mul(&numerator, number.Number_1e18)
		denominator.Mul(t.A, number.Number_2).Mul(&denominator, po)
	} else {
		numerator.Mul(&b, number.Number_1e18)
		denominator.Mul(t.A, po)
	}

	return numerator.Div(&numerator, &denominator)
}

func (t *PoolSimulator) pOracleUp(n int64) (*uint256.Int, error) {
	var power int256.Int
	power.SetInt64(-n).Mul(&power, t.LogARatio)

	expPower, err := wadExp(&power)
	if err != nil {
		return nil, err
	}

	var ret uint256.Int
	return ret.Mul(t.BasePrice, expPower).Div(&ret, number.Number_1e18), nil
}

func (t *PoolSimulator) getDynamicFee(po, poUp *uint256.Int) *uint256.Int {
	var ret, pcd, pcu uint256.Int
	pcd.Mul(po, po).Div(&pcd, poUp).Mul(&pcd, po).Div(&pcd, poUp)

	pcu.Mul(&pcd, t.A).Div(&pcu, t.Aminus1).Mul(&pcu, t.A).Div(&pcu, t.Aminus1)

	if po.Lt(&pcd) {
		ret.Sub(&pcd, po).Mul(&ret, tenPow18Div4).Div(&ret, &pcd)
	} else {
		ret.Sub(po, &pcu).Mul(&ret, tenPow18Div4).Div(&ret, po)
	}

	return &ret
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	out := params.SwapInfo.(*DetailedTrade)
	tokenInIndex := t.GetTokenIndex(params.TokenAmountIn.Token)
	inPrecision := t.getTokenPrecision(tokenInIndex)

	out.AdminFee.Div(&out.AdminFee, inPrecision)
	if tokenInIndex == 0 {
		t.AdminFeesX.Add(t.AdminFeesX, &out.AdminFee)
	} else {
		t.AdminFeesY.Add(t.AdminFeesY, &out.AdminFee)
	}

	n := kutils.Min(out.N1, out.N2)
	nDiff := kutils.Abs(out.N2 - out.N1)
	for k := range maxTicks {
		var x, y uint256.Int
		if tokenInIndex == 0 {
			x.Set(&out.TicksIn[k])
			if n == out.N2 {
				y.Set(&out.LastTickJ)
			}
		} else {
			y.Set(&out.TicksIn[nDiff-k])
			if n == out.N2 {
				x.Set(&out.LastTickJ)
			}
		}
		t.setBandX(n, &x)
		t.setBandY(n, &y)
		if k == nDiff {
			break
		}
		n += 1
	}

	t.ActiveBand = out.N2
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, _ string) interface{} {
	return Meta{
		TokenInIndex: t.GetTokenIndex(tokenIn),
		BlockNumber:  t.Info.BlockNumber,
	}
}

func (t *PoolSimulator) CanSwapFrom(address string) []string {
	switch t.GetTokenIndex(address) {
	case t.getBorrowedIndex():
		return []string{t.getCollateralToken()}
	case t.getCollateralIndex():
		return []string{t.getBorrowedToken()}
	}
	return []string{}
}

func (t *PoolSimulator) CanSwapTo(address string) []string {
	return t.CanSwapFrom(address)
}

func (t *PoolSimulator) getBorrowedIndex() int {
	return 0
}

func (t *PoolSimulator) getCollateralIndex() int {
	return 1
}

func (t *PoolSimulator) getBorrowedToken() string {
	return t.GetTokens()[t.getBorrowedIndex()]
}

func (t *PoolSimulator) getCollateralToken() string {
	return t.GetTokens()[t.getCollateralIndex()]
}

func (t *PoolSimulator) getTokenPrecision(tokenIndex int) *uint256.Int {
	switch tokenIndex {
	case t.getBorrowedIndex():
		return t.borrowedPrecision
	case t.getCollateralIndex():
		return t.collateralPrecision
	}
	return nil
}

func (t *PoolSimulator) getBandX(index int64) *uint256.Int {
	if _, ok := t.BandsX[index]; ok {
		return t.BandsX[index]
	}
	return uint256.NewInt(0)
}

func (t *PoolSimulator) setBandX(index int64, value *uint256.Int) {
	if x, ok := t.BandsX[index]; ok {
		x.Set(value)
	} else {
		t.BandsX[index] = new(uint256.Int).Set(value)
	}
}

func (t *PoolSimulator) getBandY(index int64) *uint256.Int {
	if _, ok := t.BandsY[index]; ok {
		return t.BandsY[index]
	}
	return uint256.NewInt(0)
}

func (t *PoolSimulator) setBandY(index int64, value *uint256.Int) {
	if y, ok := t.BandsY[index]; ok {
		y.Set(value)
	} else {
		t.BandsY[index] = new(uint256.Int).Set(value)
	}
}
