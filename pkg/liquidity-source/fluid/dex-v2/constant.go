package dexv2

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexType         = "fluid-dex-v2"
	graphFirstLimit = 1000
	tickChunkSize   = 100

	MIN_TICK = -524287
	MAX_TICK = -MIN_TICK

	TOKENS_DECIMALS_PRECISION = 9

	DEX_V2_TICK_LIQUIDITY_GROSS_MAPPING_SLOT = 3 // liquidityGross
	DEX_V2_TICK_DATA_MAPPING_SLOT            = 4 // liquidityNet
	LIQUIDITY_EXCHANGE_PRICES_MAPPING_SLOT   = 5

	BITS_EXCHANGE_PRICES_FEE                   = 16
	BITS_EXCHANGE_PRICES_UTILIZATION           = 30
	BITS_EXCHANGE_PRICES_LAST_TIMESTAMP        = 58
	BITS_EXCHANGE_PRICES_SUPPLY_EXCHANGE_PRICE = 91
	BITS_EXCHANGE_PRICES_SUPPLY_RATIO          = 219
	BITS_EXCHANGE_PRICES_BORROW_RATIO          = 234
)

var (
	defaultGas = Gas{BaseGas: 155000, CrossInitTickGas: 21492}

	SECONDS_PER_YEAR = big.NewInt(365 * 24 * 60 * 60)

	FOUR_DECIMALS                = bignumber.TenPowInt(4)
	LC_EXCHANGE_PRICES_PRECISION = bignumber.TenPowInt(12)
	TenPow27                     = bignumber.TenPowInt(27)
	TenPow54                     = bignumber.TenPowInt(54)

	two255 = new(big.Int).Lsh(bignumber.One, 255)
	two256 = new(big.Int).Lsh(bignumber.One, 256)

	X14  = bignumber.NewBig("0x3fff")
	X15  = bignumber.NewBig("0x7fff")
	X16  = bignumber.NewBig("0xffff")
	X33  = bignumber.NewBig("0x1ffffffff")
	X64  = bignumber.NewBig("0xffffffffffffffff")
	X86  = bignumber.NewBig("0x3fffffffffffffffffffff")
	X128 = bignumber.NewBig("0x00ffffffffffffffffffffffffffffffffffffffff")

	Q96 = new(big.Int).Lsh(bignumber.One, 96)
)
