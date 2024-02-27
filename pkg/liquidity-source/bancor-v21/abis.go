package bancor_v21

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	converterRegistryABI abi.ABI
	converterABI         abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&converterRegistryABI, converterRegistryJson},
		{&converterABI, converterJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
