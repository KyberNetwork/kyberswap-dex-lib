package brevis

import (
	"context"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = cl.RegisterHooksFactory(func(param *cl.HookParam) cl.Hook {
	return &Hook{Hook: cl.NewBaseHook(valueobject.ExchangePancakeInfinityCLBrevis, param)}
},
	common.HexToAddress("0x1A3DFBCAc585e22F993Cc8e09BcC0dB388Cc1Ca3"),
	common.HexToAddress("0x1e9c64Cad39DDD36fB808E004067Cffc710EB71D"),
	common.HexToAddress("0xF27b9134B23957D842b08fFa78b07722fB9845BD"),
	common.HexToAddress("0x60FbCAfaB24bc117b6facECd00D3e8f56ca4D5e9"),
	common.HexToAddress("0x0fcF6D110Cf96BE56D251716E69E37619932edF2"),
	common.HexToAddress("0xDfdfB2c5a717AB00B370E883021f20C2fbaEd277"),
)

type Hook struct {
	cl.Hook
}

func (h *Hook) GetDynamicFee(ctx context.Context, params *cl.HookParam, _ uint32) uint32 {
	hookCaller, err := NewBrevisCaller(params.HookAddress, params.RpcClient.GetETHClient())
	if err != nil {
		return shared.MAX_FEE_PIPS
	}

	origFee, err := hookCaller.OrigFee(&bind.CallOpts{Context: ctx})
	if err != nil {
		logger.Errorf("failed to get orig fee for hook %s: %v", params.HookAddress.Hex(), err)
		return shared.MAX_FEE_PIPS
	}

	return uint32(origFee.Uint64())
}
