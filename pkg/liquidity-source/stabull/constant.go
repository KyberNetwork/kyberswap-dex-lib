package stabull

import "math/big"

const (
	DexType = "stabull"

	// Factory methods
	// Note: Stabull factory uses event-based pool discovery (NewCurve events)
	// There is no indexed enumeration like curvesLength/curves(uint256)
	factoryMethodCurves   = "curves"   // curves(bytes32 id) returns curve address for given id
	factoryMethodGetCurve = "getCurve" // getCurve(address base, address quote) returns curve address

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
	oracleMethodLatestAnswer = "latestAnswer" // latestAnswer() returns int256 price

	// Event signatures
	// NewCurve(address indexed caller, bytes32 indexed id, address indexed curve)
	eventNewCurve = "NewCurve(address,bytes32,address)"
	newCurveTopic = "0xe7a19de9e8788cc07c144818f2945144acd6234f790b541aa1010371c8b2a73b"

	// ParametersSet event for parameter changes (admin/owner only, infrequent)
	// ParametersSet(uint256 alpha, uint256 beta, uint256 delta, uint256 epsilon, uint256 lambda)
	eventParametersSet      = "ParametersSet(uint256,uint256,uint256,uint256,uint256)"
	parametersSetEventTopic = "0xb399767364127d5a414f09f214fa5606358052b764894b1084ce5ef067c05a97"

	// Trade event (emitted when swaps occur)
	// Trade(address indexed trader, address indexed origin, address indexed target, uint256 originAmount, uint256 targetAmount, int128 rawProtocolFee)
	eventTrade      = "Trade(address,address,address,uint256,uint256,int128)"
	tradeEventTopic = "0x887adc1b38cfb756ed025ea6acd9382fbd376ede6c34bc6fa738284b09275468"

	// Chainlink Oracle events
	// AnswerUpdated(int256 indexed current, uint256 indexed roundId, uint256 updatedAt)
	// This event is emitted by both AccessControlledOCR2Aggregator (Polygon, Ethereum)
	// and AccessControlledOffchainAggregator (Base) with the same signature
	eventAnswerUpdated      = "AnswerUpdated(int256,uint256,uint256)"
	answerUpdatedEventTopic = "0x0559884fd3a460db3073b7fc896cc77986f16e378210ded43186175bf646fc5f"

	// NewTransmission event (alternative oracle update event for OCR2)
	// NewTransmission(uint32 indexed aggregatorRoundId, int192 answer, address transmitter, int192[] observations, bytes observers, bytes32 rawReportContext)
	eventNewTransmission      = "NewTransmission(uint32,int192,address,int192[],bytes,bytes32)"
	newTransmissionEventTopic = "0xf6a97944f31ea060dfde0566719c0c1d5ac5b3c3e8b4d8e2c7a6c7e1c8f0c3a8"

	// Note: Stabull pools use standard ERC20 Transfer events for LP token minting/burning
	// For liquidity tracking, monitor Transfer(address,address,uint256) events
	// Minting: Transfer(0x0, depositor, amount)
	// Burning: Transfer(withdrawer, 0x0, amount)

	defaultTokenDecimals = 18
	reserveZero          = "0"

	// Swap fee: 0.15% (15 basis points)
	// 70% goes to LPs, 30% goes to protocol
	swapFeeBps     = 15     // 0.15% = 15 basis points
	swapFeePercent = 0.0015 // 0.15%

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
