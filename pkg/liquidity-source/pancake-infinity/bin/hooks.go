package bin

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook interface {
	GetExchange() string
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

type BaseHook struct{ Exchange valueobject.Exchange }

func (h *BaseHook) GetExchange() string {
	if h != nil {
		return string(h.Exchange)
	}
	return DexType
}
