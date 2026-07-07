package calc

import (
	"errors"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

func GetSqrtPrice(point int) (*uint256.Int, error) {
	var sqrtPrice v3Utils.Uint160
	return &sqrtPrice, v3Utils.GetSqrtRatioAtTickV2(point, &sqrtPrice)
}

var (
	logSqrtPTh = []*uint256.Int{
		uint256.MustFromHex("0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"),
		uint256.MustFromHex("0xFFFFFFFFFFFFFFFF"),
		uint256.MustFromHex("0xFFFFFFFF"),
		uint256.MustFromHex("0xFFFF"),
		uint256.MustFromHex("0xFF"),
		uint256.MustFromHex("0xF"),
		uint256.MustFromHex("0x3"),
		uint256.MustFromHex("0x1"),
	}
	bitSize = []uint{128, 64, 32, 16, 8, 4, 2, 1}

	uintValue  = uint256.MustFromDecimal("255738958999603826347141")
	uintValueF = uint256.MustFromDecimal("3402992956809132418596140100660247210")
	uintValueL = uint256.MustFromDecimal("291339464771989622907027621153398088495")
)

func GetLogSqrtPriceFloor(sqrtPrice96 *uint256.Int) (int, error) {
	if sqrtPrice96.Cmp(v3Utils.MinSqrtRatioU256) <= 0 || sqrtPrice96.Cmp(v3Utils.MaxSqrtRatioU256) >= 0 {
		return 0, errors.New("R")
	}

	var s, x, m, tmp uint256.Int
	s.Lsh(sqrtPrice96, 32)
	x.Set(&s)

	for idx, size := range bitSize {
		currTh := logSqrtPTh[idx]
		if x.Cmp(currTh) > 0 {
			m.Or(&m, tmp.SetUint64(uint64(size)))
			if size > 1 {
				x.Rsh(&x, size)
			}
		}
	}

	if m.Cmp(uint256.NewInt(128)) >= 0 {
		x.Rsh(&s, uint(m.Uint64()-127))
	} else {
		x.Lsh(&s, uint(127-m.Uint64()))
	}

	l2 := s.Lsh(s.Sub(&m, s.SetUint64(128)), 64)

	// Simulate the assembly code
	for i := 63; i >= 50; i-- {
		x.Mul(&x, &x)
		x.Rsh(&x, 127)
		y := tmp.Rsh(&x, 128)
		l2 = l2.Or(l2, m.Lsh(y, uint(i)))
		if i > 50 {
			x.Rsh(&x, uint(y.Uint64()))
		}
	}

	ls10001 := l2.Mul(l2, uintValue)
	logFloor := x.Rsh(x.Sub(ls10001, uintValueF), 128)
	logUpper := m.Rsh(m.Add(ls10001, uintValueL), 128)

	logValue := logFloor
	sqrtPrice, _ := GetSqrtPrice(int(logUpper.Uint64()))
	if sqrtPrice.Cmp(sqrtPrice96) <= 0 {
		logValue = logUpper
	}

	return int(logValue.Uint64()), nil
}
