package oneinchaqua

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func guard() []byte { return ins(opTxOriginGuard, make([]byte, 20)...) }

func requireReason(t *testing.T, err error, want DiscardReason, msg string) {
	t.Helper()
	require.Error(t, err, msg)
	var ae *AdmissionError
	require.True(t, errors.As(err, &ae), msg)
	require.Equal(t, want, ae.Reason, msg)
	require.Equal(t, want, ReasonOf(err), msg)
	require.Contains(t, err.Error(), string(want))
}

func TestParseProgramAdmitted(t *testing.T) {
	t.Parallel()
	pmin, pmax := sqrtPrice(e18(1)), sqrtPrice(e18(4))
	for name, program := range map[string][]byte{
		"xyc":                          prog(iSwap()),
		"xyc salt":                     prog(iSwap(), iSalt(1)),
		"flat xyc":                     prog(iFlat(10_000_000), iSwap()),
		"concentrate flat xyc":         prog(iConcentrate(pmin, pmax), iFlat(10_000_000), iSwap()),
		"flat concentrate xyc":         prog(iFlat(10_000_000), iConcentrate(pmin, pmax), iSwap()),
		"protocol concentrate flat":    prog(iProtocol(10_000_000, testRecipient), iConcentrate(pmin, pmax), iFlat(1), iSwap(), iSalt(9)),
		"salt protocol protocol flat":  prog(iSalt(7), iProtocol(1, testRecipient), iProtocol(2, testRecipient), iFlat(3), iSwap()),
		"flat at 100 percent":          prog(iFlat(feeBpsDenominator), iSwap()),
		"protocol at 100 percent":      prog(iProtocol(feeBpsDenominator, testRecipient), iSwap()),
		"zero bps zero recipient":      prog(iProtocol(0, common.Address{}), iSwap()),
		"0x21 is a byte inside salt":   prog(iSalt(opTxOriginGuard, opTxOriginGuard, 0x14), iSwap()),
		"salt args hold a whole guard": prog(iSalt(guard()...), iSwap()),
	} {
		got, err := parseProgram(program, false)
		require.NoError(t, err, name)
		require.NotEmpty(t, got, name)
	}
}

// 0x21 anywhere excludes the strategy, whatever else the program holds.
func TestParseProgramTxOriginGuard(t *testing.T) {
	t.Parallel()
	pmin, pmax := sqrtPrice(e18(1)), sqrtPrice(e18(4))
	for name, program := range map[string][]byte{
		"first":              prog(guard(), iSwap()),
		"second":             prog(iFlat(1), guard(), iSwap()),
		"third":              prog(iConcentrate(pmin, pmax), iFlat(1), guard(), iSwap()),
		"last":               prog(iSwap(), guard()),
		"only":               guard(),
		"with unused opcode": prog(ins(0x17), guard(), iSwap()),
		"truncated after":    append(prog(iSwap(), guard()), 0x14, 0x05, 0x01),
		"truncated header":   append(prog(guard(), iSwap()), 0x14),
	} {
		_, err := parseProgram(program, false)
		requireReason(t, err, ReasonTxOriginGuard, name)
		_, err = parseProgram(program, true)
		requireReason(t, err, ReasonTxOriginGuard, name)
	}
}

func TestParseProgramUnsupportedOpcodes(t *testing.T) {
	t.Parallel()
	whitelisted := map[byte]bool{opXYCSwap: true, opXYCConcentrate: true, opSalt: true, opFlatFeeIn: true, opAquaProtocolFee: true, opTxOriginGuard: true}
	for op := range 256 {
		if whitelisted[byte(op)] {
			continue
		}
		for _, program := range [][]byte{
			prog(ins(byte(op)), iSwap()),
			prog(iSwap(), ins(byte(op), 1, 2, 3)),
			prog(ins(byte(op), make([]byte, 20)...), iSwap()),
		} {
			_, err := parseProgram(program, false)
			requireReason(t, err, ReasonUnsupportedOpcode, fmt.Sprintf("opcode %#x", op))
		}
	}
}

func TestParseProgramMalformed(t *testing.T) {
	t.Parallel()
	pmin, pmax := sqrtPrice(e18(1)), sqrtPrice(e18(4))
	_, err := parseProgram(nil, false)
	requireReason(t, err, ReasonProgramEmpty, "empty")
	for name, tc := range map[string]struct {
		program []byte
		reason  DiscardReason
	}{
		"args overrun":            {append(prog(iSwap()), 0x14, 0x05, 0x01), ReasonProgramTruncated},
		"trailing byte":           {append(prog(iSwap()), 0x14), ReasonProgramTruncated},
		"swap with args":          {prog(ins(opXYCSwap, 1)), ReasonBadArgs},
		"flat args short":         {prog(ins(opFlatFeeIn, 0, 0, 0), iSwap()), ReasonBadArgs},
		"flat args long":          {prog(ins(opFlatFeeIn, 0, 0, 0, 0, 0), iSwap()), ReasonBadArgs},
		"protocol args short":     {prog(ins(opAquaProtocolFee, make([]byte, 23)...), iSwap()), ReasonBadArgs},
		"concentrate args short":  {prog(ins(opXYCConcentrate, make([]byte, 63)...), iSwap()), ReasonBadArgs},
		"concentrate args long":   {prog(ins(opXYCConcentrate, make([]byte, 65)...), iSwap()), ReasonBadArgs},
		"flat above 100 percent":  {prog(iFlat(feeBpsDenominator+1), iSwap()), ReasonFeeBps},
		"protocol above 100":      {prog(iProtocol(feeBpsDenominator+1, testRecipient), iSwap()), ReasonFeeBps},
		"price bounds equal":      {prog(iConcentrate(pmin, pmin), iSwap()), ReasonPriceBounds},
		"price bounds inverted":   {prog(iConcentrate(pmax, pmin), iSwap()), ReasonPriceBounds},
		"price bounds both zero":  {prog(iConcentrate(new(uint256.Int), new(uint256.Int)), iSwap()), ReasonPriceBounds},
		"no swap":                 {prog(iFlat(1), iSalt()), ReasonProgramShape},
		"two swaps":               {prog(iSwap(), iSwap()), ReasonProgramShape},
		"wrapper after swap":      {prog(iSwap(), iFlat(1)), ReasonProgramShape},
		"concentrate after swap":  {prog(iSwap(), iConcentrate(pmin, pmax)), ReasonProgramShape},
		"protocol after swap":     {prog(iSwap(), iProtocol(1, testRecipient)), ReasonProgramShape},
		"salt only":               {prog(iSalt(1)), ReasonProgramShape},
		"wrapper without a swap":  {prog(iConcentrate(pmin, pmax)), ReasonProgramShape},
		"first reason wins":       {prog(iFlat(feeBpsDenominator+1), ins(0x22), iSwap()), ReasonFeeBps},
		"zero recipient non zero": {prog(iProtocol(1, common.Address{}), iSwap()), ReasonFeeRecipient},
	} {
		_, err := parseProgram(tc.program, false)
		requireReason(t, err, tc.reason, name)
	}
}

// Launch router: any zero recipient is refused, the current router refuses only a non-zero fee.
func TestParseProgramFeeRecipient(t *testing.T) {
	t.Parallel()
	for _, bps := range []uint32{0, 1, feeBpsDenominator} {
		program := prog(iProtocol(bps, common.Address{}), iFlat(1), iSwap())
		_, err := parseProgram(program, true)
		requireReason(t, err, ReasonLaunchFeeRecipent, fmt.Sprintf("launch bps %d", bps))
		_, err = parseProgram(program, false)
		if bps == 0 {
			require.NoError(t, err)
		} else {
			requireReason(t, err, ReasonFeeRecipient, fmt.Sprintf("current bps %d", bps))
		}
	}
	// A non-zero recipient is fine on both routers.
	for _, launch := range []bool{true, false} {
		_, err := parseProgram(prog(iProtocol(5, testRecipient), iSwap()), launch)
		require.NoError(t, err)
	}
}

func TestParseOrderTraits(t *testing.T) {
	t.Parallel()
	program := prog(iFlat(1), iSwap())
	with := func(f func(o *Order)) *Order {
		o := orderOf(testMaker, program)
		f(o)
		return o
	}
	cases := map[string]struct {
		order  *Order
		reason DiscardReason
	}{
		"signature order":  {with(func(o *Order) { o.Traits = uint256.Int{} }), ReasonNotAqua},
		"unwrap weth":      {with(func(o *Order) { setBit(&o.Traits, traitsShouldUnwrap) }), ReasonUnwrap},
		"zero amount in":   {with(func(o *Order) { setBit(&o.Traits, traitsAllowZeroAmountIn) }), ReasonAllowZeroAmountIn},
		"hook pre in":      {with(func(o *Order) { setBit(&o.Traits, 252) }), ReasonHooks},
		"hook post in":     {with(func(o *Order) { setBit(&o.Traits, 251) }), ReasonHooks},
		"hook pre out":     {with(func(o *Order) { setBit(&o.Traits, 250) }), ReasonHooks},
		"hook post out":    {with(func(o *Order) { setBit(&o.Traits, 249) }), ReasonHooks},
		"foreign receiver": {with(func(o *Order) { o.Traits[0] |= 0x1234 }), ReasonReceiver},
		"slice index 0":    {with(func(o *Order) { setSliceIndex(&o.Traits, 0, 1) }), ReasonSliceIndexes},
		"slice index 1":    {with(func(o *Order) { setSliceIndex(&o.Traits, 1, 1) }), ReasonSliceIndexes},
		"slice index 2":    {with(func(o *Order) { setSliceIndex(&o.Traits, 2, 1) }), ReasonSliceIndexes},
		"slice index 3":    {with(func(o *Order) { setSliceIndex(&o.Traits, 3, 1) }), ReasonSliceIndexes},
		"slice past data":  {with(func(o *Order) { setSliceIndex(&o.Traits, 3, 0xffff) }), ReasonSliceIndexes},
	}
	for name, tc := range cases {
		_, err := ParseOrder(tc.order, false)
		requireReason(t, err, tc.reason, name)
	}

	// Hook target bits without a hook flag do nothing, and the maker as receiver is the default.
	o := orderOf(testMaker, program)
	for bit := uint(245); bit <= 248; bit++ {
		setBit(&o.Traits, bit)
	}
	copy(o.Traits[:1], []uint64{0})
	receiver := o.Maker.Bytes()
	for i, b := range receiver {
		shift := uint(8 * (19 - i))
		o.Traits[shift/64] |= uint64(b) << (shift % 64)
	}
	_, err := ParseOrder(o, false)
	require.NoError(t, err)
}

func TestReasonOfOtherError(t *testing.T) {
	t.Parallel()
	require.Empty(t, ReasonOf(ErrBadProgram))
	require.Contains(t, (&AdmissionError{Reason: ReasonApp}).Error(), "unknown_app")
}

func setSliceIndex(z *uint256.Int, n uint, v uint64) {
	var field uint256.Int
	field.SetUint64(v)
	field.Lsh(&field, traitsSliceIndexesOffset+n*traitsSliceIndexBits)
	z.Or(z, &field)
}

func TestOrderEncodeDecode(t *testing.T) {
	t.Parallel()
	for _, program := range [][]byte{nil, prog(iSwap()), prog(iFlat(10), iSwap(), iSalt(1, 2, 3, 4, 5, 6, 7, 8, 9))} {
		o := orderOf(testMaker, program)
		raw := o.Encode()
		require.Zero(t, len(raw)%32)
		back, err := DecodeOrder(raw)
		require.NoError(t, err)
		require.Equal(t, o.Maker, back.Maker)
		require.True(t, o.Traits.Eq(&back.Traits))
		require.Equal(t, len(o.Data), len(back.Data))
		require.Equal(t, o.Hash(), back.Hash())
	}
}

// Live strategies: the hash of the re-encoded order equals the strategy hash Aqua keyed.
func TestDecodeLiveStrategies(t *testing.T) {
	t.Parallel()
	vectors := loadGolden(t)
	require.NotEmpty(t, vectors)
	for _, v := range vectors {
		raw := hexutil.MustDecode(v.Strategy)
		o, err := DecodeOrder(raw)
		require.NoError(t, err)
		require.Equal(t, crypto.Keccak256Hash(raw), o.Hash())
	}
}

func TestDecodeOrderNonCanonical(t *testing.T) {
	t.Parallel()
	good := orderOf(testMaker, prog(iSwap(), iSalt(1))).Encode()
	mutate := func(f func(b []byte) []byte) []byte { return f(append([]byte(nil), good...)) }
	for name, raw := range map[string][]byte{
		"empty":            nil,
		"short":            good[:100],
		"tuple offset":     mutate(func(b []byte) []byte { b[31] = 0x40; return b }),
		"tuple offset hi":  mutate(func(b []byte) []byte { b[0] = 1; return b }),
		"data offset":      mutate(func(b []byte) []byte { b[32*3+31] = 0x80; return b }),
		"maker high bytes": mutate(func(b []byte) []byte { b[32] = 1; return b }),
		"length too large": mutate(func(b []byte) []byte { b[32*4+30] = 0xff; return b }),
		"length overflow":  mutate(func(b []byte) []byte { b[32*4] = 1; return b }),
		"nonzero padding":  mutate(func(b []byte) []byte { b[len(b)-1] = 1; return b }),
		"trailing word":    append(append([]byte(nil), good...), make([]byte, 32)...),
		"missing padding":  good[:len(good)-1],
	} {
		_, err := DecodeOrder(raw)
		require.ErrorIs(t, err, ErrNonCanonical, name)
	}
}

func TestParseUint256Hex(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{"0x": "0", "0x0": "0", "0x00ff": "255", "ff": "255", "0x4000": "16384"} {
		got, err := parseUint256Hex(in)
		require.NoError(t, err, in)
		require.Equal(t, want, got.Dec(), in)
	}
	_, err := parseUint256Hex("0xzz")
	require.Error(t, err)
}

func loadGolden(t testing.TB) []goldenVector {
	t.Helper()
	b, err := os.ReadFile("testdata/golden.json")
	require.NoError(t, err)
	var out []goldenVector
	require.NoError(t, json.Unmarshal(b, &out))
	return out
}
