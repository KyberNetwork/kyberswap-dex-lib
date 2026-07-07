package cloberob

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook interface {
	GetExchange() string
}

type HookParam struct {
	Cfg         *Config
	HookAddress common.Address
}

type HookFactory func(param *HookParam) Hook

var HookFactories = map[common.Address]HookFactory{}

func RegisterHooks(hook Hook, addresses ...common.Address) bool {
	return RegisterHooksFactory(func(*HookParam) Hook {
		return hook
	}, addresses...)
}

func RegisterHooksFactory(factory HookFactory, addresses ...common.Address) bool {
	for _, address := range addresses {
		HookFactories[address] = factory
	}

	return true
}

func GetHook(hookAddress common.Address, param *HookParam) (hook Hook, ok bool) {
	hookFactory, ok := HookFactories[hookAddress]
	if hookFactory == nil {
		hook = (*BaseHook)(nil)
	} else {
		if param == nil {
			param = &HookParam{}
		}
		param.HookAddress = hookAddress
		hook = hookFactory(param)
	}

	return hook, ok
}

func GetExchangeByHook(hookAddress common.Address) string {
	hookFactory, ok := HookFactories[hookAddress]
	if hookFactory == nil || !ok {
		hook := (*BaseHook)(nil)
		return hook.GetExchange()
	}
	hook := hookFactory(nil)

	return hook.GetExchange()
}

type BaseHook struct{ Exchange valueobject.Exchange }

func (h *BaseHook) GetExchange() string {
	if h != nil {
		return string(h.Exchange)
	}

	return DexType
}
