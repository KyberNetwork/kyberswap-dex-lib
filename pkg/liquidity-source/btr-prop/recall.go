package btrprop

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// The pool recalls from a leg's yield hook inside the swap (PoolHooksLib.preOutflow, live when the
// leg's hook slot has HOOK_PRE_OUTFLOW) and prices on full reserves, so a leg's depth is its
// liquid cash plus what the hook can pay out now. Recall gas measured on an anvil fork of Monad
// (pool v2, block 122749494), as the extra over the same swap served from liquid cash.
const (
	// HookPreOutflow is HOOK_PRE_OUTFLOW (dex-evm PoolConstantsLib): the pool recalls in the swap.
	HookPreOutflow uint32 = 1 << 0

	RecallGasAave   int64 = 430_000 // measured +426k (USDC)
	RecallGasVault  int64 = 320_000 // ERC4626 / Euler, measured +314k (WETH); Morpho taken as ERC4626
	RecallGasPendle int64 = 860_000 // measured +857k (USDAT)
	// RecallGasMax prices a hook whose kind is not identified (Compound is unmeasured).
	RecallGasMax = RecallGasPendle
)

// rpcHookSlot mirrors IPool.HookSlot, in ABI order.
type rpcHookSlot struct {
	Target         common.Address `abi:"target"`
	Flags          uint32         `abi:"flags"`
	LastCreditAt   uint32         `abi:"lastCreditAt"`
	CreditBookCode uint32         `abi:"creditBookCode"`
}

// rpcHookSlotResult wraps the single unnamed tuple output, as rpcAssetResult does.
type rpcHookSlotResult struct {
	Slot rpcHookSlot
}

// rpcVenueStatus mirrors the hook's venueStatus return values.
type rpcVenueStatus struct {
	Invested        *big.Int `abi:"invested"`
	Nav             *big.Int `abi:"nav"`
	Cash            *big.Int `abi:"cash"`
	Total           *big.Int `abi:"total"`
	WithdrawableNow *big.Int `abi:"withdrawableNow"`
	PendingRecall   *big.Int `abi:"pendingRecall"`
	Distressed      bool     `abi:"distressed"`
}

// hookReads is one leg's hook reads of a refresh.
type hookReads struct {
	slot   rpcHookSlotResult
	slotOK bool

	invested   *big.Int
	investedOK bool

	status   rpcVenueStatus
	statusOK bool

	// Kind probes: each venue adapter exposes one distinguishing getter.
	market, aave, vault, morpho         common.Address
	marketOK, aaveOK, vaultOK, morphoOK bool
}

// recall is what a leg's hook adds to its depth: Room more of the token payable inside the swap,
// at Gas extra. The zero value is a leg with no usable recall.
type recall struct {
	Room uint256.Int
	Gas  int64
}

// recallGas is the recall gas of the hook's kind, RecallGasMax when none is identified.
func (h *hookReads) recallGas() int64 {
	switch {
	case h.aaveOK:
		return RecallGasAave
	case h.vaultOK || h.morphoOK:
		return RecallGasVault
	case h.marketOK:
		return RecallGasPendle
	}
	return RecallGasMax
}

// recall sizes what the leg's hook can pay out inside a swap: min(invested, withdrawableNow).
// None without a hook, without HOOK_PRE_OUTFLOW, or when the reads fail (an unreadable hook is
// not guessed). A Pendle hook is also none: its recall reverts on an exit above 5 bps of the
// market, and the getSwapQuote probe prices on full reserves, so it cannot show which sizes the
// recall honours; those legs stay liquid only.
func (h *hookReads) recall() recall {
	var r recall
	if !h.slotOK || h.slot.Slot.Target == (common.Address{}) || h.slot.Slot.Flags&HookPreOutflow == 0 ||
		!h.investedOK || !h.statusOK || h.invested == nil || h.status.WithdrawableNow == nil ||
		h.status.Distressed || h.marketOK {
		return r
	}
	var inv, w uint256.Int
	if inv.SetFromBig(h.invested) || w.SetFromBig(h.status.WithdrawableNow) {
		return r
	}
	if w.Lt(&inv) {
		inv.Set(&w)
	}
	r.Room, r.Gas = inv, h.recallGas()
	return r
}

// roomWithRecall is the most the pool can pay out of a token in one swap: liquid cash plus the
// hook's recall, less the asset's minLiquidity floor. Without recall it is liquidRoom.
func roomWithRecall(liquid *big.Int, a *rpcAsset, rec *uint256.Int) *uint256.Int {
	var liq, floor, sum uint256.Int
	if liquid == nil || a == nil || a.MinLiquidity == nil || liq.SetFromBig(liquid) || floor.SetFromBig(a.MinLiquidity) {
		return nil
	}
	if _, over := sum.AddOverflow(&liq, rec); over {
		return nil
	}
	if !sum.Gt(&floor) {
		return new(uint256.Int)
	}
	return sum.Sub(&sum, &floor)
}
