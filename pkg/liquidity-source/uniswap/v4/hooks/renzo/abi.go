package renzo

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	renzoHookABI    abi.ABI
	rateProviderABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&renzoHookABI, renzoHookABIJson},
		{&rateProviderABI, rateProviderABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
