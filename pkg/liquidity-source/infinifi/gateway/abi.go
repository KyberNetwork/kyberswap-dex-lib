package gateway

import (
	"bytes"
	_ "embed"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	gatewayABI           abi.ABI
	erc20ABI             abi.ABI
	erc4626ABI           abi.ABI
	lockingControllerABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&gatewayABI, gatewayBytes},
		{&erc20ABI, erc20Bytes},
		{&erc4626ABI, erc4626Bytes},
		{&lockingControllerABI, lockingControllerBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
