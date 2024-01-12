package bignumber_test

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGobBigIntMarshal(t *testing.T) {
	var bignum big.Int
	for i := 0; i < 100; i++ {
		number := rand.Int63()
		bi := big.NewInt(number)

		{
			// check marshal unmarshal small int
			gbi1 := (*bignumber.GobBigInt)(bi)
			bytes, err := gbi1.MarshalText()
			require.Nil(t, err)

			gbi2 := (*bignumber.GobBigInt)(new(big.Int))
			err = gbi2.UnmarshalText(bytes)
			require.Nil(t, err)
			assert.Equal(t, gbi1, gbi2)
		}

		// check marshal unmarshal big int (will start falling back to heap after sometime)
		bignum.Lsh(&bignum, 64)
		bignum.Add(&bignum, bi)
		{
			gbi1 := (*bignumber.GobBigInt)(&bignum)
			bytes, err := gbi1.MarshalText()
			require.Nil(t, err)

			gbi2 := (*bignumber.GobBigInt)(new(big.Int))
			err = gbi2.UnmarshalText(bytes)
			require.Nil(t, err)
			assert.Equal(t, gbi1, gbi2)
		}
	}
}
