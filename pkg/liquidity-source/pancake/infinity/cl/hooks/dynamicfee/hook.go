package dynamicfee

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = cl.RegisterHooksFactory(func(param *cl.HookParam) cl.Hook {
	return &Hook{Hook: cl.NewBaseHook(valueobject.ExchangePancakeInfinityCLDynamic, param)}
},
	common.HexToAddress("0x32C59D556B16DB81DFc32525eFb3CB257f7e493d"),
)

// The hook's afterInitialize sets slot0.lpFee to the pool's baseLpFee, which BaseHook.GetDynamicFee
// returns. Its beforeSwap overrides the fee only on adverse price moves (needs ewVWAP state; not modeled).
type Hook struct {
	cl.Hook
}
