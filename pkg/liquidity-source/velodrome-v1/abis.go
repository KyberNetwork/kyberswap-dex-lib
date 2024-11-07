package velodromev1

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	pairABI               abi.ABI
	pairFactoryABI        abi.ABI
	stratumPairFactoryABI abi.ABI
	nuriPairFactoryABI    abi.ABI
	lyvePairFactoryABI    abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&pairABI, pairABIJson,
		},
		{
			&pairFactoryABI, pairFactoryABIJson,
		},
		{
			&stratumPairFactoryABI, stratumPairFactoryABIJson,
		},
		{
			&nuriPairFactoryABI, nuriPairFactoryABIJson,
		},
		{
			&lyvePairFactoryABI, lyvePairFactoryABIJson,
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
