package valueobject

import (
	l1executor "github.com/KyberNetwork/aggregator-encoding/pkg/encode/l1encode/executor"
	l2executor "github.com/KyberNetwork/aggregator-encoding/pkg/encode/l2encode/executor"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	// `canReceiveTokenBeforeSwapFunctionSet` defines functions for pools that can receive token before calling swap
	canReceiveTokenBeforeSwapFunctionSet = map[string]struct{}{
		l1executor.FunctionSelectorUniswap.RawName:        {},
		l1executor.FunctionSelectorKSClassic.RawName:      {},
		l1executor.FunctionSelectorVelodrome.RawName:      {},
		l1executor.FunctionSelectorFraxSwap.RawName:       {},
		l1executor.FunctionSelectorCamelotSwap.RawName:    {},
		l1executor.FunctionSelectorMuteSwitch.RawName:     {},
		l1executor.FunctionSelectorTraderJoeV2.RawName:    {},
		l1executor.FunctionSelectorNomiswapStable.RawName: {},
		l1executor.FunctionSelectorWooFiV2.RawName:        {},
		l1executor.FunctionSelectorMaverickV2.RawName:     {},
		l1executor.FunctionSelectorKTX.RawName:            {},
		l1executor.FunctionSelectorSolidlyV2.RawName:      {},
		l1executor.FunctionSelectorMemebox.RawName:        {},
		l1executor.FunctionSelectorEulerSwap.RawName:      {},
		l1executor.FunctionSelectorBrownfi.RawName:        {},
		l1executor.FunctionSelectorBrownfiV2.RawName:      {},

		l2executor.FunctionSelectorUniswap.RawName:     {},
		l2executor.FunctionSelectorKSClassic.RawName:   {},
		l2executor.FunctionSelectorVelodrome.RawName:   {},
		l2executor.FunctionSelectorFraxSwap.RawName:    {},
		l2executor.FunctionSelectorCamelotSwap.RawName: {},
		l2executor.FunctionSelectorTraderJoeV2.RawName: {},
		l2executor.FunctionSelectorWooFiV2.RawName:     {},
		l2executor.FunctionSelectorMaverickV2.RawName:  {},
		l2executor.FunctionSelectorKTX.RawName:         {},
		l2executor.FunctionSelectorMemebox.RawName:     {},
		l2executor.FunctionSelectorEulerSwap.RawName:   {},
		l2executor.FunctionSelectorBrownFiV2.RawName:   {},

		// GMX and GMX-like exchanges are also able to receive token before calling swap.
		// However, they validate balance before swapping, so it's not possible to execute two gmx swaps consecutively
		// without transferring token back to executor.
		// We disable gmx exchanges here to reduce ad-hoc logic on back end side (do not allow two consecutive gmx swap)
		// l1executor.FunctionSelectorGMX.RawName:        {},
		// l2executor.FunctionSelectorGMX.RawName:        {},
	}

	// `canReceiveTokenBeforeSwapExchangeSet` defines exchanges that can receive token before calling swap.
	canReceiveTokenBeforeSwapExchangeSet = map[Exchange]struct{}{
		valueobject.ExchangeBabyDogeSwap: {},
		valueobject.ExchangeBakerySwap:   {},
	}
)

// CanReceiveTokenBeforeSwap returns true for exchanges that can receive token before calling swap.
func CanReceiveTokenBeforeSwap(exchange Exchange) bool {
	if _, ok := canReceiveTokenBeforeSwapExchangeSet[exchange]; ok {
		return true
	}

	l1Selector, _ := l1executor.GetFunctionSelector(exchange, false)
	if _, ok := canReceiveTokenBeforeSwapFunctionSet[l1Selector.RawName]; ok {
		return true
	}

	l2Selector, _ := l2executor.GetFunctionSelector(exchange, false)
	_, ok := canReceiveTokenBeforeSwapFunctionSet[l2Selector.RawName]
	return ok
}
