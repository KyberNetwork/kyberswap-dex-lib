package stabull

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "stabull"

	// Pool (Curve) methods
	poolMethodNumeraires     = "numeraires"     // numeraires(uint256) returns token address at index
	poolMethodAssimilator    = "assimilator"    // assimilator(address) returns assimilator address for token
	poolMethodLiquidity      = "liquidity"      // liquidity() returns (total, individual[])
	poolMethodViewCurve      = "viewCurve"      // viewCurve() returns (alpha, beta, delta, epsilon, lambda)
	poolMethodViewOriginSwap = "viewOriginSwap" // viewOriginSwap(origin, target, originAmount) returns targetAmount

	// Assimilator methods
	assimilatorMethodOracle = "oracle" // oracle() returns oracle address

	// Chainlink Oracle methods
	oracleMethodLatestAnswer    = "latestAnswer"    // latestAnswer() returns int256 price
	oracleMethodLatestRoundData = "latestRoundData" // latestRoundData() returns (roundId, answer, startedAt, updatedAt, answeredInRound)

	// Default gas costs for Stabull swaps
	// Stabull uses curve math + oracle checks, higher than simple AMMs
	defaultGas = 180000 // Approximate gas for originSwap/targetSwap with oracle checks
)

var (
	// Event topic hashes
	tradeEventTopic           = common.HexToHash("0x887adc1b38cfb756ed025ea6acd9382fbd376ede6c34bc6fa738284b09275468")
	parametersSetEventTopic   = common.HexToHash("0xb399767364127d5a414f09f214fa5606358052b764894b1084ce5ef067c05a97")
	answerUpdatedEventTopic   = common.HexToHash("0x0559884fd3a460db3073b7fc896cc77986f16e378210ded43186175bf646fc5f")
	newTransmissionEventTopic = common.HexToHash("0xf6a97944f31ea060dfde0566719c0c1d5ac5b3c3e8b4d8e2c7a6c7e1c8f0c3a8")

	Weight50             = new(uint256.Int).Div(big256.BONE, big256.U2)
	ConvergencePrecision = big256.TenPow(13)
	NumerairePrecision   = big256.TenPow(8)
)
