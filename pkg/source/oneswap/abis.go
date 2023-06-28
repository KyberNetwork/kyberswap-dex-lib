package oneswap

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	oneSwapABI        abi.ABI
	oneSwapFactoryABI abi.ABI
	erc20ABI          abi.ABI
)

func init() {
	build := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&oneSwapABI, oneSwapABIData},
		{&oneSwapFactoryABI, oneSwapFactoryABIData},
		{&erc20ABI, erc20Json},
	}

	for _, b := range build {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
