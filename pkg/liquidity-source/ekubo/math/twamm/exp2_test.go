package twamm

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func TestExp2(t *testing.T) {
	t.Parallel()
	require.Equal(t, big256.U2Pow64, exp2(new(uint256.Int)))
	require.Equal(t, new(uint256.Int).Mul(uint256.NewInt(2), big256.U2Pow64), exp2(big256.U2Pow64))
	require.Equal(t,
		big256.New("52175271301331128849"),
		exp2(new(uint256.Int).Div(
			new(uint256.Int).Lsh(
				uint256.NewInt(3),
				64,
			),
			uint256.NewInt(2),
		)),
	)
	require.Equal(t,
		big256.New("240615969168004511545033772477625056927"),
		exp2(new(uint256.Int).Add(
			new(uint256.Int).Lsh(
				uint256.NewInt(62),
				64,
			),
			new(uint256.Int).Div(
				new(uint256.Int).Lsh(
					uint256.NewInt(3),
					64,
				),
				uint256.NewInt(2),
			),
		)),
	)
	require.Equal(t,
		new(uint256.Int).Lsh(
			uint256.NewInt(4),
			64,
		),
		exp2(new(uint256.Int).Lsh(
			uint256.NewInt(2),
			64,
		)),
	)
	require.Equal(t,
		new(uint256.Int).Lsh(
			uint256.NewInt(8),
			64,
		),
		exp2(new(uint256.Int).Lsh(
			uint256.NewInt(3),
			64,
		)),
	)
	require.Equal(t,
		new(uint256.Int).Lsh(
			big256.New("9223372036854775808"),
			64,
		),
		exp2(new(uint256.Int).Lsh(
			uint256.NewInt(63),
			64,
		)),
	)
}
