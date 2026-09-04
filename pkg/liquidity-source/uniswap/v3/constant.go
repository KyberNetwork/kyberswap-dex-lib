package uniswapv3

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// DexType* are kept as the historical, backward-compatible pool-type strings for every
// Uniswap V3 fork merged into this package. Each is still registered independently (see
// pool_factory.go/pool_tracker.go/pools_list_updater.go/pool_simulator.go) so existing
// deployment configs and persisted entity.Pool.Type values keep working unchanged.
const (
	DexType           = DexTypeUniswapV3
	DexTypeUniswapV3  = "uniswapv3"
	DexTypePancakeV3  = "pancake-v3"
	DexTypeRamsesV2   = "ramses-v2"
	DexTypeSolidlyV3  = "solidly-v3"
	DexTypeSlipstream = "slipstream"
	DexTypeNuriV2     = "nuri-v2"

	graphFirstLimit      = 1000
	defaultTokenDecimals = 18
	tickChunkSize        = 100
)

const (
	methodGetLiquidity = "liquidity"
	methodGetSlot0     = "slot0"
	methodTickSpacing  = "tickSpacing"
	methodTicks        = "ticks"
	methodFee          = "fee"
	methodCurrentFee   = "currentFee"
)

var (
	zeroBI     = big.NewInt(0)
	defaultGas = Gas{BaseGas: 109334, CrossInitTickGas: 21492}

	ErrOverflow            = errors.New("bigInt overflow int/uint256")
	ErrInvalidFeeTier      = errors.New("invalid feeTier")
	ErrTickNil             = errors.WithMessage(pool.ErrUnsupported, "tick is nil")
	ErrV3TicksEmpty        = errors.WithMessage(pool.ErrUnsupported, "v3Ticks empty")
	ErrInvalidToken        = errors.New("invalid token")
	ErrZeroAmount          = errors.New("zero amount")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrBuyRestricted       = errors.New("token buy restricted")
	ErrMaxTxExceeded       = errors.New("amount exceeds launch guard maxTx")

	ErrMalformedLog = errors.New("malformed event log")
)

// PoolCreated event topic0s. Every merged fork's factory PoolCreated event decodes into one
// of exactly two shapes; in both, `token0`/`token1` are always topics[1]/topics[2] and `pool`
// is always the trailing 32-byte word of `Data`, regardless of which fields are indexed or
// whether `fee` is present at all - so decoding needs no per-fork ABI, just these two hashes.
//
//   - poolCreatedEventIDWithFee: PoolCreated(address,address,uint24,int24,address)
//     used by uniswap-v3, pancake-v3, ramses-v2, solidly-v3, nuri-v2. solidly-v3 swaps which
//     of `fee`/`tickSpacing` is indexed vs. uniswap's layout, but since neither field is read
//     at decode time (see pool_factory.go), the topic0 collision is harmless.
//   - poolCreatedEventIDNoFee: PoolCreated(address,address,int24,address), used by slipstream.
var (
	poolCreatedEventIDWithFee = crypto.Keccak256Hash([]byte("PoolCreated(address,address,uint24,int24,address)"))
	poolCreatedEventIDNoFee   = crypto.Keccak256Hash([]byte("PoolCreated(address,address,int24,address)"))
)

// Mint/Burn event topic0s, computed the same way as PoolCreated above. Burn has one shape
// across every merged fork. Mint has two: the standard shape, and ramses-v2's static-fee
// ("V3") pool, which inserts a non-indexed `index uint256` between `owner` and `tickLower`.
// `owner`/`tickLower`/`tickUpper` stay at topics[1]/[2]/[3] in both, so only the `amount`
// offset inside Data differs (see pool_tracker.go's extractEventData).
var (
	mintEventID          = crypto.Keccak256Hash([]byte("Mint(address,address,int24,int24,uint128,uint256,uint256)"))
	mintWithIndexEventID = crypto.Keccak256Hash([]byte("Mint(address,address,uint256,int24,int24,uint128,uint256,uint256)"))
	burnEventID          = crypto.Keccak256Hash([]byte("Burn(address,int24,int24,uint128,uint256,uint256)"))
)

var poolCreatedEventIDs = map[common.Hash]struct{}{
	poolCreatedEventIDWithFee: {},
	poolCreatedEventIDNoFee:   {},
}
