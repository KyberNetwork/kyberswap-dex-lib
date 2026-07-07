package brownfiv2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	brownFiV2FactoryABI abi.ABI
	brownFiV2PairABI    abi.ABI
	brownFiV2OracleABI  abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&brownFiV2FactoryABI, factoryABIJson,
		},
		{
			&brownFiV2PairABI, pairABIJson,
		},
		{
			&brownFiV2OracleABI, oracleABIJson,
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
