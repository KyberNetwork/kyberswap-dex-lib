package ambient

import (
	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type CurveState struct {
	PriceRoot    uint256.Int // uint128 — Q64.64 sqrt price
	AmbientSeeds uint256.Int // uint128
	ConcLiq      uint256.Int // uint128 (on-chain int128, always ≥ 0 in valid state)
	SeedDeflator uint64
	ConcGrowth   uint64
}

func (c CurveState) Clone() CurveState { return c } // value copy; all fields are value types

type BookLevel struct {
	BidLots     uint256.Int // uint96
	AskLots     uint256.Int // uint96
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
	maskU64  = new(uint256.Int).SetUint64(^uint64(0))
	maskU96  = func() *uint256.Int { v := new(uint256.Int).Lsh(u256.U1, 96); return v.Sub(v, u256.U1) }()
	maskU128 = u256.UMaxU128
	maskU16  = new(uint256.Int).SetUint64(0xffff)
	maskU8   = new(uint256.Int).SetUint64(0xff)
)

func DecodeCurve(slot0, slot1 [32]byte) CurveState {
	s0 := new(uint256.Int).SetBytes(slot0[:])
	s1 := new(uint256.Int).SetBytes(slot1[:])

	var c CurveState
	c.PriceRoot.And(s0, maskU128)
	c.AmbientSeeds.Rsh(s0, 128)

	c.ConcLiq.And(s1, maskU128)
	var tmp uint256.Int
	tmp.Rsh(s1, 128)
	tmp.And(&tmp, maskU64)
	c.SeedDeflator = tmp.Uint64()
	tmp.Rsh(s1, 192)
	c.ConcGrowth = tmp.Uint64()

	return c
}

func DecodeBookLevel(slot [32]byte) BookLevel {
	v := new(uint256.Int).SetBytes(slot[:])

	var bl BookLevel
	bl.BidLots.And(v, maskU96)

	var tmp uint256.Int
	tmp.Rsh(v, 96)
	bl.AskLots.And(&tmp, maskU96)

	tmp.Rsh(v, 192)
	tmp.And(&tmp, maskU64)
	bl.FeeOdometer = tmp.Uint64()

	return bl
}

func DecodePoolSpec(slot [32]byte) PoolSpec {
	v := new(uint256.Int).SetBytes(slot[:])
	var tmp uint256.Int
	field := func(rsh uint, mask *uint256.Int) uint64 {
		tmp.Rsh(v, rsh)
		tmp.And(&tmp, mask)
		return tmp.Uint64()
	}
	return PoolSpec{
		Schema:       uint8(field(0, maskU8)),
		FeeRate:      uint16(field(8, maskU16)),
		ProtocolTake: uint8(field(24, maskU8)),
		TickSize:     uint16(field(32, maskU16)),
		JitThresh:    uint8(field(48, maskU8)),
		KnockoutBits: uint8(field(56, maskU8)),
		OracleFlags:  uint8(field(64, maskU8)),
	}
}
