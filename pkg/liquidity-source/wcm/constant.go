package wcm

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexType = "wcm"

	maxOrderBookLevels = 100

	defaultBaseGas = 200000
)

var (
	DefaultGas = Gas{
		Base: defaultBaseGas,
	}

	Zero = bignumber.ZeroBI
	One  = bignumber.One
	Ten  = bignumber.Ten

	FeeDivisor = big.NewInt(100000)

	PricePrecisionMultiplier = bignumber.TenPowInt(18)

	Mask8   = big.NewInt(0xFF)
	Mask16  = big.NewInt(0xFFFF)
	Mask32  = big.NewInt(0xFFFFFFFF)
	Mask64  = new(big.Int).SetUint64(^uint64(0))
	Mask160 = new(big.Int).Sub(new(big.Int).Lsh(One, 160), One)

	VaultTokenIdShift          = 200
	VaultErc20DecimalsShift    = 184
	VaultPositionDecimalsShift = 168

	DepthQty1Shift   = 192
	DepthPrice1Shift = 128
	DepthQty2Shift   = 64

	ConfigTakerFeeShift   = 64
	ConfigToMaxFeeShift   = 120
	ConfigFromMaxFeeShift = 184
)
