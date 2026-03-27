package hiddenocean

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	poolABI     abi.ABI
	registryABI abi.ABI
	erc20ABI    abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&poolABI, poolABIBytes},
		{&registryABI, registryABIBytes},
		{&erc20ABI, erc20ABIBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
