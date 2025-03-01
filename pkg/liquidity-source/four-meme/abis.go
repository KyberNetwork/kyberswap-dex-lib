package fourmeme

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
)

var (
	erc20ABI              abi.ABI
	tokenManagerABI       abi.ABI
	tokenManager2ABI      abi.ABI
	tokenManagerHelperABI abi.ABI
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{
			&tokenManagerABI, tokenManagerABIJson,
		},
		{
			&tokenManager2ABI, tokenManager2ABIJson,
		},
		{
			&tokenManagerHelperABI, tokenManagerHelperABIJson,
		},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}

	// https://www.notion.so/kybernetwork/four-meme-reverse-engineer-19d26751887e80119fc9d113074b2365?pvs=4#19d26751887e80488cf8e96d0d41794c

	// method for query templates
	tokenManager2ABI.Methods["_templates"] = abi.Method{
		Inputs: abi.Arguments{
			{
				Name: "",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
		},
		Outputs: abi.Arguments{
			{
				Name: "",
				Type: lo.Must(abi.NewType("address", "", nil)),
			},
			{
				Name: "",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "minTradeFee",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
		},
	}

	// method for query tradingHalted
	tokenManager2ABI.Methods["tradingHalted"] = abi.Method{
		ID:     common.Hex2Bytes("088c5d0b"),
		Inputs: abi.Arguments{},
		Outputs: abi.Arguments{
			{
				Name: "",
				Type: lo.Must(abi.NewType("bool", "", nil)),
			},
		},
	}

	// method for query tokenTxFee
	tokenManager2ABI.Methods["tokenTxFee"] = abi.Method{
		ID: common.Hex2Bytes("9f266331"),
		Inputs: abi.Arguments{
			{
				Name: "",
				Type: lo.Must(abi.NewType("address", "", nil)),
			},
		},
		Outputs: abi.Arguments{
			{
				Name: "tokenTxFee",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
		},
	}

	tokenManager2ABI.Methods["tokenData"] = abi.Method{
		ID: common.Hex2Bytes("e684626b"),
		Inputs: abi.Arguments{
			{
				Name: "",
				Type: lo.Must(abi.NewType("address", "", nil)),
			},
		},
		Outputs: abi.Arguments{
			{
				Name: "token",
				Type: lo.Must(abi.NewType("address", "", nil)),
			},
			{
				Name: "raisedToken",
				Type: lo.Must(abi.NewType("address", "", nil)),
			},
			{
				Name: "templateId",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "field3",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "maxOffers",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "maxFunds",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "LaunchTime",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "Offers",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "Funds",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "Price",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "field10",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "field11",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
			{
				Name: "TradingDisabled",
				Type: lo.Must(abi.NewType("uint256", "", nil)),
			},
		},
	}
}
