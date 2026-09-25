package gblin

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	vaultABI        abi.ABI
	lensABI         abi.ABI
	aggregatorV3ABI abi.ABI
	multicall3ABI   abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&vaultABI, vaultABIData},
		{&lensABI, lensABIData},
		{&aggregatorV3ABI, aggregatorV3ABIData},
		{&multicall3ABI, multicall3ABIData},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
