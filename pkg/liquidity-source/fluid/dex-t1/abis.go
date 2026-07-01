package dexT1

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	DexReservesResolverABI abi.ABI
	erc20                  abi.ABI
	StorageReadABI         abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&DexReservesResolverABI, dexReservesResolverJSON},
		{&erc20, erc20JSON},
		{&StorageReadABI, storageReadJSON},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
