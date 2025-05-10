package eulerswap

import (
	"errors"

	"github.com/holiman/uint256"

	bignumber "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	maxUint112 = new(uint256.Int).Sub(new(uint256.Int).Lsh(bignumber.One, 112), bignumber.One) // 2^112 - 1
	MaxUint248 = new(uint256.Int).Sub(new(uint256.Int).Lsh(bignumber.One, 248), bignumber.One) // 2^248 - 1
	maxUint256 = new(uint256.Int).Sub(new(uint256.Int).Lsh(bignumber.One, 256), bignumber.One) // 2^256 - 1
	hundred    = uint256.NewInt(100)
	sixtyThree = uint256.NewInt(63)
	ten        = uint256.NewInt(10)

	ErrSwapLimitExceeded = errors.New("swap limit exceed")
)

func BinarySearch(
	reserve0 *uint256.Int,
	reserve1 *uint256.Int,
	amount *uint256.Int,
	exactIn bool,
	asset0IsInput bool,
	verify func(a, b *uint256.Int) bool,
) (*uint256.Int, error) {

	dx := new(uint256.Int)
	dy := new(uint256.Int)
	reserve0New := new(uint256.Int)
	reserve1New := new(uint256.Int)
	low := new(uint256.Int)
	high := new(uint256.Int).Set(maxUint112)
	mid := new(uint256.Int)
	a := new(uint256.Int)
	b := new(uint256.Int)
	output := new(uint256.Int)

	if exactIn {
		if asset0IsInput {
			dx.Set(amount)
			dy.Set(bignumber.ZeroBI)
		} else {
			dx.Set(bignumber.ZeroBI)
			dy.Set(amount)
		}
	} else {
		if asset0IsInput {
			dx.Set(bignumber.ZeroBI)
			dy.Neg(amount)
		} else {
			dx.Neg(amount)
			dy.Set(bignumber.ZeroBI)
		}
	}

	reserve0New.Add(reserve0, dx)
	reserve1New.Add(reserve1, dy)

	if reserve0New.Sign() <= 0 || reserve1New.Sign() <= 0 {
		return nil, ErrSwapLimitExceeded
	}

	for low.Cmp(high) < 0 {
		mid.Add(low, high)
		mid.Div(mid, bignumber.Two)

		if mid.Sign() <= 0 {
			return nil, ErrSwapLimitExceeded
		}

		if dy.Sign() == 0 {
			a.Set(reserve0New)
			b.Set(mid)
		} else {
			a.Set(mid)
			b.Set(reserve1New)
		}

		if verify(a, b) {
			high.Set(mid)
		} else {
			low.Add(mid, bignumber.One)
		}
	}

	if high.Cmp(maxUint112) >= 0 {
		return nil, ErrSwapLimitExceeded
	}

	if dx.Sign() != 0 {
		dy.Sub(low, reserve1New)
	} else {
		dx.Sub(low, reserve0New)
	}

	if exactIn {
		if asset0IsInput {
			output.Neg(dy)
		} else {
			output.Neg(dx)
		}
	} else {
		if asset0IsInput {
			if dx.Sign() >= 0 {
				output.Set(dx)
			} else {
				output.SetUint64(0)
			}
		} else {
			if dy.Sign() >= 0 {
				output.Set(dy)
			} else {
				output.SetUint64(0)
			}
		}
	}

	return output, nil
}
