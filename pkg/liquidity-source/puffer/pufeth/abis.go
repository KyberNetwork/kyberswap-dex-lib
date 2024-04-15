package pufeth

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	pufferVaultABI abi.ABI
	lidoABI        abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&pufferVaultABI, pufferVaultABIJson},
		{&lidoABI, lidoABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
