package helper

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

var (
	// UINT_160_MAX represents the maximum value for a uint160
	UINT_160_MAX = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 160), big.NewInt(1))
	// UINT_40_MAX represents the maximum value for a uint40
	UINT_40_MAX = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 40), big.NewInt(1))
)

func TestMakerTraits_NewWithHexValue(t *testing.T) {
	hexValue := "0x4000000000000000000000000000000000006777c0f900000000000000000000"
	traits := NewMakerTraits(hexValue)

	// Test that the value was correctly parsed
	assert.Equal(t, hexValue[2:], traits.Build().Text(16))

	// Test specific bits and values that should be set based on this hex value
	// The hex value 0x4000... has the following properties:
	// - Bit 254 (ALLOW_MULTIPLE_FILLS_FLAG) is set to 1
	assert.True(t, traits.IsMultipleFillsAllowed())

	expectedExpiration := big.NewInt(1735901433)
	assert.Equal(t, expectedExpiration, traits.Expiration())

	// Test that other flags are not set
	assert.True(t, traits.IsPartialFillAllowed())
	assert.False(t, traits.HasPreInteraction())
	assert.False(t, traits.HasPostInteraction())
	assert.False(t, traits.IsEpochManagerEnabled())
	assert.False(t, traits.HasExtension())
	assert.False(t, traits.IsPermit2())
	assert.False(t, traits.IsNativeUnwrapEnabled())
}

func TestMakerTraits_AllowedSender(t *testing.T) {
	traits := DefaultMakerTraits()

	// Create an address with value 1337
	sender := common.BigToAddress(big.NewInt(1337))

	traits.WithAllowedSender(sender)
	senderHalf := traits.AllowedSender()

	// Compare the last 10 bytes
	assert.Equal(t, sender.Bytes()[10:], senderHalf.Bytes()[10:])
}

func TestMakerTraits_Nonce(t *testing.T) {
	traits := DefaultMakerTraits()

	// Test normal nonce (1 << 10)
	nonce := new(big.Int).Lsh(big.NewInt(1), 10)
	traits.WithNonce(nonce)
	assert.Equal(t, nonce, traits.NonceOrEpoch())

	// Test too large nonce (1 << 50)
	bigNonce := new(big.Int).Lsh(big.NewInt(1), 50)
	traits.WithNonce(bigNonce)
	// In Go, we handle overflow by masking, so we should get the masked value
	masked := new(big.Int).And(bigNonce, new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 40), big.NewInt(1)))
	assert.Equal(t, masked, traits.NonceOrEpoch())
}

func TestMakerTraits_Expiration(t *testing.T) {
	traits := DefaultMakerTraits()
	expiration := big.NewInt(1000000)

	traits.WithExpiration(expiration)
	assert.Equal(t, expiration, traits.Expiration())
}

func TestMakerTraits_Epoch(t *testing.T) {
	traits := DefaultMakerTraits()
	series := big.NewInt(100)
	epoch := big.NewInt(1)

	traits.AllowPartialFills().AllowMultipleFills().WithEpoch(series, epoch)
	assert.Equal(t, series, traits.Series())
	assert.Equal(t, epoch, traits.NonceOrEpoch())
	assert.True(t, traits.IsEpochManagerEnabled())
}

func TestMakerTraits_Extension(t *testing.T) {
	traits := DefaultMakerTraits()
	assert.False(t, traits.HasExtension())

	traits.WithExtension()
	assert.True(t, traits.HasExtension())
}

func TestMakerTraits_PartialFills(t *testing.T) {
	traits := DefaultMakerTraits()
	assert.True(t, traits.IsPartialFillAllowed())

	traits.DisablePartialFills()
	assert.False(t, traits.IsPartialFillAllowed())

	traits.AllowPartialFills()
	assert.True(t, traits.IsPartialFillAllowed())
}

func TestMakerTraits_MultipleFills(t *testing.T) {
	traits := DefaultMakerTraits()
	assert.False(t, traits.IsMultipleFillsAllowed())

	traits.AllowMultipleFills()
	assert.True(t, traits.IsMultipleFillsAllowed())

	traits.DisableMultipleFills()
	assert.False(t, traits.IsMultipleFillsAllowed())
}

func TestMakerTraits_PreInteraction(t *testing.T) {
	traits := DefaultMakerTraits()
	assert.False(t, traits.HasPreInteraction())

	traits.EnablePreInteraction()
	assert.True(t, traits.HasPreInteraction())

	traits.DisablePreInteraction()
	assert.False(t, traits.HasPreInteraction())
}

func TestMakerTraits_PostInteraction(t *testing.T) {
	traits := DefaultMakerTraits()
	assert.False(t, traits.HasPostInteraction())

	traits.EnablePostInteraction()
	assert.True(t, traits.HasPostInteraction())

	traits.DisablePostInteraction()
	assert.False(t, traits.HasPostInteraction())
}

func TestMakerTraits_Permit2(t *testing.T) {
	traits := DefaultMakerTraits()
	assert.False(t, traits.IsPermit2())

	traits.EnablePermit2()
	assert.True(t, traits.IsPermit2())

	traits.DisablePermit2()
	assert.False(t, traits.IsPermit2())
}

func TestMakerTraits_NativeUnwrap(t *testing.T) {
	traits := DefaultMakerTraits()
	assert.False(t, traits.IsNativeUnwrapEnabled())

	traits.EnableNativeUnwrap()
	assert.True(t, traits.IsNativeUnwrapEnabled())

	traits.DisableNativeUnwrap()
	assert.False(t, traits.IsNativeUnwrapEnabled())
}

func TestMakerTraits_IsExpired(t *testing.T) {
	traits := DefaultMakerTraits()

	// Test with no expiration set
	assert.False(t, traits.IsExpired(1704279454)) // current timestamp

	// Test with future expiration
	futureTime := int64(1704279454 + 3600) // current time + 1 hour
	traits.WithExpiration(big.NewInt(futureTime))
	assert.False(t, traits.IsExpired(1704279454))

	// Test with past expiration
	pastTime := int64(1704279454 - 3600) // current time - 1 hour
	traits.WithExpiration(big.NewInt(pastTime))
	assert.True(t, traits.IsExpired(1704279454))

	// Test at exact expiration time
	exactTime := int64(1704279454)
	traits.WithExpiration(big.NewInt(exactTime))
	assert.False(t, traits.IsExpired(exactTime))  // should not be expired at exact time
	assert.True(t, traits.IsExpired(exactTime+1)) // should be expired one second later
}

func TestMakerTraits_All(t *testing.T) {
	traits := DefaultMakerTraits().
		WithAllowedSender(common.BigToAddress(UINT_160_MAX)).
		AllowPartialFills().
		AllowMultipleFills().
		WithEpoch(UINT_40_MAX, UINT_40_MAX).
		WithExpiration(UINT_40_MAX).
		WithExtension().
		EnablePermit2().
		EnableNativeUnwrap().
		EnablePreInteraction().
		EnablePostInteraction()

	expected := "5f800000000000ffffffffffffffffffffffffffffffffffffffffffffffffff"
	assert.Equal(t, expected, traits.Build().Text(16))
}
