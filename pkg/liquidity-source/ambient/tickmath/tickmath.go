package tickmath

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ambient/types"
)

func GetSqrtRatioAtTick(tick types.Int24) (sqrtPriceX64 *big.Int, err error) {
	// Set to unchecked, but the original UniV3 library was written in a pre-checked version of Solidity
	// require(tick >= MIN_TICK && tick <= MAX_TICK);
	if tick < MIN_TICK || tick > MAX_TICK {
		return nil, fmt.Errorf("Tick out of range")
	}

	// uint256 absTick = tick < 0 ? uint256(-int256(tick)) : uint256(int256(tick));
	var absTick *big.Int
	if tick < 0 {
		absTick = big.NewInt(int64(-tick))
	} else {
		absTick = big.NewInt(int64(tick))
	}

	var (
		ratio   = new(big.Int)
		condTmp = new(big.Int)
	)
	// uint256 ratio = absTick & 0x1 != 0 ? 0xfffcb933bd6fad37aa2d162d1a594001 : 0x100000000000000000000000000000000;
	if condTmp.And(absTick, big0x1).Cmp(big0x0) != 0 {
		ratio.Set(big0xfffcb933bd6fad37aa2d162d1a594001)
	} else {
		ratio.Set(big0x10000000000000000000000000000000)
	}

	// if (absTick & 0x2 != 0) ratio = (ratio * 0xfff97272373d413259a46990580e213a) >> 128;
	if condTmp.And(absTick, big0x2).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0xfff97272373d413259a46990580e213a)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x4 != 0) ratio = (ratio * 0xfff2e50f5f656932ef12357cf3c7fdcc) >> 128;
	if condTmp.And(absTick, big0x4).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0xfff2e50f5f656932ef12357cf3c7fdcc)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x8 != 0) ratio = (ratio * 0xffe5caca7e10e4e61c3624eaa0941cd0) >> 128;
	if condTmp.And(absTick, big0x8).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0xffe5caca7e10e4e61c3624eaa0941cd0)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x10 != 0) ratio = (ratio * 0xffcb9843d60f6159c9db58835c926644) >> 128;
	if condTmp.And(absTick, big0x10).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0xffcb9843d60f6159c9db58835c926644)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x20 != 0) ratio = (ratio * 0xff973b41fa98c081472e6896dfb254c0) >> 128;
	if condTmp.And(absTick, big0x10).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0xff973b41fa98c081472e6896dfb254c0)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x40 != 0) ratio = (ratio * 0xff2ea16466c96a3843ec78b326b52861) >> 128;
	if condTmp.And(absTick, big0x40).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0xff2ea16466c96a3843ec78b326b52861)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x80 != 0) ratio = (ratio * 0xfe5dee046a99a2a811c461f1969c3053) >> 128;
	if condTmp.And(absTick, big0x80).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0xfe5dee046a99a2a811c461f1969c3053)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x100 != 0) ratio = (ratio * 0xfcbe86c7900a88aedcffc83b479aa3a4) >> 128;
	if condTmp.And(absTick, big0x100).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0xfcbe86c7900a88aedcffc83b479aa3a4)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x200 != 0) ratio = (ratio * 0xf987a7253ac413176f2b074cf7815e54) >> 128;
	if condTmp.And(absTick, big0x200).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0xf987a7253ac413176f2b074cf7815e54)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x400 != 0) ratio = (ratio * 0xf3392b0822b70005940c7a398e4b70f3) >> 128;
	if condTmp.And(absTick, big0x400).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0xf3392b0822b70005940c7a398e4b70f3)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x800 != 0) ratio = (ratio * 0xe7159475a2c29b7443b29c7fa6e889d9) >> 128;
	if condTmp.And(absTick, big0x800).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0xe7159475a2c29b7443b29c7fa6e889d9)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x1000 != 0) ratio = (ratio * 0xd097f3bdfd2022b8845ad8f792aa5825) >> 128;
	if condTmp.And(absTick, big0x1000).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0xd097f3bdfd2022b8845ad8f792aa5825)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x2000 != 0) ratio = (ratio * 0xa9f746462d870fdf8a65dc1f90e061e5) >> 128;
	if condTmp.And(absTick, big0x2000).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0xa9f746462d870fdf8a65dc1f90e061e5)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x4000 != 0) ratio = (ratio * 0x70d869a156d2a1b890bb3df62baf32f7) >> 128;
	if condTmp.And(absTick, big0x4000).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0x70d869a156d2a1b890bb3df62baf32f7)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x8000 != 0) ratio = (ratio * 0x31be135f97d08fd981231505542fcfa6) >> 128;
	if condTmp.And(absTick, big0x8000).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0x31be135f97d08fd981231505542fcfa6)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x10000 != 0) ratio = (ratio * 0x9aa508b5b7a84e1c677de54f3e99bc9) >> 128;
	if condTmp.And(absTick, big0x10000).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0x9aa508b5b7a84e1c677de54f3e99bc9)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x20000 != 0) ratio = (ratio * 0x5d6af8dedb81196699c329225ee604) >> 128;
	if condTmp.And(absTick, big0x20000).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0x5d6af8dedb81196699c329225ee604)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x40000 != 0) ratio = (ratio * 0x2216e584f5fa1ea926041bedfe98) >> 128;
	if condTmp.And(absTick, big0x40000).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0x2216e584f5fa1ea926041bedfe98)
		ratio.Rsh(ratio, 128)
	}

	// if (absTick & 0x80000 != 0) ratio = (ratio * 0x48a170391f7dc42444e8fa2) >> 128;
	if condTmp.And(absTick, big0x80000).Cmp(big0x0) != 0 {
		ratio.Mul(ratio, big0x48a170391f7dc42444e8fa2)
		ratio.Rsh(ratio, 128)
	}

	// if (tick > 0) ratio = type(uint256).max / ratio;
	if tick > 0 {
		ratio.Div(bigMaxUint256, ratio)
	}

	// this divides by 1<<64 rounding up to go from a Q128.128 to a Q64.64
	// we then downcast because we know the result always fits within 128 bits due to our tick input constraint
	// we round up in the division so getTickAtSqrtRatio of the output price is always consistent
	// sqrtPriceX64 = uint128((ratio >> 64) + (ratio % (1 << 64) == 0 ? 0 : 1));

	sqrtPriceX64 = new(big.Int).Set(ratio)
	sqrtPriceX64.Rsh(sqrtPriceX64, 64)
	if ratio.Mod(ratio, big2Pow64).Cmp(big0x0) != 0 {
		sqrtPriceX64.Add(sqrtPriceX64, big0x1)
	}

	return sqrtPriceX64, nil
}
