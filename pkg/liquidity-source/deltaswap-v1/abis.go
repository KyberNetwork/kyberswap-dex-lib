package deltaswapv1

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	deltaSwapV1FactoryABI abi.ABI
	deltaSwapV1PairABI    abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&deltaSwapV1FactoryABI, deltaSwapV1FactoryABIJson,
		},
		{
			&deltaSwapV1PairABI, DeltaSwapV1PairABIJson,
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
