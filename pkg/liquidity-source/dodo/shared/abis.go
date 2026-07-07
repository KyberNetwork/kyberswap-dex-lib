package shared

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	v1PoolABI abi.ABI
	V2PoolABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&v1PoolABI, v1PoolABIJson,
		},
		{
			&V2PoolABI, v2PoolABIJson,
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
