package aevmclient

import aevmclient "github.com/KyberNetwork/aevm/client"

type IAEVMClientUseCase interface {
	aevmclient.Client

	ApplyConfig(config Config)
	Close()
}
