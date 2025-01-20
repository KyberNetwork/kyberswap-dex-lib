package integral

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	erc20ABI               abi.ABI
	algebraIntegralPoolABI abi.ABI
	algebraBasePluginV2ABI abi.ABI
	ticklensABI            abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&erc20ABI, erc20Json},
		{&algebraIntegralPoolABI, algebraIntegralPoolJson},
		{&algebraBasePluginV2ABI, algebraBasePluginV2Json},
		{&ticklensABI, ticklenJson},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
