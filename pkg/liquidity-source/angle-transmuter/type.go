package angletransmuter

import (
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
		Swap uint64 `json:"swap,omitempty"`
	}

	Extra struct {
		Gas        Gas             `json:"gas"`
		Transmuter TransmuterState `json:"transmuter"`
	}

	TransmuterState struct {
		Collaterals           map[string]CollateralState
		IsWhitelisted         map[int][]string // TODO: map of what (OracleQuoteType ??????????)
		XRedemptionCurve      []uint64
		YRedemptionCurve      []int64
		TotalStablecoinIssued *uint256.Int
	}

	CollateralState struct {
		Whitelisted       bool
		WhitelistData     []byte
		Fees              Fees
		StablecoinsIssued *uint256.Int
		Config            Oracle
		StablecoinCap     *uint256.Int
	}

	Fees struct {
		XFeeMint []*uint256.Int
		XFeeBurn []*uint256.Int
		YFeeMint []*uint256.Int
		YFeeBurn []*uint256.Int
	}

	Oracle struct {
		OracleType      OracleReadType
		TargetType      OracleReadType
		ExternalOracle  common.Address
		OracleFeed      OracleFeed
		TargetFeed      OracleFeed
		Hyperparameters Hyperparameters
	}
	Hyperparameters struct {
		UserDeviation      *uint256.Int
		BurnRatioDeviation *uint256.Int
	}
	OracleFeed struct {
		IsPyth      bool
		IsChainLink bool
		IsMorpho    bool
		Pyth        Pyth
		Chainlink   Chainlink
	}

	Pyth struct {
		Pyth         common.Address
		FeedIds      []string
		StalePeriods []uint32
		IsMultiplied []uint8
		QuoteType    uint8
		PythState    []PythState
		Active       bool
	}
	PythState struct {
		Price     *uint256.Int
		Expo      *uint256.Int
		Timestamp *uint256.Int
	}

	Chainlink struct {
		CircuitChainlink         []common.Address
		StalePeriods             []uint32
		CircuitChainIsMultiplied []uint8
		ChainlinkDecimals        []uint8
		QuoteType                uint8
		Active                   bool
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
