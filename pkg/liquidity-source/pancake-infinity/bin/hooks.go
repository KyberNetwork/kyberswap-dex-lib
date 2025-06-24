package bin

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake-infinity/bin/hooks/brevis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake-infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook interface {
	GetExchange() string
	GetDynamicFee(ctx context.Context, ethrpcClient *ethrpc.Client,
		binPoolManager string, hookAddress common.Address, lpFee uint32) uint32
}

var Hooks = map[common.Address]Hook{}

var _ = RegisterHooks(&BaseHook{valueobject.ExchangePancakeInfinityBinBrevis}, brevis.BinHookAddresses...)

func RegisterHooks(hook Hook, addresses ...common.Address) bool {
	for _, address := range addresses {
		Hooks[address] = hook
	}
	return true
}

func GetHook(hookAddress common.Address) (hook Hook, ok bool) {
	hook, ok = Hooks[hookAddress]
	if hook == nil {
		hook = (*BaseHook)(nil)
	}
	return hook, ok
}

type BaseHook struct{ Exchange valueobject.Exchange }

func (h *BaseHook) GetExchange() string {
	if h != nil {
		return string(h.Exchange)
	}
	return valueobject.ExchangePancakeInfinityBin
}

func (h *BaseHook) GetDynamicFee(_ context.Context, _ *ethrpc.Client, _ string, _ common.Address, _ uint32) uint32 {
	return shared.MAX_FEE_PIPS
}
