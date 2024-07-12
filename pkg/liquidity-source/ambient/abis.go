package ambient

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	erc20ABI     abi.ABI
	queryABI     abi.ABI
	multicallABI abi.ABI
)

func init() {
	var build = []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&erc20ABI, erc20ABIBytes},
		{&queryABI, queryABIBytes},
		{&multicallABI, multicallABIBytes},
	}

	for _, b := range build {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
