package ambient

import (
	"context"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// ChainBitmapView implements BitmapView by reading the CrocSwapDex storage
// mappings (mezzanine_, terminus_, levels_) at a pinned block. Mirrors
// CrocImpact.sol's pinBitmap/seekMezzSpill/queryLevel exactly.
//
// The BitmapView interface does not return errors, so readSlot failures are
// captured on the struct and surfaced via Err(). Once an error is set,
// subsequent slot reads short-circuit (returning zero) so the containing
// SweepSwap call terminates quickly; callers should check Err() afterwards.
type ChainBitmapView struct {
	Ctx      context.Context
	Client   *ethclient.Client
	DexAddr  common.Address
	PoolHash common.Hash
	Block    *big.Int // nil → latest

	err error
}

// Err returns the first readSlot error encountered, or nil if none.
func (v *ChainBitmapView) Err() error { return v.err }

func (v *ChainBitmapView) readSlot(slot common.Hash) *big.Int {
	if v.err != nil {
		return new(big.Int)
	}
	f := &TickFetcher{Client: v.Client, DexAddr: v.DexAddr}
	val, err := f.readSlot(v.ctx(), slot, v.Block)
	if err != nil {
		v.err = err
		return new(big.Int)
	}
	return val
}

func (v *ChainBitmapView) ctx() context.Context {
	if v.Ctx != nil {
		return v.Ctx
	}
	return context.Background()
}

// PinBitmap mirrors CrocImpact.sol pinBitmap → pinTermMezz.
func (v *ChainBitmapView) PinBitmap(isBuy bool, startTick int32) (int32, bool) {
	termBitmap := v.readSlot(TerminusSlot(v.PoolHash, startTick))
	shiftTerm := uint(TermBump(startTick, isBuy))
	tickMezz := MezzKey(startTick)

	nextTerm, spillTrunc := BitAfterTrunc(termBitmap, shiftTerm, isBuy)
	spillBit := doesSpillBit(isBuy, spillTrunc, termBitmap)
	if spillBit {
		return chainSpillOverPin(isBuy, tickMezz), true
	}
	return WeldMezzTerm(tickMezz, nextTerm), false
}

// doesSpillBit mirrors CrocImpact.sol doesSpillBit. For sell direction, if
// bit 0 is already set we are AT a bump, so do not spill.
func doesSpillBit(isBuy bool, spillTrunc bool, termBitmap *big.Int) bool {
	if isBuy {
		return spillTrunc
	}
	if IsBitSet(termBitmap, 0) {
		return false
	}
	return spillTrunc
}

// chainSpillOverPin mirrors CrocImpact.sol spillOverPin (uses zeroTerm(!isBuy)
// for the welded terminus bit so a buy spillover lands at the top of the next
// terminus).
func chainSpillOverPin(isBuy bool, tickMezz int16) int32 {
	if isBuy {
		if tickMezz == math.MaxInt16 {
			return zeroTick(true)
		}
		return WeldMezzTerm(tickMezz+1, zeroTerm(!isBuy))
	}
	return WeldMezzTerm(tickMezz, 0)
}

func zeroTerm(isUpper bool) uint8 {
	if isUpper {
		return 255
	}
	return 0
}

// SeekMezzSpill mirrors CrocImpact.sol seekMezzSpill → seekAtTerm/seekAtMezz/
// seekOverLobby.
func (v *ChainBitmapView) SeekMezzSpill(borderTick int32, isBuy bool) int32 {
	lobbyBorder, mezzBorder := rootsForBorder(borderTick, isBuy)

	if pin, ok := v.seekAtTerm(lobbyBorder, mezzBorder, isBuy); ok {
		return pin
	}
	if pin, ok := v.seekAtMezz(lobbyBorder, mezzBorder, isBuy); ok {
		return pin
	}
	return v.seekOverLobby(lobbyBorder, isBuy)
}

func (v *ChainBitmapView) seekAtTerm(lobbyBit, mezzBit uint8, isBuy bool) (int32, bool) {
	lobbyIdx := UncastBitmapIndex(lobbyBit)
	mezzIdx := WeldLobbyMezz(lobbyIdx, mezzBit)
	probeTick := int32(mezzIdx) << 8
	termBitmap := v.readSlot(TerminusSlot(v.PoolHash, probeTick))
	termBit, spills := BitAfterTrunc(termBitmap, 0, isBuy)
	if spills {
		return 0, false
	}
	return weldLobbyPosMezzTerm(lobbyBit, mezzBit, termBit), true
}

func (v *ChainBitmapView) seekAtMezz(lobbyBit, mezzBorder uint8, isBuy bool) (int32, bool) {
	lobbyIdx := UncastBitmapIndex(lobbyBit)
	probeTick := int32(lobbyIdx) << 16
	mezzBitmap := v.readSlot(MezzSlot(v.PoolHash, probeTick))
	mezzShift := uint(bitRelate(mezzBorder, isBuy))
	mezzBit, spills := BitAfterTrunc(mezzBitmap, mezzShift, isBuy)
	if spills {
		return 0, false
	}
	return v.seekAtTerm(lobbyBit, mezzBit, isBuy)
}

func (v *ChainBitmapView) seekOverLobby(lobbyBit uint8, isBuy bool) int32 {
	if isBuy {
		// Walk up through adjacent lobby words; wrap-around terminates.
		for i := uint16(lobbyBit) + 1; i < 256; i++ {
			if pin, ok := v.seekAtMezz(uint8(i), 0, true); ok {
				return pin
			}
		}
		return zeroTick(true)
	}
	// sell: walk down through adjacent lobby words.
	for i := int16(lobbyBit) - 1; i >= 0; i-- {
		if pin, ok := v.seekAtMezz(uint8(i), 255, false); ok {
			return pin
		}
	}
	return zeroTick(false)
}

// rootsForBorder mirrors CrocImpact.sol rootsForBorder.
//
//	pinTick = isBuy ? borderTick : (borderTick - 1)
//	lobbyBit = pinTick.lobbyBit()
//	mezzBit  = pinTick.mezzBit()
func rootsForBorder(borderTick int32, isBuy bool) (lobbyBit, mezzBit uint8) {
	pinTick := borderTick
	if !isBuy {
		pinTick = borderTick - 1
	}
	lobbyBit = LobbyBit(pinTick)
	mezzBit = MezzBit(pinTick)
	return
}

func weldLobbyPosMezzTerm(lobbyWord, mezzBit, termBit uint8) int32 {
	return WeldLobbyMezzTerm(UncastBitmapIndex(lobbyWord), mezzBit, termBit)
}

// QueryLevel reads (bidLots, askLots) from levels_[poolHash, tick] using the
// same layout as CrocImpact.sol queryLevel:
//
//	askLots = (val << 64)  >> 160   → bits [96, 191]  from LSB-0 perspective
//	bidLots = (val << 160) >> 160   → bits [0, 95]
func (v *ChainBitmapView) QueryLevel(tick int32) (bidLots, askLots *big.Int) {
	val := v.readSlot(LevelSlot(v.PoolHash, tick))
	mask96 := new(big.Int).Lsh(bignum.One, 96)
	mask96.Sub(mask96, bignum.One)

	bidLots = new(big.Int).And(val, mask96)
	askLots = new(big.Int).And(new(big.Int).Rsh(val, 96), mask96)
	return
}
