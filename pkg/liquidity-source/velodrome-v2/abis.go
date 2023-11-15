package velodromev2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	poolABI        abi.ABI
	poolFactoryABI abi.ABI
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
			&poolFactoryABI, poolFactoryABIJson,
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
