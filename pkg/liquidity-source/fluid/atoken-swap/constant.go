package atokenswap

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexType = "fluid-atoken-swap"

	defaultGas = 150000

	AEthWETH   = "0x4d5f47fa6a74757f35c14fd3a6ef8e3c9bc514e8"
	AEthwstETH = "0x0b925ed163218f6662a35e0f0371ac234f9e9371"
	AEthweETH  = "0xbdfa7b7893081b35fb54027489e2bc7a38275129"
)

var (
	PremiumPrecision = bignumber.TenPowInt(6)

	ErrInvalidAmountIn       = errors.New("invalid amountIn")
	ErrInvalidAmountOut      = errors.New("invalid amount out")
	ErrInvalidToken          = errors.New("invalid token")
	ErrContractPaused        = errors.New("contract is paused")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrExcessiveSwapAmount   = errors.New("excessive swap amount")
)
