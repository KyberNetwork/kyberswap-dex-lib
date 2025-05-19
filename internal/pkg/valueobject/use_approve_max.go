package valueobject

import (
	l1executor "github.com/KyberNetwork/aggregator-encoding/pkg/encode/l1encode/executor"
)

// `useApproveMaxFunctionSet` defines set of functions that we should track if executor `approveMax` for the pools,
// to support optimizing gas cost when encode (not all pools need to call `approveMax` when swap with executor).
// This data can be added by check the SC code, if the function of each pool type reads the SHOULD_APPROVE_MAX flag.
// If value is true, use the pool address as the approval address.
// If value is false, use a custom approval address.
var useApproveMaxFunctionSet = map[string]bool{
	// ExecutorHelper1
	l1executor.FunctionSelectorLimitOrder.RawName: false,

	// ExecutorHelper2
	l1executor.FunctionSelectorStableSwap.RawName:        true,
	l1executor.FunctionSelectorCurveSwap.RawName:         true,
	l1executor.FunctionSelectorPancakeStableSwap.RawName: true,
	l1executor.FunctionSelectorBalancerV2.RawName:        false,
	l1executor.FunctionSelectorDODO.RawName:              true,
	l1executor.FunctionSelectorWSTETH.RawName:            true,
	l1executor.FunctionSelectorPlatypus.RawName:          true,
	l1executor.FunctionSelectorPSM.RawName:               true,

	// ExecutorHelper3
	l1executor.FunctionSelectorMantisSwap.RawName: true,
	l1executor.FunctionSelectorWombat.RawName:     true,
	l1executor.FunctionSelectorKyberPMM.RawName:   false,
	l1executor.FunctionSelectorBancorV3.RawName:   true,
	l1executor.FunctionSelectorAmbient.RawName:    true,

	// ExecutorHelper4
	l1executor.FunctionSelectorVooi.RawName:          true,
	l1executor.FunctionSelectorVelocoreV2.RawName:    false,
	l1executor.FunctionSelectorMaticMigrate.RawName:  true,
	l1executor.FunctionSelectorSmardex.RawName:       true,
	l1executor.FunctionSelectorKokonutCrypto.RawName: true,
	l1executor.FunctionSelectorBalancerV1.RawName:    true,
	l1executor.FunctionSelectorDexalot.RawName:       false,
	l1executor.FunctionSelectorBancorV21.RawName:     true,

	// ExecutorHelper5
	l1executor.FunctionSelectorNative.RawName:          false,
	l1executor.FunctionSelectorUniswapV1.RawName:       false,
	l1executor.FunctionSelectorEtherFiWeETH.RawName:    true,
	l1executor.FunctionSelectorKelp.RawName:            true,
	l1executor.FunctionSelectorEthenaSusde.RawName:     true,
	l1executor.FunctionSelectorRocketPool.RawName:      false,
	l1executor.FunctionSelectorMakerSavingsDai.RawName: false,
	l1executor.FunctionSelectorEtherfiVampire.RawName:  true,

	// ExecutorHelper6
	l1executor.FunctionSelectorRenzo.RawName:            true,
	l1executor.FunctionSelectorWBETH.RawName:            true,
	l1executor.FunctionSelectorSfrxETHConvertor.RawName: true,
	l1executor.FunctionSelectorHashflow.RawName:         false,
	l1executor.FunctionSelectorPufferFinance.RawName:    true,
	l1executor.FunctionSelectorIntegral.RawName:         false,
	l1executor.FunctionSelectorUsd0PP.RawName:           true,

	// ExecutorHelper7
	l1executor.FunctionSelectorEtherVista.RawName:   false,
	l1executor.FunctionSelectorFluidVaultT1.RawName: false,
	l1executor.FunctionSelectorLitePSM.RawName:      true,
	l1executor.FunctionSelectorMkrSky.RawName:       true,
	l1executor.FunctionSelectorDaiUsds.RawName:      true,
	l1executor.FunctionSelectorLO1inch.RawName:      false,
	l1executor.FunctionSelectorFluidDex.RawName:     false,
	l1executor.FunctionSelectorOndoUSDY.RawName:     true,
	l1executor.FunctionSelectorRingSwap.RawName:     false,

	// ExecutorHelper8
	l1executor.FunctionSelectorLimitOrderDS.RawName: false,
	l1executor.FunctionSelectorEtherFieBTC.RawName:  true,
	l1executor.FunctionSelectorUniswapV4.RawName:    false, // Deprecated

	// ExecutorHelper9
	l1executor.FunctionSelectorSavingsUSDS.RawName:   false,
	l1executor.FunctionSelectorHoney.RawName:         true,
	l1executor.FunctionSelectorOvernightUsdp.RawName: true,
	l1executor.FunctionSelectorSwaapV2.RawName:       false,
	l1executor.FunctionSelectorPanda.RawName:         true,
	l1executor.FunctionSelectorGeneric.RawName:       false,

	l1executor.FunctionSelectorBalancerV3Batch.RawName: false,
	l1executor.FunctionSelectorCurveLlamma.RawName:     true,
	l1executor.FunctionSelectorHyETH.RawName:           false,

	// ExecutorHelper10
	l1executor.FunctionSelectorBeefySonic.RawName: true,
}

// IsApproveMaxExchange returns true if we should track if executor `approveMax` for the provided exchange,
// to support optimizing gas cost when encoding (not all pools need to call `approveMax` when swap with executor).
// This data can be added by check the SC code, if the function of each pool type reads the SHOULD_APPROVE_MAX flag.
// Only L1 chains check SHOULD_APPROVE_MAX flag.
func IsApproveMaxExchange(exchange Exchange) (bool, bool) {
	l1Selector, _ := l1executor.GetFunctionSelector(exchange, false)
	usePoolAsApprovalAddress, ok := useApproveMaxFunctionSet[l1Selector.RawName]
	return ok, usePoolAsApprovalAddress
}
