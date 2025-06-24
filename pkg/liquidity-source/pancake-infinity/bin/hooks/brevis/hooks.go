package brevis

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake-infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type BrevisHook struct{}

func (h *BrevisHook) GetExchange() string {
	return valueobject.ExchangePancakeInfinityBinBrevis
}

func (h *BrevisHook) GetDynamicFee(ctx context.Context, hookAddress common.Address, ethrpcClient *ethrpc.Client) uint32 {
	hookCaller, err := NewBrevisCaller(hookAddress, ethrpcClient.GetETHClient())
	if err != nil {
		return shared.MAX_FEE_PIPS
	}

	origFee, err := hookCaller.OrigFee(&bind.CallOpts{Context: ctx})
	if err != nil {
		logger.Errorf("failed to get orig fee for hook %s: %v", hookAddress.Hex(), err)
		return shared.MAX_FEE_PIPS
	}

	return uint32(origFee.Uint64())
}
