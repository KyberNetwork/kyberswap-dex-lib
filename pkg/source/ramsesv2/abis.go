package ramsesv2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	ramsesV2PoolABI    abi.ABI
	ramsesV3PoolABI    abi.ABI
	pharaohV3PoolABI   abi.ABI
	ramsesV2FactoryABI abi.ABI
	ramsesV3FactoryABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&ramsesV2PoolABI, ramsesV2PoolJson},
		{&ramsesV3PoolABI, ramsesV3PoolJson},
		{&pharaohV3PoolABI, pharaohV3PoolJson},
		{&ramsesV2FactoryABI, factoryV2Json},
		{&ramsesV3FactoryABI, factoryV3Json},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
