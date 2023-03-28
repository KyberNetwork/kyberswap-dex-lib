package executor

import (
	"github.com/KyberNetwork/router-service/internal/pkg/utils/abi"
)

// FunctionSelector function to execute swap on aggregation executor contract
// light version of go-ethereum/abi/method https://github.com/ethereum/go-ethereum/blob/master/accounts/abi/method.go#L52
type FunctionSelector struct {
	RawName string
	Types   []string
	ID      [4]byte
}

func NewFunctionSelector(rawName string, types []string) FunctionSelector {
	return FunctionSelector{
		RawName: rawName,
		Types:   types,
		ID:      abi.GenMethodID(rawName, types),
	}
}

var (
	FunctionSelectorUniSwap        FunctionSelector
	FunctionSelectorStableSwap     FunctionSelector
	FunctionSelectorCurveSwap      FunctionSelector
	FunctionSelectorUniSwapV3ProMM FunctionSelector
	FunctionSelectorBalancerV2     FunctionSelector
	FunctionSelectorDODO           FunctionSelector
	FunctionSelectorGMX            FunctionSelector
	FunctionSelectorSynthetix      FunctionSelector
	FunctionSelectorPSM            FunctionSelector
	FunctionSelectorWSTETH         FunctionSelector
	FunctionSelectorKyberDMM       FunctionSelector
	FunctionSelectorVelodrome      FunctionSelector
	FunctionSelectorPlatypus       FunctionSelector
	FunctionSelectorFraxSwap       FunctionSelector
	FunctionSelectorCamelotSwap    FunctionSelector
	FunctionSelectorLimitOrder     FunctionSelector
)

func init() {
	FunctionSelectorUniSwap = NewFunctionSelector("executeUniSwap", []string{"uint256", "bytes", "uint256"})
	FunctionSelectorStableSwap = NewFunctionSelector("executeStableSwap", []string{"uint256", "bytes", "uint256"})
	FunctionSelectorCurveSwap = NewFunctionSelector("executeCurveSwap", []string{"uint256", "bytes", "uint256"})
	FunctionSelectorUniSwapV3ProMM = NewFunctionSelector("executeUniV3ProMMSwap", []string{"uint256", "bytes", "uint256"})
	FunctionSelectorBalancerV2 = NewFunctionSelector("executeBalV2Swap", []string{"uint256", "bytes", "uint256"})
	FunctionSelectorDODO = NewFunctionSelector("executeDODOSwap", []string{"uint256", "bytes", "uint256"})
	FunctionSelectorGMX = NewFunctionSelector("executeGMXSwap", []string{"uint256", "bytes", "uint256"})
	FunctionSelectorSynthetix = NewFunctionSelector("executeSynthetixSwap", []string{"uint256", "bytes", "uint256"})
	FunctionSelectorPSM = NewFunctionSelector("executePSMSwap", []string{"uint256", "bytes", "uint256"})
	FunctionSelectorWSTETH = NewFunctionSelector("executeWrappedstETHSwap", []string{"uint256", "bytes", "uint256"})
	FunctionSelectorKyberDMM = NewFunctionSelector("executeKyberDMMSwap", []string{"uint256", "bytes", "uint256"})
	FunctionSelectorVelodrome = NewFunctionSelector("executeVelodromeSwap", []string{"uint256", "bytes", "uint256"})
	FunctionSelectorPlatypus = NewFunctionSelector("executePlatypusSwap", []string{"uint256", "bytes", "uint256"})
	FunctionSelectorFraxSwap = NewFunctionSelector("executeFraxSwap", []string{"uint256", "bytes", "uint256"})
	FunctionSelectorCamelotSwap = NewFunctionSelector("executeCamelotSwap", []string{"uint256", "bytes", "uint256"})
	// Reference from SC
	// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/edd5870ecd990313cb9ab984b7d6a4f16ad6ed9b/contracts/executor-helpers/ExecutorHelper1.sol#L583
	FunctionSelectorLimitOrder = NewFunctionSelector("executeKyberLimitOrder", []string{"uint256", "bytes", "uint256"})
}
