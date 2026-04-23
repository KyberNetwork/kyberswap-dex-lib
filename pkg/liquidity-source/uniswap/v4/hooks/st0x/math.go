package st0x

import (
	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	uPriceScale = u256.TenPow(18)
	uBpsDenom   = uint256.NewInt(bpsDenom)
)

func calcAmountOut(price, spreadBps uint64, zeroForOne bool, amountIn *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	eff, err := effectivePrice(price, spreadBps, zeroForOne)
	if err != nil {
		return nil, nil, err
	}

	out := u256.MulDiv(amountIn, eff, uPriceScale)
	if !zeroForOne {
		out = u256.MulDiv(amountIn, uPriceScale, eff)
	}
	return out, eff, nil
}

func calcAmountIn(price, spreadBps uint64, zeroForOne bool, amountOut *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	eff, err := effectivePrice(price, spreadBps, zeroForOne)
	if err != nil {
		return nil, nil, err
	}

	in := u256.MulDiv(amountOut, uPriceScale, eff)
	if !zeroForOne {
		in = u256.MulDiv(amountOut, eff, uPriceScale)
	}
	return in, eff, nil
}

// effectivePrice replicates PropAMMHook.beforeSwap lines 167–172:
//
//	zeroForOne  →  P * (10_000 − spreadBps/2) / 10_000
//	oneForZero  →  P * (10_000 + spreadBps/2) / 10_000
func effectivePrice(price, spreadBps uint64, zeroForOne bool) (*uint256.Int, error) {
	if spreadBps >= bpsDenom*2 {
		return nil, ErrInvalidSpread
	}
	halfSpread := spreadBps / 2
	var mul uint64
	if zeroForOne {
		mul = bpsDenom - halfSpread
	} else {
		mul = bpsDenom + halfSpread
	}
	eff := new(uint256.Int).SetUint64(price)
	eff.Mul(eff, uint256.NewInt(mul))
	eff.Div(eff, uBpsDenom)
	return eff, nil
}
