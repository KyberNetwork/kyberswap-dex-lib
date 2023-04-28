package abis

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	MetaAggregationRouterV2 abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&MetaAggregationRouterV2, metaAggregationRouterV2},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
