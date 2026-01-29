package someswapv2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var factoryABI abi.ABI

func FactoryABI() abi.ABI {
	return factoryABI
}

var poolABI abi.ABI

func PoolABI() abi.ABI {
	return poolABI
}

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&factoryABI, factoryABIJson},
		{&poolABI, poolABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}

