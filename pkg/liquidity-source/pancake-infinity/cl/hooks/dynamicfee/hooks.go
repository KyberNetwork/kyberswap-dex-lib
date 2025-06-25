package dynamicfee

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake-infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// CLHookAddresses https://github.com/pancakeswap/pancake-frontend/blob/c9d4f1fb71f122aa7e3d735b6f0e5475953c8d2b/packages/infinity-sdk/src/constants/hooksList/bsc.ts#L6-L25
var CLHookAddresses = []common.Address{
	common.HexToAddress("0x32c59d556b16db81dfc32525efb3cb257f7e493d"),
}

var defaultMaxFee uint32 = 5000 // A maximum fee cap of 5%

type DynamicFeeHook struct {
	Exchange valueobject.Exchange
}

func NewHook(exchange valueobject.Exchange) *DynamicFeeHook {
	return &DynamicFeeHook{Exchange: exchange}
}

func (h *DynamicFeeHook) GetExchange() string {
	return string(h.Exchange)
}

func (h *DynamicFeeHook) GetDynamicFee(ctx context.Context, ethrpcClient *ethrpc.Client,
	clPoolManager string, hookAddress common.Address, lpFee uint32) uint32 {
	if !shared.IsDynamicFee(lpFee) {
		return lpFee
	}

	rpcRequests := ethrpcClient.NewRequest().SetContext(ctx)
	var result struct {
		LpFee uint32 `json:"lpFee"`
	}
	rpcRequests.AddCall(&ethrpc.Call{
		ABI:    shared.CLPoolManagerABI,
		Target: clPoolManager,
		Method: shared.CLPoolManagerMethodGetSlot0,
		Params: []any{hookAddress},
	}, []any{&result})

	_, err := rpcRequests.Aggregate()
	if err != nil {
		return defaultMaxFee
	}

	return result.LpFee
}
