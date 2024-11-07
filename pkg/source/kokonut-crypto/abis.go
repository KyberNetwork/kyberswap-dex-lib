package kokonutcrypto

import (
	"bytes"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	erc20ABI           abi.ABI
	poolRegistryABI    abi.ABI
	cryptoSwap2PoolABI abi.ABI
)

func init() {
	var build = []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&erc20ABI, erc20ABIBytes},
		{&poolRegistryABI, poolRegistryABIBytes},
		{&cryptoSwap2PoolABI, cryptoSwap2PoolABIBytes},
	}

	for _, b := range build {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
