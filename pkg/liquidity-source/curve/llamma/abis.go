package llamma

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	llammaABI  abi.ABI
	factoryABI abi.ABI
)

func init() {
	var build = []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&llammaABI, llammaABIBytes},
		{&factoryABI, factoryABIBytes},
	}

	for _, b := range build {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
