package brownfiv3

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	brownFiV3FactoryABI    abi.ABI
	brownFiV3PairABI       abi.ABI
	brownFiV3PairConfigABI abi.ABI
	brownFiV3OracleABI     abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&brownFiV3FactoryABI, factoryABIJson},
		{&brownFiV3PairABI, pairABIJson},
		{&brownFiV3PairConfigABI, pairConfigABIJson},
		{&brownFiV3OracleABI, oracleABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
