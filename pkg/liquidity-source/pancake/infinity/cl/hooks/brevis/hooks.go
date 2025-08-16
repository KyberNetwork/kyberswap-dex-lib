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

var CLHookAddresses = []common.Address{
	common.HexToAddress("0x1a3dfbcac585e22f993cc8e09bcc0db388cc1ca3"),
	common.HexToAddress("0x1e9c64cad39ddd36fb808e004067cffc710eb71d"),
	common.HexToAddress("0xf27b9134b23957d842b08ffa78b07722fb9845bd"),
	common.HexToAddress("0x0fcf6d110cf96be56d251716e69e37619932edf2"),
	common.HexToAddress("0xdfdfb2c5a717ab00b370e883021f20c2fbaed277"),
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
