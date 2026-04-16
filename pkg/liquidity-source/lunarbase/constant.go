package lunarbase

import (
	"math"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const (
	DexType = "lunarbase"

	defaultGas = 120000
)

var (
	fQ48 = math.Pow(2, 48)
	q96  = new(uint256.Int).Lsh(big256.U1, 96)

	topicStateUpdated      = crypto.Keccak256Hash([]byte("StateUpdated((uint160,uint48))"))
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
