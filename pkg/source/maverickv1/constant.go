package maverickv1

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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

var (
	DefaultGas              = Gas{Swap: 125000, CrossBin: 20000}
	zeroBI                  = big.NewInt(0)
	zeroString              = "0"
	defaultTokenWeight uint = 50

	MaxTickBI = big.NewInt(int64(MaxTick))

	CompareConst1 = bignumber.NewBig("0xfffcb933bd6fad9d3af5f0b9f25db4d6")
	CompareConst2 = bignumber.NewBig("0x100000000000000000000000000000000")

	MulConst1  = bignumber.NewBig("0xfff97272373d41fd789c8cb37ffcaa1c")
	MulConst2  = bignumber.NewBig("0xfff2e50f5f656ac9229c67059486f389")
	MulConst3  = bignumber.NewBig("0xffe5caca7e10e81259b3cddc7a064941")
	MulConst4  = bignumber.NewBig("0xffcb9843d60f67b19e8887e0bd251eb7")
	MulConst5  = bignumber.NewBig("0xff973b41fa98cd2e57b660be99eb2c4a")
	MulConst6  = bignumber.NewBig("0xff2ea16466c9838804e327cb417cafcb")
	MulConst7  = bignumber.NewBig("0xfe5dee046a99d51e2cc356c2f617dbe0")
	MulConst8  = bignumber.NewBig("0xfcbe86c7900aecf64236ab31f1f9dcb5")
	MulConst9  = bignumber.NewBig("0xf987a7253ac4d9194200696907cf2e37")
	MulConst10 = bignumber.NewBig("0xf3392b0822b88206f8abe8a3b44dd9be")
	MulConst11 = bignumber.NewBig("0xe7159475a2c578ef4f1d17b2b235d480")
	MulConst12 = bignumber.NewBig("0xd097f3bdfd254ee83bdd3f248e7e785e")
	MulConst13 = bignumber.NewBig("0xa9f746462d8f7dd10e744d913d033333")
	MulConst14 = bignumber.NewBig("0x70d869a156ddd32a39e257bc3f50aa9b")
	MulConst15 = bignumber.NewBig("0x31be135f97da6e09a19dc367e3b6da40")
	MulConst16 = bignumber.NewBig("0x9aa508b5b7e5a9780b0cc4e25d61a56")
	MulConst17 = bignumber.NewBig("0x5d6af8dedbcb3a6ccb7ce618d14225")
	MulConst18 = bignumber.NewBig("0x2216e584f630389b2052b8db590e")
	MulConst19 = bignumber.NewBig("0x48a1703920644d4030024fe")
	MulConst20 = bignumber.NewBig("0x149b34ee7b4532")

	XAuxConst64 = bignumber.NewBig("0x100000000000000000000000000000000")
	XAuxConst32 = bignumber.NewBig("0x10000000000000000")
	XAuxConst16 = bignumber.NewBig("0x100000000")
	XAuxConst8  = bignumber.NewBig("0x10000")
	XAuxConst4  = bignumber.NewBig("0x100")
	XAuxConst2  = bignumber.NewBig("0x10")
	XAuxConst1  = bignumber.NewBig("0x8")

	MulConst49_17 = bignumber.NewBig("499999999999999999")
)

var (
	MaxTick                     = 460540
	MaxSwapIterationCalculation = 50
	OffsetMask                  = big.NewInt(255)
	Kinds                       = big.NewInt(4)
	Mask                        = big.NewInt(15)
	BitMask                     = new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(256), nil), big.NewInt(1))
	WordSize                    = big.NewInt(256)
	One                         = bignumber.BONE
	Unit                        = bignumber.BONE
)
