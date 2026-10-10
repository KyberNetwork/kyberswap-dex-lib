package lotflow

import (
	"math/big"

	"github.com/holiman/uint256"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

// Exact integer helpers. Everything on the arithmetic path is math/big so that every Solidity
// mulDiv/floor/ceil is reproduced bit for bit; tick math goes through dex-lib's uniswap v3 TickMath.

var (
	bigBps  = big.NewInt(10_000)
	bigE18  = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	bigE36  = new(big.Int).Exp(big.NewInt(10), big.NewInt(36), nil)
	bigQ96  = new(big.Int).Lsh(big.NewInt(1), 96)
	bigQ128 = new(big.Int).Lsh(big.NewInt(1), 128)
	bigQ192 = new(big.Int).Lsh(big.NewInt(1), 192)
	maxU128 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))

	MinSqrtPrice = uniswapv3.MinSqrtRatioU256.ToBig()
	MaxSqrtPrice = uniswapv3.MaxSqrtRatioU256P1.ToBig() // TickMath.MAX_SQRT_PRICE (exclusive bound)
)

func pow10(n int) *big.Int { return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil) }

// mulDiv = floor(a*b/d).
func mulDiv(a, b, d *big.Int) *big.Int {
	z := new(big.Int).Mul(a, b)
	return z.Quo(z, d)
}

// mulDivUp = ceil(a*b/d) for non-negative operands.
func mulDivUp(a, b, d *big.Int) *big.Int { return divUp(new(big.Int).Mul(a, b), d) }

// divUp = ceil(a/d) for non-negative operands.
func divUp(a, d *big.Int) *big.Int {
	q, r := new(big.Int).QuoRem(a, d, new(big.Int))
	if r.Sign() != 0 {
		q.Add(q, big.NewInt(1))
	}
	return q
}

// bpsUp = floor(x*(1e4+bps)/1e4); bpsDown = floor(x*(1e4-bps)/1e4).
func bpsUp(x *big.Int, bps uint64) *big.Int {
	return mulDiv(x, new(big.Int).Add(bigBps, new(big.Int).SetUint64(bps)), bigBps)
}

func bpsDown(x *big.Int, bps uint64) *big.Int {
	return mulDiv(x, new(big.Int).Sub(bigBps, new(big.Int).SetUint64(bps)), bigBps)
}

// SqrtPriceAtTick is TickMath.getSqrtPriceAtTick.
func SqrtPriceAtTick(tick int) (*big.Int, error) {
	var r uint256.Int
	if err := uniswapv3.GetSqrtRatioAtTick(tick, &r); err != nil {
		return nil, err
	}
	return r.ToBig(), nil
}

// TickAtSqrtPrice is TickMath.getTickAtSqrtPrice.
func TickAtSqrtPrice(sqrtP *big.Int) (int, error) {
	u, overflow := uint256.FromBig(sqrtP)
	if overflow {
		return 0, ErrLiquidityCast
	}
	return uniswapv3.GetTickAtSqrtRatio(u)
}

// amount0Delta / amount1Delta are SqrtPriceMath.getAmount0Delta / getAmount1Delta (unsigned form).
func amount0Delta(sa, sb, liq *big.Int, roundUp bool) *big.Int {
	if sa.Cmp(sb) > 0 {
		sa, sb = sb, sa
	}
	n1 := new(big.Int).Lsh(liq, 96)
	n2 := new(big.Int).Sub(sb, sa)
	if roundUp {
		return divUp(mulDivUp(n1, n2, sb), sa)
	}
	return new(big.Int).Quo(mulDiv(n1, n2, sb), sa)
}

func amount1Delta(sa, sb, liq *big.Int, roundUp bool) *big.Int {
	if sa.Cmp(sb) > 0 {
		sa, sb = sb, sa
	}
	if roundUp {
		return mulDivUp(liq, new(big.Int).Sub(sb, sa), bigQ96)
	}
	return mulDiv(liq, new(big.Int).Sub(sb, sa), bigQ96)
}

// positionAmounts is v4 Pool.modifyLiquidity's amount calculation for a position [lower, upper]
// with |liquidityDelta| = liq at (sqrtP, tick): rounded up when adding, down when removing.
func positionAmounts(sqrtP *big.Int, tick, lower, upper int, liq *big.Int, add bool) (amt0, amt1 *big.Int,
	err error) {
	sl, err := SqrtPriceAtTick(lower)
	if err != nil {
		return nil, nil, err
	}
	su, err := SqrtPriceAtTick(upper)
	if err != nil {
		return nil, nil, err
	}
	switch {
	case tick < lower:
		return amount0Delta(sl, su, liq, add), new(big.Int), nil
	case tick < upper:
		return amount0Delta(sqrtP, su, liq, add), amount1Delta(sl, sqrtP, liq, add), nil
	default:
		return new(big.Int), amount1Delta(sl, su, liq, add), nil
	}
}
