package twamm

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestExp2(t *testing.T) {
	t.Parallel()
	require.Equal(t, math.TwoPow64, exp2(new(big.Int)))
	require.Equal(t, new(big.Int).Mul(big.NewInt(2), math.TwoPow64), exp2(math.TwoPow64))
	require.Equal(t,
		bignum.NewBig("52175271301331128849"),
		exp2(new(big.Int).Div(
			new(big.Int).Lsh(
				big.NewInt(3),
				64,
			),
			big.NewInt(2),
		)),
	)
	require.Equal(t,
		bignum.NewBig("240615969168004511545033772477625056927"),
		exp2(new(big.Int).Add(
			new(big.Int).Lsh(
				big.NewInt(62),
				64,
			),
			new(big.Int).Div(
				new(big.Int).Lsh(
					big.NewInt(3),
					64,
				),
				big.NewInt(2),
			),
		)),
	)
	require.Equal(t,
		new(big.Int).Lsh(
			big.NewInt(4),
			64,
		),
		exp2(new(big.Int).Lsh(
			big.NewInt(2),
			64,
		)),
	)
	require.Equal(t,
		new(big.Int).Lsh(
			big.NewInt(8),
			64,
		),
		exp2(new(big.Int).Lsh(
			big.NewInt(3),
			64,
		)),
	)
	require.Equal(t,
		new(big.Int).Lsh(
			bignum.NewBig("9223372036854775808"),
			64,
		),
		exp2(new(big.Int).Lsh(
			big.NewInt(63),
			64,
		)),
	)
}
