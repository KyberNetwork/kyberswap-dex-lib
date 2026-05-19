package lunarbase

import (
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	DexType = "lunarbase"

	defaultGas = 120000

	// fQ24 is 2^24 — used to render Q24 directional fees as fractional
	// `SwapFee` on the entity.
	fQ24 = 1 << 24
)

var (
	topicStateUpdated      = crypto.Keccak256Hash([]byte("StateUpdated(uint160,uint24,uint24)"))
	topicSync              = crypto.Keccak256Hash([]byte("Sync(uint128,uint128)"))
	topicSwapExecuted      = crypto.Keccak256Hash([]byte("SwapExecuted(address,bool,uint256,uint256,uint256)"))
	topicConcentrationKSet = crypto.Keccak256Hash([]byte("ConcentrationKSet(uint32)"))
	topicBlockDelaySet     = crypto.Keccak256Hash([]byte("BlockDelaySet(uint48)"))

	ErrStalePool             = errors.WithMessage(pool.ErrUnsupported, "stale pool")
	ErrInvalidToken          = errors.New("invalid token")
	ErrPoolPaused            = errors.New("pool is paused")
	ErrZeroPrice             = errors.New("pool price is zero")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrQuoteFailed           = errors.New("quote failed")
)
