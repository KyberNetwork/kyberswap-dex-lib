package llamma

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/int256"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	A              *uint256.Int
	Aminus1        *uint256.Int
	maxOracleDnPow *uint256.Int
	logARatio      *int256.Int

	po *uint256.Int

	activeBand *int256.Int
	minBand    *int256.Int
	maxBand    *int256.Int
	bandsX     map[int64]*uint256.Int
	bandsY     map[int64]*uint256.Int

	fee        *uint256.Int
	adminFee   *uint256.Int
	adminFeesX *uint256.Int
	adminFeesY *uint256.Int

	basePrice *uint256.Int

	collateralPrecision *uint256.Int
	stableCoinPrecision *uint256.Int

	gas int64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	A := staticExtra.A

	Aminus1 := new(uint256.Int).Sub(A, number.Number_1)

	ARatio, overflow := new(uint256.Int).MulDivOverflow(tenPow18, A, Aminus1)
	if overflow {
		return nil, ErrMulDivOverflow
	}

	maxOracleDnPow := tenPow18.Clone()
	for range maxTicks {
		maxOracleDnPow.MulDivOverflow(A, maxOracleDnPow, Aminus1)
	}

	logARatio := lnInt(ARatio)

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(ep.Reserves, func(item string, index int) *big.Int { return utils.NewBig(item) }),
			BlockNumber: ep.BlockNumber,
		}},

		A:              A,
		Aminus1:        Aminus1,
		logARatio:      logARatio,
		maxOracleDnPow: maxOracleDnPow,

		stableCoinPrecision: big256.TenPowInt(18 - ep.Tokens[0].Decimals),
		collateralPrecision: big256.TenPowInt(18 - ep.Tokens[1].Decimals),

		basePrice:  extra.BasePrice,
		po:         extra.PriceOracle,
		fee:        extra.Fee,
		adminFee:   extra.AdminFee,
		adminFeesX: extra.AdminFeesX,
		adminFeesY: extra.AdminFeesY,

		activeBand: extra.ActiveBand,
		minBand:    extra.MinBand,
		maxBand:    extra.MaxBand,
		bandsX:     extra.BandsX,
		bandsY:     extra.BandsY,

		gas: defaultGas,
	}, nil
}

func (t *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenOut
	tokenInIdx, tokenToIdx := t.GetTokenIndex(tokenIn), t.GetTokenIndex(tokenOut)
	amountIn := uint256.MustFromBig(params.TokenAmountIn.Amount)

	_, amountOutDone, out, err := t.exchange(tokenInIdx, tokenToIdx, amountIn, true)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOutDone.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: integer.Zero(),
		},
		SwapInfo: out,
		Gas:      t.gas,
	}, nil
}

func (t *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn, tokenOut := params.TokenIn, params.TokenAmountOut.Token
	tokenInIdx, tokenOutIdx := t.GetTokenIndex(tokenIn), t.GetTokenIndex(tokenOut)
	amountOut := uint256.MustFromBig(params.TokenAmountOut.Amount)

	amountInDone, _, out, err := t.exchange(tokenInIdx, tokenOutIdx, amountOut, false)
	if err != nil {
		return nil, err
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountInDone.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: integer.Zero(),
		},
		SwapInfo: out,
		Gas:      t.gas,
	}, nil
}

func (t *PoolSimulator) exchange(i, j int, amount *uint256.Int, useAmountIn bool) (*uint256.Int, *uint256.Int, *DetailedTrade, error) {
	if i^j != 1 {
		return nil, nil, nil, ErrWrongIndex
	}

	inIdx, outIdx := t.getStableCoinIdx(), t.getCollateralIdx()
	inPrecision, outPrecision := t.collateralPrecision, t.stableCoinPrecision
	if i == 1 {
		inIdx, outIdx = outIdx, inIdx
		inPrecision, outPrecision = outPrecision, inPrecision
	}

	out := &DetailedTrade{}
	var err error
	if useAmountIn {
		out, err = t.calcSwapOut(
			i == t.getStableCoinIdx(),
			new(uint256.Int).Mul(amount, inPrecision),
			t.po,
			inPrecision, outPrecision,
		)
	} else {
		out, err = t.calcSwapIn(
			i == t.getStableCoinIdx(),
			new(uint256.Int).Mul(amount, outPrecision),
			t.po,
			inPrecision, outPrecision,
		)
	}
	if err != nil {
		return nil, nil, nil, err
	}

	var amountIn, amountOut uint256.Int
	amountIn.Div(&out.InAmount, inPrecision)
	amountOut.Div(&out.OutAmount, outPrecision)

	if out.InAmount.Sign() == 0 || out.OutAmount.Sign() == 0 {
		return nil, nil, nil, ErrZeroSwapAmount
	}

	return &amountIn, &amountOut, out, nil
}

func (t *PoolSimulator) calcSwapOut(
	pump bool, inAmount, po, inPrecision, outPrecision *uint256.Int,
) (*DetailedTrade, error) {
	minBand := t.minBand
	maxBand := t.maxBand

	out := &DetailedTrade{}
	out.N2.Set(t.activeBand)
	poUp, err := t.pOracleUp(&out.N2)
	if err != nil {
		return nil, err
	}
	x := t.bandsX[out.N2.Int64()]
	y := t.bandsY[out.N2.Int64()]

	inAmountLeft := new(uint256.Int).Set(inAmount)
	fee := t.fee
	adminFee := t.adminFee

	var temp uint256.Int
	j := maxTicksUnit
	for i := range maxTicks + maxSkipTicks {
		var (
			y0, f, g, inv uint256.Int
			dynamicFee    uint256.Int
		)

		dynamicFee.Set(fee)

		if x.Sign() > 0 || y.Sign() > 0 {
			if j == maxTicksUnit {
				out.N1.Set(&out.N2)
				j = 0
			}

			y0.Set(t.getY0(x, y, po, poUp))
			f.Mul(t.A, &y0).Mul(&f, po).Div(&f, poUp).Mul(&f, po).Div(&f, tenPow18)
			g.Mul(t.Aminus1, &y0).Mul(&g, poUp).Div(&g, po)
			inv.Add(&f, x).Mul(&inv, temp.Add(&g, y))
			dynamicFee.Set(maxUint256(t.getDynamicFee(po, poUp), fee))
		}

		antifee := new(uint256.Int).Div(
			tenPow36,
			temp.Sub(tenPow18, minUint256(&dynamicFee, tenPow18Minus1)),
		)

		if j != maxTicksUnit {
			var tick uint256.Int
			tick.Set(y)
			if pump {
				tick.Set(x)
			}
			out.TicksIn = append(out.TicksIn, tick)
		}

		// Need this to break if price is too far
		var pRatio uint256.Int
		pRatio.Mul(poUp, tenPow18).Div(&pRatio, po)

		if pump {
			if y.Sign() != 0 {
				if g.Sign() != 0 {
					var xDest, dx uint256.Int
					xDest.Div(&inv, &g).Sub(&xDest, &f).Sub(&xDest, x)
					dx.Mul(&xDest, antifee).Div(&dx, tenPow18)

					if dx.Cmp(inAmountLeft) >= 0 {
						// This is the last band
						xDest.Mul(inAmountLeft, tenPow18).Div(&xDest, antifee)

						out.LastTickJ.Div(&inv, temp.Add(x, &xDest).Add(&temp, &f)).
							Sub(&out.LastTickJ, &g).Add(&out.LastTickJ, number.Number_1)
						if out.LastTickJ.Cmp(y) > 0 {
							out.LastTickJ.Set(y)
						}

						xDest.Sub(inAmountLeft, &xDest).Mul(&xDest, adminFee).Div(&xDest, tenPow18)
						x.Add(x, inAmountLeft)

						// Round down the output
						out.OutAmount.Add(&out.OutAmount, y).Sub(&out.OutAmount, &out.LastTickJ)
						out.TicksIn[j].Sub(x, &xDest)
						out.InAmount.Set(inAmount)
						out.AdminFee.Add(&out.AdminFee, &xDest)
						break
					} else { // We go into the next band
						// Prevents from leaving dust in the band
						dx.Set(maxUint256(&dx, number.Number_1))

						xDest.Sub(&dx, &xDest).Mul(&xDest, adminFee).Div(&xDest, tenPow18)
						inAmountLeft.Sub(inAmountLeft, &dx)

						out.TicksIn[j].Add(x, &dx).Sub(&out.TicksIn[j], &xDest)
						out.InAmount.Add(&out.InAmount, &dx)
						out.OutAmount.Add(&out.OutAmount, y)
						out.AdminFee.Add(&out.AdminFee, &xDest)
					}
				}
			}

			if i != maxTicks+maxSkipTicks-1 {
				if out.N2.Eq(maxBand) {
					break
				}
				if j == maxTicksUnit-1 {
					break
				}
				if pRatio.Lt(temp.Div(tenPow36, t.maxOracleDnPow)) {
					break
				}
				out.N2.Add(&out.N2, i256One)
				poUp.Mul(poUp, t.Aminus1).Div(poUp, t.A)
				x.Set(number.Zero)
				y.Set(t.bandsY[out.N2.Int64()])
			}
		} else { // dump
			if x.Sign() != 0 {
				if f.Sign() != 0 {
					var yDest, dy uint256.Int
					yDest.Div(&inv, &f).Sub(&yDest, &g).Sub(&yDest, y)
					dy.Mul(&yDest, antifee).Div(&dy, tenPow18)

					if dy.Cmp(inAmountLeft) >= 0 {
						// This is the last band
						yDest.Mul(inAmountLeft, tenPow18).Div(&yDest, antifee)

						out.LastTickJ.Div(&inv, temp.Add(y, &yDest).Add(&temp, &g)).
							Sub(&out.LastTickJ, &f).Add(&out.LastTickJ, number.Number_1)
						if out.LastTickJ.Cmp(x) > 0 {
							out.LastTickJ.Set(x)
						}

						yDest.Sub(inAmountLeft, &yDest).Mul(&yDest, antifee).Div(&yDest, tenPow18)
						y.Add(y, inAmountLeft)

						// Round down the output
						out.OutAmount.Add(&out.OutAmount, x).Sub(&out.OutAmount, &out.LastTickJ)
						out.TicksIn[j].Sub(y, &yDest)
						out.InAmount.Set(inAmount)
						out.AdminFee.Add(&out.AdminFee, &yDest)
						break
					} else { // We go into the next band
						// Prevents from leaving dust in the band
						dy.Set(maxUint256(&dy, number.Number_1))

						yDest.Sub(&dy, &yDest).Mul(&yDest, adminFee).Div(&yDest, tenPow18)
						inAmountLeft.Sub(inAmountLeft, &dy)

						out.TicksIn[j].Add(y, &dy).Sub(&out.TicksIn[j], &yDest)
						out.InAmount.Add(&out.InAmount, &dy)
						out.OutAmount.Add(&out.OutAmount, x)
						out.AdminFee.Add(&out.AdminFee, &yDest)
					}
				}
			}
			if i != maxTicks+maxSkipTicks-1 {
				if out.N2.Eq(minBand) {
					break
				}
				if j == maxTicksUnit-1 {
					break
				}
				if pRatio.Gt(t.maxOracleDnPow) {
					// Don't allow to be away by more than ~50 ticks
					break
				}
				out.N2.Sub(&out.N2, i256One)
				poUp.Mul(poUp, t.A).Div(poUp, t.Aminus1)
				x.Set(t.bandsX[out.N2.Int64()])
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
	pump bool, outAmount, po, inPrecision, outPrecision *uint256.Int,
) (*DetailedTrade, error) {
	minBand := t.minBand
	maxBand := t.maxBand

	out := &DetailedTrade{}
	out.N2.Set(t.activeBand)
	poUp, err := t.pOracleUp(&out.N2)
	if err != nil {
		return nil, err
	}
	x := t.bandsX[out.N2.Int64()]
	y := t.bandsY[out.N2.Int64()]

	outAmountLeft := new(uint256.Int).Set(outAmount)
	fee := t.fee
	adminFee := t.adminFee

	var temp uint256.Int
	j := maxTicksUnit
	for i := range maxTicks + maxSkipTicks {
		var (
			y0, f, g, inv uint256.Int
			dynamicFee    uint256.Int
		)

		dynamicFee.Set(fee)

		if x.Sign() > 0 || y.Sign() > 0 {
			if j == maxTicksUnit {
				out.N1.Set(&out.N2)
				j = 0
			}

			y0.Set(t.getY0(x, y, po, poUp))
			f.Mul(t.A, &y0).Mul(&f, po).Div(&f, poUp).Mul(&f, po).Div(&f, tenPow18)
			g.Mul(t.Aminus1, &y0).Mul(&g, poUp).Div(&g, po)
			inv.Add(&f, x).Mul(&inv, temp.Add(&g, y))
			dynamicFee.Set(maxUint256(t.getDynamicFee(po, poUp), fee))
		}

		antifee := new(uint256.Int).Div(
			tenPow36,
			temp.Sub(tenPow18, minUint256(&dynamicFee, tenPow18Minus1)),
		)

		if j != maxTicksUnit {
			var tick uint256.Int
			tick.Set(y)
			if pump {
				tick.Set(x)
			}
			out.TicksIn = append(out.TicksIn, tick)
		}

		// Need this to break if price is too far
		var pRatio uint256.Int
		pRatio.Mul(poUp, tenPow18).Div(&pRatio, po)

		if pump {
			if y.Sign() != 0 {
				if g.Sign() != 0 {
					if !y.Lt(outAmountLeft) {
						// This is the last band
						out.LastTickJ.Sub(y, outAmountLeft)
						var xDest, dx uint256.Int
						xDest.Add(&g, &out.LastTickJ).Div(&inv, &xDest).Sub(&xDest, &f).Sub(&xDest, x)
						dx.Mul(&xDest, antifee).Div(&dx, tenPow18)
						out.OutAmount.Set(outAmount)
						out.InAmount.Add(&out.InAmount, &dx)
						xDest.Sub(&dx, &xDest).Mul(&xDest, adminFee).Div(&xDest, tenPow18)
						out.TicksIn[j].Add(x, &dx).Sub(&out.TicksIn[j], &xDest)
						out.AdminFee.Add(&out.AdminFee, &xDest)
						break
					} else {
						// We go into the next band
						var xDest, dx uint256.Int
						xDest.Div(&inv, &g).Sub(&xDest, &f).Sub(&xDest, x)
						dx.Set(maxUint256(dx.Mul(&xDest, antifee).Div(&dx, tenPow18), number.Number_1))
						outAmountLeft.Sub(outAmountLeft, y)
						out.InAmount.Add(&out.InAmount, &dx)
						out.OutAmount.Add(&out.OutAmount, y)
						xDest.Sub(&dx, &xDest).Mul(&xDest, adminFee).Div(&xDest, tenPow18)
						out.TicksIn[j].Add(x, &dx).Sub(&out.TicksIn[j], &xDest)
						out.AdminFee.Add(&out.AdminFee, &xDest)
					}
				}
			}

			if i != maxTicks+maxSkipTicks-1 {
				if out.N2.Eq(maxBand) {
					break
				}
				if j == maxTicksUnit-1 {
					break
				}
				if pRatio.Lt(temp.Div(tenPow36, t.maxOracleDnPow)) {
					break
				}
				out.N2.Add(&out.N2, i256One)
				poUp.Mul(poUp, t.Aminus1).Div(poUp, t.A)
				x.Set(number.Zero)
				y.Set(t.bandsY[out.N2.Int64()])
			}
		} else { // dump
			if x.Sign() != 0 {
				if f.Sign() != 0 {
					if !x.Lt(outAmountLeft) {
						// This is the last band
						out.LastTickJ.Sub(x, outAmountLeft)
						var yDest, dy uint256.Int
						yDest.Add(&f, &out.LastTickJ).Div(&inv, &yDest).Sub(&yDest, &g).Sub(&yDest, y)
						dy.Mul(&yDest, antifee).Div(&dy, tenPow18)
						out.OutAmount.Set(outAmount)
						out.InAmount.Add(&out.InAmount, &dy)
						yDest.Sub(&dy, &yDest).Mul(&yDest, adminFee).Div(&yDest, tenPow18)
						out.TicksIn[j].Add(y, &dy).Sub(&out.TicksIn[j], &yDest)
						out.AdminFee.Add(&out.AdminFee, &yDest)
						break
					} else {
						// We go into the next band
						var yDest, dy uint256.Int
						yDest.Div(&inv, &f).Sub(&yDest, &g).Sub(&yDest, y)
						dy.Set(maxUint256(dy.Mul(&yDest, antifee).Div(&dy, tenPow18), number.Number_1))
						outAmountLeft.Sub(outAmountLeft, x)
						out.InAmount.Add(&out.InAmount, &dy)
						out.OutAmount.Add(&out.OutAmount, x)
						yDest.Sub(&dy, &yDest).Mul(&yDest, adminFee).Div(&yDest, tenPow18)
						out.TicksIn[j].Add(y, &dy).Sub(&out.TicksIn[j], &yDest)
						out.AdminFee.Add(&out.AdminFee, &yDest)
					}
				}
			}
			if i != maxTicks+maxSkipTicks-1 {
				if out.N2.Eq(minBand) {
					break
				}
				if j == maxTicksUnit-1 {
					break
				}
				if pRatio.Gt(t.maxOracleDnPow) {
					// Don't allow to be away by more than ~50 ticks
					break
				}
				out.N2.Sub(&out.N2, i256One)
				poUp.Mul(poUp, t.A).Div(poUp, t.Aminus1)
				x.Set(t.bandsX[out.N2.Int64()])
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
		temp.Mul(t.A, po).Mul(&temp, po).Div(&temp, poUp).Mul(&temp, y).Div(&temp, tenPow18)
		b.Add(&b, &temp)
	}
	var num, den uint256.Int
	if x.Sign() > 0 && y.Sign() > 0 {
		var D uint256.Int
		D.Mul(big256.Four, t.A).Mul(&D, po).Mul(&D, y).Div(&D, tenPow18)
		D.Mul(&D, x)

		D.Add(&D, temp.Mul(&b, &b))

		num.Add(&b, temp.Sqrt(&D)).Mul(&num, tenPow18)
		den.Mul(t.A, big256.Two).Mul(&den, po)
	} else {
		num.Mul(&b, tenPow18)
		den.Mul(t.A, po)
	}
	return num.Div(&num, &den)
}

func (t *PoolSimulator) pOracleUp(n *int256.Int) (*uint256.Int, error) {
	var power int256.Int
	power.Neg(n).Mul(&power, t.logARatio)

	expPower, err := wadExp(&power)
	if err != nil {
		return nil, err
	}

	uint256ExpPower := uint256.MustFromBig(expPower.ToBig())

	return uint256ExpPower.Mul(t.basePrice, uint256ExpPower).Div(uint256ExpPower, tenPow18), nil
}

func (t *PoolSimulator) limitPO(n *int256.Int) (*uint256.Int, error) {
	var power int256.Int
	power.Neg(n).Mul(&power, t.logARatio)

	expPower, err := wadExp(&power)
	if err != nil {
		return nil, err
	}

	uint256ExpPower := uint256.MustFromBig(expPower.ToBig())

	return uint256ExpPower.Mul(t.basePrice, uint256ExpPower).Div(uint256ExpPower, tenPow18), nil
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

	out.AdminFee.Div(&out.OutPrecision, &out.InPrecision)
	if out.TokenInIdx == 0 {
		t.adminFeesX.Add(t.adminFeesX, &out.AdminFee)
	} else {
		t.adminFeesY.Add(t.adminFeesY, &out.AdminFee)
	}

	n := minInt256(&out.N1, &out.N2)
	nDiff := new(int256.Int).Sub(&out.N2, &out.N1)

	for k := range maxTicks {
		var x, y uint256.Int
		if out.TokenInIdx == 0 {
			x.Set(&out.TicksIn[k])
			if n.Eq(&out.N2) {
				y.Set(&out.LastTickJ)
			}
		} else {
			y.Set(&out.TicksIn[nDiff.Int64()-k])
			if n.Eq(&out.N2) {
				x.Set(&out.LastTickJ)
			}
		}
		t.bandsX[n.Int64()].Set(&x)
		t.bandsY[n.Int64()].Set(&y)
		if k == nDiff.Int64() {
			break
		}
		n.Add(n, i256One)
	}

	t.activeBand.Set(&out.N2)
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}

func (t *PoolSimulator) CanSwapFrom(address string) []string {
	if t.GetTokenIndex(address) == t.getCollateralIdx() {
		return []string{t.GetTokens()[t.getStableCoinIdx()]}
	}
	return []string{}
}

func (t *PoolSimulator) CanSwapTo(address string) []string {
	if t.GetTokenIndex(address) == t.getStableCoinIdx() {
		return []string{t.GetTokens()[t.getCollateralIdx()]}
	}
	return []string{}
}

func (t *PoolSimulator) getCollateralIdx() int {
	return 1
}

func (t *PoolSimulator) getStableCoinIdx() int {
	return 0
}
