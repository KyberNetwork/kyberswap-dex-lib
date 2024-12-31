package maverickv1

import (
	"time"

	"github.com/holiman/uint256"

	bignumber "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexTypeMaverickV1     = "maverick-v1"
	graphQLRequestTimeout = 20 * time.Second
	defaultChunk          = 200

	poolMethodFee         = "fee"
	poolMethodGetState    = "getState"
	poolMethodBinBalanceA = "binBalanceA"
	poolMethodBinBalanceB = "binBalanceB"
	poolMethodTokenAScale = "tokenAScale"
	poolMethodTokenBScale = "tokenBScale"

	poolMethodGetBin = "getBin"
)

const (
	GasSwap     = 125000
	GasCrossBin = 20000

	zeroString              = "0"
	defaultTokenWeight uint = 50

	MaxTick                   = 460540
	MaxSwapCalcIter           = 150
	MaxSwapCalcIterForPricing = 50

	Kinds      = 4
	KindMask   = 1<<Kinds - 1
	Offsets    = 8
	OffsetMask = 1<<Offsets - 1
	WordSize   = 256
)

var (
	uZero = new(uint256.Int)

	CompareConst1, _ = uint256.FromHex("0xfffcb933bd6fad9d3af5f0b9f25db4d6")
	CompareConst2, _ = uint256.FromHex("0x100000000000000000000000000000000")

	MulConst1, _  = uint256.FromHex("0xfff97272373d41fd789c8cb37ffcaa1c")
	MulConst2, _  = uint256.FromHex("0xfff2e50f5f656ac9229c67059486f389")
	MulConst3, _  = uint256.FromHex("0xffe5caca7e10e81259b3cddc7a064941")
	MulConst4, _  = uint256.FromHex("0xffcb9843d60f67b19e8887e0bd251eb7")
	MulConst5, _  = uint256.FromHex("0xff973b41fa98cd2e57b660be99eb2c4a")
	MulConst6, _  = uint256.FromHex("0xff2ea16466c9838804e327cb417cafcb")
	MulConst7, _  = uint256.FromHex("0xfe5dee046a99d51e2cc356c2f617dbe0")
	MulConst8, _  = uint256.FromHex("0xfcbe86c7900aecf64236ab31f1f9dcb5")
	MulConst9, _  = uint256.FromHex("0xf987a7253ac4d9194200696907cf2e37")
	MulConst10, _ = uint256.FromHex("0xf3392b0822b88206f8abe8a3b44dd9be")
	MulConst11, _ = uint256.FromHex("0xe7159475a2c578ef4f1d17b2b235d480")
	MulConst12, _ = uint256.FromHex("0xd097f3bdfd254ee83bdd3f248e7e785e")
	MulConst13, _ = uint256.FromHex("0xa9f746462d8f7dd10e744d913d033333")
	MulConst14, _ = uint256.FromHex("0x70d869a156ddd32a39e257bc3f50aa9b")
	MulConst15, _ = uint256.FromHex("0x31be135f97da6e09a19dc367e3b6da40")
	MulConst16, _ = uint256.FromHex("0x9aa508b5b7e5a9780b0cc4e25d61a56")
	MulConst17, _ = uint256.FromHex("0x5d6af8dedbcb3a6ccb7ce618d14225")
	MulConst18, _ = uint256.FromHex("0x2216e584f630389b2052b8db590e")
	MulConst19, _ = uint256.FromHex("0x48a1703920644d4030024fe")
	MulConst20, _ = uint256.FromHex("0x149b34ee7b4532")
	MaxUint256    = new(uint256.Int).SetAllOne()

	XAuxConst64, _ = uint256.FromHex("0x100000000000000000000000000000000")
	XAuxConst32, _ = uint256.FromHex("0x10000000000000000")
	XAuxConst16, _ = uint256.FromHex("0x100000000")
	XAuxConst8, _  = uint256.FromHex("0x10000")
	XAuxConst4, _  = uint256.FromHex("0x100")
	XAuxConst2, _  = uint256.FromHex("0x10")
	XAuxConst1, _  = uint256.FromHex("0x8")

	MulConst49_17, _ = uint256.FromDecimal("499999999999999999")

	Bone = bignumber.BONE
)
