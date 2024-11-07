package woofiv2

import (
	"bytes"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

var (
	WooPPV2ABI           abi.ABI
	IntegrationHelperABI abi.ABI
	WooracleV2ABI        abi.ABI
	Erc20ABI             abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&WooPPV2ABI, WooPPV2ABIBytes},
		{&IntegrationHelperABI, IntegrationHelperABIBytes},
		{&WooracleV2ABI, WooracleV2ABIBytes},
		{&Erc20ABI, Erc20ABIBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}
}
