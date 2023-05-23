package saddle

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	swapFlashLoanABI abi.ABI
	erc20ABI         abi.ABI
)

func init() {
	build := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&swapFlashLoanABI, swapFlashLoanData},
		{&erc20ABI, erc20Data},
	}

	for _, b := range build {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
