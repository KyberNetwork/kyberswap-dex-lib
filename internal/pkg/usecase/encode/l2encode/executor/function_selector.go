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
	FunctionSelectorUniswap        FunctionSelector
	FunctionSelectorKSClassic      FunctionSelector
	FunctionSelectorVelodrome      FunctionSelector
	FunctionSelectorFraxSwap       FunctionSelector
	FunctionSelectorLimitOrder     FunctionSelector
	FunctionSelectorLimitOrderDS   FunctionSelector
	FunctionSelectorSynthetix      FunctionSelector
	FunctionSelectorUniV3KSElastic FunctionSelector
	FunctionSelectorGMX            FunctionSelector
	FunctionSelectorGmxGlp         FunctionSelector
	FunctionSelectorStableSwap     FunctionSelector
	FunctionSelectorCurveSwap      FunctionSelector
	FunctionSelectorBalancerV2     FunctionSelector
	FunctionSelectorDODO           FunctionSelector
	FunctionSelectorCamelotSwap    FunctionSelector
	FunctionSelectorMaverickV1     FunctionSelector
	FunctionSelectorAlgebraV1      FunctionSelector
	FunctionSelectorWombat         FunctionSelector
	FunctionSelectorWooFiV2        FunctionSelector
	FunctionSelectorIZiSwap        FunctionSelector
)

func init() {
	FunctionSelectorUniswap = NewFunctionSelector("executeUniSwap", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorKSClassic = NewFunctionSelector("executeKyberClassic", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorVelodrome = NewFunctionSelector("executeVelodromeSwap", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorFraxSwap = NewFunctionSelector("executeFraxSwap", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorLimitOrder = NewFunctionSelector("executeKyberLimitOrder", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorLimitOrderDS = NewFunctionSelector("executeKyberDSLO", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorSynthetix = NewFunctionSelector("executeSynthetixSwap", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorUniV3KSElastic = NewFunctionSelector("executeUniV3KSElastic", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorGMX = NewFunctionSelector("executeGMXSwap", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorGmxGlp = NewFunctionSelector("executeGMXGLP", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorStableSwap = NewFunctionSelector("executeStableSwap", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorCurveSwap = NewFunctionSelector("executeCurveSwap", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorBalancerV2 = NewFunctionSelector("executeBalV2Swap", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorDODO = NewFunctionSelector("executeDODOSwap", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorCamelotSwap = NewFunctionSelector("executeCamelotSwap", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorMaverickV1 = NewFunctionSelector("executeMaverick", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorAlgebraV1 = NewFunctionSelector("executeAlgebraV1", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorWombat = NewFunctionSelector("executeWombat", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorWooFiV2 = NewFunctionSelector("executeWooFiV2", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
	FunctionSelectorIZiSwap = NewFunctionSelector("executeIziSwap", []string{"uint256", "bytes", "uint256", "address", "bool", "address"})
}
