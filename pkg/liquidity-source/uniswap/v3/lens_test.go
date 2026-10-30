package uniswapv3

// The fixtures are real revert payloads of V3Lens on Ethereum USDC/WETH 0.05%
// (0x88e6a0c2ddd26feeb64f039a2c41296fcb3f5640) at block 26224192, captured with eth_call and
// decoded independently with `cast abi-decode` for the expected values below:
//
//	full.hex     words 70..72, no gas limit
//	partial.hex  words 70..85 under a 600k gas limit: stopped at word 73

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
	"github.com/ethereum/go-ethereum/common"
)

func lensFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/lens/" + name)
	if err != nil {
		t.Fatal(err)
	}
	b, err := hex.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func checkLensState(t *testing.T, r ephemeral.TickSnapshot) {
	t.Helper()
	if r.TickSpacing.Int64() != 10 || r.Tick.Int64() != 198408 || r.BlockNumber.Uint64() != 26224192 {
		t.Errorf("spacing %d, tick %d, block %s", r.TickSpacing, r.Tick, r.BlockNumber)
	}
	if got := r.SqrtPriceX96.String(); got != "1610793979119022655101904473140671" {
		t.Errorf("sqrtPriceX96 %s", got)
	}
	if got := r.Liquidity.String(); got != "2310548941712730099" {
		t.Errorf("liquidity %s", got)
	}
	if len(r.Ticks) != 24 {
		t.Fatalf("%d ticks, want 24", len(r.Ticks))
	}
	first, last := r.Ticks[0], r.Ticks[23]
	if first.Tick.Int64() != 179440 || first.LiquidityNet.String() != "60114662787448" ||
		first.LiquidityGross.String() != "60114662787448" {
		t.Errorf("first tick %+v", first)
	}
	if last.Tick.Int64() != 186450 || last.LiquidityNet.String() != "249188824324359" ||
		last.LiquidityGross.String() != "249189419780545" {
		t.Errorf("last tick %+v", last)
	}
	sum := new(big.Int)
	for _, tk := range r.Ticks {
		sum.Add(sum, tk.LiquidityNet)
	}
	if sum.String() != "11026110364234256" {
		t.Errorf("liquidityNet sums to %s", sum)
	}
}

func TestLensDecodesFullSnapshot(t *testing.T) {
	var r ephemeral.TickSnapshot
	if err := lens.UnpackCall(lensFixture(t, "full.hex"), &r); err != nil {
		t.Fatal(err)
	}
	checkLensState(t, r)
	if !r.Done || r.NextWordPos != 72 {
		t.Errorf("done %v, nextWordPos %d", r.Done, r.NextWordPos)
	}
}

// TestLensDecodesPartialSnapshot: a scan that stopped short reports it and where to resume.
func TestLensDecodesPartialSnapshot(t *testing.T) {
	var r ephemeral.TickSnapshot
	if err := lens.UnpackCall(lensFixture(t, "partial.hex"), &r); err != nil {
		t.Fatal(err)
	}
	checkLensState(t, r)
	if r.Done || r.NextWordPos != 73 {
		t.Errorf("done %v, nextWordPos %d", r.Done, r.NextWordPos)
	}
}

// TestLensRejectsWhatIsNotAResult: each payload here is the real one with one thing wrong, or a
// different kind of revert; reading any of them as a result would hand the tracker made-up state.
func TestLensRejectsWhatIsNotAResult(t *testing.T) {
	good := lensFixture(t, "full.hex")
	with := func(word int, value []byte) []byte {
		b := bytes.Clone(good)
		copy(b[word*32:(word+1)*32], value)
		return b
	}
	word := func(hexWord string) []byte {
		b, _ := hex.DecodeString(strings.Repeat("0", 64-len(hexWord)) + hexWord)
		return b
	}
	reason, _ := hex.DecodeString("08c379a0" +
		"0000000000000000000000000000000000000000000000000000000000000020" +
		"0000000000000000000000000000000000000000000000000000000000000004" +
		"6e6f706500000000000000000000000000000000000000000000000000000000")

	for name, payload := range map[string][]byte{
		"empty":                 nil,
		"head only":             good[:256],
		"trailing word":         append(bytes.Clone(good), make([]byte, 32)...),
		"truncated entry":       good[:len(good)-32],
		"Error(string)":         reason,
		"offset elsewhere":      with(7, word("120")),
		"count too large":       with(8, word("19")),
		"count overflows":       with(8, word("ffffffffffffffffffffffffffffffff")),
		"bool not 0 or 1":       with(6, word("2")),
		"tick past int24":       with(2, word("800000")),
		"cursor past int16":     with(5, word("8000")),
		"tick entry past int24": with(9, word("ff800000")),
		"spacing high bits":     with(0, word("10000000000000000000000000000000a")),
		"price past uint160":    with(1, word("10000000000000000000000000000000000000000")),
	} {
		var r ephemeral.TickSnapshot
		if err := lens.UnpackCall(payload, &r); err == nil {
			t.Errorf("%s: decoded", name)
		}
	}
}

// TestLensErrors: the lens's own errors decode by name.
func TestLensErrors(t *testing.T) {
	for selector, name := range map[string]string{
		"2fd3dd54": "UnexpectedPoolResponse",
		"51a31485": "InvalidWordRange",
	} {
		raw, _ := hex.DecodeString(selector)
		got, err := lens.UnpackRevert(raw)
		if err != nil || got.Name != name {
			t.Errorf("%s decoded as %v, %v", selector, got, err)
		}
	}
	if _, err := lens.UnpackRevert([]byte{0xde, 0xad, 0xbe, 0xef}); err == nil {
		t.Error("an unknown selector decoded")
	}
	if _, err := lens.UnpackRevert(nil); err == nil {
		t.Error("an empty revert decoded")
	}
}

// TestLensPackCall: the arguments the fixtures were captured with encode to the same words.
func TestLensPackCall(t *testing.T) {
	pool := common.HexToAddress("0x88e6a0c2ddd26feeb64f039a2c41296fcb3f5640")
	data, err := lens.PackCall(pool, int16(70), int16(-3))
	if err != nil {
		t.Fatal(err)
	}
	got := hex.EncodeToString(data[len(data)-96:])
	want := "00000000000000000000000088e6a0c2ddd26feeb64f039a2c41296fcb3f5640" +
		"0000000000000000000000000000000000000000000000000000000000000046" +
		"fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffd"
	if got != want {
		t.Errorf("arguments\n got %s\nwant %s", got, want)
	}
}
