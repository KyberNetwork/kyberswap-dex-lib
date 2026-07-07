package shared

import "github.com/holiman/uint256"

const (
	HOOKS_BEFORE_INITIALIZE_OFFSET = iota
	HOOKS_AFTER_INITIALIZE_OFFSET
	HOOKS_BEFORE_MINT_OFFSET
	HOOKS_AFTER_MINT_OFFSET
	HOOKS_BEFORE_BURN_OFFSET
	HOOKS_AFTER_BURN_OFFSET
	HOOKS_BEFORE_SWAP_OFFSET
	HOOKS_AFTER_SWAP_OFFSET
	HOOKS_BEFORE_DONATE_OFFSET
	HOOKS_AFTER_DONATE_OFFSET
	HOOKS_BEFORE_SWAP_RETURNS_DELTA_OFFSET
	HOOKS_AFTER_SWAP_RETURNS_DELTA_OFFSET
	HOOKS_AFTER_MINT_RETURNS_DELTA_OFFSET
	HOOKS_AFTER_BURN_RETURNS_DELTA_OFFSET
)

var (
	_MASK1 = uint256.NewInt(0x1)

	MAX_FEE_PIPS     uint32 = 1000000 // 100%
	DYNAMIC_FEE_FLAG uint32 = 0x800000
)

func hasOffsetEnabled(data []byte, offset int) bool {
	res := new(uint256.Int).SetBytes32(data)
	res.Rsh(res, uint(offset))

	return res.And(res, _MASK1).Sign() != 0
}

func HasSwapPermissions(parameters []byte) bool {
	return hasOffsetEnabled(parameters, HOOKS_BEFORE_SWAP_OFFSET) || hasOffsetEnabled(parameters, HOOKS_AFTER_SWAP_OFFSET)
}

func IsDynamicFee(fee uint32) bool {
	return fee == DYNAMIC_FEE_FLAG
}
