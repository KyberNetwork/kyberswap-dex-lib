package nightlyprop

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	poolmath "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/nightly-prop/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var fuzzABI = lo.Must(abi.JSON(strings.NewReader(`[
 {"type":"function","name":"defaultAdmin","inputs":[],"outputs":[{"type":"address"}]},
 {"type":"function","name":"getOracleConfig","inputs":[],"outputs":[{"type":"address"},{"type":"bytes32"},{"type":"uint32"}]},
 {"type":"function","name":"getSlot","inputs":[{"type":"uint8"}],"outputs":[{"type":"bytes32"}]},
 {"type":"function","name":"updatePrices","inputs":[{"type":"bytes32[]"}],"outputs":[]},
 {"type":"function","name":"grantRole","inputs":[{"type":"bytes32"},{"type":"address"}],"outputs":[]},
 {"type":"function","name":"mint","inputs":[{"type":"address"},{"type":"uint256"}],"outputs":[]},
 {"type":"function","name":"approve","inputs":[{"type":"address"},{"type":"uint256"}],"outputs":[{"type":"bool"}]},
 {"type":"function","name":"removeLiquidity","inputs":[{"type":"int16"},{"type":"uint16"},{"type":"address"}],"outputs":[{"type":"uint256"},{"type":"uint256"}]},
 {"type":"function","name":"addLiquidity","inputs":[{"type":"int16"},{"type":"uint256"},{"type":"uint256"},{"type":"uint256"},{"type":"uint256"}],"outputs":[{"type":"uint256"},{"type":"uint256"}]},
 {"type":"function","name":"swapWithFees","inputs":[{"type":"address"},{"type":"address"},{"type":"uint256"},{"type":"uint256"},{"type":"uint32"}],"outputs":[{"type":"uint256"},{"type":"uint256"}]}
]`)))

// FuzzNightlyPoolAgainstEVM builds one WETH/USDC state per corpus entry. Each
// entry sets the oracle, the balances at levels -1, 0 and 1, and two exact-in
// swaps. Amounts are unsigned, big-endian native token units (up to 128 bits).
// The second swap starts where the first finished. A new case starts empty.
func FuzzNightlyPoolAgainstEVM(f *testing.F) {
	const eth2000 = uint32(11<<27 | 97_656_250) // oracle wire price for 2,000 USDC/WETH
	cash, base := evmUnits(20_000, 6).Bytes(), evmUnits(10, 18).Bytes()
	one, zero := []byte{1}, []byte{}
	add := func(mid uint32, bid, ask uint8, ageMs uint16, below, active, above []byte, buy1 bool, in1 []byte, buy2 bool, in2 []byte) {
		f.Add(mid, bid, ask, ageMs, below, active, above, buy1, in1, buy2, in2)
	}
	// Land on a boundary, cross by one unit, reverse inside a mixed level,
	// exhaust all levels, and follow a failed swap with a valid one.
	add(eth2000, 0, 0, 0, cash, base, base, true, evmUnits(20_010, 6).Bytes(), true, one)
	add(eth2000, 100, 200, 0, cash, base, base, true, evmUnits(7_000, 6).Bytes(), false, evmUnits(1, 18).Bytes())
	add(eth2000, 0, 0, 0, cash, base, base, true, evmUnits(40_040, 6).Bytes(), true, one)
	add(eth2000, 0, 0, 0, cash, base, base, false, evmUnits(11, 18).Bytes(), false, evmUnits(5, 18).Bytes())
	// Empty levels, dust, zero input, and the largest spread codes.
	add(eth2000, 255, 250, 0, cash, zero, base, true, one, false, one)
	add(eth2000, 0, 0, 0, zero, zero, zero, true, zero, true, one)
	add(1, 0, 255, 0, one, one, one, false, one, true, one)
	wide := bytes.Repeat([]byte{0xff}, 16)
	add(^uint32(0), 250, 255, 0, wide, wide, base, true, wide, false, wide)
	// At expiry and one millisecond past it. A zero oracle price disables the feed.
	add(eth2000, 0, 0, 2500, cash, base, base, true, evmUnits(1_000, 6).Bytes(), false, evmUnits(1, 18).Bytes())
	add(eth2000, 0, 0, 2501, cash, base, base, false, evmUnits(1, 18).Bytes(), true, evmUnits(1_000, 6).Bytes())
	add(0, 0, 0, 0, cash, base, base, true, one, false, zero)

	evm := loadEVMFixture(f)
	a := evm.accounts
	// Set the EVM clock to the start of Go's synctest clock. Only CalcAmountOut
	// runs in a clock bubble. The tracker uses its usual local RPC connection.
	evm.accounts.BlockTimestamp = uint64(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
	admin := evm.fuzzMustCall(f, common.Address{}, a.Router, "defaultAdmin")[0].(common.Address)
	config := evm.fuzzMustCall(f, common.Address{}, a.WethUsdc, "getOracleConfig")
	oracle, feed := config[0].(common.Address), config[1].([32]byte)
	slot, position := feed[30], int(feed[31])
	require.Less(f, position, 4)
	slotWord := evm.fuzzMustCall(f, common.Address{}, oracle, "getSlot", slot)[0].([32]byte)

	// Clear the three initially one-sided levels through the pool. Fund the
	// fixture partner for all deposits and swaps, so a token allowance or
	// balance cannot mask a pricing or liquidity mismatch.
	evm.fuzzMustCall(f, admin, a.WethUsdc, "grantRole", crypto.Keccak256Hash([]byte("LIQUIDITY_MANAGER_ROLE")), admin)
	for level := int16(-1); level <= 1; level++ {
		evm.fuzzMustCall(f, admin, a.WethUsdc, "removeLiquidity", level, uint16(10_000), a.Partner)
	}
	funds := new(big.Int).Lsh(big.NewInt(1), 160)
	for _, token := range []common.Address{a.Weth, a.Usdc} {
		evm.fuzzMustCall(f, admin, token, "mint", a.Partner, funds)
		evm.fuzzMustCall(f, a.Partner, token, "approve", a.Router, funds)
		evm.fuzzMustCall(f, a.Partner, token, "approve", a.WethUsdc, funds)
	}
	empty := evm.state.Copy()

	f.Fuzz(func(t *testing.T, mid uint32, bid, ask uint8, ageMs uint16, below, active, above []byte, buy1 bool, in1 []byte, buy2 bool, in2 []byte) {
		// Go runs cases serially inside each fuzz worker. The server reads the
		// same pointer under its mutex, and each worker owns a separate fixture.
		evm.mu.Lock()
		evm.state = empty.Copy()
		evm.mu.Unlock()

		// Keep the other feeds in the fixture slot. Codes 251..254 are reserved;
		// map them to the highest valid spread, so every mutation can be posted.
		word := slotWord
		lane := word[position*6 : (position+1)*6]
		binary.BigEndian.PutUint32(lane, mid)
		lane[4], lane[5] = fuzzSpreadCode(bid), fuzzSpreadCode(ask)
		timestamp := evm.accounts.BlockTimestamp*1000 - uint64(ageMs)
		binary.BigEndian.PutUint64(word[24:], timestamp<<8|uint64(slot))
		evm.fuzzMustCall(t, admin, oracle, "updatePrices", [][32]byte{word})

		for i, raw := range [][]byte{below, active, above} {
			amount := fuzzAmount(raw)
			if amount.IsZero() {
				continue
			}
			baseAmount, quoteAmount := amount.ToBig(), new(big.Int)
			if i == 0 { // level -1 holds USDC; levels 0 and 1 hold WETH
				baseAmount, quoteAmount = quoteAmount, baseAmount
			}
			evm.fuzzMustCall(t, a.Partner, a.WethUsdc, "addLiquidity", int16(i-1),
				baseAmount, quoteAmount, baseAmount, quoteAmount)
		}

		record := evm.track(t, a.WethUsdc, a.Weth, a.Usdc)
		sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: record, Opts: pool.FactoryOpts{StaleCheck: true}})
		require.NoError(t, err)
		// Verify that the tracker saw the requested on-chain starting state.
		expectedMid := uint64(mid&0x07ff_ffff) << (mid >> 27)
		require.Equal(t, expectedMid != 0, sim.Extra.PriceAvailable)
		require.Equal(t, expectedMid, sim.Extra.Price.MidPrice)
		if expectedMid != 0 {
			require.Equal(t, timestamp, sim.Extra.Price.TimestampMs)
		}
		levels := sim.Extra.Pool.Levels
		require.Len(t, levels, 3)
		require.Equal(t, *fuzzAmount(below), levels[0].Liquidity.QuoteAmount)
		require.Equal(t, *fuzzAmount(active), levels[1].Liquidity.BaseAmount)
		require.Equal(t, *fuzzAmount(above), levels[2].Liquidity.BaseAmount)

		evm.fuzzSwap(t, sim, buy1, fuzzAmount(in1))
		evm.fuzzSwap(t, sim, buy2, fuzzAmount(in2))
	})
}

func fuzzAmount(raw []byte) *uint256.Int {
	return new(uint256.Int).SetBytes(raw[:min(len(raw), 16)])
}

func fuzzSpreadCode(code uint8) uint8 {
	if code > 250 {
		return 255
	}
	return code
}

func (f *evmFixture) fuzzCall(t testing.TB, from, to common.Address, method string, args ...any) ([]byte, error) {
	t.Helper()
	input, err := fuzzABI.Pack(method, args...)
	require.NoError(t, err)
	f.mu.Lock()
	defer f.mu.Unlock()
	output, _, err := runtime.Call(to, input, f.config(from, f.state))
	f.state.Finalise(true)
	return output, err
}

func (f *evmFixture) fuzzMustCall(t testing.TB, from, to common.Address, method string, args ...any) []any {
	t.Helper()
	output, err := f.fuzzCall(t, from, to, method, args...)
	require.NoError(t, err, "%s returned %x", method, output)
	values, err := fuzzABI.Unpack(method, output)
	require.NoError(t, err, method)
	return values
}

func (f *evmFixture) fuzzSwap(t *testing.T, sim *PoolSimulator, buy bool, amount *uint256.Int) {
	t.Helper()
	a := f.accounts
	in, out := a.Weth, a.Usdc
	if buy {
		in, out = out, in
	}
	var got *pool.CalcAmountOutResult
	var simErr error
	synctest.Test(t, func(t *testing.T) {
		got, simErr = quote(sim, hexAddress(in), amount, hexAddress(out))
	})
	output, evmErr := f.fuzzCall(t, a.Partner, a.Router, "swapWithFees", in, out, amount.ToBig(), new(big.Int), uint32(0))
	if evmErr != nil {
		require.ErrorIs(t, evmErr, vm.ErrExecutionReverted)
		require.GreaterOrEqual(t, len(output), 4, "empty EVM revert")
		want := fuzzSwapErrors[[4]byte(output[:4])]
		require.NotNil(t, want, "unexpected EVM revert %x, Go error %v", output, simErr)
		// Go checks availability before entering the walk. The pool checks zero
		// input first. Both conditions independently reject this particular case.
		if want == poolmath.ErrZeroSwapAmount && !sim.Extra.PriceAvailable && amount.IsZero() {
			want = ErrPriceUnavailable
		}
		require.ErrorIs(t, simErr, want, "input %s, EVM revert %x", amount.Dec(), output)
	} else {
		require.NoError(t, simErr, "EVM accepted %s of %s", amount.Dec(), in)
		values, err := fuzzABI.Unpack("swapWithFees", output)
		require.NoError(t, err)
		require.Equal(t, values[0].(*big.Int).String(), got.TokenAmountOut.Amount.String())
		require.Zero(t, values[1].(*big.Int).Sign())
		sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: got.SwapInfo})
	}
	onChain := f.simulator(t, a.WethUsdc, a.Weth, a.Usdc)
	require.Equal(t, onChain.Extra.Pool.Position, sim.Extra.Pool.Position)
	require.Equal(t, onChain.Extra.Pool.Levels, sim.Extra.Pool.Levels)
	for i, reserve := range onChain.Info.Reserves {
		require.Equal(t, reserve.String(), sim.Info.Reserves[i].String(), "reserve of token %d", i)
	}
}

var fuzzSwapErrors = func() map[[4]byte]error {
	signatures := map[string]error{
		"NightlyPoolZeroSwapAmount()":                     poolmath.ErrZeroSwapAmount,
		"NightlyPoolAmountTooSmall()":                     poolmath.ErrAmountTooSmall,
		"NightlyPoolInsufficientLiquidity(uint256)":       poolmath.ErrInsufficientLiquidity,
		"NightlyPoolStaleOraclePrice(uint56,uint32)":      poolmath.ErrStaleOraclePrice,
		"NightlyPriceOraclePriceUnavailable(uint8,uint8)": ErrPriceUnavailable,
	}
	result := make(map[[4]byte]error, len(signatures))
	for signature, reason := range signatures {
		result[[4]byte(crypto.Keccak256([]byte(signature)))] = reason
	}
	return result
}()
