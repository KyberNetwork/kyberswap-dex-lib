package lunarbase

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

const (
	DexType = "lunarbase"

	defaultCoreAddress      = "0xeccd5b11549140c67fa2e8b6028bc86a4f9cab6d"
	defaultPeripheryAddress = "0x110ab7d4a269cc0e94b6f56926186ec4716edb1b"
	defaultPermit2Address   = "0x000000000022d473030f116ddee9f6b43ac78ba3"

	defaultGas = 120000
)

var (
	pmmSlotState    = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002")
	pmmSlotReserves = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000003")
	q96             = new(uint256.Int).Lsh(uint256.NewInt(1), 96)
	// The contract exposes fee as uint48 in state() and an immutable 5000 getter on the live Base deployment.
	// We treat it as 1e6-based fee precision here; this still needs confirmation from verified source code.
	feePrecision = uint256.NewInt(1_000_000)

	ErrInvalidToken          = errors.New("invalid token")
	ErrPoolPaused            = errors.New("pool is paused")
	ErrZeroPrice             = errors.New("pool price is zero")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrQuoteFailed           = errors.New("quote failed")
)
