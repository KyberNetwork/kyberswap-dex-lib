package angletransmuter

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

const (
	CHAINLINK_FEEDS OracleReadType = iota
	EXTERNAL
	NO_ORACLE
	STABLE
	WSTETH
	CBETH
	RETH
	SFRXETH
	PYTH
	MAX
	MORPHO_ORACLE
)
const (
	UNIT OracleQuoteType = iota
	TARGET
)
const (
	MintExactInput QuoteType = iota
	MintExactOutput
	BurnExactInput
	BurnExactOutput
)

type (
	OracleReadType  uint8
	OracleQuoteType int
	QuoteType       uint8

	Gas struct {
		Mint uint64 `json:"mint,omitempty"`
		Burn uint64 `json:"burn,omitempty"`
	}

	Extra struct {
		Gas        Gas             `json:"gas"`
		Transmuter TransmuterState `json:"transmuter"`
	}

	TransmuterState struct {
		Collaterals           map[string]CollateralState `json:"collaterals,omitempty"`
		IsWhitelisted         map[int][]string           `json:"isWhitelisted,omitempty"`
		XRedemptionCurve      []uint64                   `json:"xRedemptionCurve,omitempty"`
		YRedemptionCurve      []int64                    `json:"yRedemptionCurve,omitempty"`
		TotalStablecoinIssued *uint256.Int               `json:"totalStablecoinIssued,omitempty"`
	}

	CollateralState struct {
		Whitelisted       bool         `json:"whitelisted,omitempty"`
		WhitelistData     []byte       `json:"whitelistData,omitempty"`
		Fees              Fees         `json:"fees,omitempty"`
		StablecoinsIssued *uint256.Int `json:"stablecoinsIssued,omitempty"`
		Config            Oracle       `json:"config,omitempty"`
		StablecoinCap     *uint256.Int `json:"stablecoinCap,omitempty"`
	}

	Fees struct {
		XFeeMint []*uint256.Int
		XFeeBurn []*uint256.Int
		YFeeMint []*uint256.Int
		YFeeBurn []*uint256.Int
	}

	Oracle struct {
		OracleType      OracleReadType  `json:"oracleType,omitempty"`
		TargetType      OracleReadType  `json:"targetType,omitempty"`
		ExternalOracle  common.Address  `json:"externalOracle,omitempty"`
		OracleFeed      OracleFeed      `json:"oracleFeed,omitempty"`
		TargetFeed      OracleFeed      `json:"targetFeed,omitempty"`
		Hyperparameters Hyperparameters `json:"hyperparameters,omitempty"`
	}
	Hyperparameters struct {
		UserDeviation      *uint256.Int
		BurnRatioDeviation *uint256.Int
	}
	OracleFeed struct {
		IsPyth      bool         `json:"isPyth,omitempty"`
		IsChainLink bool         `json:"isChainLink,omitempty"`
		IsMorpho    bool         `json:"isMorpho,omitempty"`
		Pyth        Pyth         `json:"pyth,omitempty"`
		Chainlink   Chainlink    `json:"chainlink,omitempty"`
		Max         *uint256.Int `json:"max,omitempty"`
		Morpho      Morpho
	}

	Pyth struct {
		Pyth         common.Address          `json:"pyth,omitempty"`
		FeedIds      []string                `json:"feedIds,omitempty"`
		StalePeriods []uint32                `json:"stalePeriods,omitempty"`
		IsMultiplied []uint8                 `json:"isMultiplied,omitempty"`
		QuoteType    uint8                   `json:"quoteType,omitempty"`
		PythState    []PythState             `json:"pythState,omitempty"`
		Active       bool                    `json:"active,omitempty"`
		RawStates    []DecodedPythStateTuple `json:"-"`
	}
	PythState struct {
		Price     *uint256.Int `json:"price,omitempty"`
		Expo      *uint256.Int `json:"expo,omitempty"`
		Timestamp *uint256.Int `json:"timestamp,omitempty"`
	}

	Chainlink struct {
		CircuitChainlink         []common.Address   `json:"circuitChainlink,omitempty"`
		StalePeriods             []uint32           `json:"stalePeriods,omitempty"`
		CircuitChainIsMultiplied []uint8            `json:"circuitChainIsMultiplied,omitempty"`
		ChainlinkDecimals        []uint8            `json:"chainlinkDecimals,omitempty"`
		QuoteType                uint8              `json:"quoteType,omitempty"`
		Answers                  []*uint256.Int     `json:"answers,omitempty"`
		UpdatedAt                []uint64           `json:"updatedAt,omitempty"`
		Active                   bool               `json:"active,omitempty"`
		RawStates                []DecodedChainlink `json:"-"`
	}
	Morpho struct {
		Oracle              common.Address `json:"oracle,omitempty"`
		NormalizationFactor *uint256.Int   `json:"normalizationFactor,omitempty"`
		Price               *uint256.Int   `json:"price,omitempty"`
		Active              bool           `json:"active,omitempty"`
		RawState            *big.Int       `json:"-"`
	}
)

var (
	Uint256, _    = abi.NewType("uint256", "", nil)
	Uint160, _    = abi.NewType("uint160", "", nil)
	Uint32, _     = abi.NewType("uint32", "", nil)
	Uint32Arr, _  = abi.NewType("uint32[]", "", nil)
	Uint16, _     = abi.NewType("uint16", "", nil)
	Uint8, _      = abi.NewType("uint8", "", nil)
	Uint8Arr, _   = abi.NewType("uint8[]", "", nil)
	String, _     = abi.NewType("string", "", nil)
	Bool, _       = abi.NewType("bool", "", nil)
	Bytes, _      = abi.NewType("bytes", "", nil)
	Bytes32, _    = abi.NewType("bytes32", "", nil)
	Bytes32Arr, _ = abi.NewType("bytes32[]", "", nil)
	Address, _    = abi.NewType("address", "", nil)
	Uint64Arr, _  = abi.NewType("uint64[]", "", nil)
	Uint256Arr, _ = abi.NewType("uint256[]", "", nil)
	AddressArr, _ = abi.NewType("address[]", "", nil)
	BytesArr, _   = abi.NewType("bytes[]", "", nil)
	Int8, _       = abi.NewType("int8", "", nil)
	Int24, _      = abi.NewType("int24", "", nil)
	Int128, _     = abi.NewType("int128", "", nil)
)
