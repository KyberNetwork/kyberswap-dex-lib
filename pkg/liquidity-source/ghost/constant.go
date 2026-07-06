package ghost

import "errors"

const (
	DexType = "ghost"

	defaultReserves = "0"

	DefaultGas int64 = 250_000

	feeMethodMaxFee     = "maxFee"
	feeMethodHalfAmount = "halfAmount"
	feeMethodQuotes     = "quotes"

	feeTypeCrossCollateralRouting uint8 = 5
	feeTypeOffchainQuotedLinear   uint8 = 6

	// GhostFeeDenominator mirrors ExecutorV3Helper7's GHOST_FEE_DENOMINATOR — the
	// denominator executeGhost uses to recover the principal from totalFeeBps on-chain.
	GhostFeeDenominator int64 = 1_000_000
)

// DEFAULT_ROUTER is the fallback bytes32 key used in feeContracts() lookups.
// 0x6e086cd647d6eb8b516856666e2c1465fb8a6a58d3a75938362acc674eacaf47
var defaultRouterKey = [32]byte{
	0x6e, 0x08, 0x6c, 0xd6, 0x47, 0xd6, 0xeb, 0x8b,
	0x51, 0x68, 0x56, 0x66, 0x6e, 0x2c, 0x14, 0x65,
	0xfb, 0x8a, 0x6a, 0x58, 0xd3, 0xa7, 0x59, 0x38,
	0x36, 0x2a, 0xcc, 0x67, 0x4e, 0xac, 0xaf, 0x47,
}

var (
	ErrInsufficientLiquidity = errors.New("ghost: insufficient liquidity")
	ErrInvalidToken          = errors.New("ghost: invalid token")
	ErrOverflow              = errors.New("ghost: uint256 overflow")
	ErrUnsupportedFeeType    = errors.New("ghost: unsupported fee contract type")
	ErrNoFeeContract         = errors.New("ghost: could not resolve fee contract")
)
