package cl

import (
	"context"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake-infinity/cl/hooks/brevis"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake-infinity/cl/hooks/dynamicfee"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake-infinity/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook interface {
	GetExchange() string
	GetDynamicFee(ctx context.Context, ethrpcClient *ethrpc.Client,
		clPoolManager string, hookAddress common.Address, lpFee uint32) uint32
	RFQ(context.Context, pool.RFQParams, *PoolMetaInfo, *pool.RFQResult) (any, error)
}

var Hooks = map[common.Address]Hook{}

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

var _ = RegisterHooks(&BaseHook{valueobject.ExchangePancakeInfinityCLBrevis}, brevis.CLHookAddresses...)
var _ = RegisterHooks(&BaseHook{valueobject.ExchangePancakeInfinityCLDynamicFee}, dynamicfee.CLHookAddresses...)

type BaseHook struct{ Exchange valueobject.Exchange }

func (h *BaseHook) GetExchange() string {
	if h != nil {
		return string(h.Exchange)
	}
	return valueobject.ExchangePancakeInfinityCL
}

func (*BaseHook) RFQ(context.Context, pool.RFQParams, *PoolMetaInfo, *pool.RFQResult) (any, error) {
	return nil, nil
}

func (h *BaseHook) GetDynamicFee(_ context.Context, _ *ethrpc.Client, _ string, _ common.Address, _ uint32) uint32 {
	return shared.MAX_FEE_PIPS
}
