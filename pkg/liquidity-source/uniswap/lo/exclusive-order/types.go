package uniswaplo

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type OrderInfo struct {
	Reactor                      common.Address
	Swapper                      common.Address
	Nonce                        *big.Int
	Deadline                     *big.Int
	AdditionalValidationContract common.Address
	AdditionalValidationData     []uint8
}

type DutchOutput struct {
	Token       common.Address
	StartAmount *big.Int
	EndAmount   *big.Int
	Recipient   common.Address
}

type ExclusiveDutchOrder struct {
	Info                   OrderInfo
	DecayStartTime         *big.Int
	DecayEndTime           *big.Int
	ExclusiveFiller        common.Address
	ExclusivityOverrideBps *big.Int
	InputToken             common.Address
	InputStartAmount       *big.Int
	InputEndAmount         *big.Int
	Outputs                []DutchOutput
}

// orderTuple
// reference: https://github.com/Uniswap/UniswapX/blob/e451175f285cfc7e11ae253b3b88b87e08cd1d91/src/lib/ExclusiveDutchOrderLib.sol#L8
var orderTuple, _ = abi.NewType("tuple", "", []abi.ArgumentMarshaling{
	{Name: "info", Type: "tuple", Components: []abi.ArgumentMarshaling{
		{Name: "reactor", Type: "address"},
		{Name: "swapper", Type: "address"},
		{Name: "nonce", Type: "uint256"},
		{Name: "deadline", Type: "uint256"},
		{Name: "additionalValidationContract", Type: "address"},
		{Name: "additionalValidationData", Type: "bytes"}},
	},
	{Name: "decayStartTime", Type: "uint256"},
	{Name: "decayEndTime", Type: "uint256"},
	{Name: "exclusiveFiller", Type: "address"},
	{Name: "exclusivityOverrideBps", Type: "uint256"},
	{Name: "inputToken", Type: "address"},
	{Name: "inputStartAmount", Type: "uint256"},
	{Name: "inputEndAmount", Type: "uint256"},
	{Name: "outputs", Type: "tuple[]", Components: []abi.ArgumentMarshaling{
		{Name: "token", Type: "address"},
		{Name: "startAmount", Type: "uint256"},
		{Name: "endAmount", Type: "uint256"},
		{Name: "recipient", Type: "address"},
	}},
})

var orderArguments = abi.Arguments{{Type: orderTuple}}

func DecodeOrder(order []byte) (ExclusiveDutchOrder, error) {
	inputs, err := orderArguments.Unpack(order)
	if err != nil {
		return ExclusiveDutchOrder{}, err
	}

	input := struct {
		Order ExclusiveDutchOrder
	}{}
	if err = orderArguments.Copy(&input, inputs); err != nil {
		return ExclusiveDutchOrder{}, err
	}
	return input.Order, nil
}
