package feemanager

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = cl.RegisterHooksFactory(func(param *cl.HookParam) cl.Hook {
	return &Hook{Hook: cl.NewBaseHook(valueobject.ExchangePancakeInfinityCLFeeManager, param)}
},
	common.HexToAddress("0x51854e4BAA4A7c9653B52e42021F1B2B35D31651"),
)

type Hook struct {
	cl.Hook
}

// GetDynamicFee reads the hook contract's own lpFees(poolId) mapping, which
// this hook's on-chain beforeSwap uses (via LPFeeLibrary.OVERRIDE_FEE_FLAG)
// as the pool's actual LP fee whenever enableLPFeeOverride is true. If the
// override is disabled, the hook falls back to the last known lpFee instead
// of the on-chain default of 0.
func (h *Hook) GetDynamicFee(ctx context.Context, params *cl.HookParam, lpFee uint32) uint32 {
	var enableLPFeeOverride bool
	var overrideFee uint32

	req := params.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(params.BlockNumber)
	req.AddCall(&ethrpc.Call{
		ABI:    Abi,
		Target: hexutil.Encode(params.HookAddress[:]),
		Method: "enableLPFeeOverride",
	}, []any{&enableLPFeeOverride})
	req.AddCall(&ethrpc.Call{
		ABI:    Abi,
		Target: hexutil.Encode(params.HookAddress[:]),
		Method: "lpFees",
		Params: []any{common.HexToHash(params.Pool.Address)},
	}, []any{&overrideFee})

	if _, err := req.Aggregate(); err != nil {
		return lpFee
	}

	if !enableLPFeeOverride {
		return lpFee
	}

	return overrideFee
}
