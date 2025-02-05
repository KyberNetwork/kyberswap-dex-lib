package solidlyv2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	poolABI     abi.ABI
	factoryABI  abi.ABI
	memecoreABI abi.ABI
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
			&factoryABI, factoryABIJson,
		},
		{
			&memecoreABI, memecoreABIJson,
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
