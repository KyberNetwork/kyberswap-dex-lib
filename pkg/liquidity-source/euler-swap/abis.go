package eulerswap

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	poolABI    abi.ABI
	factoryABI abi.ABI
	vaultABI   abi.ABI
	erc20ABI   abi.ABI
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
			&erc20ABI, erc20ABIJson,
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
