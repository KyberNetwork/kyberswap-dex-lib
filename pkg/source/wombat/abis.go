package wombat

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	PoolV2ABI         abi.ABI
	DynamicAssetABI   abi.ABI
	AssetABI          abi.ABI
	CrossChainPoolABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&PoolV2ABI, PoolV2ABIData},
		{&DynamicAssetABI, DynamicAssetABIData},
		{&AssetABI, AssetABIData},
		{&CrossChainPoolABI, CrossChainPoolABIData},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
