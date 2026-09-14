package valueobject

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Arc's native token (18 decimals) and its wrapped-native entry, the ERC-20 USDC interface
// (6 decimals), share one balance - unlike every other chain, where wrapped-native matches
// native's decimals 1:1. WrapNativeAmount/UnwrapNativeAmount are the single conversion point;
// every dex-lib package that substitutes native for wrapped-native must scale amounts through
// these instead of assuming 1:1, or Arc quotes end up off by 1e12.
func TestWrapUnwrapNativeAmount(t *testing.T) {
	oneNative := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	oneWrapped := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)

	t.Run("Arc scales native down to wrapped-native decimals", func(t *testing.T) {
		assert.Equal(t, oneWrapped, WrapNativeAmount(ChainIDArc, oneNative))
	})

	t.Run("Arc scales wrapped-native up to native decimals", func(t *testing.T) {
		assert.Equal(t, oneNative, UnwrapNativeAmount(ChainIDArc, oneWrapped))
	})

	t.Run("non-Arc chains are 1:1", func(t *testing.T) {
		assert.Equal(t, oneNative, WrapNativeAmount(ChainIDEthereum, oneNative))
		assert.Equal(t, oneNative, UnwrapNativeAmount(ChainIDEthereum, oneNative))
	})

	t.Run("nil amount is untouched", func(t *testing.T) {
		assert.Nil(t, WrapNativeAmount(ChainIDArc, nil))
		assert.Nil(t, UnwrapNativeAmount(ChainIDArc, nil))
	})
}
