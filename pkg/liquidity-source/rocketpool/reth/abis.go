package reth

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	RocketDAOProtocolSettingsDepositABI abi.ABI
	RocketDepositPoolABI                abi.ABI
	RocketMinipoolQueueABI              abi.ABI
	RocketNetworkBalancesABI            abi.ABI
	RocketTokenRETHABI                  abi.ABI
	RocketVaultABI                      abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&RocketDAOProtocolSettingsDepositABI, rocketDAOProtocolSettingsDepositABIJson},
		{&RocketDepositPoolABI, rocketDepositPoolABIJSON},
		{&RocketMinipoolQueueABI, rocketMinipoolQueueABIJSON},
		{&RocketNetworkBalancesABI, rocketNetworkBalancesABIJSON},
		{&RocketTokenRETHABI, rocketTokenRETHABIJSON},
		{&RocketVaultABI, rocketVaultABIJSON},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
