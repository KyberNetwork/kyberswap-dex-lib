package bunniv2

import (
	"math/big"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/ethereum/go-ethereum/common"

	"github.com/holiman/uint256"
)

const (
	_MAX_OBSERVATION_BATCH_SIZE = 1000

	_BEFORE_SWAP_GAS = 525000
)

const (
	STATIC               uint8 = iota // LDF does not change ever
	DYNAMIC_NOT_STATEFUL              // LDF can change, does not use ldfState
	DYNAMIC_AND_STATEFUL              // LDF can change, uses ldfState
)

var (
	ZERO_BALANCE = [32]byte{
		0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}

	OBSERVATION_BASE_SLOT   = common.LeftPadBytes(big.NewInt(6).Bytes(), 32)
	OBSERVATION_STATE_SLOT  = common.LeftPadBytes(big.NewInt(7).Bytes(), 32)
	VAULT_SHARE_PRICES_SLOT = common.LeftPadBytes(big.NewInt(11).Bytes(), 32)
	CURATOR_FEES_SLOT       = common.LeftPadBytes(big.NewInt(15).Bytes(), 32)
	HOOK_FEE_SLOT           = common.LeftPadBytes(big.NewInt(16).Bytes(), 32)

	SWAP_FEE_BASE        = uint256.NewInt(1e6)
	MODIFIER_BASE        = SWAP_FEE_BASE
	RAW_TOKEN_RATIO_BASE = SWAP_FEE_BASE

	CURATOR_FEE_BASE      = uint256.NewInt(1e5)
	MAX_SWAP_FEE_RATIO, _ = u256.NewUint256("28800000000000000000000") // 2.88e20
	MAX_SWAP_FEE          = SWAP_FEE_BASE                              // 1e6
	MIN_FEE_AMOUNT        = u256.U1
	EPSILON_FEE           = uint256.NewInt(30)
	SWAP_FEE_BASE_SQUARED = uint256.NewInt(1e12)
	LN2_WAD               = uint256.NewInt(693147180559945309)
	WAD                   = uint256.NewInt(1e18)
	Q96                   = new(uint256.Int).Lsh(u256.U1, 96)
)
