package llamma

import (
	"errors"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "curve-llamma"

	factoryMethodNCollaterals = "n_collaterals"
	factoryMethodCollaterals  = "collaterals"
	factoryMethodAmms         = "amms"

	llammaMethodA            = "A"
	llammaMethodGetBasePrice = "get_base_price"
	llammaMethodPriceOracle  = "price_oracle"
	llammaMethodActiveBand   = "active_band"
	llammaMethodMinBand      = "min_band"
	llammaMethodMaxBand      = "max_band"
	llammaMethodBandsX       = "bands_x"
	llammaMethodBandsY       = "bands_y"
	llammaMethodFee          = "fee"
	llammaMethodAdminFee     = "admin_fee"
	llammaMethodAdminFeesX   = "admin_fees_x"
	llammaMethodAdminFeesY   = "admin_fees_y"

	curveLlammaHelperMethodGet = "get"

	maxTicksUnit int64 = 50
	maxTicks     int64 = 50
	maxSkipTicks int64 = 1024
)

var (
	defaultGas = int64(1) // TODO:
)

var (
	tenPow18       = big256.TenPowInt(18)
	tenPow18Minus1 = new(uint256.Int).Sub(tenPow18, big256.One)
	tenPow18Div4   = new(uint256.Int).Div(tenPow18, big256.Four)
	tenPow36       = big256.TenPowInt(36)
	u256Fifty      = new(uint256.Int).SetUint64(50)

	i256Zero = new(int256.Int).SetInt64(0)
	i256One  = new(int256.Int).SetInt64(1)
)

var (
	ErrMulDivOverflow = errors.New("mul div overflow")
	ErrWrongIndex     = errors.New("wrong index")
	ErrInvalidPO      = errors.New("invalid po")
	ErrZeroSwapAmount = errors.New("zero swap amount")
	ErrWadExpOverflow = errors.New("wad_exp overflow")
)
