package llamma

import (
	"errors"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "curve-llamma"

	factoryMethodNCollaterals = "n_collaterals"
	factoryMethodCollaterals  = "collaterals"
	factoryMethodAmms         = "amms"

	LlammaMethodA            = "A"
	llammaMethodGetBasePrice = "get_base_price"
	llammaMethodFee          = "fee"
	llammaMethodAdminFee     = "admin_fee"
	llammaMethodAdminFeesX   = "admin_fees_x"
	llammaMethodAdminFeesY   = "admin_fees_y"
	llammaMethodPriceOracle  = "price_oracle"
	llammaMethodActiveBand   = "active_band"
	llammaMethodMinBand      = "min_band"
	llammaMethodMaxBand      = "max_band"
	llammaMethodBandsX       = "bands_x"
	llammaMethodBandsY       = "bands_y"

	maxTicksUnit int64 = 50
	maxTicks     int64 = 50
	maxSkipTicks int64 = 1024
)

var (
	defaultGas = Gas{Exchange: 310000}

	Number_1e36 = big256.TenPowInt(36)

	tenPow18Minus1 = new(uint256.Int).Sub(number.Number_1e18, number.Number_1)
	tenPow18Div4   = new(uint256.Int).Div(number.Number_1e18, number.Number_4)
)

var (
	ErrMulDivOverflow      = errors.New("mul div overflow")
	ErrWrongIndex          = errors.New("wrong index")
	ErrZeroSwapAmount      = errors.New("zero swap amount")
	ErrWadExpOverflow      = errors.New("wad_exp overflow")
	ErrInsufficientBalance = errors.New("insufficient balance")
)
