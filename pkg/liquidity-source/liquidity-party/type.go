package liquidityparty

import "math/big"

// PoolStateSnapshotRPC decodes IPartyInfo.fetchPoolState's single tuple return.
//
// go-ethereum copies a single-tuple output into the destination struct's field 0 and then copies
// the tuple components into that field's struct fields POSITIONALLY (see accounts/abi/reflect.go
// setStruct). So (a) the fetch must pass a wrapper whose field 0 is this struct, and (b) the fields
// below must stay in the exact order of the Solidity struct components (IPartyInfo.sol:59). int128
// and uint256 both decode to *big.Int.
type PoolStateSnapshotRPC struct {
	Kappa                    *big.Int
	EffectiveSigmaQ          *big.Int
	QInternal                []*big.Int
	Bases                    []*big.Int
	FeesPpm                  []*big.Int
	CachedBalances           []*big.Int
	LpSupply                 *big.Int
	SigmaSwap                *big.Int
	SigmaSwapLastUpdateBlock uint64
	PrevBlockEndSigmaQ       *big.Int
	GammaAccum               *big.Int
	GammaAccumLastBlock      uint64
	MaxGammaPerWindowPpm     uint32
	MintDeviationPpm         uint32
	EmaShiftBlocks           uint8
	CurrentBlock             *big.Int
}

// fetchPoolStateResult is the ethrpc destination wrapper: field 0 receives the fetchPoolState tuple.
type fetchPoolStateResult struct {
	State PoolStateSnapshotRPC
}

// Extra is the mutable per-refresh swap state the simulator needs to reproduce the LMSR swap kernel
// to the wei. Everything here comes from one PartyInfo.fetchPoolState snapshot.
// Kappa and Bases/FeesPpm are construction-immutable on-chain but are carried here (rather than in
// StaticExtra) because the on-demand snapshot returns them anyway, keeping state self-consistent.
//
// All values are Q64.64 numerators (Kappa, EffectiveSigmaQ, QInternal) or plain uints (Bases, in
// wei-denominator units) except FeesPpm (per-asset parts-per-million). Reserves are stored on
// entity.Pool (from CachedBalances), not here.
type Extra struct {
	Kappa           *big.Int   `json:"kappa"`
	EffectiveSigmaQ *big.Int   `json:"eSigmaQ"`
	QInternal       []*big.Int `json:"q"`
	Bases           []*big.Int `json:"bases"`
	FeesPpm         []uint64   `json:"fees"`
	Killed          bool       `json:"killed,omitempty"`
}

// SwapInfo is emitted by CalcAmountOut/CalcAmountIn and consumed verbatim by UpdateBalance (never
// recomputed). It carries the gross kernel output and the internal input delta so the LMSR q-vector
// update in UpdateBalance mirrors LMSRKernel.applySwap exactly.
type SwapInfo struct {
	TokenInIndex  int      `json:"i"`
	TokenOutIndex int      `json:"j"`
	DeltaInternal *big.Int `json:"dIn"`  // Q64.64: qInternal[i] += this
	GrossInternal *big.Int `json:"gOut"` // Q64.64: qInternal[j] -= this (gross, pre-fee)
}

// Meta is returned by GetMetaInfo so the aggregator can build the adapter calldata —
// word-aligned abi.encode(pool, indexIn, indexOut), matching the in-repo adapter convention.
type Meta struct {
	TokenInIndex  int `json:"i"`
	TokenOutIndex int `json:"j"`
}
