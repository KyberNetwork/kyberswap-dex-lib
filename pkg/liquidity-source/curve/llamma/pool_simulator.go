package llamma

import (
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	inPrecision  *uint256.Int
	outPrecision *uint256.Int

	po *PriceOracle

	minBand    *int256.Int
	maxBand    *int256.Int
	activeBand *int256.Int
	bandsX     map[int64]*uint256.Int
	bandsY     map[int64]*uint256.Int

	fee      *uint256.Int
	adminFee *uint256.Int

	A              *uint256.Int
	Aminus1        *uint256.Int
	maxOracleDnPow *uint256.Int

	collateralPrecision *uint256.Int
	borrowedPrecision   *uint256.Int

	gas int64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	Aminus1 := new(uint256.Int).Sub(staticExtra.A, big256.One)

	var maxOracleDnPow uint256.Int
	maxOracleDnPow.Div(staticExtra.A, Aminus1).Exp(&maxOracleDnPow, u256Fifty) // (A/(A-1))**50

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(ep.Reserves, func(item string, index int) *big.Int { return utils.NewBig(item) }),
			BlockNumber: ep.BlockNumber,
		}},

		A:                   staticExtra.A,
		Aminus1:             Aminus1,
		maxOracleDnPow:      &maxOracleDnPow,
		collateralPrecision: big256.TenPowInt(ep.Tokens[0].Decimals),
		borrowedPrecision:   big256.TenPowInt(ep.Tokens[1].Decimals),

		gas: defaultGas,
	}, nil
}

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut := param.TokenAmountIn.Token, param.TokenOut

	tokenInIdx, tokenToIdx := t.GetTokenIndex(tokenIn), t.GetTokenIndex(tokenOut)
	if !(tokenInIdx == 0 && tokenToIdx == 1) {
		return nil, ErrWrongIndex
	}
	// inPrecision, outPrecision := t.inPrecision, t.outPrecision

	var amountIn, amountOut, fee uint256.Int
	amountIn.SetFromBig(param.TokenAmountIn.Amount)

	amountOut.SetUint64(100)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: fee.ToBig(),
		},
		Gas: t.gas,
	}, nil
}

func (t *PoolSimulator) calcSwapOut(pump bool, po *PriceOracle, inAmount, inPrecision, outPrecision *uint256.Int) (out *DetailedTrade) {
	// pump = true: borrowable in, collateral out
	// pump = false: collateral in, borrowable out

	minBand := t.minBand
	maxBand := t.maxBand
	out.N2 = t.activeBand
	poUp := t.pOracleUp(out.N2)

	x := t.bandsX[out.N2.Int64()]
	y := t.bandsY[out.N2.Int64()]
	inAmountLeft := new(uint256.Int).Set(inAmount)

	antifee := new(uint256.Int).Div(tenPow36, MaxUint256(t.fee, tenPow18))

	j := maxTicksUnit
	for i := range maxTicks + maxSkipTicks {
		var y0, f, g, inv uint256.Int

		var temp1, temp2 uint256.Int

		if x.Sign() > 0 || y.Sign() > 0 {
			if j == maxTicksUnit {
				out.N1.Set(out.N2)
				j = 0
			}

			y0.Set(t.getY0(x, y, po.Price, poUp))

			temp1.Mul(t.A, &y0).Mul(&temp1, po.Price)
			temp2.Mul(poUp, po.Price)
			f.Div(&temp1, &temp2).Sub(&f, tenPow18)

			g.Mul(t.Aminus1, &y0).Mul(&g, poUp).Div(&g, poUp)

			inv.Add(&f, x).Mul(&inv, temp1.Add(&g, y))
		}

		if j != maxTicksUnit {
			var tick uint256.Int
			tick.Set(y)
			if pump {
				tick.Set(x)
			}
			out.TicksIn = append(out.TicksIn, &tick)
		}

		var pRatio uint256.Int
		pRatio.Mul(poUp, tenPow18).Div(&pRatio, po.Price)

		if pump {
			if y.Sign() != 0 {
				if g.Sign() != 0 {
					var xDest, dx uint256.Int
					xDest.Div(&inv, &g).Sub(&xDest, &f).Sub(&xDest, x)
					dx.Mul(&xDest, antifee).Div(&dx, tenPow18)

					if dx.Cmp(inAmountLeft) >= 0 {
						// This is the last band
						xDest.Mul(inAmountLeft, tenPow18).Div(&xDest, tenPow18)

						temp1.Add(&xDest, x).Add(&temp1, &f)
						out.LastTickJ = new(uint256.Int).Div(&inv, &temp1).Sub(out.LastTickJ, &g).Add(out.LastTickJ, big256.One)
						if out.LastTickJ.Cmp(y) > 0 {
							out.LastTickJ.Set(y)
						}

						xDest.Sub(inAmountLeft, &xDest).Mul(&xDest, t.adminFee).Div(&xDest, tenPow18)
						x.Add(x, inAmountLeft)

						// Round down the output
						out.OutAmount.Add(out.OutAmount, y).Sub(out.OutAmount, out.LastTickJ)
						out.TicksIn[j].Sub(x, &xDest)
						out.InAmount.Set(inAmount)
						out.AdminFee.Add(out.AdminFee, &xDest)

						break
					} else {
						// We go into the next band
						if dx.Lt(big256.One) { // Prevents from leaving dust in the band
							dx.Set(big256.One)
						}

						xDest.Sub(&dx, &xDest).Mul(&xDest, t.adminFee).Div(&xDest, tenPow18)
						inAmountLeft.Sub(inAmountLeft, &dx)

						out.TicksIn[j].Add(x, &dx).Sub(out.TicksIn[j], &xDest)
						out.InAmount.Add(out.InAmount, &dx)
						out.OutAmount.Add(out.OutAmount, y)
						out.AdminFee.Add(out.AdminFee, &xDest)
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
				if pRatio.Lt(temp1.Div(tenPow36, t.maxOracleDnPow)) {
					break
				}
				out.N2.Add(out.N2, i256One)
				poUp.Mul(poUp, t.Aminus1).Div(poUp, t.A)
				x.Set(big256.ZeroBI)
				y.Set(t.bandsY[out.N2.Int64()])
			}
		} else {
			if x.Sign() != 0 {
				if f.Sign() != 0 {
					var yDest, dy uint256.Int
					yDest.Div(&inv, &f).Sub(&yDest, &g).Sub(&yDest, y)
					dy.Mul(&yDest, antifee).Div(&dy, tenPow18)

					if dy.Cmp(inAmountLeft) >= 0 {
						// This is the last band
						yDest.Mul(inAmountLeft, tenPow18).Div(&yDest, tenPow18)

						temp1.Add(&yDest, x).Add(&temp1, &f)
						out.LastTickJ = new(uint256.Int).Div(&inv, &temp1).Sub(out.LastTickJ, &g).Add(out.LastTickJ, big256.One)
						if out.LastTickJ.Cmp(y) > 0 {
							out.LastTickJ.Set(y)
						}

						yDest.Sub(inAmountLeft, &yDest).Mul(&yDest, antifee).Div(&yDest, tenPow18)
						x.Add(x, inAmountLeft)

						// Round down the output
						out.OutAmount.Add(out.OutAmount, y).Sub(out.OutAmount, out.LastTickJ)
						out.TicksIn[j].Sub(x, &yDest)
						out.InAmount.Set(inAmount)
						out.AdminFee.Add(out.AdminFee, &yDest)

						break
					} else {
						// We go into the next band
						if dy.Lt(big256.One) { // Prevents from leaving dust in the band
							dy.Set(big256.One)
						}

						yDest.Sub(&dy, &yDest).Mul(&yDest, antifee).Div(&yDest, tenPow18)
						inAmountLeft.Sub(inAmountLeft, &dy)

						out.TicksIn[j].Add(x, &dy).Sub(out.TicksIn[j], &yDest)
						out.InAmount.Add(out.InAmount, &dy)
						out.OutAmount.Add(out.OutAmount, y)
						out.AdminFee.Add(out.AdminFee, &yDest)
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
				if pRatio.Lt(t.maxOracleDnPow) {
					// Don't allow to be away by more than ~50 ticks
					break
				}
				out.N2.Sub(out.N2, i256One)
				poUp.Mul(poUp, t.A).Div(poUp, t.Aminus1)
				x.Set(t.bandsX[out.N2.Int64()])
				y.Set(big256.ZeroBI)
			}
		}

		if j != maxTicksUnit {
			j += 1
		}
	}

	out.InAmount = new(uint256.Int).Mul(new(uint256.Int).Div(new(uint256.Int).Add(out.InAmount, new(uint256.Int).Sub(inPrecision, tenPow18)), inPrecision), inPrecision)
	out.OutAmount = new(uint256.Int).Mul(new(uint256.Int).Div(out.OutAmount, outPrecision), outPrecision)

	return out
}

func (t *PoolSimulator) getY0(x, y, po, poUp *uint256.Int) *uint256.Int {
	return nil
}

func (t *PoolSimulator) pOracleUp(n *int256.Int) *uint256.Int {
	return nil
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {

}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	meta := curve.Meta{
		TokenInIndex:  t.GetTokenIndex(tokenIn),
		TokenOutIndex: t.GetTokenIndex(tokenOut),
		Underlying:    false,
	}
	return meta
}

func (t *PoolSimulator) CanSwapFrom(address string) []string {
	tokenIdx := t.GetTokenIndex(address)
	if tokenIdx == 0 {
		return []string{t.GetTokens()[1]}
	}
	return []string{}
}

func (t *PoolSimulator) CanSwapTo(address string) []string {
	tokenIdx := t.GetTokenIndex(address)
	if tokenIdx == 1 {
		return []string{t.GetTokens()[0]}
	}
	return []string{}
}
