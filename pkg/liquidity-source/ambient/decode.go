package ambient

import (
	"math/big"
)

type CurveState struct {
	PriceRoot    *big.Int // uint128 — Q64.64 sqrt price
	AmbientSeeds *big.Int // uint128
	ConcLiq      *big.Int // uint128
	SeedDeflator uint64
	ConcGrowth   uint64
}

type BookLevel struct {
	BidLots     *big.Int // uint96
	AskLots     *big.Int // uint96
	FeeOdometer uint64
}

type PoolSpec struct {
	Schema       uint8
	FeeRate      uint16
	ProtocolTake uint8
	TickSize     uint16
	JitThresh    uint8
	KnockoutBits uint8
	OracleFlags  uint8
}

type KnockoutPivot struct {
	Lots       *big.Int // uint96
	PivotTime  uint32
	RangeTicks uint16
}

type KnockoutMerkle struct {
	MerkleRoot *big.Int // uint160
	PivotTime  uint32
	FeeMileage uint64
}

type KnockoutPos struct {
	Lots       *big.Int // uint96
	FeeMileage uint64
	Timestamp  uint32
}

type RangePosition struct {
	Liquidity  *big.Int // uint128
	FeeMileage uint64
	Timestamp  uint32
	AtomicLiq  bool
}

type AmbientPosition struct {
	Seeds     *big.Int // uint128
	Timestamp uint32
}

func slotToBig(slot [32]byte) *big.Int {
	return new(big.Int).SetBytes(slot[:])
}

func DecodeCurve(slot0, slot1 [32]byte) CurveState {
	v0 := slotToBig(slot0)
	v1 := slotToBig(slot1)

	mask128 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
	mask64 := new(big.Int).SetUint64(^uint64(0))

	priceRoot := new(big.Int).And(v0, mask128)
	ambientSeeds := new(big.Int).Rsh(v0, 128)

	concLiq := new(big.Int).And(v1, mask128)
	seedDeflator := new(big.Int).And(new(big.Int).Rsh(v1, 128), mask64).Uint64()
	concGrowth := new(big.Int).Rsh(v1, 192).Uint64()

	return CurveState{
		PriceRoot:    priceRoot,
		AmbientSeeds: ambientSeeds,
		ConcLiq:      concLiq,
		SeedDeflator: seedDeflator,
		ConcGrowth:   concGrowth,
	}
}

func DecodeBookLevel(slot [32]byte) BookLevel {
	v := slotToBig(slot)

	mask96 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 96), big.NewInt(1))
	mask64 := new(big.Int).SetUint64(^uint64(0))

	bidLots := new(big.Int).And(v, mask96)
	askLots := new(big.Int).And(new(big.Int).Rsh(v, 96), mask96)
	feeOdometer := new(big.Int).And(new(big.Int).Rsh(v, 192), mask64).Uint64()

	return BookLevel{
		BidLots:     bidLots,
		AskLots:     askLots,
		FeeOdometer: feeOdometer,
	}
}

func DecodePoolSpec(slot [32]byte) PoolSpec {
	v := slotToBig(slot)
	return PoolSpec{
		Schema:       uint8(new(big.Int).And(v, big.NewInt(0xff)).Uint64()),
		FeeRate:      uint16(new(big.Int).And(new(big.Int).Rsh(v, 8), big.NewInt(0xffff)).Uint64()),
		ProtocolTake: uint8(new(big.Int).And(new(big.Int).Rsh(v, 24), big.NewInt(0xff)).Uint64()),
		TickSize:     uint16(new(big.Int).And(new(big.Int).Rsh(v, 32), big.NewInt(0xffff)).Uint64()),
		JitThresh:    uint8(new(big.Int).And(new(big.Int).Rsh(v, 48), big.NewInt(0xff)).Uint64()),
		KnockoutBits: uint8(new(big.Int).And(new(big.Int).Rsh(v, 56), big.NewInt(0xff)).Uint64()),
		OracleFlags:  uint8(new(big.Int).And(new(big.Int).Rsh(v, 64), big.NewInt(0xff)).Uint64()),
	}
}

func DecodeKnockoutPivot(slot [32]byte) KnockoutPivot {
	v := slotToBig(slot)
	mask96 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 96), big.NewInt(1))
	return KnockoutPivot{
		Lots:       new(big.Int).And(v, mask96),
		PivotTime:  uint32(new(big.Int).And(new(big.Int).Rsh(v, 96), big.NewInt(0xffffffff)).Uint64()),
		RangeTicks: uint16(new(big.Int).Rsh(v, 128).Uint64()),
	}
}

func DecodeKnockoutMerkle(slot [32]byte) KnockoutMerkle {
	v := slotToBig(slot)
	mask160 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 160), big.NewInt(1))
	mask64 := new(big.Int).SetUint64(^uint64(0))
	return KnockoutMerkle{
		MerkleRoot: new(big.Int).And(v, mask160),
		PivotTime:  uint32(new(big.Int).And(new(big.Int).Rsh(v, 160), big.NewInt(0xffffffff)).Uint64()),
		FeeMileage: new(big.Int).And(new(big.Int).Rsh(v, 192), mask64).Uint64(),
	}
}

func DecodeKnockoutPos(slot [32]byte) KnockoutPos {
	v := slotToBig(slot)
	mask96 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 96), big.NewInt(1))
	mask64 := new(big.Int).SetUint64(^uint64(0))
	return KnockoutPos{
		Lots:       new(big.Int).And(v, mask96),
		FeeMileage: new(big.Int).And(new(big.Int).Rsh(v, 96), mask64).Uint64(),
		Timestamp:  uint32(new(big.Int).Rsh(v, 160).Uint64()),
	}
}

func DecodeRangePosition(slot [32]byte) RangePosition {
	v := slotToBig(slot)
	mask128 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
	mask64 := new(big.Int).SetUint64(^uint64(0))
	return RangePosition{
		Liquidity:  new(big.Int).And(v, mask128),
		FeeMileage: new(big.Int).And(new(big.Int).Rsh(v, 128), mask64).Uint64(),
		Timestamp:  uint32(new(big.Int).And(new(big.Int).Rsh(v, 192), big.NewInt(0xffffffff)).Uint64()),
		AtomicLiq:  new(big.Int).Rsh(v, 224).Sign() > 0,
	}
}

func DecodeAmbientPosition(slot [32]byte) AmbientPosition {
	v := slotToBig(slot)
	mask128 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
	return AmbientPosition{
		Seeds:     new(big.Int).And(v, mask128),
		Timestamp: uint32(new(big.Int).Rsh(v, 128).Uint64()),
	}
}
