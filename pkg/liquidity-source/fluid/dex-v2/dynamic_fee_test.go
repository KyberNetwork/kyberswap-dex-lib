package dexv2

import (
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func dynamicSnapshot(t *testing.T, factor uint64) (*PoolSimulator, entity.Pool) {
	t.Helper()
	_, p := snapshotSimulator(t)
	var e Extra
	if err := json.Unmarshal([]byte(p.Extra), &e); err != nil {
		t.Fatal(err)
	}
	var packed uint256.Int
	if packed.SetFromBig(e.DexVariables2) {
		t.Fatal("invalid packed state")
	}
	putPackedBits(&packed, 152, 4, 1)
	putPackedBits(&packed, 156, 12, 600)
	putPackedBits(&packed, 168, 8, factor)
	putPackedBits(&packed, 176, 16, 100)
	putPackedBits(&packed, 192, 16, 1000)
	putPackedBits(&packed, 208, 1, 1)
	putPackedBits(&packed, 209, 20, 0)
	putPackedBits(&packed, 229, 15, e.SnapshotTimestamp&32767)
	putPackedBits(&packed, 244, 12, 0)
	e.DexVariables2 = packed.ToBig()
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	p.Extra = string(b)
	sim, err := NewPoolSimulator(p, valueobject.ChainIDEthereum)
	if err != nil {
		t.Fatal(err)
	}
	return sim, p
}

func TestDynamicConstantFeeMatchesStaticAndPreservesClone(t *testing.T) {
	static, p := snapshotSimulator(t)
	dynamic, _ := dynamicSnapshot(t, 0)
	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[0].Address, Amount: big.NewInt(1_000_000_000)}, TokenOut: p.Tokens[1].Address}
	original, err := static.CalcAmountOut(params)
	if err != nil {
		t.Fatal(err)
	}
	quote, err := dynamic.CalcAmountOut(params)
	if err != nil {
		t.Fatal(err)
	}
	if original.TokenAmountOut.Amount.Cmp(quote.TokenAmountOut.Amount) != 0 {
		t.Fatalf("dynamic constant %s static %s", quote.TokenAmountOut.Amount, original.TokenAmountOut.Amount)
	}
	before := dynamic.V3Pool.dynamicPacked
	repeated, err := dynamic.CalcAmountOut(params)
	if err != nil {
		t.Fatal(err)
	}
	if quote.TokenAmountOut.Amount.Cmp(repeated.TokenAmountOut.Amount) != 0 || !before.Eq(&dynamic.V3Pool.dynamicPacked) {
		t.Fatal("quote mutated dynamic state")
	}
	clone := dynamic.CloneState().(*PoolSimulator)
	dynamic.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: params.TokenAmountIn, TokenAmountOut: *quote.TokenAmountOut, SwapInfo: quote.SwapInfo})
	if before.Eq(&dynamic.V3Pool.dynamicPacked) {
		t.Fatal("dynamic state not updated")
	}
	if !before.Eq(&clone.V3Pool.dynamicPacked) {
		t.Fatal("clone dynamic state changed")
	}
}

func TestDynamicFeeDecayAndTimestampWrap(t *testing.T) {
	var packed, sqrt uint256.Int
	sqrt.SetOne().Lsh(&sqrt, 96)
	putPackedBits(&packed, 156, 12, 600)
	putPackedBits(&packed, 168, 8, 4)
	putPackedBits(&packed, 176, 16, 100)
	putPackedBits(&packed, 192, 16, 1000)
	putPackedBits(&packed, 208, 1, 0)
	putPackedBits(&packed, 209, 20, 1000)
	putPackedBits(&packed, 229, 15, 32760)
	putPackedBits(&packed, 244, 12, 100)
	d, err := prepareDynamicFee(&packed, &sqrt, true, 12)
	if err != nil {
		t.Fatal(err)
	}
	if got := packedBits(&d.packed, 209, 20); got != 800 {
		t.Fatalf("decayed impact %d", got)
	}
	if got := packedBits(&d.packed, 244, 12); got != 80 {
		t.Fatalf("remaining %d", got)
	}
	if got := packedBits(&d.packed, 229, 15); got != 12 {
		t.Fatalf("timestamp %d", got)
	}
	d, err = prepareDynamicFee(&packed, &sqrt, true, 120)
	if err != nil {
		t.Fatal(err)
	}
	if packedBits(&d.packed, 209, 20) != 0 || packedBits(&d.packed, 244, 12) != 0 {
		t.Fatal("expired impact persisted")
	}
}

// Each output and packed post-state was independently checked through the
// deployed DEX and router on a Tenderly fork at the fixture block and timestamp.
func TestDynamicOracleFixture(t *testing.T) {
	sim, p := dynamicSnapshot(t, 20)
	putPackedBits(&sim.V3Pool.dynamicPacked, 176, 16, 10)
	putPackedBits(&sim.V3Pool.dynamicPacked, 192, 16, 200)
	forward := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[0].Address, Amount: big.NewInt(1_000_000_000)}, TokenOut: p.Tokens[1].Address}
	reverse := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[1].Address, Amount: big.NewInt(10_000_000)}, TokenOut: p.Tokens[0].Address}
	check := func(params pool.CalcAmountOutParams, wantOut int64, wantState string) {
		t.Helper()
		q, err := sim.CalcAmountOut(params)
		if err != nil {
			t.Fatal(err)
		}
		if q.TokenAmountOut.Amount.Cmp(big.NewInt(wantOut)) != 0 {
			t.Fatalf("output %s want %d", q.TokenAmountOut.Amount, wantOut)
		}
		si := q.SwapInfo.(SwapInfo)
		got := fmt.Sprintf("%064x", si.nextDynamicPacked.ToBig())
		if got != wantState {
			t.Fatalf("packed state %s want %s", got, wantState)
		}
	}
	check(forward, 1000431588, "258dbd60053c00c8000a1425810000000000000003f1e24b8aedafd980000000")
	check(reverse, 9991086, "258dbd60001700c8000a14258100000000000000015e38a1491fd89980000000")
	// This swap crosses both kinks and exercises all three fee zones.
	putPackedBits(&sim.V3Pool.dynamicPacked, 168, 8, 10)
	putPackedBits(&sim.V3Pool.dynamicPacked, 176, 16, 5)
	putPackedBits(&sim.V3Pool.dynamicPacked, 192, 16, 20)
	check(forward, 1000435451, "258dbd60053c001400050a25810000000000000003f1e24b8aedafd980000000")
}

func TestFeeLegsMatchSwapOutput(t *testing.T) {
	for _, factor := range []uint64{0, 20} {
		sim, _ := dynamicSnapshot(t, factor)
		var amount int256.Int
		amount.SetFromBig(big.NewInt(1_000_000_000_000))
		var priceLimit uint256.Int
		if err := sim.GetSqrtPriceLimit(true, &priceLimit); err != nil {
			t.Fatal(err)
		}
		result, err := sim.V3Pool.GetOutputAmountV2(&amount, true, &priceLimit)
		if err != nil {
			t.Fatal(err)
		}
		var net, fees, sum uint256.Int
		if net.SetFromBig(result.ReturnedAmount.ToBig()) {
			t.Fatal("output overflow")
		}
		fees.Add(&result.protocolFeeOut, &result.lpFeeOut)
		sum.Add(&net, &fees)
		if !sum.Eq(&result.grossOut) {
			t.Fatalf("factor %d gross %s net %s fees %s", factor, &result.grossOut, &net, &fees)
		}
	}
}

func TestDynamicFeeSignedZero(t *testing.T) {
	var packed, sqrt uint256.Int
	sqrt.SetOne().Lsh(&sqrt, 96)
	putPackedBits(&packed, 209, 20, 1)
	putPackedBits(&packed, 229, 15, 10)
	putPackedBits(&packed, 244, 12, 2)
	d, err := prepareDynamicFee(&packed, &sqrt, true, 11)
	if err != nil {
		t.Fatal(err)
	}
	if packedBits(&d.packed, 209, 20) != 0 || packedBits(&d.packed, 208, 1) != 1 {
		t.Fatal("negative impact rounded to zero must store positive zero")
	}
	final, err := finishDynamicFee(&d, &sqrt)
	if err != nil {
		t.Fatal(err)
	}
	if packedBits(&final, 209, 20) != 0 || packedBits(&final, 208, 1) != 1 {
		t.Fatal("zero final impact must store positive zero")
	}
}

func TestDynamicFeeFinishResetsDecayOnSignChange(t *testing.T) {
	var packed, sqrt, up uint256.Int
	sqrt.SetOne().Lsh(&sqrt, 96)
	putPackedBits(&packed, 156, 12, 600)
	putPackedBits(&packed, 208, 1, 0)
	putPackedBits(&packed, 209, 20, 1000)
	putPackedBits(&packed, 229, 15, 20)
	putPackedBits(&packed, 244, 12, 100)
	d, err := prepareDynamicFee(&packed, &sqrt, false, 20)
	if err != nil {
		t.Fatal(err)
	}
	// Anchor lies above the current price for a negative initial impact.
	up.Mul(&sqrt, uint256.NewInt(1001)).Div(&up, uint256.NewInt(1000))
	state, err := finishDynamicFee(&d, &up)
	if err != nil {
		t.Fatal(err)
	}
	if packedBits(&state, 208, 1) != 1 || packedBits(&state, 244, 12) != 600 {
		t.Fatalf("sign or decay incorrect: %x", state.ToBig())
	}
}

// Both directions were verified against the deployed router with a fee-v1
// storage override at this snapshot block and timestamp.
func TestDynamicD4OracleFixture(t *testing.T) {
	b, err := os.ReadFile("testdata/ethereum_d4_dynamic_snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Extra       Extra
		StaticExtra StaticExtra
		Address     string
		Tokens      []string
	}
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	extra, err := json.Marshal(f.Extra)
	if err != nil {
		t.Fatal(err)
	}
	static, err := json.Marshal(f.StaticExtra)
	if err != nil {
		t.Fatal(err)
	}
	p := entity.Pool{Address: f.Address, Extra: string(extra), StaticExtra: string(static), Tokens: []*entity.PoolToken{{Address: f.Tokens[0], Decimals: 6}, {Address: f.Tokens[1], Decimals: 6}}, Reserves: []string{"10000000000", "10000000000"}}
	sim, err := NewPoolSimulator(p, valueobject.ChainIDEthereum)
	if err != nil {
		t.Fatal(err)
	}
	// The deployed D4 swap event reports 1,010 raw USDT LP fee and zero
	// protocol fee for the forward quote; fees convert separately from output.
	c, err := _calculateVars(f.Extra.DexVariables2, f.Extra.Token0ExchangePricesAndConfig, f.Extra.Token1ExchangePricesAndConfig, f.Extra.SnapshotTimestamp)
	if err != nil {
		t.Fatal(err)
	}
	adjusted := amountToAdjusted(big.NewInt(100_000_000), c.Token0NumeratorPrecision, c.Token0DenominatorPrecision, c.Token0BorrowExchangePrice)
	var normalized uint256.Int
	if normalized.SetFromBig(adjusted) {
		t.Fatal("input overflow")
	}
	roundNormalizedDown(&normalized)
	var signed int256.Int
	if signed.SetFromBig(normalized.ToBig()) {
		t.Fatal("input signed overflow")
	}
	var limit uint256.Int
	if err := sim.GetSqrtPriceLimit(true, &limit); err != nil {
		t.Fatal(err)
	}
	result, err := sim.V3Pool.GetOutputAmountV2(&signed, true, &limit)
	if err != nil {
		t.Fatal(err)
	}
	lpFee := adjustedToAmount(result.lpFeeOut.ToBig(), c.Token1NumeratorPrecision, c.Token1DenominatorPrecision, c.Token1BorrowExchangePrice)
	if lpFee.Cmp(big.NewInt(1010)) != 0 || !result.protocolFeeOut.IsZero() {
		t.Fatalf("fee legs: LP %s protocol %s", lpFee, &result.protocolFeeOut)
	}
	for _, tc := range []struct {
		in           int
		amount, want int64
	}{{0, 100_000_000, 99_971_351}, {1, 10_000_000, 10_001_696}} {
		q, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[tc.in].Address, Amount: big.NewInt(tc.amount)}, TokenOut: p.Tokens[1-tc.in].Address})
		if err != nil {
			t.Fatal(err)
		}
		if q.TokenAmountOut.Amount.Cmp(big.NewInt(tc.want)) != 0 {
			t.Fatalf("direction %d output %s want %d", tc.in, q.TokenAmountOut.Amount, tc.want)
		}
	}
}

// Tenderly 0bc5abd6-637e-4abe-a1db-43c10edfb6f0 exercises both a
// nonzero protocol fee and a 63% cut of LP fees on the deployed D4 module.
func TestD4ProtocolCutOracle(t *testing.T) {
	b, err := os.ReadFile("testdata/ethereum_d4_dynamic_snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Extra       Extra
		StaticExtra StaticExtra
		Address     string
		Tokens      []string
	}
	if err = json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	var packed uint256.Int
	if packed.SetFromBig(f.Extra.DexVariables2) {
		t.Fatal("packed state overflow")
	}
	putPackedBits(&packed, 24, 6, 63)
	putPackedBits(&packed, 0, 12, 123)
	f.Extra.DexVariables2 = packed.ToBig()
	extra, err := json.Marshal(f.Extra)
	if err != nil {
		t.Fatal(err)
	}
	static, err := json.Marshal(f.StaticExtra)
	if err != nil {
		t.Fatal(err)
	}
	p := entity.Pool{Address: f.Address, Extra: string(extra), StaticExtra: string(static), Tokens: []*entity.PoolToken{{Address: f.Tokens[0], Decimals: 6}, {Address: f.Tokens[1], Decimals: 6}}, Reserves: []string{"10000000000", "10000000000"}}
	sim, err := NewPoolSimulator(p, valueobject.ChainIDEthereum)
	if err != nil {
		t.Fatal(err)
	}
	c, err := _calculateVars(f.Extra.DexVariables2, f.Extra.Token0ExchangePricesAndConfig, f.Extra.Token1ExchangePricesAndConfig, f.Extra.SnapshotTimestamp)
	if err != nil {
		t.Fatal(err)
	}
	adjusted := amountToAdjusted(big.NewInt(100_000_000), c.Token0NumeratorPrecision, c.Token0DenominatorPrecision, c.Token0BorrowExchangePrice)
	var amount uint256.Int
	if amount.SetFromBig(adjusted) {
		t.Fatal("input overflow")
	}
	roundNormalizedDown(&amount)
	var signed int256.Int
	if signed.SetFromBig(amount.ToBig()) {
		t.Fatal("signed input overflow")
	}
	var limit uint256.Int
	if err = sim.GetSqrtPriceLimit(true, &limit); err != nil {
		t.Fatal(err)
	}
	r, err := sim.V3Pool.GetOutputAmountV2(&signed, true, &limit)
	if err != nil {
		t.Fatal(err)
	}
	for _, leg := range []struct {
		amount *uint256.Int
		want   int64
	}{{&r.protocolFeeOut, 12933}, {&r.lpFeeOut, 373}} {
		got := adjustedToAmount(leg.amount.ToBig(), c.Token1NumeratorPrecision, c.Token1DenominatorPrecision, c.Token1BorrowExchangePrice)
		if got.Cmp(big.NewInt(leg.want)) != 0 {
			t.Fatalf("fee %s want %d", got, leg.want)
		}
	}
	q, err := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[0].Address, Amount: big.NewInt(100_000_000)}, TokenOut: p.Tokens[1].Address})
	if err != nil {
		t.Fatal(err)
	}
	if q.TokenAmountOut.Amount.Cmp(big.NewInt(99_959_055)) != 0 {
		t.Fatalf("output %s", q.TokenAmountOut.Amount)
	}
}
