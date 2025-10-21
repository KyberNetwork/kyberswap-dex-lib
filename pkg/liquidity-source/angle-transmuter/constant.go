package angletransmuter

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType        = "angle-transmuter"
	defaultReserve = "100000000000000000000000000"
)

var oracleTypeMapping = map[valueobject.Exchange]map[uint8]OracleReadType{
	valueobject.ExchangeParallelParallelizer: {
		0: CHAINLINK_FEEDS,
		1: EXTERNAL,
		2: NO_ORACLE,
		3: STABLE,
		4: WSTETH,
		5: CBETH,
		6: RETH,
		7: SFRXETH,
		8: MAX,
		9: MORPHO_ORACLE,
	},
}

func convertOracleType(exchange valueobject.Exchange, e uint8) OracleReadType {
	if mapping, ok := oracleTypeMapping[exchange]; ok {
		if oracleType, exists := mapping[e]; exists {
			return oracleType
		}
	}

	return OracleReadType(e)
}

var (
	ErrInvalidToken            = errors.New("invalid token")
	ErrInvalidAmountIn         = errors.New("invalid amount in")
	ErrInsufficientInputAmount = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInvalidOracle           = errors.New("invalid oracle compared to oracle type")
	ErrUnimplemented           = errors.New("unimplemented")
	ErrInvalidSwap             = errors.New("invalid swap")
	ErrMulOverflow             = errors.New("MUL_OVERFLOW")
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

var HyperparametersArgument = abi.Arguments{
	{Name: "userDeviation", Type: Int128},
	{Name: "burnRatioDeviation", Type: Int128},
}

var MaxArgument = abi.Arguments{
	{Name: "maxValue", Type: Uint256},
}

var MorphoArgument = abi.Arguments{
	{Name: "oracle", Type: Address},
	{Name: "normalizationFactor", Type: Uint256},
}
