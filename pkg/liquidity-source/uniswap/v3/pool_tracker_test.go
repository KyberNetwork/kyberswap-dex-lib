package uniswapv3

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
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
		got, err := resolveSlot0(slot0RawKatana{}, std, slot0RawSlipstream{}, slot0RawSolidly{})
		require.NoError(t, err)
		assert.Equal(t, sqrtP, got.SqrtPriceX96)
		assert.Nil(t, got.Fee)
	})

	t.Run("katana shape used when standard did not decode", func(t *testing.T) {
		t.Parallel()
		kat := slot0RawKatana{SqrtPriceX96: sqrtP, Tick: tick, Unlocked: true}
		got, err := resolveSlot0(kat, slot0RawStandard{}, slot0RawSlipstream{}, slot0RawSolidly{})
		require.NoError(t, err)
		assert.Equal(t, sqrtP, got.SqrtPriceX96)
		assert.Nil(t, got.Fee)
	})

	t.Run("slipstream shape used when standard and katana did not decode", func(t *testing.T) {
		t.Parallel()
		slip := slot0RawSlipstream{SqrtPriceX96: sqrtP, Tick: tick, Unlocked: true}
		got, err := resolveSlot0(slot0RawKatana{}, slot0RawStandard{}, slip, slot0RawSolidly{})
		require.NoError(t, err)
		assert.Equal(t, sqrtP, got.SqrtPriceX96)
		assert.Nil(t, got.Fee)
	})

	t.Run("solidly shape carries Fee - only trusted when it's the one that actually decoded", func(t *testing.T) {
		t.Parallel()
		fee := big.NewInt(10000)
		solid := slot0RawSolidly{SqrtPriceX96: sqrtP, Tick: tick, Unlocked: true, Fee: fee}
		got, err := resolveSlot0(slot0RawKatana{}, slot0RawStandard{}, slot0RawSlipstream{}, solid)
		require.NoError(t, err)
		assert.Equal(t, sqrtP, got.SqrtPriceX96)
		assert.Equal(t, fee, got.Fee)
	})

	t.Run("errors when nothing decoded", func(t *testing.T) {
		t.Parallel()
		_, err := resolveSlot0(slot0RawKatana{}, slot0RawStandard{}, slot0RawSlipstream{}, slot0RawSolidly{})
		assert.Error(t, err)
	})
}

// TestStatelessPoolsAreScheduled: a pool created from its factory log whose lens read the guard
// declined is due again as soon as the guard would take it, at once if not tried, after the
// cooldown if tried; any other pool needs no scheduled update.
func TestStatelessPoolsAreScheduled(t *testing.T) {
	t.Parallel()

	tr, err := NewTracker(&Config{UseLens: true, LensGuard: ephemeral.GuardConfig{CooldownSecs: 30}}, nil, nil)
	require.NoError(t, err)
	before := time.Now().Unix()
	at := tr.NextUpdateAt(entity.Pool{Extra: `{}`})
	assert.GreaterOrEqual(t, at, before)
	assert.LessOrEqual(t, at, time.Now().Unix(), "a pool not tried yet is due at once")

	_, _ = tr.lensGuard.Try(t.Context(), "", func(context.Context) error { return nil })
	at = tr.NextUpdateAt(entity.Pool{Extra: `{}`})
	assert.GreaterOrEqual(t, at, before+30)
	assert.LessOrEqual(t, at, time.Now().Unix()+30, "a pool just tried waits out its cooldown")
	assert.Zero(t, tr.NextUpdateAt(entity.Pool{Extra: `{"liquidity":0,"sqrtPriceX96":1}`}))

	plain, err := NewTracker(&Config{}, nil, nil)
	require.NoError(t, err)
	assert.Zero(t, plain.NextUpdateAt(entity.Pool{Extra: `{}`}))
}

// TestLogsAtOrBelowTheLensBlockAreLate: a Mint at or below the lens snapshot's block is late, read
// as the absolute values of the ticks it touches rather than added. It may be one the snapshot holds,
// replayed by a backfill, which adding would count twice without changing the sum of liquidityNet;
// or one a reorg brought in below a snapshot read on the abandoned fork, with no log of this pool
// reverted, which dropping would lose.
func TestLogsAtOrBelowTheLensBlockAreLate(t *testing.T) {
	t.Parallel()

	pool := common.HexToAddress("0x3333333333333333333333333333333333333333")
	at := func(block uint64) ethtypes.Log { return ethtypes.Log{Address: pool, BlockNumber: block} }
	extra, err := json.Marshal(Extra{LensBlock: 100})
	require.NoError(t, err)
	p, err := tickspkg.NewTicksBasedPool(entity.Pool{Address: strings.ToLower(pool.Hex()), Extra: string(extra)})
	require.NoError(t, err)

	held := p.Holds([]ethtypes.Log{at(99), at(110)}, 0)
	assert.Equal(t, uint64(100), held, "the snapshot is newer than the pool's block")
	assert.True(t, hasLateLog([]ethtypes.Log{at(99), at(110)}, held), "the reorged-in or replayed log at 99")
	assert.False(t, hasLateLog([]ethtypes.Log{at(110)}, held), "one after the snapshot applies")
}

// TestLateLogs: a log at or below the pool's block, already applied or a retry's, is read as the
// absolute values of the ticks it touches; one above it, the removed one of a reorg, and the empty
// one that triggers an update, are not late.
func TestLateLogs(t *testing.T) {
	t.Parallel()

	pool := common.HexToAddress("0x3333333333333333333333333333333333333333")
	at := func(block uint64) ethtypes.Log { return ethtypes.Log{Address: pool, BlockNumber: block} }

	assert.True(t, hasLateLog([]ethtypes.Log{at(101), at(100)}, 100))
	assert.True(t, hasLateLog([]ethtypes.Log{at(90)}, 100))
	assert.False(t, hasLateLog([]ethtypes.Log{at(101)}, 100))
	assert.False(t, hasLateLog([]ethtypes.Log{{Address: pool, BlockNumber: 90, Removed: true}}, 100))
	assert.False(t, hasLateLog([]ethtypes.Log{{}, at(101)}, 100))
	assert.False(t, hasLateLog([]ethtypes.Log{at(90)}, 0), "a pool with no block has seen no log")
}

// TestResetPoolState: with the lens a reset pool is one without state, read whole on its next
// update; without it the pool is kept, as nothing could read it whole.
func TestResetPoolState(t *testing.T) {
	t.Parallel()

	p := entity.Pool{Extra: `{"liquidity":7,"sqrtPriceX96":1,"ticks":[]}`}
	lens, err := NewTracker(&Config{UseLens: true}, nil, nil)
	require.NoError(t, err)
	got, ok := lens.ResetPoolState(p, 100)
	assert.True(t, ok)
	assert.False(t, tickspkg.HasState(got))
	assert.NotZero(t, lens.NextUpdateAt(got), "it is due for its lens read")

	plain, err := NewTracker(&Config{}, nil, nil)
	require.NoError(t, err)
	got, ok = plain.ResetPoolState(p, 100)
	assert.False(t, ok)
	assert.Equal(t, p, got)
}
