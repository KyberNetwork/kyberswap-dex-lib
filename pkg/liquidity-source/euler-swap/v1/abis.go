package v1

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	poolABI    abi.ABI
	factoryABI abi.ABI
	vaultABI   abi.ABI
	evcABI     abi.ABI
	routerABI  abi.ABI
)

func init() {
	builder := []struct {
		abiVal *abi.ABI
		json   []byte
	}{
		{&poolABI, poolABIJson},
		{&factoryABI, factoryABIJson},
		{&vaultABI, vaultABIJson},
		{&evcABI, evcABIJson},
		{&routerABI, routerABIJson},
	}

	for _, item := range builder {
		var err error
		*item.abiVal, err = abi.JSON(bytes.NewReader(item.json))
		if err != nil {
			panic(err)
		}
	}
}
