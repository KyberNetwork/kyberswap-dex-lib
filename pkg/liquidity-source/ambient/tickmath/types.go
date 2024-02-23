package tickmath

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ambient/types"
)

const (
	/// @dev The minimum tick that may be passed to #getSqrtRatioAtTick computed from log base 1.0001 of 2**-96
	MIN_TICK types.Int24 = -665454
	/// @dev The maximum tick that may be passed to #getSqrtRatioAtTick computed from log base 1.0001 of 2**120
	MAX_TICK types.Int24 = 831818
)

var (
	/// @dev The minimum value that can be returned from #getSqrtRatioAtTick. Equivalent to getSqrtRatioAtTick(MIN_TICK). The reason we don't set this as min(uint128) is so that single precicion moves represent a small fraction.
	MIN_SQRT_RATIO = big.NewInt(65538)
	/// @dev The maximum value that can be returned from #getSqrtRatioAtTick. Equivalent to getSqrtRatioAtTick(MAX_TICK)
	MAX_SQRT_RATIO, _ = new(big.Int).SetString("21267430153580247136652501917186561138", 10)
)

var (
	bigMaxUint256, _ = new(big.Int).SetString("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 0)
	big0x0           = big.NewInt(0)

	big0x1, _     = new(big.Int).SetString("0x1", 0)
	big0x2, _     = new(big.Int).SetString("0x2", 0)
	big0x4, _     = new(big.Int).SetString("0x4", 0)
	big0x8, _     = new(big.Int).SetString("0x8", 0)
	big0x10, _    = new(big.Int).SetString("0x10", 0)
	big0x20, _    = new(big.Int).SetString("0x20", 0)
	big0x40, _    = new(big.Int).SetString("0x40", 0)
	big0x80, _    = new(big.Int).SetString("0x80", 0)
	big0x100, _   = new(big.Int).SetString("0x100", 0)
	big0x200, _   = new(big.Int).SetString("0x200", 0)
	big0x400, _   = new(big.Int).SetString("0x400", 0)
	big0x800, _   = new(big.Int).SetString("0x800", 0)
	big0x1000, _  = new(big.Int).SetString("0x1000", 0)
	big0x2000, _  = new(big.Int).SetString("0x2000", 0)
	big0x4000, _  = new(big.Int).SetString("0x4000", 0)
	big0x8000, _  = new(big.Int).SetString("0x8000", 0)
	big0x10000, _ = new(big.Int).SetString("0x10000", 0)
	big0x20000, _ = new(big.Int).SetString("0x20000", 0)
	big0x40000, _ = new(big.Int).SetString("0x40000", 0)
	big0x80000, _ = new(big.Int).SetString("0x80000", 0)

	big0xfffcb933bd6fad37aa2d162d1a594001, _ = new(big.Int).SetString("0xfffcb933bd6fad37aa2d162d1a594001", 0)
	big0x10000000000000000000000000000000, _ = new(big.Int).SetString("0x100000000000000000000000000000000", 0)
	big0xfff97272373d413259a46990580e213a, _ = new(big.Int).SetString("0xfff97272373d413259a46990580e213a", 0)
	big0xfff2e50f5f656932ef12357cf3c7fdcc, _ = new(big.Int).SetString("0xfff2e50f5f656932ef12357cf3c7fdcc", 0)
	big0xffe5caca7e10e4e61c3624eaa0941cd0, _ = new(big.Int).SetString("0xffe5caca7e10e4e61c3624eaa0941cd0", 0)
	big0xffcb9843d60f6159c9db58835c926644, _ = new(big.Int).SetString("0xffcb9843d60f6159c9db58835c926644", 0)
	big0xff973b41fa98c081472e6896dfb254c0, _ = new(big.Int).SetString("0xff973b41fa98c081472e6896dfb254c0", 0)
	big0xff2ea16466c96a3843ec78b326b52861, _ = new(big.Int).SetString("0xff2ea16466c96a3843ec78b326b52861", 0)
	big0xfe5dee046a99a2a811c461f1969c3053, _ = new(big.Int).SetString("0xfe5dee046a99a2a811c461f1969c3053", 0)
	big0xfcbe86c7900a88aedcffc83b479aa3a4, _ = new(big.Int).SetString("0xfcbe86c7900a88aedcffc83b479aa3a4", 0)
	big0xf987a7253ac413176f2b074cf7815e54, _ = new(big.Int).SetString("0xf987a7253ac413176f2b074cf7815e54", 0)
	big0xf3392b0822b70005940c7a398e4b70f3, _ = new(big.Int).SetString("0xf3392b0822b70005940c7a398e4b70f3", 0)
	big0xe7159475a2c29b7443b29c7fa6e889d9, _ = new(big.Int).SetString("0xe7159475a2c29b7443b29c7fa6e889d9", 0)
	big0xd097f3bdfd2022b8845ad8f792aa5825, _ = new(big.Int).SetString("0xd097f3bdfd2022b8845ad8f792aa5825", 0)
	big0xa9f746462d870fdf8a65dc1f90e061e5, _ = new(big.Int).SetString("0xa9f746462d870fdf8a65dc1f90e061e5", 0)
	big0x70d869a156d2a1b890bb3df62baf32f7, _ = new(big.Int).SetString("0x70d869a156d2a1b890bb3df62baf32f7", 0)
	big0x31be135f97d08fd981231505542fcfa6, _ = new(big.Int).SetString("0x31be135f97d08fd981231505542fcfa6", 0)
	big0x9aa508b5b7a84e1c677de54f3e99bc9, _  = new(big.Int).SetString("0x9aa508b5b7a84e1c677de54f3e99bc9", 0)
	big0x5d6af8dedb81196699c329225ee604, _   = new(big.Int).SetString("0x5d6af8dedb81196699c329225ee604", 0)
	big0x2216e584f5fa1ea926041bedfe98, _     = new(big.Int).SetString("0x2216e584f5fa1ea926041bedfe98", 0)
	big0x48a170391f7dc42444e8fa2, _          = new(big.Int).SetString("0x48a170391f7dc42444e8fa2", 0)
)

var (
	big2Pow64, _ = new(big.Int).SetString("18446744073709551616", 10)
)
