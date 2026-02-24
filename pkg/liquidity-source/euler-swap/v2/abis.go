package v2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	poolABI     abi.ABI
	registryABI abi.ABI
	vaultABI    abi.ABI
	evcABI      abi.ABI
	routerABI   abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&poolABI, readABI("EulerSwap.json")},
		{&registryABI, readABI("EulerSwapRegistry.json")},
		{&vaultABI, readABI("EVault.json")},
		{&evcABI, readABI("EthereumVaultConnector.json")},
		{&routerABI, readABI("EulerRouter.json")},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}

func readABI(name string) []byte {
	data, err := abiData.ReadFile("abis/" + name)
	if err != nil {
		panic(err)
	}
	return data
}
