package bancorv3

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	bancorNetworkABI  abi.ABI
	poolCollectionABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&bancorNetworkABI, bancorNetworkJSON},
		{&poolCollectionABI, poolCollectionJSON},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
