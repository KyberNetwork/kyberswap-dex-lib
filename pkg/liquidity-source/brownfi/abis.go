package brownfi

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	brownFiV1PairABI    abi.ABI
	brownFiV1FactoryABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&brownFiV1PairABI, pairABIJson,
		},
		{
			&brownFiV1FactoryABI, factoryABIJson,
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
