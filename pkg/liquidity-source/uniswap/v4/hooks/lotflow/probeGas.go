package lotflow

// Probe gas (DESIGN REV 6 G1). Every probe's inner frame gets exactly PROBE_GAS; a leg that needs
// more is a skip, whatever the transaction's gas. The simulator cannot meter EVM gas, so it bounds a
// leg's gas from the swap's shape: a base per source kind (swap, callbacks, transfers, cold reads)
// plus a cost per initialized tick crossed and per empty bitmap word walked.
//
// A point estimate is not good enough. What one tick crossing costs depends on whether the slots it
// writes (feeGrowthOutside, v3's oracle-outside fields) go zero -> non-zero, which pool state alone
// does not say: in the P2-FIX fixtures it ranges ~16k-55k per tick on v4 and ~24k-53k on v3 (MEASURED,
// legGas of the 376 filling probes). So the model carries a [Lo, Hi] envelope:
//   - Lo >= PROBE_GAS: the probe certainly exhausts the stipend -> skip (GasSkip);
//   - Hi >= PROBE_GAS - InexactGasMargin: the verdict is the model's, not the chain's -> Inexact,
//     and the leg is still treated as a fill (Lo < PROBE_GAS). This covers the 1.1M-1.2M band;
//   - otherwise the verdict is exact.
// TestProbesReproduced asserts Lo <= legGas <= Hi on every filling probe.

// GasModel bounds one leg's gas.
type GasModel struct {
	BaseV4Lo, BaseV4Hi uint64
	BaseV3Lo, BaseV3Hi uint64
	TickV4Lo, TickV4Hi uint64 // per initialized tick crossed
	TickV3Lo, TickV3Hi uint64
	WordLo, WordHi     uint64 // per empty bitmap word walked
}

// DefaultGasModel: envelope around the fixtures' measured leg gas (v4 base 45,298-71,135 incl. a
// dynamic-fee hook; v3 base 157,136-187,139; per tick as above), widened to the EVM worst case of a
// crossing that turns its written slots non-zero (2 x 22.1k on v4, more on v3).
var DefaultGasModel = GasModel{
	BaseV4Lo: 45_000, BaseV4Hi: 75_000,
	BaseV3Lo: 150_000, BaseV3Hi: 195_000,
	TickV4Lo: 15_000, TickV4Hi: 60_000,
	TickV3Lo: 18_000, TickV3Hi: 70_000,
	WordLo: 2_000, WordHi: 6_000,
}

// Bounds is the [lo, hi] gas of a leg on a source of kind `kind`.
func (g GasModel) Bounds(kind, ticks, words int) (lo, hi uint64) {
	t, w := uint64(ticks), uint64(words)
	if kind == KindV4 {
		return g.BaseV4Lo + t*g.TickV4Lo + w*g.WordLo, g.BaseV4Hi + t*g.TickV4Hi + w*g.WordHi
	}
	return g.BaseV3Lo + t*g.TickV3Lo + w*g.WordLo, g.BaseV3Hi + t*g.TickV3Hi + w*g.WordHi
}

// applyGas sets r's gas bounds, and for a leg that fills by the maths its stipend skip and Inexact
// flag (a leg that cannot fill is a no-fill whatever its gas).
func (cfg *Config) applyGas(s Source, r *LegResult) {
	r.GasLo, r.GasHi = cfg.Gas.Bounds(s.Kind, r.CrossedTicks, r.CrossedWords)
	if !r.Fill {
		return
	}
	if r.GasLo >= cfg.ProbeGas {
		r.GasSkip = true
		return
	}
	near := uint64(0)
	if cfg.ProbeGas > cfg.InexactGasMargin {
		near = cfg.ProbeGas - cfg.InexactGasMargin
	}
	if r.GasHi >= near {
		r.Inexact = true
		if cfg.NearStipend == NearStipendSkip {
			r.GasSkip = true
		}
	}
}

// mayExhaustStipend: a probe the chain may run out of its PROBE_GAS stipend (a GasSkip, or a leg-gas
// envelope reaching PROBE_GAS - InexactGasMargin, whatever the maths' verdict). Such a probe burns the
// whole stipend on chain (NavJitLegs.sol:87), so SwapGasFor counts it in full (StipendProbes).
func (cfg *Config) mayExhaustStipend(r *LegResult) bool {
	return r.GasSkip || r.GasHi+cfg.InexactGasMargin >= cfg.ProbeGas
}
