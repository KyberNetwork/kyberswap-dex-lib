package elfomofi

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	factoryABI abi.ABI
	helperABI  abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&factoryABI, factoryABIData},
		{&helperABI, helperABIData},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
