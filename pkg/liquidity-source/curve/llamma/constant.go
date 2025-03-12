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

	llammaMethodA                   = "A"
	llammaMethodPriceOracleContract = "price_oracle_contract"
	llammaMethodGetBasePrice        = "get_base_price"
	llammaMethodFee                 = "fee"
	llammaMethodAdminFeesX          = "admin_fees_x"
	llammaMethodAdminFeesY          = "admin_fees_y"

	priceOracleMethodPriceW = "price_w"

	curveLlammaHelperMethodGet = "get"

	maxTicksUnit int64 = 50
	maxTicks     int64 = 50
	maxSkipTicks int64 = 1024

	defaultGas int64 = 91000
)

var (
	tenPow18       = big256.TenPowInt(18)
	tenPow36       = big256.TenPowInt(36)
	tenPow18Minus1 = new(uint256.Int).Sub(tenPow18, big256.One)
	tenPow18Div4   = new(uint256.Int).Div(tenPow18, big256.Four)
	i256One        = new(int256.Int).SetInt64(1)
)

var (
	ErrNotEnoughData  = errors.New("not enough data")
	ErrMulDivOverflow = errors.New("mul div overflow")
	ErrWrongIndex     = errors.New("wrong index")
	ErrZeroSwapAmount = errors.New("zero swap amount")
	ErrWadExpOverflow = errors.New("wad_exp overflow")
)
