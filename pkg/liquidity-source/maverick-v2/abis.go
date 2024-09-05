package maverickv2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	maverickV2FactoryABI abi.ABI
	maverickV2PoolABI    abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&maverickV2FactoryABI, maverickV2FactoryABIJson,
		},
		{
			&maverickV2PoolABI, maverickV2PoolABIJson,
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
