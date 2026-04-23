package ambient

import (
	"math/big"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type CurveState struct {
	PriceRoot    *big.Int // uint128 — Q64.64 sqrt price
	AmbientSeeds *big.Int // uint128
	ConcLiq      *big.Int // uint128
	SeedDeflator uint64
	ConcGrowth   uint64
}

func (c CurveState) Clone() CurveState {
	return CurveState{
		PriceRoot:    new(big.Int).Set(c.PriceRoot),
		AmbientSeeds: new(big.Int).Set(c.AmbientSeeds),
		ConcLiq:      new(big.Int).Set(c.ConcLiq),
		SeedDeflator: c.SeedDeflator,
		ConcGrowth:   c.ConcGrowth,
	}
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

var (
	maskU64  = new(big.Int).SetUint64(^uint64(0))
	maskU96  = new(big.Int).Sub(new(big.Int).Lsh(bignum.One, 96), bignum.One)
	maskU128 = bignum.MaxUint128
	maskU16  = big.NewInt(0xffff)
	maskU8   = big.NewInt(0xff)
)

func slotToBig(slot [32]byte) *big.Int {
	return new(big.Int).SetBytes(slot[:])
}

func DecodeCurve(slot0, slot1 [32]byte) CurveState {
	v0 := slotToBig(slot0)
	v1 := slotToBig(slot1)

	priceRoot := new(big.Int).And(v0, maskU128)
	ambientSeeds := new(big.Int).Rsh(v0, 128)

	concLiq := new(big.Int).And(v1, maskU128)
	seedDeflator := new(big.Int).And(new(big.Int).Rsh(v1, 128), maskU64).Uint64()
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

	bidLots := new(big.Int).And(v, maskU96)
	askLots := new(big.Int).And(new(big.Int).Rsh(v, 96), maskU96)
	feeOdometer := new(big.Int).And(new(big.Int).Rsh(v, 192), maskU64).Uint64()

	return BookLevel{
		BidLots:     bidLots,
		AskLots:     askLots,
		FeeOdometer: feeOdometer,
	}
}

func DecodePoolSpec(slot [32]byte) PoolSpec {
	v := slotToBig(slot)
	return PoolSpec{
		Schema:       uint8(new(big.Int).And(v, maskU8).Uint64()),
		FeeRate:      uint16(new(big.Int).And(new(big.Int).Rsh(v, 8), maskU16).Uint64()),
		ProtocolTake: uint8(new(big.Int).And(new(big.Int).Rsh(v, 24), maskU8).Uint64()),
		TickSize:     uint16(new(big.Int).And(new(big.Int).Rsh(v, 32), maskU16).Uint64()),
		JitThresh:    uint8(new(big.Int).And(new(big.Int).Rsh(v, 48), maskU8).Uint64()),
		KnockoutBits: uint8(new(big.Int).And(new(big.Int).Rsh(v, 56), maskU8).Uint64()),
		OracleFlags:  uint8(new(big.Int).And(new(big.Int).Rsh(v, 64), maskU8).Uint64()),
	}
}
