package dodo

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	v1PoolABI abi.ABI
	v2PoolABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&v1PoolABI, v1PoolData,
		},
		{
			&v2PoolABI, v2PoolData,
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
