package stabull

import "math/big"

const (
	DexType = "stabull"

	// Pool (Curve) methods
	poolMethodNumeraires     = "numeraires"     // numeraires(uint256) returns token address at index
	poolMethodReserves       = "reserves"       // reserves(uint256) returns reserve token address at index
	poolMethodAssimilator    = "assimilator"    // assimilator(address) returns assimilator address for token
	poolMethodLiquidity      = "liquidity"      // liquidity() returns (total, individual[])
	poolMethodViewCurve      = "viewCurve"      // viewCurve() returns (alpha, beta, delta, epsilon, lambda)
	poolMethodViewOriginSwap = "viewOriginSwap" // viewOriginSwap(origin, target, originAmount) returns targetAmount

	// Assimilator methods
	assimilatorMethodOracle = "oracle" // oracle() returns oracle address

	// Chainlink Oracle methods
	oracleMethodLatestAnswer    = "latestAnswer"    // latestAnswer() returns int256 price
	oracleMethodLatestRoundData = "latestRoundData" // latestRoundData() returns (roundId, answer, startedAt, updatedAt, answeredInRound)

	// Event topic hashes
	newCurveTopic             = "0xe7a19de9e8788cc07c144818f2945144acd6234f790b541aa1010371c8b2a73b"
	parametersSetEventTopic   = "0xb399767364127d5a414f09f214fa5606358052b764894b1084ce5ef067c05a97"
	tradeEventTopic           = "0x887adc1b38cfb756ed025ea6acd9382fbd376ede6c34bc6fa738284b09275468"
	answerUpdatedEventTopic   = "0x0559884fd3a460db3073b7fc896cc77986f16e378210ded43186175bf646fc5f"
	newTransmissionEventTopic = "0xf6a97944f31ea060dfde0566719c0c1d5ac5b3c3e8b4d8e2c7a6c7e1c8f0c3a8"

	reserveZero = "0"

	// Precision for curve parameters
	ONE = 1e18
)

// Factory addresses per chain (from official Stabull documentation)
// https://docs.stabull.finance/amm/contracts
var FactoryAddresses = map[string]string{
	"ethereum": "0x2e9E34b5Af24b66F12721113C1C8FFcbB7Bc8051",
	"polygon":  "0x3c60234db40e6e5b57504e401b1cdc79d91faf89",
	"base":     "0x86Ba17ebf8819f7fd32Cf1A43AbCaAe541A5BEbf",
}

var (
	// Default gas costs for Stabull swaps
	// Stabull uses curve math + oracle checks, higher than simple AMMs
	defaultGas = Gas{
		Swap: 180000, // Approximate gas for originSwap/targetSwap with oracle checks
	}

	// Big number constants
	BigOne = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
)
