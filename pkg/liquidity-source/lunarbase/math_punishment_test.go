package lunarbase

import (
	"testing"

	"github.com/holiman/uint256"
)

// bscPunishmentParams reproduces live state pinned at BSC block 119902916 for
// pool 0x00007904d186680C709519e71f4Dc3e2DF8f1b99, which has upgraded to the
// lunarbase-pmm-math v0.4.0 linear-anchor / directional-punishment model
// (concentrationK() reverts on-chain; maxPunishmentX24() returns 83886).
func bscPunishmentParams() *PoolParams {
	return &PoolParams{
		SqrtPriceX96:     uint256.MustFromDecimal("2124565750896485315338783686656"),
		FeeAskX24:        7975,
		FeeBidX24:        2188,
		ReserveX:         uint256.MustFromDecimal("39134580821500176234"),
		ReserveY:         uint256.MustFromDecimal("42770804199297762732014"),
		ConcentrationK:   0,
		MaxPunishmentX24: 83886,
	}
}

// TestPunishmentQuoteMatchesOnChain cross-checks quoteXToYInto against the
// same swap replayed by hand against on-chain `quoteXToY(dx)` at the pinned
// block, with the caller's fee multiplier (blacklistFeeMultiplier=4 for a
// non-whitelisted caller) divided back out to isolate the base (multiplier=1)
// math this package implements.
func TestPunishmentQuoteMatchesOnChain(t *testing.T) {
	params := bscPunishmentParams()
	dx := uint256.MustFromDecimal("1000000000000000000")

	result := quoteXToY(params, dx)

	if got, want := result.AmountOut.Dec(), "718956327424094777629"; got != want {
		t.Errorf("amountOut = %s, want %s", got, want)
	}
	if got, want := result.Fee.Dec(), "130254275905269392"; got != want {
		t.Errorf("fee = %s, want %s", got, want)
	}
	if got, want := result.SqrtPriceNext.Dec(), "2124565750896485315338783686656"; got != want {
		t.Errorf("sqrtPriceNext = %s, want %s", got, want)
	}
	if got, want := params.FeeBidX24, uint32(3039); got != want {
		t.Errorf("post-swap FeeBidX24 = %d, want %d (punishment must persist onto the pool params)", got, want)
	}
	if got, want := params.FeeAskX24, uint32(7975); got != want {
		t.Errorf("post-swap FeeAskX24 = %d, want %d (opposite direction must not move)", got, want)
	}
}

// TestPunishmentSaturatesOnlySameDirection ports mechanism.rs's
// punishment_saturates_only_same_direction: applying a punishment near the
// uint24 ceiling must saturate at MAX_U24 on the swapped direction and leave
// the other direction's fee untouched.
func TestPunishmentSaturatesOnlySameDirection(t *testing.T) {
	feeAsk, feeBid := uint32(7), uint32(maxU24-5)
	effective := applyPunishment(&feeAsk, &feeBid, 100, true)

	if effective != maxU24 {
		t.Errorf("effective fee = %d, want %d", effective, maxU24)
	}
	if feeBid != maxU24 {
		t.Errorf("feeBid = %d, want %d", feeBid, maxU24)
	}
	if feeAsk != 7 {
		t.Errorf("feeAsk = %d, want unchanged 7", feeAsk)
	}
}

// TestPunishmentSplitYieldsMoreOutput ports mechanism.rs's
// full_immediate_punishment_split_regression_matches_solidity: splitting one
// large swap into chunks through a punishment pool, persisting the ratcheting
// fee between chunks via UpdateBalance's contract (mutate-in-place
// PoolParams), must yield strictly more total output than a single swap of
// the same size. If UpdateBalance fails to persist the punishment, router
// split-planning would over-estimate output and under-deliver on execution.
func TestPunishmentSplitYieldsMoreOutput(t *testing.T) {
	reserve := uint256.MustFromDecimal("1000000000000000000000000")
	single := &PoolParams{
		SqrtPriceX96:     new(uint256.Int).Lsh(uint256.NewInt(1), 96),
		ReserveX:         new(uint256.Int).Set(reserve),
		ReserveY:         new(uint256.Int).Set(reserve),
		MaxPunishmentX24: maxU24,
	}
	split := &PoolParams{
		SqrtPriceX96:     new(uint256.Int).Set(single.SqrtPriceX96),
		ReserveX:         new(uint256.Int).Set(reserve),
		ReserveY:         new(uint256.Int).Set(reserve),
		MaxPunishmentX24: maxU24,
	}

	total := reserve
	singleResult := quoteXToY(single, total)

	chunk := uint256.MustFromDecimal("100000000000000000000000")
	splitTotalOut := new(uint256.Int)
	for i := 0; i < 10; i++ {
		r := quoteXToY(split, chunk)
		splitTotalOut.Add(splitTotalOut, r.AmountOut)
		split.ReserveX.Add(split.ReserveX, chunk)
		split.ReserveY.Sub(split.ReserveY, new(uint256.Int).Add(r.AmountOut, r.Fee))
	}

	if got, want := singleResult.AmountOut.Dec(), "500000000000000000000000"; got != want {
		t.Fatalf("single amountOut = %s, want %s", got, want)
	}
	if got, want := splitTotalOut.Dec(), "724999934434890747070315"; got != want {
		t.Fatalf("split total amountOut = %s, want %s", got, want)
	}
	if got, want := single.FeeBidX24, uint32(8_388_608); got != want {
		t.Errorf("single post-swap FeeBidX24 = %d, want %d", got, want)
	}
	if got, want := split.FeeBidX24, uint32(8_388_610); got != want {
		t.Errorf("split post-swap FeeBidX24 = %d, want %d", got, want)
	}
	if !splitTotalOut.Gt(singleResult.AmountOut) {
		t.Errorf("split total %s must exceed single-swap amountOut %s", splitTotalOut.Dec(), singleResult.AmountOut.Dec())
	}
}
