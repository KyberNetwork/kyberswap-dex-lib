package biconomyprop

import (
	"math/big"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	testVenue  = "0x000000Da21a0f02b2626874870b6447Db220C1EF"
	testToken0 = "0x4200000000000000000000000000000000000006"
	testToken1 = "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
)

func u(dec string) *uint256.Int {
	v, err := uint256.FromDecimal(dec)
	if err != nil {
		panic(err)
	}
	return v
}

func farExpiry() uint64 { return uint64(time.Now().Add(time.Hour).Unix()) }

// liveBoard builds a board as the venue reports it, with remaining = top size - filled.
func liveBoard(sizes, prices []string, filled string, expiresAt uint64) Board {
	if len(sizes) == 0 || len(sizes) != len(prices) {
		panic("liveBoard: sizes and prices must be non-empty and equal length")
	}
	b := Board{
		Sizes:     make([]*uint256.Int, len(sizes)),
		Prices:    make([]*uint256.Int, len(prices)),
		Filled:    u(filled),
		ExpiresAt: expiresAt,
	}
	for i := range sizes {
		b.Sizes[i] = u(sizes[i])
		b.Prices[i] = u(prices[i])
	}
	top := b.Sizes[len(b.Sizes)-1]
	b.Remaining = new(uint256.Int)
	if b.Filled.Cmp(top) < 0 {
		b.Remaining.Sub(top, b.Filled)
	}
	return b
}

// darkBoard is what the venue returns for a direction a maker does not quote.
func darkBoard() Board {
	return Board{Filled: u("0"), Remaining: u("0")}
}

func buildPool(t *testing.T, members []MemberExtra) *PoolSimulator {
	t.Helper()
	extra, err := json.Marshal(Extra{Members: members})
	require.NoError(t, err)
	staticExtra, err := json.Marshal(StaticExtra{Venue: testVenue})
	require.NoError(t, err)

	sim, err := NewPoolSimulator(entity.Pool{
		Address:     testVenue + "_" + testToken0 + "_" + testToken1,
		Exchange:    "biconomy-prop",
		Type:        DexType,
		Reserves:    entity.PoolReserves{"0", "0"},
		Tokens:      []*entity.PoolToken{{Address: testToken0}, {Address: testToken1}},
		Extra:       string(extra),
		StaticExtra: string(staticExtra),
		BlockNumber: 12345,
	})
	require.NoError(t, err)
	return sim
}

// twoMakers: A quotes 1e18 @ 2000 then 2e18 more @ 1999, B quotes 2e18 @ 2000. Neither quotes
// the reverse direction.
func twoMakers() []MemberExtra {
	exp := farExpiry()
	return []MemberExtra{
		{
			Maker: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Dir0: liveBoard(
				[]string{"1000000000000000000", "3000000000000000000"},
				[]string{"2000000000000000000000", "1999000000000000000000"},
				"0", exp,
			),
			Dir1: darkBoard(),
		},
		{
			Maker: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Dir0: liveBoard(
				[]string{"2000000000000000000"},
				[]string{"2000000000000000000000"},
				"0", exp,
			),
			Dir1: darkBoard(),
		},
	}
}

func calc(t *testing.T, sim *PoolSimulator, amountIn string) (*pool.CalcAmountOutResult, error) {
	t.Helper()
	in, ok := new(big.Int).SetString(amountIn, 10)
	require.True(t, ok)
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken0, Amount: in},
		TokenOut:      testToken1,
	})
}

func calcReverse(t *testing.T, sim *PoolSimulator, amountIn string) (*pool.CalcAmountOutResult, error) {
	t.Helper()
	in, ok := new(big.Int).SetString(amountIn, 10)
	require.True(t, ok)
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken1, Amount: in},
		TokenOut:      testToken0,
	})
}

// Equal prices fill in registry order: 2e18 in takes 1e18 from A, then 1e18 from B.
func TestCalcAmountOut_TieBreaksByRegistryOrder(t *testing.T) {
	sim := buildPool(t, twoMakers())
	res, err := calc(t, sim, "2000000000000000000")
	require.NoError(t, err)
	assert.Equal(t, "4000000000000000000000", res.TokenAmountOut.Amount.String())

	swapInfo, ok := res.SwapInfo.(SwapInfo)
	require.True(t, ok)
	require.Len(t, swapInfo.Takes, 2)
	assert.Equal(t, 0, swapInfo.Takes[0].Member)
	assert.Equal(t, "1000000000000000000", swapInfo.Takes[0].AmountIn.Dec())
	assert.Equal(t, 1, swapInfo.Takes[1].Member)
	assert.Equal(t, "1000000000000000000", swapInfo.Takes[1].AmountIn.Dec())
}

// 1e18@2000 (A) + 2e18@2000 (B) + 1e18@1999 (A) = 7999e18 out for 4e18 in.
func TestCalcAmountOut_SplitsAcrossMakersBestPriceFirst(t *testing.T) {
	sim := buildPool(t, twoMakers())
	res, err := calc(t, sim, "4000000000000000000")
	require.NoError(t, err)
	assert.Equal(t, "7999000000000000000000", res.TokenAmountOut.Amount.String())

	swapInfo := res.SwapInfo.(SwapInfo)
	require.Len(t, swapInfo.Takes, 2)
	assert.Equal(t, "2000000000000000000", swapInfo.Takes[0].AmountIn.Dec()) // member A
	assert.Equal(t, "2000000000000000000", swapInfo.Takes[1].AmountIn.Dec()) // member B
}

// floor(3 * 1333333333333333333 / 1e18) = 3.
func TestCalcAmountOut_FloorsPerSegment(t *testing.T) {
	sim := buildPool(t, []MemberExtra{{
		Maker: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Dir0:  liveBoard([]string{"1000000"}, []string{"1333333333333333333"}, "0", farExpiry()),
		Dir1:  darkBoard(),
	}})
	res, err := calc(t, sim, "3")
	require.NoError(t, err)
	assert.Equal(t, "3", res.TokenAmountOut.Amount.String())
}

// The venue reverts when the book cannot cover amountIn; it never fills partially.
func TestCalcAmountOut_RejectsBeyondDepth(t *testing.T) {
	sim := buildPool(t, twoMakers())
	_, err := calc(t, sim, "5000000000000000001") // total live depth is 5e18
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
}

// B quotes 0.5: one wei past A's depth pays floor(1 * 0.5) = 0, so B is dropped and the size is
// not covered; two wei pay 1.
func TestCalcAmountOut_ZeroOutputAllocationIsNotCoverage(t *testing.T) {
	exp := farExpiry()
	sim := buildPool(t, []MemberExtra{
		{
			Maker: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Dir0:  liveBoard([]string{"1000000000000000000"}, []string{"2000000000000000000000"}, "0", exp),
			Dir1:  darkBoard(),
		},
		{
			Maker: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Dir0:  liveBoard([]string{"1000000000000000000"}, []string{"500000000000000000"}, "0", exp),
			Dir1:  darkBoard(),
		},
	})

	_, err := calc(t, sim, "1000000000000000001") // A's top size + 1 wei
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)

	res, err := calc(t, sim, "1000000000000000002") // A's top size + 2 wei
	require.NoError(t, err)
	assert.Equal(t, "2000000000000000000001", res.TokenAmountOut.Amount.String())
	swapInfo := res.SwapInfo.(SwapInfo)
	require.Len(t, swapInfo.Takes, 2)
	assert.Equal(t, 0, swapInfo.Takes[0].Member)
	assert.Equal(t, "1000000000000000000", swapInfo.Takes[0].AmountIn.Dec())
	assert.Equal(t, 1, swapInfo.Takes[1].Member)
	assert.Equal(t, "2", swapInfo.Takes[1].AmountIn.Dec())

	// Exactly A's top size never touches B and is unaffected by the rule.
	res, err = calc(t, sim, "1000000000000000000")
	require.NoError(t, err)
	assert.Equal(t, "2000000000000000000000", res.TokenAmountOut.Amount.String())
}

// An expired board contributes nothing.
func TestCalcAmountOut_SkipsExpiredBoards(t *testing.T) {
	members := twoMakers()
	members[0].Dir0.ExpiresAt = uint64(time.Now().Add(-time.Minute).Unix())
	sim := buildPool(t, members)

	res, err := calc(t, sim, "2000000000000000000") // only B's 2e18 remains
	require.NoError(t, err)
	assert.Equal(t, "4000000000000000000000", res.TokenAmountOut.Amount.String())
	swapInfo := res.SwapInfo.(SwapInfo)
	require.Len(t, swapInfo.Takes, 1)
	assert.Equal(t, 1, swapInfo.Takes[0].Member)

	_, err = calc(t, sim, "2000000000000000001")
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
}

// An exhausted board (remaining == 0) and dark boards are skipped.
func TestCalcAmountOut_SkipsDarkAndExhaustedBoards(t *testing.T) {
	members := twoMakers()
	// Exhaust A: meter at the top size, so the venue reports remaining 0.
	members[0].Dir0 = liveBoard(
		[]string{"1000000000000000000", "3000000000000000000"},
		[]string{"2000000000000000000000", "1999000000000000000000"},
		"3000000000000000000", farExpiry(),
	)
	require.True(t, members[0].Dir0.Remaining.IsZero())
	sim := buildPool(t, members)

	res, err := calc(t, sim, "2000000000000000000") // only B's 2e18
	require.NoError(t, err)
	assert.Equal(t, "4000000000000000000000", res.TokenAmountOut.Amount.String())
	swapInfo := res.SwapInfo.(SwapInfo)
	require.Len(t, swapInfo.Takes, 1)
	assert.Equal(t, 1, swapInfo.Takes[0].Member)

	_, err = calcReverse(t, sim, "1")
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
}

// Filled depth is skipped.
func TestCalcAmountOut_RespectsFilledCursor(t *testing.T) {
	members := twoMakers()
	members[0].Dir0 = liveBoard( // A's 2000 rung already spent
		[]string{"1000000000000000000", "3000000000000000000"},
		[]string{"2000000000000000000000", "1999000000000000000000"},
		"1000000000000000000", farExpiry(),
	)
	sim := buildPool(t, members)

	res, err := calc(t, sim, "3000000000000000000")
	require.NoError(t, err)
	// B 2e18@2000 (4000e18) + A 1e18@1999 (1999e18)
	assert.Equal(t, "5999000000000000000000", res.TokenAmountOut.Amount.String())
}

// UpdateBalance advances the meters, so the next quote walks the book like a second fill.
func TestUpdateBalance_AdvancesCursors(t *testing.T) {
	sim := buildPool(t, twoMakers())
	res, err := calc(t, sim, "2000000000000000000")
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})

	// A cursor now 1e18 (its 2000 rung gone), B cursor 1e18 (1e18 left @2000).
	res2, err := calc(t, sim, "2000000000000000000")
	require.NoError(t, err)
	// B 1e18@2000 (2000e18) + A 1e18@1999 (1999e18)
	assert.Equal(t, "3999000000000000000000", res2.TokenAmountOut.Amount.String())
}

// Simulated fills can exhaust a board; the tracked Board itself is left unchanged.
func TestUpdateBalance_ExhaustsBoard(t *testing.T) {
	sim := buildPool(t, twoMakers())
	res, err := calc(t, sim, "5000000000000000000") // the whole book
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})

	_, err = calc(t, sim, "1")
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
	assert.Equal(t, "0", sim.Info.Reserves[1].String())
	assert.True(t, sim.members[0].Dir0.Filled.IsZero(), "Board.Filled stays as tracked")
}

// Clones do not share meters with the original.
func TestCloneState_IsIndependent(t *testing.T) {
	sim := buildPool(t, twoMakers())
	clone := sim.CloneState().(*PoolSimulator)

	res, err := calc(t, sim, "2000000000000000000")
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})

	resClone, err := calc(t, clone, "2000000000000000000")
	require.NoError(t, err)
	assert.Equal(t, "4000000000000000000000", resClone.TokenAmountOut.Amount.String())

	resOrig, err := calc(t, sim, "2000000000000000000")
	require.NoError(t, err)
	assert.Equal(t, "3999000000000000000000", resOrig.TokenAmountOut.Amount.String())
}

func TestDeterminism(t *testing.T) {
	simA := buildPool(t, twoMakers())
	simB := buildPool(t, twoMakers())
	for i := 0; i < 3; i++ {
		resA, errA := calc(t, simA, "1500000000000000000")
		resB, errB := calc(t, simB, "1500000000000000000")
		require.NoError(t, errA)
		require.NoError(t, errB)
		assert.Equal(t, resA.TokenAmountOut.Amount.String(), resB.TokenAmountOut.Amount.String())
		simA.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: resA.SwapInfo})
		simB.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: resB.SwapInfo})
	}
}

func TestGetMetaInfo(t *testing.T) {
	sim := buildPool(t, twoMakers())
	meta, ok := sim.GetMetaInfo(testToken0, testToken1).(MetaInfo)
	require.True(t, ok)
	assert.Equal(t, testVenue, meta.Venue)
	assert.Equal(t, uint64(12345), meta.BlockNumber)
}

// A per-block cap shows up as Remaining below the top level. The venue fills a maker only up to
// Filled + Remaining, so A gives 1.5e18 here and B covers the rest.
func TestCalcAmountOut_BlockCapLimitsMaker(t *testing.T) {
	members := twoMakers()
	members[0].Dir0.Remaining = u("1500000000000000000")
	sim := buildPool(t, members)

	res, err := calc(t, sim, "3500000000000000000")
	require.NoError(t, err)
	// A 1e18@2000 + B 2e18@2000 + A 0.5e18@1999
	assert.Equal(t, "6999500000000000000000", res.TokenAmountOut.Amount.String())
	swapInfo := res.SwapInfo.(SwapInfo)
	assert.Equal(t, "1500000000000000000", swapInfo.Takes[0].AmountIn.Dec())

	_, err = calc(t, sim, "3500000000000000001")
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)
	assert.Equal(t, "6999500000000000000000", deliverable(members, 0, uint64(time.Now().Unix())).Dec())
}

// A maker whose allocation pays out nothing is dropped and the order is allocated again, so a
// third maker can cover the size instead of the quote failing.
func TestCalcAmountOut_DustMakerDroppedAndReallocated(t *testing.T) {
	exp := farExpiry()
	sim := buildPool(t, []MemberExtra{
		{
			Maker: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Dir0:  liveBoard([]string{"1000000000000000000"}, []string{"2000000000000000000000"}, "0", exp),
			Dir1:  darkBoard(),
		},
		{
			Maker: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Dir0:  liveBoard([]string{"1000000000000000000"}, []string{"500000000000000000"}, "0", exp),
			Dir1:  darkBoard(),
		},
		{
			Maker: "0xcccccccccccccccccccccccccccccccccccccccc",
			Dir0:  liveBoard([]string{"1000000000000000000"}, []string{"400000000000000000"}, "0", exp),
			Dir1:  darkBoard(),
		},
	})

	// One wei past A lands on B at 0.5 and pays 0, so B is dropped; C at 0.4 pays 0 too.
	_, err := calc(t, sim, "1000000000000000001")
	assert.ErrorIs(t, err, ErrInsufficientLiquidity)

	// Three wei: B pays floor(3 * 0.5) = 1, so B stays.
	res, err := calc(t, sim, "1000000000000000003")
	require.NoError(t, err)
	assert.Equal(t, "2000000000000000000001", res.TokenAmountOut.Amount.String())
}

func TestCalcAmountOut_DustMakerDroppedOtherMakerCovers(t *testing.T) {
	exp := farExpiry()
	sim := buildPool(t, []MemberExtra{
		{
			Maker: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			Dir0:  liveBoard([]string{"1"}, []string{"500000000000000000"}, "0", exp),
			Dir1:  darkBoard(),
		},
		{
			Maker: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			Dir0:  liveBoard([]string{"10"}, []string{"400000000000000000"}, "0", exp),
			Dir1:  darkBoard(),
		},
	})

	// 5 wei: A takes 1 wei and pays 0, so A is dropped and B covers all 5 for floor(5 * 0.4) = 2.
	res, err := calc(t, sim, "5")
	require.NoError(t, err)
	assert.Equal(t, "2", res.TokenAmountOut.Amount.String())
	swapInfo := res.SwapInfo.(SwapInfo)
	require.Len(t, swapInfo.Takes, 1)
	assert.Equal(t, 1, swapInfo.Takes[0].Member)
}

func TestCalcAmountOut_ProtocolFee(t *testing.T) {
	extra, err := json.Marshal(Extra{Members: twoMakers(), FeeBps: 30})
	require.NoError(t, err)
	staticExtra, err := json.Marshal(StaticExtra{Venue: testVenue})
	require.NoError(t, err)
	sim, err := NewPoolSimulator(entity.Pool{
		Address:     testVenue + "_" + testToken0 + "_" + testToken1,
		Type:        DexType,
		Reserves:    entity.PoolReserves{"0", "0"},
		Tokens:      []*entity.PoolToken{{Address: testToken0}, {Address: testToken1}},
		Extra:       string(extra),
		StaticExtra: string(staticExtra),
	})
	require.NoError(t, err)

	res, err := calc(t, sim, "2000000000000000000")
	require.NoError(t, err)
	// 4000e18 gross, 0.3% to the protocol
	assert.Equal(t, "3988000000000000000000", res.TokenAmountOut.Amount.String())
	assert.Equal(t, "12000000000000000000", res.Fee.Amount.String())
}

func TestCalcAmountOut_GasPerMaker(t *testing.T) {
	sim := buildPool(t, twoMakers())
	one, err := calc(t, sim, "1000000000000000000")
	require.NoError(t, err)
	two, err := calc(t, sim, "2000000000000000000")
	require.NoError(t, err)
	assert.Equal(t, int64(defaultGas), one.Gas)
	assert.Equal(t, int64(defaultGas+perMemberGas), two.Gas)
}
