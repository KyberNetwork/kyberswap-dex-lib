package dynamicfee

import (
	"context"
	"math/big"

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

type DynamicFeeHook struct{}

func (h *DynamicFeeHook) GetExchange() string {
	return valueobject.ExchangePancakeInfinityCLDynamicFee
}

func (h *DynamicFeeHook) GetDynamicFee(ctx context.Context, poolManager, hookAddress string, ethrpcClient *ethrpc.Client, lpFee *big.Int) uint32 {
	if lpFee != nil && !shared.IsDynamicFee(uint32(lpFee.Uint64())) {
		return uint32(lpFee.Uint64())
	}

	rpcRequests := ethrpcClient.NewRequest().SetContext(ctx)
	var result struct {
		SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
		Tick         *big.Int `json:"tick"`
		ProtocolFee  *big.Int `json:"protocolFee"`
		LpFee        *big.Int `json:"lpFee"`
	}
	rpcRequests.AddCall(&ethrpc.Call{
		ABI:    shared.CLPoolManagerABI,
		Target: poolManager,
		Method: shared.CLPoolManagerMethodGetSlot0,
		Params: []any{common.HexToAddress(hookAddress)},
	}, []any{&result})

	_, err := rpcRequests.Aggregate()
	if err != nil {
		return defaultMaxFee
	}

	return uint32(result.LpFee.Uint64())
}
