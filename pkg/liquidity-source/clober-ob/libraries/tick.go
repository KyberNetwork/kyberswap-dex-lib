package cloberlib

import (
	"errors"

	"github.com/holiman/uint256"
	"github.com/samber/lo"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type Tick = int32

var (
	ErrInvalidTick  = errors.New("invalid tick")
	ErrInvalidPrice = errors.New("invalid price")

	MaxTick Tick = 1<<19 - 1
	MinTick      = -MaxTick

	r0  = uint256.MustFromHex("0xfff97272373d413259a46990")
	r1  = uint256.MustFromHex("0xfff2e50f5f656932ef12357c")
	r2  = uint256.MustFromHex("0xffe5caca7e10e4e61c3624ea")
	r3  = uint256.MustFromHex("0xffcb9843d60f6159c9db5883")
	r4  = uint256.MustFromHex("0xff973b41fa98c081472e6896")
	r5  = uint256.MustFromHex("0xff2ea16466c96a3843ec78b3")
	r6  = uint256.MustFromHex("0xfe5dee046a99a2a811c461f1")
	r7  = uint256.MustFromHex("0xfcbe86c7900a88aedcffc83b")
	r8  = uint256.MustFromHex("0xf987a7253ac413176f2b074c")
	r9  = uint256.MustFromHex("0xf3392b0822b70005940c7a39")
	r10 = uint256.MustFromHex("0xe7159475a2c29b7443b29c7f")
	r11 = uint256.MustFromHex("0xd097f3bdfd2022b8845ad8f7")
	r12 = uint256.MustFromHex("0xa9f746462d870fdf8a65dc1f")
	r13 = uint256.MustFromHex("0x70d869a156d2a1b890bb3df6")
	r14 = uint256.MustFromHex("0x31be135f97d08fd981231505")
	r15 = uint256.MustFromHex("0x9aa508b5b7a84e1c677de54")
	r16 = uint256.MustFromHex("0x5d6af8dedb81196699c329")
	r17 = uint256.MustFromHex("0x2216e584f5fa1ea92604")
	r18 = uint256.MustFromHex("0x48a170391f7dc42")

	q96  = new(uint256.Int).Lsh(u256.U1, 96)
	q192 = new(uint256.Int).Lsh(u256.U1, 192)
)

func ValidateTick(tick Tick) error {
	if tick > MaxTick || tick < MinTick {
		return ErrInvalidTick
	}
	return nil
}

func ToPrice(tick Tick) (*uint256.Int, error) {
	if err := ValidateTick(tick); err != nil {
		return nil, err
	}
	absTick := lo.Ternary(tick < 0, -tick, tick)

	var price uint256.Int
	if absTick&0x1 != 0 {
		price.Set(r0)
	} else {
		price.Set(q96)
	}

	if absTick&0x2 != 0 {
		price.Rsh(price.Mul(&price, r1), 96)
	}
	if absTick&0x4 != 0 {
		price.Rsh(price.Mul(&price, r2), 96)
	}
	if absTick&0x8 != 0 {
		price.Rsh(price.Mul(&price, r3), 96)
	}
	if absTick&0x10 != 0 {
		price.Rsh(price.Mul(&price, r4), 96)
	}
	if absTick&0x20 != 0 {
		price.Rsh(price.Mul(&price, r5), 96)
	}
	if absTick&0x40 != 0 {
		price.Rsh(price.Mul(&price, r6), 96)
	}
	if absTick&0x80 != 0 {
		price.Rsh(price.Mul(&price, r7), 96)
	}
	if absTick&0x100 != 0 {
		price.Rsh(price.Mul(&price, r8), 96)
	}
	if absTick&0x200 != 0 {
		price.Rsh(price.Mul(&price, r9), 96)
	}
	if absTick&0x400 != 0 {
		price.Rsh(price.Mul(&price, r10), 96)
	}
	if absTick&0x800 != 0 {
		price.Rsh(price.Mul(&price, r11), 96)
	}
	if absTick&0x1000 != 0 {
		price.Rsh(price.Mul(&price, r12), 96)
	}
	if absTick&0x2000 != 0 {
		price.Rsh(price.Mul(&price, r13), 96)
	}
	if absTick&0x4000 != 0 {
		price.Rsh(price.Mul(&price, r14), 96)
	}
	if absTick&0x8000 != 0 {
		price.Rsh(price.Mul(&price, r15), 96)
	}
	if absTick&0x10000 != 0 {
		price.Rsh(price.Mul(&price, r16), 96)
	}
	if absTick&0x20000 != 0 {
		price.Rsh(price.Mul(&price, r17), 96)
	}
	if absTick&0x40000 != 0 {
		price.Rsh(price.Mul(&price, r18), 96)
	}

	if tick > 0 {
		price.Div(q192, &price)
	}

	return &price, nil
}

func BaseToQuote(tick Tick, base *uint256.Int, roundingUp bool) (*uint256.Int, error) {
	res, err := ToPrice(tick)
	if err != nil {
		return nil, err
	}

	return u256.MulDivRounding(res, base, res, q96, roundingUp), nil
}

func QuoteToBase(tick Tick, quote *uint256.Int, roundingUp bool) (*uint256.Int, error) {
	res, err := ToPrice(tick)
	if err != nil {
		return nil, err
	}

	return u256.MulDivRounding(res, quote, q96, res, roundingUp), nil
}
