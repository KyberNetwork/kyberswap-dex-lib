package swaplimit

import (
	"math/big"

	"github.com/holiman/uint256"
)

// ToU256Balances converts legacy big.Int balances to uint256. A negative or nil balance becomes zero. A balance
// above 2^256 is not a cap, so it saturates to MaxUint256.
func ToU256Balances(balance map[string]*big.Int) map[string]*uint256.Int {
	out := make(map[string]*uint256.Int, len(balance))
	for k, v := range balance {
		switch {
		case v == nil || v.Sign() < 0:
			out[k] = new(uint256.Int)
		case v.BitLen() > 256:
			out[k] = new(uint256.Int).SetAllOne()
		default:
			out[k] = uint256.MustFromBig(v)
		}
	}
	return out
}
