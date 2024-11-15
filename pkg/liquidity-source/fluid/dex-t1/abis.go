package dexT1

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	dexReservesResolverABI abi.ABI
	erc20                  abi.ABI
	storageReadABI         abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&dexReservesResolverABI, dexReservesResolverJSON},
		{&erc20, erc20JSON},
		{&storageReadABI, storageReadJSON},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
