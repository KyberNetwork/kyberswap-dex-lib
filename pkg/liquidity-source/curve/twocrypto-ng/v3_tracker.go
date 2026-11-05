package twocryptong

import (
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
)

// VersionV3 is Yield Basis twocrypto v3.0.0.
const VersionV3 = "v3.0.0"

// V3RPC receives v3.0.0's profit and fee-split views and, for a known policy, the policy's state and immutables.
type V3RPC struct {
	LpXcpProfit, ReservedProfitFraction, AdminFee *big.Int

	policyQueried string
	State         struct {
		LastUpdateTs, LastPrices, FastEma, SlowEma, PriceScale *big.Int
	}
	FastHalfLife, SlowHalfLife, Kappa, Deadband, MinCap, MaxCap *big.Int
}

// PreviousPolicy is the policy a stored pool state recorded, or "".
func PreviousPolicy(extra string) string {
	var e struct{ Policy string }
	if json.Unmarshal([]byte(extra), &e) != nil {
		return ""
	}
	return e.Policy
}

// AddV3Calls adds the reads of lp_xcp_profit, reserved_profit_fraction and admin_fee and, when previousPolicy
// is a YBTwocryptoPolicy (nativeFeePolicies), of its state and immutables. The policy is the one the pool had at
// the previous refresh: it is read in the same multicall, and Apply keeps its state only if the pool still
// points to it.
func AddV3Calls(add func(*ethrpc.Call, []any), poolAddress, previousPolicy string, r *V3RPC) {
	for _, c := range []struct {
		method string
		out    any
	}{
		{poolMethodLpXcpProfit, &r.LpXcpProfit},
		{poolMethodReservedProfitFraction, &r.ReservedProfitFraction},
		{poolMethodAdminFee, &r.AdminFee},
	} {
		add(&ethrpc.Call{ABI: CurveTwocryptoNGABI, Target: poolAddress, Method: c.method}, []any{c.out})
	}
	if _, ok := nativeFeePolicies[previousPolicy]; !ok {
		return
	}
	r.policyQueried = previousPolicy
	for _, c := range []struct {
		method string
		out    any
	}{
		{policyMethodFastHalfLife, &r.FastHalfLife},
		{policyMethodSlowHalfLife, &r.SlowHalfLife},
		{policyMethodKappa, &r.Kappa},
		{policyMethodDeadband, &r.Deadband},
		{policyMethodMinCap, &r.MinCap},
		{policyMethodMaxCap, &r.MaxCap},
	} {
		add(&ethrpc.Call{ABI: YBTwocryptoPolicyABI, Target: previousPolicy, Method: c.method}, []any{c.out})
	}
	// state() is a static tuple of five words; the ABI lists them as five named outputs, unpacked by name.
	add(&ethrpc.Call{ABI: YBTwocryptoPolicyABI, Target: previousPolicy, Method: policyMethodState}, []any{&r.State})
}

// Apply stores the reads in extra, after Version and Policy are set: the v3.0.0 fields and the step range
// [adjustment_step_min, adjustment_step_max] from packed_rebalancing_params, and the policy state when complete
// and still the pool's policy.
func (r *V3RPC) Apply(extra *Extra, rebalancingParams *big.Int) {
	if extra.Version != VersionV3 {
		return
	}
	if r.LpXcpProfit != nil && r.ReservedProfitFraction != nil && r.AdminFee != nil {
		extra.LpXcpProfit = uint256.MustFromBig(r.LpXcpProfit)
		extra.ReservedProfitFraction = uint256.MustFromBig(r.ReservedProfitFraction)
		extra.AdminFee = uint256.MustFromBig(r.AdminFee)
	}
	if rebalancingParams != nil {
		packed := uint256.MustFromBig(rebalancingParams)
		extra.AdjustmentStepMin = new(uint256.Int).Rsh(packed, 128)
		extra.AdjustmentStepMax = new(uint256.Int).Rsh(packed, 64)
		extra.AdjustmentStepMax.And(extra.AdjustmentStepMax, uint256.NewInt(^uint64(0)))
	}
	s := &r.State
	if r.policyQueried == "" || r.policyQueried != extra.Policy || s.LastUpdateTs == nil || s.LastPrices == nil ||
		s.FastEma == nil || s.SlowEma == nil || s.PriceScale == nil || r.FastHalfLife == nil ||
		r.SlowHalfLife == nil || r.Kappa == nil || r.Deadband == nil || r.MinCap == nil || r.MaxCap == nil {
		return
	}
	extra.PolicyState = &PolicyState{
		LastUpdateTs: s.LastUpdateTs.Int64(),
		LastPrices:   uint256.MustFromBig(s.LastPrices),
		FastEma:      uint256.MustFromBig(s.FastEma),
		SlowEma:      uint256.MustFromBig(s.SlowEma),
		PriceScale:   uint256.MustFromBig(s.PriceScale),
		FastHalfLife: uint256.MustFromBig(r.FastHalfLife),
		SlowHalfLife: uint256.MustFromBig(r.SlowHalfLife),
		Kappa:        uint256.MustFromBig(r.Kappa),
		Deadband:     uint256.MustFromBig(r.Deadband),
		MinCap:       uint256.MustFromBig(r.MinCap),
		MaxCap:       uint256.MustFromBig(r.MaxCap),
	}
}
