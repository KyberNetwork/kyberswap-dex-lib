package brevis

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var BinHookAddresses = []common.Address{
	common.HexToAddress("0x60fbcafab24bc117b6facecd00d3e8f56ca4d5e9"),
}

type BrevisHook struct {
	Exchange valueobject.Exchange
}

func NewHook(exchange valueobject.Exchange) *BrevisHook {
	return &BrevisHook{Exchange: exchange}
}

func (h *BrevisHook) GetExchange() string {
	return string(h.Exchange)
}

func (h *BrevisHook) GetDynamicFee(ctx context.Context, ethrpcClient *ethrpc.Client,
	_ string, hookAddress common.Address, _ uint32) uint32 {
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
