package infinitypools

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	erc20ABI        abi.ABI
	infinityPoolABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&erc20ABI, erc20ABIJson},
		{&infinityPoolABI, infinityPoolABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
