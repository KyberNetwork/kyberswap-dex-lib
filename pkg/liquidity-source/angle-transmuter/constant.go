package angletransmuter

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const (
	DexType = "angle-transmuter"
)

var (
	ErrInvalidToken              = errors.New("invalid token")
	ErrInvalidAmountIn           = errors.New("invalid amount in")
	ErrInsufficientInputAmount   = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrUnsupportedSwap           = errors.New("unsupported swap")
	ErrERC4626DepositMoreThanMax = errors.New("ERC4626: deposit more than max")
	ErrERC4626RedeemMoreThanMax  = errors.New("ERC4626: redeem more than max")
)

var PythArgument = abi.Arguments{
	{Name: "pyth", Type: Address},
	{Name: "feedIds", Type: Bytes32Arr},
	{Name: "stalePeriods", Type: Uint32Arr},
	{Name: "isMultiplied", Type: Uint8Arr},
	{Name: "quoteType", Type: Uint8},
}

var ChainlinkArgument = abi.Arguments{
	{Name: "circuitChainlink", Type: AddressArr},
	{Name: "stalePeriods", Type: Uint32Arr},
	{Name: "circuitChainIsMultiplied", Type: Uint8Arr},
	{Name: "chainlinkDecimals", Type: Uint8Arr},
	{Name: "quoteType", Type: Uint8},
}
