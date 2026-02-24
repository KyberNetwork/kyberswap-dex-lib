package someswapv1

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	PairABI    abi.ABI
	factoryABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&PairABI, pairABIJson,
		},
		{
			&factoryABI, factoryABIJson,
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
