package twocryptong

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
)

// Yield Basis oraclized fee policy (YBOraclizedPolicy.get_fee). The pool charges
// policy.get_fee(xp), clamped to [MinFee, MaxFee]; 0 means the pool's native fee.

const (
	policyMinFee = 1e5  // pool MIN_FEE
	policyMaxFee = 1e10 // pool MAX_FEE = FEE_PRECISION

	// A swap executes after the quote. The discount only shrinks with report age, so the
	// quote adds this margin (about two blocks) to the age.
	policyAgeMarginSec = 24
)

// PolicyState is the policy storage get_fee reads. Nil in Extra when the policy is
// absent, killed, or returns 0 (native fee), so the pool's own fee curve applies.
type PolicyState struct {
	BaseFee     *uint256.Int // FEE_PRECISION: floor of a discounted fee
	Capture     *uint256.Int // WAD: share of the reported edge above BaseFee
	FallbackFee *uint256.Int // FEE_PRECISION: fee without a discount, and the cap
	RelaxTime   uint64       // seconds: report age at which the discount is gone

	ReportPrice  *uint256.Int // coin1 in coin0, WAD; 0 when no live report
	ReportObsUs  uint64       // report observation time, microseconds
	CachedA      *uint256.Int // policy cached_A (not the pool's live A)
	SnapXp       [NumTokens]uint256.Int
	SnapD        *uint256.Int
	SnapPriceScl *uint256.Int
}

// policyClockSec is the time the report age is measured at: wall clock plus a margin.
var policyClockSec = func() uint64 { return uint64(time.Now().Unix()) + policyAgeMarginSec }

type policyConfig struct {
	BaseFee     *big.Int `abi:"base_fee"`
	Capture     *big.Int `abi:"capture"`
	FallbackFee *big.Int `abi:"fallback_fee"`
	RelaxTime   *big.Int `abi:"relax_time"`
	FastHL      *big.Int `abi:"fast_half_life"`
	SlowHL      *big.Int `abi:"slow_half_life"`
	TrendGate   *big.Int `abi:"trend_gate"`
	StepCap     *big.Int `abi:"step_cap"`
	ReportLife  *big.Int `abi:"report_lifetime"`
	RatioTol    *big.Int `abi:"ratio_tolerance"`
}

type policyReport struct {
	AssetPrice    *big.Int `abi:"asset_price"`
	ObservationUs *big.Int `abi:"observation_us"`
}

type policySnapshot struct {
	LastUpdateTs *big.Int    `abi:"last_update_ts"`
	FastEma      *big.Int    `abi:"fast_ema"`
	SlowEma      *big.Int    `abi:"slow_ema"`
	PriceScale   *big.Int    `abi:"price_scale"`
	Xp           [2]*big.Int `abi:"xp"`
	D            *big.Int    `abi:"D"`
	LastPrices   *big.Int    `abi:"last_prices"`
}

// Policies whose source is verified (Sourcify/Blockscout). Any other policy is rejected: its
// get_fee logic is unknown, so no quote can be trusted.
var (
	// YBOraclizedPolicy: identical _get_fee in all three deployments.
	oraclizedPolicies = map[common.Address]struct{}{
		common.HexToAddress("0xAEa95d551154443E73565bE724d59D63d72f8A7d"): {}, // Yield Basis WETH
		common.HexToAddress("0xb83697796A35998aF07B01258bBdA3a04caDe241"): {}, // oraclized crvUSD/WETH
		common.HexToAddress("0xD746C005A827b7eA2D7D1fA80ce35e4E90eC1384"): {}, // oraclized crvUSD/WETH
	}
	// YBTwocryptoPolicy: get_fee returns 0, so the pool keeps its native fee.
	nativeFeePolicies = map[common.Address]struct{}{
		common.HexToAddress("0x0436c0648AfD66de23f86f23cB42884304743EfA"): {}, // Yield Basis WBTC
		common.HexToAddress("0x038f549D21b9C22a26AdA4da2ebF7cFD1C3CA22d"): {}, // Yield Basis cbBTC
		common.HexToAddress("0xE99ee179f1Cc62d4A1d3e67dD5541Ecb63acA396"): {}, // Yield Basis tBTC
	}
)

// FetchPolicy reads the policy state get_fee needs, at block (nil = latest). It returns nil for
// a native-fee policy or a killed one, and ErrUnsupportedPolicy for an unverified policy.
func FetchPolicy(
	ctx context.Context,
	client *ethrpc.Client,
	overrides map[common.Address]gethclient.OverrideAccount,
	policy common.Address,
	block *big.Int,
) (*PolicyState, error) {
	if _, ok := nativeFeePolicies[policy]; ok {
		return nil, nil
	}
	if _, ok := oraclizedPolicies[policy]; !ok {
		return nil, ErrUnsupportedPolicy
	}
	var (
		cfg    policyConfig
		rep    policyReport
		snap   policySnapshot
		cached *big.Int
		killed bool
	)
	call := func(m string) *ethrpc.Call {
		return &ethrpc.Call{ABI: YBPolicyABI, Target: policy.Hex(), Method: m}
	}
	if _, err := client.NewRequest().SetContext(ctx).SetOverrides(overrides).SetFrom(shared.AddrDummy).
		SetBlockNumber(block).SetRequireSuccess(true).
		AddCall(call("is_killed"), []any{&killed}).
		AddCall(call("config"), []any{&cfg}).
		AddCall(call("report"), []any{&rep}).
		AddCall(call("state"), []any{&snap}).
		AddCall(call("cached_A"), []any{&cached}).
		TryBlockAndAggregate(); err != nil {
		return nil, err
	}
	if cfg.FallbackFee == nil || rep.AssetPrice == nil || snap.D == nil || cached == nil {
		return nil, ErrUnsupportedPolicy
	}
	if killed { // get_fee returns 0: the pool's native fee
		return nil, nil
	}

	state := &PolicyState{
		BaseFee:      number.SetFromBig(cfg.BaseFee),
		Capture:      number.SetFromBig(cfg.Capture),
		FallbackFee:  number.SetFromBig(cfg.FallbackFee),
		RelaxTime:    cfg.RelaxTime.Uint64(),
		ReportPrice:  number.SetFromBig(rep.AssetPrice),
		ReportObsUs:  rep.ObservationUs.Uint64(),
		CachedA:      number.SetFromBig(cached),
		SnapD:        number.SetFromBig(snap.D),
		SnapPriceScl: number.SetFromBig(snap.PriceScale),
	}
	for i := range NumTokens {
		state.SnapXp[i].SetFromBig(snap.Xp[i])
	}
	return state, nil
}

// policyFee ports YBOraclizedPolicy._get_fee for the swap leaving the pool at xp (coin1 scaled
// by price_scale, after the swap and before its fee). D is the pool's stored D.
func (p *PolicyState) fee(xp []uint256.Int, D *uint256.Int, fee *uint256.Int) error {
	fee.Set(p.FallbackFee)
	old := &p.SnapXp
	// A discount requires one balance to rise and the other to fall: a swap.
	if xp[0].Cmp(&old[0]) >= 0 && xp[1].Cmp(&old[1]) >= 0 ||
		xp[0].Cmp(&old[0]) <= 0 && xp[1].Cmp(&old[1]) <= 0 {
		return nil
	}
	nowSec := policyClockSec()
	nowUs := nowSec * 1_000_000
	var ageUs uint64 // a report observed after the block has age 0
	if nowUs > p.ReportObsUs {
		ageUs = nowUs - p.ReportObsUs
	}
	if p.ReportPrice.IsZero() || ageUs >= p.RelaxTime*1_000_000 {
		return nil
	}
	// The snapshot D must equal the pool's stored D.
	if p.SnapD.IsZero() || !p.SnapD.Eq(D) {
		return nil
	}

	coinIn, coinOut := 0, 1
	if xp[0].Cmp(&old[0]) <= 0 {
		coinIn, coinOut = 1, 0
	}
	// _reported_edge. Products stay below 2^256: xp < MaxD (1e33), price <= 1e30, fee unit 1e10.
	var in, out, edge uint256.Int
	in.Sub(&xp[coinIn], &old[coinIn])
	out.Sub(&old[coinOut], &xp[coinOut])
	if coinIn == 1 {
		in.Div(in.Mul(&in, p.ReportPrice), p.SnapPriceScl)
	} else {
		out.Div(out.Mul(&out, p.ReportPrice), p.SnapPriceScl)
	}
	if in.Cmp(&out) >= 0 {
		return nil
	}
	edge.Div(in.Mul(&in, U_1e10), &out)
	edge.Sub(U_1e10, &edge) // rounds up
	if edge.IsZero() {
		return nil
	}

	// _reported_fee: fresh = min(fallback, base + capture * (edge - min(edge, base)) / 1e18).
	var fresh, disc uint256.Int
	if edge.Gt(p.BaseFee) {
		fresh.Sub(&edge, p.BaseFee)
	}
	fresh.Add(p.BaseFee, fresh.Div(fresh.Mul(&fresh, p.Capture), U_1e18))
	if fresh.Gt(p.FallbackFee) {
		fresh.Set(p.FallbackFee)
	}
	if ageUs != 0 && fresh.Lt(p.FallbackFee) {
		// fresh + isqrt((fallback-fresh)^2 * age / relax_us) + 1; disc <= 1e10, age < relax_us.
		disc.Sub(p.FallbackFee, &fresh)
		disc.Mul(&disc, &disc)
		disc.Div(disc.Mul(&disc, edge.SetUint64(ageUs)), edge.SetUint64(p.RelaxTime*1_000_000))
		fresh.Add(&fresh, disc.Sqrt(&disc))
		fresh.AddUint64(&fresh, 1)
	}
	if !fresh.Lt(p.FallbackFee) || !p.isCorrective(xp, coinIn) {
		return nil
	}
	fee.Set(&fresh)
	return nil
}

// isCorrective ports _is_corrective_swap: the marginal price moves toward the report
// and does not pass it.
func (p *PolicyState) isCorrective(xp []uint256.Int, coinIn int) bool {
	if xp[0].IsZero() || xp[1].IsZero() {
		return false
	}
	var pBefore, pAfter [1]uint256.Int
	if _get_p_custom(p.SnapXp, p.SnapD, p.CachedA, pBefore[:]) != nil ||
		_get_p_custom([NumTokens]uint256.Int{xp[0], xp[1]}, p.SnapD, p.CachedA, pAfter[:]) != nil {
		return false
	}
	pBefore[0].Div(pBefore[0].Mul(&pBefore[0], p.SnapPriceScl), U_1e18)
	pAfter[0].Div(pAfter[0].Mul(&pAfter[0], p.SnapPriceScl), U_1e18)
	if coinIn == 0 {
		return pBefore[0].Lt(&pAfter[0]) && !pAfter[0].Gt(p.ReportPrice)
	}
	return pBefore[0].Gt(&pAfter[0]) && !pAfter[0].Lt(p.ReportPrice)
}
