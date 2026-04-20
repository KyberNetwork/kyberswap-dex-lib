package atokenswap

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
)

const (
	DexType = "fluid-atoken-swap"

	defaultGas = 150000
)

var (
	// AEthWETH - Input token
	AEthWETH = common.HexToAddress("0x4d5F47FA6A74757f35C14fD3a6Ef8E3C9BC514E8")

	// Output tokens
	AEthwstETH = common.HexToAddress("0x0B925eD163218f6662a35e0f0371Ac234f9E9371")
	AEthweETH  = common.HexToAddress("0xBdfa7b7893081B35Fb54027489e2Bc7A38275129")

	// Map output token addresses to their contract functions
	tokenFunctions = map[string]struct {
		rateWithPremiumFunc string
		liquidityFunc       string
		maxSwapFunc         string
	}{
		hexutil.Encode(AEthwstETH[:]): {
			rateWithPremiumFunc: "getWstETHRateWithPremium",
			liquidityFunc:       "availableWstETHLiquidity",
			maxSwapFunc:         "maxSwapToWstETH",
		},
		hexutil.Encode(AEthweETH[:]): {
			rateWithPremiumFunc: "getWeETHRateWithPremium",
			liquidityFunc:       "availableWeETHLiquidity",
			maxSwapFunc:         "maxSwapToWeETH",
		},
	}

	// Constants for calculations
	OneEth = uint256.NewInt(1e18)

	// Error definitions
	ErrInvalidAmountIn       = errors.New("invalid amountIn")
	ErrInvalidAmountOut      = errors.New("invalid amount out")
	ErrInvalidToken          = errors.New("invalid token")
	ErrContractPaused        = errors.New("contract is paused")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrExcessiveSwapAmount   = errors.New("excessive swap amount")
)
