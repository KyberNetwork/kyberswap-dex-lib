package iziswap

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var iZiSwapPoolABI abi.ABI
var erc20ABI abi.ABI

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&iZiSwapPoolABI, iZiSwapPoolJson},
		{&erc20ABI, erc20Json},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
