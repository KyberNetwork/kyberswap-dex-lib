package crowdswapv2

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	crowdswapV2PairABI    abi.ABI
	crowdswapV2FactoryABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&crowdswapV2PairABI, pairABIJson,
		},
		{
			&crowdswapV2FactoryABI, factoryABIJson,
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
