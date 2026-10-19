package hyfiprop

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The Arbitrum HyFi bytecode differs from Base/Robinhood (unverified, 56 bytes larger). Every case is
// quoteDirect on the ARB/USDC pair 0x52c29eb2... at block 515714410, 2s after the first book push
// (only the ask side holds ticks). Book ends between 150 and 200 USDC.
func TestPriceExactIn_ArbitrumQuoteDirect(t *testing.T) {
	cfg := PairConfig{TickWidth: u("1000000"), BaseLiqUnit: u("105000000000000000000"), FeePerSecond: 50, BaseIsC0: true}
	book := &[2]Side{{Slot0: u("6692968426461250899748")},
		{Slot0: u("242639520004355758152003135783176676062734299809558120756393743142792")}}
	const now = 1792252029
	for _, c := range [][3]string{{"1000000", "5428338762214983712", "542888165038003"},
		{"10000000", "54283387622149837133", "5428881650380022"},
		{"50000000", "271416938110749185667", "27144408251900109"},
		{"100000000", "542833876221498371334", "54288816503800218"},
		{"150000000", "814250814332247557002", "81433224755700326"}} {
		in := u(c[0])
		res, err := priceExactIn(book, &cfg, false, &in, now)
		require.NoError(t, err, c[0])
		require.Equal(t, c[1], res.amountOut.Dec(), "out %s", c[0])
		require.Equal(t, c[2], res.fee.Dec(), "fee %s", c[0])
	}
	for _, x := range []string{"200000000", "500000000"} { // past the book: quoteDirect reverts
		in := u(x)
		_, err := priceExactIn(book, &cfg, false, &in, now)
		require.ErrorIs(t, err, ErrInsufficientLiquidity, x)
	}
	in := u("1000000000000000000") // ARB in: the bid side holds no ticks
	_, err := priceExactIn(book, &cfg, true, &in, now)
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
}
