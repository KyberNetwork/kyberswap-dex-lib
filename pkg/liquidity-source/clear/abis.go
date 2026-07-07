package clear

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	clearSwapABI    abi.ABI
	clearFactoryABI abi.ABI
	clearVaultABI   abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&clearSwapABI, clearSwapABIJson},
		{&clearFactoryABI, clearFactoryABIJson},
		{&clearVaultABI, clearVaultABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
