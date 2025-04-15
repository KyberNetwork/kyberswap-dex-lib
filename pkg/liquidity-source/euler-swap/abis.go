package eulerswap

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
)

var (
	poolABI    abi.ABI
	factoryABI abi.ABI
	vaultABI   abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&poolABI, poolABIJson,
		},
		{
			&factoryABI, factoryABIJson,
		},
		{
			&vaultABI, vaultABIJson,
		},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}

	factoryABI.Methods["allPools"] = abi.Method{
		ID: common.Hex2Bytes("041d1ad0"),
		Inputs: abi.Arguments{
			{
				Name: "start",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "end",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
		},
		Outputs: abi.Arguments{
			{
				Name: "",
				Type: lo.Must(abi.NewType("address[]", "", nil)),
			},
		},
	}
}
