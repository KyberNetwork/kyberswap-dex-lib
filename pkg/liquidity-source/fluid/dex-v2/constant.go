package dexv2

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexType         = "fluid-dex-v2"
	graphFirstLimit = 1000

	TOKENS_DECIMALS_PRECISION              = 9
	LIQUIDITY_EXCHANGE_PRICES_MAPPING_SLOT = 5

	BITS_EXCHANGE_PRICES_SUPPLY_EXCHANGE_PRICE = 91
	BITS_EXCHANGE_PRICES_BORROW_RATIO          = 234
)

var (
	// TODO: Revise this value
	defaultGas = Gas{BaseGas: 109334, CrossInitTickGas: 21492}

	LC_EXCHANGE_PRICES_PRECISION = bignumber.TenPowInt(12)

	X15 = bignumber.NewBig("0x7fff")
	X16 = bignumber.NewBig("0xffff")
	X64 = bignumber.NewBig("0xffffffffffffffff")
)
