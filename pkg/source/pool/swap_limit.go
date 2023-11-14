package pool

import "math/big"

type SwapLimit interface {
	GetLimit(key string) *big.Int
	UpdateLimit(decreaseKey, increasedKey string, decreasedDelta, increasedDelta *big.Int) (increasedLimitAfter *big.Int, decreasedLimitAfter *big.Int, err error)
}
