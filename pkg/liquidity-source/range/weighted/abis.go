package weighted

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	rangePoolABI   abi.ABI
	rangeVaultABI  abi.ABI
	rangeRouterABI abi.ABI
)

func init() {
	for _, b := range []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&rangePoolABI, rangePoolJson},
		{&rangeVaultABI, rangeVaultJson},
		{&rangeRouterABI, rangeRouterJson},
	} {
		parsed, err := abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
		*b.ABI = parsed
	}
}
