package bignumber

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestTenPowDecimals(t *testing.T) {
	expected1, _ := new(big.Float).SetString("1000000000000000000")
	expected2, _ := new(big.Float).SetString("1")
	expected3, _ := new(big.Float).SetString("10000000000")

	type test struct {
		decimal  uint8
		expected *big.Float
	}

	tests := []test{
		{decimal: 18, expected: expected1},
		{decimal: 0, expected: expected2},
		{decimal: 10, expected: expected3},
	}

	for _, tc := range tests {
		actual := TenPowDecimals(tc.decimal)

		assert.Equal(t, 0, tc.expected.Cmp(actual))
	}

}

func TestFillBig(t *testing.T) {
	var bi big.Int
	for i := 0; i < 500; i++ {
		number := testutil.RandNumberHexString(64)
		t.Run(fmt.Sprintf("test %s", number), func(t *testing.T) {
			u := uint256.MustFromHex("0x" + number)
			FillBig(u, &bi)
			assert.Equal(t, u.Dec(), bi.Text(10))
		})
	}
}
