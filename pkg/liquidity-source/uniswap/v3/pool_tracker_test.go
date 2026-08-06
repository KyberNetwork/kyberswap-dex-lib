package uniswapv3

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// word encodes v as a left-padded 32-byte big-endian word, the same shape both indexed
// topics and non-indexed data words take for value types.
func word(v int64) common.Hash {
	var h common.Hash
	if v < 0 {
		// two's complement over 256 bits
		b := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(v))
		b.FillBytes(h[:])
		return h
	}
	big.NewInt(v).FillBytes(h[:])
	return h
}

func addrWord(addr common.Address) common.Hash {
	var h common.Hash
	copy(h[12:], addr[:])
	return h
}

// TestExtractEventData_MintWithIndex uses a real ramses-v2 static-fee ("V3") pool Mint log,
// which inserts a non-indexed `index` field before `amount` in Data - the case that makes
// Mint need two topic0-keyed decode paths instead of one.
// https://arbiscan.io/tx/... (fixture carried over from the pre-merge ramsesv2 tracker tests)
func TestExtractEventData_MintWithIndex(t *testing.T) {
	t.Parallel()

	event := ethtypes.Log{
		Address: common.HexToAddress("0xee02e3a3034e9ef3bd569b140bc9911fcf1ba067"),
		Topics: []common.Hash{
			mintWithIndexEventID,
			common.HexToHash("0x000000000000000000000000b3f77c5134d643483253d22e0ca24627ae42ed51"),
			common.HexToHash("0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc57e8"),
			common.HexToHash("0xfffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc57f2"),
		},
		Data: common.FromHex("0x000000000000000000000000b3f77c5134d643483253d22e0ca24627ae42ed51000000000000000000000000000000000000000000000000000000000000247d000000000000000000000000000000000000000000000000006195d80fdba6de0000000000000000000000000000000000000000000000001e709140f362c0ad0000000000000000000000000000000000000000000000000000000000000000"),
	}

	tr := &Tracker{}
	lower, upper, delta, err := tr.extractEventData(event)
	require.NoError(t, err)

	assert.Equal(t, -239640, lower)
	assert.Equal(t, -239630, upper)
	// amount sits at word index 2 here (sender, index, amount, ...), not word index 1
	// (sender, amount, ...) like the standard Mint shape - this is exactly what would
	// silently read the `index` field as if it were the liquidity delta if the two Mint
	// shapes were not decoded via distinct topic0s.
	assert.Equal(t, big.NewInt(27467827952461534), delta)
}

func TestExtractEventData_MintStandard(t *testing.T) {
	t.Parallel()

	sender := common.HexToAddress("0x1111111111111111111111111111111111111111")
	owner := common.HexToAddress("0x2222222222222222222222222222222222222222")

	event := ethtypes.Log{
		Address: common.HexToAddress("0x3333333333333333333333333333333333333333"),
		Topics: []common.Hash{
			mintEventID,
			addrWord(owner),
			word(-100),
			word(200),
		},
		Data: append(append(append(
			addrWord(sender).Bytes(),
			word(555).Bytes()...),
			word(0).Bytes()...),
			word(0).Bytes()...),
	}

	tr := &Tracker{}
	lower, upper, delta, err := tr.extractEventData(event)
	require.NoError(t, err)

	assert.Equal(t, -100, lower)
	assert.Equal(t, 200, upper)
	assert.Equal(t, big.NewInt(555), delta)
}

func TestExtractEventData_Burn(t *testing.T) {
	t.Parallel()

	event := ethtypes.Log{
		Address: common.HexToAddress("0x3333333333333333333333333333333333333333"),
		Topics: []common.Hash{
			burnEventID,
			addrWord(common.HexToAddress("0x2222222222222222222222222222222222222222")),
			word(-100),
			word(200),
		},
		Data: append(append(
			word(555).Bytes(),
			word(0).Bytes()...),
			word(0).Bytes()...),
	}

	tr := &Tracker{}
	lower, upper, delta, err := tr.extractEventData(event)
	require.NoError(t, err)

	assert.Equal(t, -100, lower)
	assert.Equal(t, 200, upper)
	// Burn amount is negated: it represents liquidity leaving the tick range.
	assert.Equal(t, big.NewInt(-555), delta)
}

func TestResolveFee(t *testing.T) {
	t.Parallel()

	fee := big.NewInt(3000)
	currentFee := big.NewInt(500)
	slot0Fee := big.NewInt(10000)

	t.Run("currentFee wins over fee when both succeed - e.g. ramses-v2 dynamic/nuri-v2, whose fee() is a stale deploy-time value", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, currentFee, resolveFee(fee, currentFee, nil))
	})

	t.Run("fee used when currentFee reverts - static pools, and slipstream's single always-current fee()", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, fee, resolveFee(fee, nil, nil))
	})

	t.Run("slot0 fee used when neither method exists - solidly-v3", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, slot0Fee, resolveFee(nil, nil, slot0Fee))
	})

	t.Run("zero when nothing resolved", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, zeroBI, resolveFee(nil, nil, nil))
	})
}

func TestResolveSlot0(t *testing.T) {
	t.Parallel()

	sqrtP := big.NewInt(12345)
	tick := big.NewInt(-100)

	t.Run("standard shape wins when it decodes", func(t *testing.T) {
		t.Parallel()
		std := slot0RawStandard{SqrtPriceX96: sqrtP, Tick: tick, Unlocked: true}
		got, err := resolveSlot0(std, slot0RawSlipstream{}, slot0RawSolidly{})
		require.NoError(t, err)
		assert.Equal(t, sqrtP, got.SqrtPriceX96)
		assert.Nil(t, got.Fee)
	})

	t.Run("slipstream shape used when standard did not decode", func(t *testing.T) {
		t.Parallel()
		slip := slot0RawSlipstream{SqrtPriceX96: sqrtP, Tick: tick, Unlocked: true}
		got, err := resolveSlot0(slot0RawStandard{}, slip, slot0RawSolidly{})
		require.NoError(t, err)
		assert.Equal(t, sqrtP, got.SqrtPriceX96)
		assert.Nil(t, got.Fee)
	})

	t.Run("solidly shape carries Fee - only trusted when it's the one that actually decoded", func(t *testing.T) {
		t.Parallel()
		fee := big.NewInt(10000)
		solid := slot0RawSolidly{SqrtPriceX96: sqrtP, Tick: tick, Unlocked: true, Fee: fee}
		got, err := resolveSlot0(slot0RawStandard{}, slot0RawSlipstream{}, solid)
		require.NoError(t, err)
		assert.Equal(t, sqrtP, got.SqrtPriceX96)
		assert.Equal(t, fee, got.Fee)
	})

	t.Run("errors when nothing decoded", func(t *testing.T) {
		t.Parallel()
		_, err := resolveSlot0(slot0RawStandard{}, slot0RawSlipstream{}, slot0RawSolidly{})
		assert.Error(t, err)
	})
}
