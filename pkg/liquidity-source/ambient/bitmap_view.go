package ambient

import (
	"context"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/holiman/uint256"
)

// ChainBitmapView reads CrocSwapDex storage at a pinned block.
// readSlot errors are captured; callers must check Err() after SweepSwap.
type ChainBitmapView struct {
	Ctx      context.Context
	Client   *ethclient.Client
	DexAddr  common.Address
	PoolHash common.Hash
	Block    *big.Int // nil → latest

	err error
}

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

func doesSpillBit(isBuy bool, spillTrunc bool, termBitmap *big.Int) bool {
	if isBuy {
		return spillTrunc
	}
	if IsBitSet(termBitmap, 0) {
		return false
	}
	return spillTrunc
}

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
		for i := uint16(lobbyBit) + 1; i < 256; i++ {
			if pin, ok := v.seekAtMezz(uint8(i), 0, true); ok {
				return pin
			}
		}
		return zeroTick(true)
	}
	for i := int16(lobbyBit) - 1; i >= 0; i-- {
		if pin, ok := v.seekAtMezz(uint8(i), 255, false); ok {
			return pin
		}
	}
	return zeroTick(false)
}

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

// QueryLevel reads (bidLots, askLots) from levels_[poolHash, tick].
func (v *ChainBitmapView) QueryLevel(tick int32) (bidLots, askLots uint256.Int) {
	val := v.readSlot(LevelSlot(v.PoolHash, tick))
	var valU uint256.Int
	var buf [32]byte
	val.FillBytes(buf[:])
	valU.SetBytes(buf[:])
	bidLots.And(&valU, maskU96)
	var tmp uint256.Int
	tmp.Rsh(&valU, 96)
	askLots.And(&tmp, maskU96)
	return
}
