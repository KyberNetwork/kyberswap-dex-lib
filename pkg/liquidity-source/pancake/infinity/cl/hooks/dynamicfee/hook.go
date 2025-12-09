package dynamicfee

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var defaultMaxFee uint32 = 5000 // A maximum fee cap of 5%

var _ = cl.RegisterHooksFactory(func(param *cl.HookParam) cl.Hook {
	return &Hook{Hook: cl.NewBaseHook(valueobject.ExchangePancakeInfinityCLDynamic, param)}
},
	common.HexToAddress("0x32C59D556B16DB81DFc32525eFb3CB257f7e493d"),
)

type Hook struct {
	cl.Hook
}

func (h *Hook) GetDynamicFee(ctx context.Context, params *cl.HookParam, lpFee uint32) uint32 {
	if !shared.IsDynamicFee(lpFee) {
		return lpFee
	}

	rpcRequests := params.RpcClient.NewRequest().SetContext(ctx)
	var result struct {
		LpFee uint32 `json:"lpFee"`
	}
	rpcRequests.AddCall(&ethrpc.Call{
		ABI:    shared.CLPoolManagerABI,
		Target: params.Cfg.CLPoolManagerAddress,
		Method: shared.CLPoolManagerMethodGetSlot0,
		Params: []any{params.HookAddress},
	}, []any{&result})

	_, err := rpcRequests.Aggregate()
	if err != nil {
		return defaultMaxFee
	}

	return result.LpFee
}
