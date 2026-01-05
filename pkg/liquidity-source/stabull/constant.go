package stabull

import "math/big"

const (
	DexType = "stabull"

	// Pool (Curve) methods
	poolMethodNumeraires     = "numeraires"     // numeraires(uint256) returns token address at index
	poolMethodReserves       = "reserves"       // reserves(uint256) returns reserve token address at index
	poolMethodLiquidity      = "liquidity"      // liquidity() returns (total, individual[])
	poolMethodViewCurve      = "viewCurve"      // viewCurve() returns (alpha, beta, delta, epsilon, lambda)
	poolMethodViewOriginSwap = "viewOriginSwap" // viewOriginSwap(origin, target, originAmount) returns targetAmount

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

// Factory addresses per chain (from DefiLlama adapter)
var FactoryAddresses = map[string]string{
	"ethereum": "0x2e9E34b5Af24b66F12721113C1C8FFcbB7Bc8051",
	"polygon":  "0x3c60234db40e6e5b57504e401b1cdc79d91faf89",
	"base":     "0x86Ba17ebf8819f7fd32Cf1A43AbCaAe541A5BEbf",
	"arbitrum": "0xArbitrumFactoryAddress", // TODO: Add when Stabull deploys to Arbitrum
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
