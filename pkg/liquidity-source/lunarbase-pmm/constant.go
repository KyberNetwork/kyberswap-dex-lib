package lunarbase

import (
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
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
	q96 = new(uint256.Int).Lsh(uint256.NewInt(1), 96)

	topicStateUpdated     = crypto.Keccak256Hash([]byte("StateUpdated((uint160,uint48))"))
	topicSync             = crypto.Keccak256Hash([]byte("Sync(uint128,uint128)"))
	topicSwapExecuted     = crypto.Keccak256Hash([]byte("SwapExecuted(address,bool,uint256,uint256,uint256)"))
	topicConcentrationKSet = crypto.Keccak256Hash([]byte("ConcentrationKSet(uint32)"))
	topicBlockDelaySet    = crypto.Keccak256Hash([]byte("BlockDelaySet(uint48)"))

	ErrInvalidToken          = errors.New("invalid token")
	ErrPoolPaused            = errors.New("pool is paused")
	ErrZeroPrice             = errors.New("pool price is zero")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrQuoteFailed           = errors.New("quote failed")
)
