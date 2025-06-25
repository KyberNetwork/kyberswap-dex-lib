package euler

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	poolABI    abi.ABI
	factoryABI abi.ABI
	vaultABI   abi.ABI
	evcABI     abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&poolABI, poolABIJson,
		},
		{
			&factoryABI, factoryABIJson,
		},
		{
			&vaultABI, vaultABIJson,
		},
		{
			&evcABI, evcABIJson,
		},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
