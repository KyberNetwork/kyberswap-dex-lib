package everlongcollvault

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	rebalancerABI abi.ABI
	swapperABI    abi.ABI
	almABI        abi.ABI
	collVaultABI  abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&rebalancerABI, rebalancerABIJson},
		{&swapperABI, swapperABIJson},
		{&almABI, almABIJson},
		{&collVaultABI, collVaultABIJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
