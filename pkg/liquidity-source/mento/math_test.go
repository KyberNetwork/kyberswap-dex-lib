package mento

import (
	"testing"
	"time"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func ts(s string) uint64 {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return uint64(t.Unix())
}

func TestIsFXMarketOpen(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"2026-09-22T12:00:00Z": true,  // Tuesday
		"2026-09-25T20:59:59Z": true,  // Friday before close
		"2026-09-25T21:00:00Z": false, // Friday 21:00 UTC close
		"2026-09-26T12:00:00Z": false, // Saturday
		"2026-09-27T22:59:59Z": false, // Sunday before open
		"2026-09-27T23:00:00Z": true,  // Sunday 23:00 UTC open
		"2026-12-24T21:59:59Z": true,  // Christmas Eve before early close
		"2026-12-24T22:00:00Z": false, // Christmas Eve early close (Thursday)
		"2026-12-25T10:00:00Z": false, // Christmas (Friday)
		"2026-12-31T21:00:00Z": true,  // New Year's Eve before early close (Thursday)
		"2026-12-31T22:00:00Z": false, // New Year's Eve early close
		"2027-01-01T12:00:00Z": false, // New Year's Day (Friday)
		"2024-01-06T00:00:00Z": false, // closedMarketProbeTimestamp is a Saturday
	}
	for at, open := range cases {
		assert.Equalf(t, open, isFXMarketOpen(ts(at)), "at %s", at)
	}
	assert.Equal(t, uint64(closedMarketProbeTimestamp), ts("2024-01-06T00:00:00Z"))
}

func TestConvertWithRateAndFee_ReproducesOnChainQuote(t *testing.T) {
	t.Parallel()
	// Monad USDC/USDm pool, block 107045873: medianRate 999935510000000000/1e18,
	// lpFee 3, protocolFee 2. getAmountOut(1e6, USDC) = 999435542245000000.
	num := big256.New("999935510000000000")
	den := big256.New("1000000000000000000")
	dec0, dec1 := big256.TenPow(6), big256.TenPow(18)
	feeNum := uint256.NewInt(bps - 5)

	out, err := convertWithRateAndFee(uint256.NewInt(1_000_000), dec0, dec1, num, den, feeNum, uBps)
	require.NoError(t, err)
	assert.Equal(t, "999435542245000000", out.Dec())

	out, err = convertWithRateAndFee(big256.New("1000000000000000000"), dec1, dec0, den, num, feeNum, uBps)
	require.NoError(t, err)
	assert.Equal(t, "999564", out.Dec())

	gross, err := convertWithRate(uint256.NewInt(1_000_000), dec0, dec1, num, den)
	require.NoError(t, err)
	assert.Equal(t, "999935510000000000", gross.Dec())
}

func TestConvertWithRate_Overflow(t *testing.T) {
	t.Parallel()
	huge := new(uint256.Int).Sub(big256.UMax, uint256.NewInt(1))
	_, err := convertWithRate(huge, big256.TenPow(6), big256.TenPow(18), uint256.NewInt(2), uint256.NewInt(1))
	assert.ErrorIs(t, err, ErrOverflow)
	_, err = convertWithRateAndFee(huge, big256.TenPow(6), big256.TenPow(18), uint256.NewInt(2), uint256.NewInt(1),
		uint256.NewInt(9995), uBps)
	assert.ErrorIs(t, err, ErrOverflow)
}

func TestScaleValue(t *testing.T) {
	t.Parallel()
	v, err := scaleValue(uint256.NewInt(1_000_000), 6)
	require.NoError(t, err)
	assert.Equal(t, "1000000000000000", v.Dec()) // 1 token -> 1e15
	v, err = scaleValue(big256.New("1500000000000000000"), 18)
	require.NoError(t, err)
	assert.Equal(t, "1500000000000000", v.Dec()) // 1.5 token -> 1.5e15
	v, err = scaleValue(new(uint256.Int), 18)
	require.NoError(t, err)
	assert.True(t, v.IsZero())
}
