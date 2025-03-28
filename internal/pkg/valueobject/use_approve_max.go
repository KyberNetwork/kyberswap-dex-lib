package valueobject

import (
	l1executor "github.com/KyberNetwork/aggregator-encoding/pkg/encode/l1encode/executor"
	l2executor "github.com/KyberNetwork/aggregator-encoding/pkg/encode/l2encode/executor"
)

// `useApproveMaxFunctionSet` defines set of functions that we should track if executor `approveMax` for the pools,
// to support optimizing gas cost when encode (not all pools need to call `approveMax` when swap with executor).
// This data can be added by check the SC code, if the function of each pool type reads the SHOULD_APPROVE_MAX flag.
var useApproveMaxFunctionSet = map[string]struct{}{
	l1executor.FunctionSelectorStableSwap.RawName:        {},
	l1executor.FunctionSelectorCurveSwap.RawName:         {},
	l1executor.FunctionSelectorKokonutCrypto.RawName:     {},
	l1executor.FunctionSelectorPancakeStableSwap.RawName: {},
	l1executor.FunctionSelectorBalancerV1.RawName:        {},
	l1executor.FunctionSelectorBalancerV2.RawName:        {},
	l1executor.FunctionSelectorBalancerV3.RawName:        {},
	l1executor.FunctionSelectorDODO.RawName:              {},
	l1executor.FunctionSelectorPSM.RawName:               {},
	l1executor.FunctionSelectorWSTETH.RawName:            {},
	l1executor.FunctionSelectorPlatypus.RawName:          {},
	l1executor.FunctionSelectorWombat.RawName:            {},
	l1executor.FunctionSelectorMantisSwap.RawName:        {},
	l1executor.FunctionSelectorLimitOrderDS.RawName:      {},
	l1executor.FunctionSelectorKyberPMM.RawName:          {},
	l1executor.FunctionSelectorVooi.RawName:              {},
	l1executor.FunctionSelectorMaticMigrate.RawName:      {},
	l1executor.FunctionSelectorSmardex.RawName:           {},
	l1executor.FunctionSelectorIntegral.RawName:          {},
	l1executor.FunctionSelectorVelocoreV2.RawName:        {},
	l1executor.FunctionSelectorSwaapV2.RawName:           {},
	l1executor.FunctionSelectorBancorV3.RawName:          {},
	l1executor.FunctionSelectorEtherFiWeETH.RawName:      {},
	l1executor.FunctionSelectorKelp.RawName:              {},
	l1executor.FunctionSelectorRocketPool.RawName:        {},
	l1executor.FunctionSelectorEthenaSusde.RawName:       {},
	l1executor.FunctionSelectorMakerSavingsDai.RawName:   {},
	l1executor.FunctionSelectorHashflow.RawName:          {},
	l1executor.FunctionSelectorNative.RawName:            {},
	l1executor.FunctionSelectorRenzo.RawName:             {},
	l1executor.FunctionSelectorPufferFinance.RawName:     {},
	l1executor.FunctionSelectorAmbient.RawName:           {},
	l1executor.FunctionSelectorEtherVista.RawName:        {},
	l1executor.FunctionSelectorLitePSM.RawName:           {},
	l1executor.FunctionSelectorMkrSky.RawName:            {},
	l1executor.FunctionSelectorDaiUsds.RawName:           {},
	l1executor.FunctionSelectorFluidVaultT1.RawName:      {},
	l1executor.FunctionSelectorFluidDex.RawName:          {},
	l1executor.FunctionSelectorUsd0PP.RawName:            {},
	l1executor.FunctionSelectorBebop.RawName:             {},
	l1executor.FunctionSelectorWBETH.RawName:             {},
	l1executor.FunctionSelectorUniswapV1.RawName:         {},
	l1executor.FunctionSelectorOndoUSDY.RawName:          {},
	l1executor.FunctionSelectorDexalot.RawName:           {},
	l1executor.FunctionSelectorRingSwap.RawName:          {},
	l1executor.FunctionSelectorSfrxETHConvertor.RawName:  {},
	l1executor.FunctionSelectorEtherfiVampire.RawName:    {},
	l1executor.FunctionSelectorLO1inch.RawName:           {},
	l1executor.FunctionSelectorBeetsSS.RawName:           {},
	l1executor.FunctionSelectorEtherFieBTC.RawName:       {},
	l1executor.FunctionSelectorUniswapV4.RawName:         {},
	l1executor.FunctionSelectorOvernightUsdp.RawName:     {},
	l1executor.FunctionSelectorSavingsUSDS.RawName:       {},
	l1executor.FunctionSelectorPanda.RawName:             {},
	l1executor.FunctionSelectorHoney.RawName:             {},
	l1executor.FunctionSelectorCurveLlamma.RawName:       {},

	l2executor.FunctionSelectorLimitOrderDS.RawName:      {},
	l2executor.FunctionSelectorStableSwap.RawName:        {},
	l2executor.FunctionSelectorCurveSwap.RawName:         {},
	l2executor.FunctionSelectorPancakeStableSwap.RawName: {},
	l2executor.FunctionSelectorBalancerV2.RawName:        {},
	l2executor.FunctionSelectorBalancerV3.RawName:        {},
	l2executor.FunctionSelectorDODO.RawName:              {},
	l2executor.FunctionSelectorWombat.RawName:            {},
	l2executor.FunctionSelectorSmardex.RawName:           {},
	l2executor.FunctionSelectorIntegral.RawName:          {},
	l2executor.FunctionSelectorEtherVista.RawName:        {},
	l2executor.FunctionSelectorSwaapV2.RawName:           {},
	l2executor.FunctionSelectorHashflow.RawName:          {},
	l2executor.FunctionSelectorNative.RawName:            {},
	l2executor.FunctionSelectorFluidDex.RawName:          {},
	l2executor.FunctionSelectorBebop.RawName:             {},
	l2executor.FunctionSelectorRingSwap.RawName:          {},
	l2executor.FunctionSelectorDexalot.RawName:           {},
	l2executor.FunctionSelectorLO1inch.RawName:           {},
	l2executor.FunctionSelectorVirtualFun.RawName:        {},
	l2executor.FunctionSelectorEtherFieBTC.RawName:       {},
	l2executor.FunctionSelectorUniswapV4.RawName:         {},
	l2executor.FunctionSelectorOvernightUsdp.RawName:     {},
	l2executor.FunctionSelectorSkyPSM.RawName:            {},
	l2executor.FunctionSelectorCurveLlamma.RawName:       {},
}

// IsApproveMaxExchange returns true if we should track if executor `approveMax` for the provided exchange,
// to support optimizing gas cost when encoding (not all pools need to call `approveMax` when swap with executor).
// This data can be added by check the SC code, if the function of each pool type reads the SHOULD_APPROVE_MAX flag.
func IsApproveMaxExchange(exchange Exchange) bool {
	l1Selector, _ := l1executor.GetFunctionSelector(exchange, false)
	if _, ok := useApproveMaxFunctionSet[l1Selector.RawName]; ok {
		return true
	}

	l2Selector, _ := l2executor.GetFunctionSelector(exchange, false)
	_, ok := useApproveMaxFunctionSet[l2Selector.RawName]
	return ok
}
