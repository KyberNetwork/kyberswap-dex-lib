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
	FunctionSelectorStableSwap     FunctionSelector
	FunctionSelectorCurveSwap      FunctionSelector
	FunctionSelectorUniV3KSElastic FunctionSelector
	FunctionSelectorBalancerV2     FunctionSelector
	FunctionSelectorDODO           FunctionSelector
	FunctionSelectorGMX            FunctionSelector
	FunctionSelectorSynthetix      FunctionSelector
	FunctionSelectorPSM            FunctionSelector
	FunctionSelectorWSTETH         FunctionSelector
	FunctionSelectorSTETH          FunctionSelector
	FunctionSelectorKSClassic      FunctionSelector
	FunctionSelectorVelodrome      FunctionSelector
	FunctionSelectorPlatypus       FunctionSelector
	FunctionSelectorFraxSwap       FunctionSelector
	FunctionSelectorCamelotSwap    FunctionSelector
	FunctionSelectorMuteSwitch     FunctionSelector
	FunctionSelectorSyncSwap       FunctionSelector
	FunctionSelectorLimitOrder     FunctionSelector
	FunctionSelectorLimitOrderDS   FunctionSelector
	FunctionSelectorMaverickV1     FunctionSelector
	FunctionSelectorAlgebraV1      FunctionSelector
	FunctionSelectorTraderJoeV2    FunctionSelector
	FunctionSelectorKyberPMM       FunctionSelector
	FunctionSelectorIZiSwap        FunctionSelector
)

func init() {
	FunctionSelectorUniswap = NewFunctionSelector("executeUniswap", []string{"bytes", "uint256"})
	FunctionSelectorStableSwap = NewFunctionSelector("executeStableSwap", []string{"bytes", "uint256"})
	FunctionSelectorCurveSwap = NewFunctionSelector("executeCurve", []string{"bytes", "uint256"})
	FunctionSelectorUniV3KSElastic = NewFunctionSelector("executeUniV3KSElastic", []string{"bytes", "uint256"})
	FunctionSelectorBalancerV2 = NewFunctionSelector("executeBalV2", []string{"bytes", "uint256"})
	FunctionSelectorDODO = NewFunctionSelector("executeDODO", []string{"bytes", "uint256"})
	FunctionSelectorGMX = NewFunctionSelector("executeGMX", []string{"bytes", "uint256"})
	FunctionSelectorSynthetix = NewFunctionSelector("executeSynthetix", []string{"bytes", "uint256"})
	FunctionSelectorPSM = NewFunctionSelector("executePSM", []string{"bytes", "uint256"})
	FunctionSelectorWSTETH = NewFunctionSelector("executeWrappedstETH", []string{"bytes", "uint256"})
	FunctionSelectorSTETH = NewFunctionSelector("executeStEth", []string{"bytes", "uint256"})
	FunctionSelectorKSClassic = NewFunctionSelector("executeKSClassic", []string{"bytes", "uint256"})
	FunctionSelectorVelodrome = NewFunctionSelector("executeVelodrome", []string{"bytes", "uint256"})
	FunctionSelectorPlatypus = NewFunctionSelector("executePlatypus", []string{"bytes", "uint256"})
	FunctionSelectorFraxSwap = NewFunctionSelector("executeFrax", []string{"bytes", "uint256"})
	FunctionSelectorCamelotSwap = NewFunctionSelector("executeCamelot", []string{"bytes", "uint256"})

	FunctionSelectorMuteSwitch = NewFunctionSelector("executeMuteSwitchSwap", []string{"bytes", "uint256"})
	FunctionSelectorSyncSwap = NewFunctionSelector("executeSyncSwap", []string{"bytes", "uint256"})
	FunctionSelectorMaverickV1 = NewFunctionSelector("executeMaverick", []string{"bytes", "uint256"})
	FunctionSelectorAlgebraV1 = NewFunctionSelector("executeAlgebraV1", []string{"bytes", "uint256"})
	// Reference from SC
	// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/921725af2a121e023945fa46669c3ea5343ecd37/contracts/executor-helpers/ExecutorHelper2.sol#LL724C1-L724C1
	FunctionSelectorLimitOrder = NewFunctionSelector("executeKyberLimitOrder", []string{"bytes", "uint256"})
	FunctionSelectorLimitOrderDS = NewFunctionSelector("executeKyberDSLO", []string{"bytes", "uint256"})
	FunctionSelectorTraderJoeV2 = NewFunctionSelector("executeTraderJoeV2", []string{"bytes", "uint256"})

	// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/fda542505b49252f6c59273d9ee542377be6c3a9/contracts/executor-helpers/ExecutorHelper2.sol#L375-L411
	FunctionSelectorKyberPMM = NewFunctionSelector("executeRfq", []string{"bytes", "uint256"})

	// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/29d63d83cac3011067bcfd3b9597239745f848d9/contracts/executor-helpers/ExecutorHelper3.sol#L363-L417
	FunctionSelectorIZiSwap = NewFunctionSelector("executeIziSwap", []string{"bytes", "uint256"})
}
