package integral

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	erc20ABI               abi.ABI
	algebraIntegralPoolABI abi.ABI
	algebraBasePluginV1ABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&erc20ABI, erc20Json},
		{&algebraIntegralPoolABI, algebraIntegralPoolJson},
		{&algebraBasePluginV1ABI, algebraBasePluginV1Json},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
