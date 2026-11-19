package btrprop

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var someHook = common.HexToAddress("0x00000000000000000000000000000000000000aa")

// hooked is a leg with a live pre-outflow hook holding inv, of which w is withdrawable now.
func hooked(inv, w int64) *hookReads {
	h := &hookReads{slotOK: true, investedOK: true, statusOK: true, invested: big.NewInt(inv)}
	h.slot.Slot = rpcHookSlot{Target: someHook, Flags: HookPreOutflow}
	h.status.WithdrawableNow = big.NewInt(w)
	return h
}

func TestRecallRoom(t *testing.T) {
	a := &rpcAsset{MinLiquidity: big.NewInt(10)}
	liquid := big.NewInt(100)

	// No hook: liquid only.
	none := (&hookReads{slotOK: true, investedOK: true, invested: big.NewInt(500)}).recall()
	require.True(t, isZero(none))
	require.Equal(t, "90", roomWithRecall(liquid, a, &none.Room).Dec())

	// Hook without HOOK_PRE_OUTFLOW: the pool does not recall in the swap.
	h := hooked(500, 500)
	h.slot.Slot.Flags = 0
	require.True(t, isZero(h.recall()))

	// min(invested, withdrawable): withdrawable binds, then invested binds.
	h = hooked(500, 200)
	h.aaveOK = true
	r := h.recall()
	require.Equal(t, "200", r.Room.Dec())
	require.Equal(t, RecallGasAave, r.Gas)
	require.Equal(t, "290", roomWithRecall(liquid, a, &r.Room).Dec(), "liquid + recall - minLiquidity")
	r = hooked(150, 900).recall()
	require.Equal(t, "150", r.Room.Dec())

	// Liquid under the floor still opens room through the recall.
	r = hooked(500, 500).recall()
	require.Equal(t, "495", roomWithRecall(big.NewInt(5), a, &r.Room).Dec())

	// Unreadable or distressed hook: not guessed.
	h = hooked(500, 500)
	h.statusOK = false
	require.True(t, isZero(h.recall()))
	h = hooked(500, 500)
	h.status.Distressed = true
	require.True(t, isZero(h.recall()))
}

func TestRecallGasByKind(t *testing.T) {
	require.Equal(t, RecallGasAave, (&hookReads{aaveOK: true}).recallGas())
	require.Equal(t, RecallGasVault, (&hookReads{vaultOK: true}).recallGas())
	require.Equal(t, RecallGasVault, (&hookReads{morphoOK: true}).recallGas())
	require.Equal(t, RecallGasPendle, (&hookReads{marketOK: true}).recallGas())
	require.Equal(t, RecallGasMax, (&hookReads{}).recallGas(), "unidentified kind prices at the max")
}

// A Pendle leg's recall-backed sizes stay out of the ladder: liquid only.
func TestPendleLegLiquidOnly(t *testing.T) {
	h := hooked(500, 500)
	h.marketOK = true
	r := h.recall()
	require.True(t, isZero(r))
	a := &rpcAsset{MinLiquidity: big.NewInt(10)}
	room := roomWithRecall(big.NewInt(100), a, &r.Room)
	require.Equal(t, "90", room.Dec())
	require.Len(t, cutAbove(pts(10, 9, 20, 80, 30, 200), room), 2)
}

func TestRecallRoomWidensLadder(t *testing.T) {
	a := &rpcAsset{MinLiquidity: big.NewInt(10)}
	r := hooked(500, 200).recall()
	room := roomWithRecall(big.NewInt(100), a, &r.Room)
	require.Len(t, cutAbove(pts(10, 9, 20, 80, 30, 200, 40, 400), room), 3)
}

func TestHookABI(t *testing.T) {
	call, err := poolABI.Pack("getAssetHook", someHook)
	require.NoError(t, err)
	require.Len(t, call, 36)
	raw, err := poolABI.Methods["getAssetHook"].Outputs.Pack(rpcHookSlot{Target: someHook, Flags: 1})
	require.NoError(t, err)
	var out rpcHookSlotResult
	require.NoError(t, poolABI.UnpackIntoInterface(&out, "getAssetHook", raw))
	require.Equal(t, someHook, out.Slot.Target)
	require.Equal(t, HookPreOutflow, out.Slot.Flags)

	raw, err = hookABI.Methods["venueStatus"].Outputs.Pack(
		big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4), big.NewInt(5), big.NewInt(6), true)
	require.NoError(t, err)
	var vs rpcVenueStatus
	require.NoError(t, hookABI.UnpackIntoInterface(&vs, "venueStatus", raw))
	require.Equal(t, int64(5), vs.WithdrawableNow.Int64())
	require.True(t, vs.Distressed)
}

// Recall gas is added only on the sizes that reach past liquid cash, counting what earlier swaps
// of the same quote session already took.
func TestRecallGasOnlyAboveLiquid(t *testing.T) {
	e := Extra{Cash: make([]uint256.Int, 2), RecallGas: make([]int64, 2)}
	e.Cash[1], e.RecallGas[1] = *uint256.NewInt(100), RecallGasAave
	zero := new(uint256.Int)
	require.Equal(t, int64(DefaultGas), e.swapGas(0, 1, zero, uint256.NewInt(100)), "exactly the cash: no recall")
	require.Equal(t, int64(DefaultGas)+RecallGasAave, e.swapGas(0, 1, zero, uint256.NewInt(101)))
	require.Equal(t, int64(DefaultGas)+RecallGasAave, e.swapGas(0, 1, uint256.NewInt(60), uint256.NewInt(41)), "earlier takes count")
	require.Equal(t, int64(DefaultGas), e.swapGas(1, 0, zero, uint256.NewInt(1000)), "leg without a hook")
	require.Equal(t, int64(DefaultGas), (&Extra{}).swapGas(0, 1, zero, uint256.NewInt(1000)))
}

func TestSimulatorAddsRecallGas(t *testing.T) {
	extra, err := json.Marshal(Extra{
		Flags: []uint16{SwapEnabledBit, SwapEnabledBit}, GateOK: true,
		Ladders:   [][]Ladder{{{}, newLadder(pts(10, 9, 200, 180))}, {{}, {}}},
		Cash:      []uint256.Int{{}, *uint256.NewInt(100)},
		RecallGas: []int64{0, RecallGasVault},
	})
	require.NoError(t, err)
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entity.Pool{
		Address: CreditPoolCore, Exchange: DexType, Type: DexType, Extra: string(extra),
		Tokens:   []*entity.PoolToken{{Address: usdc, Swappable: true}, {Address: wmon, Swappable: true}},
		Reserves: entity.PoolReserves{"1000000", "1000000"},
	}})
	require.NoError(t, err)
	gas := func(in int64) int64 {
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: usdc, Amount: big.NewInt(in)}, TokenOut: wmon,
		})
		require.NoError(t, err)
		return res.Gas
	}
	require.Equal(t, int64(DefaultGas), gas(10))
	require.Equal(t, int64(DefaultGas)+RecallGasVault, gas(150))
}

func isZero(r recall) bool { return r.Room.IsZero() }
