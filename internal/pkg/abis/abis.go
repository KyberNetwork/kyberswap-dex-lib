package abis

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	MetaAggregationRouterV2           abi.ABI
	ERC20                             abi.ABI
	MetaAggregationRouterV2Optimistic abi.ABI
	ScrolL1GasPriceOracle             abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&MetaAggregationRouterV2, metaAggregationRouterV2},
		{&ERC20, erc20},
		{&MetaAggregationRouterV2Optimistic, metaAggregationRouterV2Optimistic},
		{&ScrolL1GasPriceOracle, scrolL1GasPriceOracle},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
