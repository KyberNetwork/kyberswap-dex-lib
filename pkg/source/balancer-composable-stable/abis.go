package balancercomposablestable

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	vaultABI                abi.ABI
	balancerPoolABI         abi.ABI
	composableStablePoolABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&vaultABI, balancerVaultJson},
		{&balancerPoolABI, balancerPoolJson},
		{&composableStablePoolABI, balancerComposableStablePoolJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
