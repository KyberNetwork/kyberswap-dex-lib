package twocryptong

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/holiman/uint256"
)

// VersionV21d is the donation-era twocrypto ("Yield Basis" v2.1.0d), whose tweak_price differs from v2.0.0's.
const VersionV21d = "v2.1.0d"

// V21dRPC receives the views v2.1.0d's rebalance reads. A pool without a view leaves its field nil.
type V21dRPC struct {
	Version                                                                string
	LastTimestamp, DonationShares, DonationDuration, LastDonationReleaseTs *big.Int
	DonationProtectionExpiryTs, DonationProtectionPeriod                   *big.Int
}

// AddV21dCalls adds the reads of version(), last_timestamp and the donation views.
func AddV21dCalls(add func(*ethrpc.Call, []any), poolAddress string, r *V21dRPC) {
	for _, c := range []struct {
		method string
		out    any
	}{
		{poolMethodVersion, &r.Version},
		{poolMethodLastTimestamp, &r.LastTimestamp},
		{poolMethodDonationShares, &r.DonationShares},
		{poolMethodDonationDuration, &r.DonationDuration},
		{poolMethodLastDonationReleaseTs, &r.LastDonationReleaseTs},
		{poolMethodDonationProtectionExpiryTs, &r.DonationProtectionExpiryTs},
		{poolMethodDonationProtectionPeriod, &r.DonationProtectionPeriod},
	} {
		add(&ethrpc.Call{ABI: CurveTwocryptoNGABI, Target: poolAddress, Method: c.method}, []any{c.out})
	}
}

// Apply stores the reads in extra: the version always, the rest only when the pool has the views.
func (r *V21dRPC) Apply(extra *Extra) {
	extra.Version = r.Version
	if r.LastTimestamp != nil {
		extra.LastTimestamp = r.LastTimestamp.Int64()
	}
	if r.DonationShares == nil || r.DonationDuration == nil || r.LastDonationReleaseTs == nil ||
		r.DonationProtectionExpiryTs == nil || r.DonationProtectionPeriod == nil {
		return
	}
	extra.DonationShares = uint256.MustFromBig(r.DonationShares)
	extra.DonationDuration = uint256.MustFromBig(r.DonationDuration)
	extra.LastDonationReleaseTs = r.LastDonationReleaseTs.Int64()
	extra.DonationProtectionExpiryTs = r.DonationProtectionExpiryTs.Int64()
	extra.DonationProtectionPeriod = uint256.MustFromBig(r.DonationProtectionPeriod)
}

// v21dCappedStepPools run the later v2.1.0d build, which steps by min(adjustment_step, norm / 5). The earlier
// build, under the same version string, steps by max(adjustment_step, norm / 5) as v2.0.0 does. Both are
// verified on Ethereum (eth.blockscout.com): 0x6e5492f8 is the later one (runtime code 23613 bytes); 0x83f24023,
// 0xd9ff8396 and 0xf1f435b0 are the earlier one (23655 bytes), and their tweak_price differs from the later
// one's only in that step.
var v21dCappedStepPools = map[string]struct{}{
	"0x6e5492f8ea2370844ee098a56dd88e1717e4a9c2": {}, // crvUSD/WETH
}

func (t *PoolSimulator) isV21d() bool {
	return t.Extra.Version == VersionV21d
}

// xcpV21d is v2.1.0d's _xcp: D * 1e18 // 2 // isqrt(1e18 * price_scale). It rounds differently from v2.0.0's
// geometric mean of the scaled balances.
func xcpV21d(D, priceScale *uint256.Int) *uint256.Int {
	var root, xcp uint256.Int
	root.Mul(U_1e18, priceScale)
	root.Sqrt(&root)
	xcp.Mul(D, U_1e18)
	xcp.Div(&xcp, NumTokensU256)
	return xcp.Div(&xcp, &root)
}

// donationSharesAt is v2.1.0d's _donation_shares(protection) at now: the donation shares released by time,
// damped while add_liquidity's protection lasts. Zero without donation state.
func (t *PoolSimulator) donationSharesAt(now int64, protection bool) *uint256.Int {
	e := &t.Extra
	if e.DonationShares == nil || e.DonationShares.IsZero() || e.DonationDuration == nil || e.DonationDuration.IsZero() {
		return new(uint256.Int)
	}
	var unlocked uint256.Int
	unlocked.Mul(e.DonationShares, uint256.NewInt(uint64(now-e.LastDonationReleaseTs)))
	unlocked.Div(&unlocked, e.DonationDuration)
	if unlocked.Gt(e.DonationShares) {
		unlocked.Set(e.DonationShares)
	}
	if !protection {
		return &unlocked
	}
	var factor uint256.Int
	if e.DonationProtectionExpiryTs > now && e.DonationProtectionPeriod != nil && !e.DonationProtectionPeriod.IsZero() {
		factor.Mul(uint256.NewInt(uint64(e.DonationProtectionExpiryTs-now)), Precision)
		factor.Div(&factor, e.DonationProtectionPeriod)
		if factor.Gt(Precision) {
			factor.Set(Precision)
		}
	}
	factor.Sub(Precision, &factor)
	unlocked.Mul(&unlocked, &factor)
	return unlocked.Div(&unlocked, Precision)
}

// tweakPriceV21d is v2.1.0d's tweak_price (verified source of 0x6e5492f8). Against v2.0.0:
//   - xcp_profit grows by virtual_price - old_virtual_price, not by their ratio;
//   - virtual_price may not drop unless A/gamma ramp (equal is allowed);
//   - the rebalance runs once per block, when the virtual price over the supply less unlocked donation shares
//     exceeds max(1e18, (xcp_profit + 1e18) / 2) + allowed_extra_profit;
//   - the step is min(adjustment_step, norm / 5);
//   - a rebalance that would drop virtual_price below max(threshold, virtual_price) burns donation shares, and is
//     kept only if the new virtual_price exceeds 1e18 and the threshold.
func (t *PoolSimulator) tweakPriceV21d(A, gamma *uint256.Int, _xp [NumTokens]uint256.Int, K0_prev *uint256.Int,
	s *SwapInfo) error {
	e := &t.Extra
	now := t.chainNow()
	var err error
	if s.PriceOracleTimestamp, err = t.priceOracleAt(now, s.PriceOracle[:]); err != nil {
		return err
	}
	priceScale, priceOracle := &e.PriceScale[0], &s.PriceOracle[0]

	D, err := newton_D(A, gamma, K0_prev, _xp[:], e.UseCustomMath)
	if err != nil {
		return err
	}
	if err = get_p(e.UseCustomMath, _xp, D, A, gamma, s.LastPrices[:]); err != nil {
		return err
	}
	s.LastPrices[0].Div(new(uint256.Int).Mul(&s.LastPrices[0], priceScale), U_1e18)

	totalSupply := e.LpSupply
	donationShares := t.donationSharesAt(now, true)
	if !donationShares.Lt(totalSupply) {
		return ErrNegativeDonation
	}
	var lockedSupply uint256.Int
	lockedSupply.Sub(totalSupply, donationShares)

	xcp := xcpV21d(D, priceScale)
	var virtualPrice uint256.Int
	virtualPrice.Div(new(uint256.Int).Mul(U_1e18, xcp), totalSupply)
	if virtualPrice.Lt(e.VirtualPrice) && e.FutureAGammaTime <= now {
		return ErrLoss
	}
	xcpProfit := new(uint256.Int).Add(e.XcpProfit, &virtualPrice)
	if xcpProfit.Lt(e.VirtualPrice) {
		return ErrLoss
	}
	xcpProfit.Sub(xcpProfit, e.VirtualPrice)

	threshold := new(uint256.Int).Add(xcpProfit, U_1e18)
	threshold.Rsh(threshold, 1)
	if threshold.Lt(U_1e18) {
		threshold.Set(U_1e18)
	}
	var vpBoosted uint256.Int
	vpBoosted.Div(new(uint256.Int).Mul(U_1e18, xcp), &lockedSupply)
	if vpBoosted.Lt(&virtualPrice) {
		return ErrNegativeDonation
	}

	// Without a rebalance: price_scale kept, D and virtual_price from the trade.
	s.XcpProfit.Set(xcpProfit)
	s.PriceScale[0].Set(priceScale)
	s.D.Set(D)
	s.VirtualPrice.Set(&virtualPrice)
	s.LpSupply.Set(totalSupply)
	s.DonationShares.Clear()
	if e.DonationShares != nil {
		s.DonationShares.Set(e.DonationShares)
	}
	s.LastDonationReleaseTs = e.LastDonationReleaseTs
	s.LastTimestamp = now

	if !vpBoosted.Gt(new(uint256.Int).Add(threshold, e.AllowedExtraProfit)) || e.LastTimestamp >= now {
		return nil
	}
	return t.rebalanceV21d(A, gamma, _xp, priceScale, priceOracle, totalSupply, donationShares, threshold,
		&virtualPrice, now, s)
}

// rebalanceV21d is the price_scale adjustment in v2.1.0d's tweak_price; it writes s only when the pool keeps it.
func (t *PoolSimulator) rebalanceV21d(A, gamma *uint256.Int, _xp [NumTokens]uint256.Int,
	priceScale, priceOracle, totalSupply, donationShares, threshold, virtualPrice *uint256.Int, now int64,
	s *SwapInfo) error {
	e := &t.Extra
	var norm uint256.Int
	norm.Div(new(uint256.Int).Mul(priceOracle, U_1e18), priceScale)
	if norm.Gt(U_1e18) {
		norm.Sub(&norm, U_1e18)
	} else {
		norm.Sub(U_1e18, &norm)
	}
	step := new(uint256.Int).Div(&norm, number.Number_5)
	if _, capped := v21dCappedStepPools[t.Info.Address]; capped == e.AdjustmentStep.Lt(step) {
		step.Set(e.AdjustmentStep) // capped: min(adjustment_step, norm / 5); otherwise max
	}
	if !norm.Gt(step) {
		return nil
	}

	var pNew, tmp uint256.Int
	pNew.Mul(priceScale, tmp.Sub(&norm, step))
	pNew.Add(&pNew, tmp.Mul(step, priceOracle))
	pNew.Div(&pNew, &norm)

	xp := []uint256.Int{_xp[0], {}}
	xp[1].Div(new(uint256.Int).Mul(&_xp[1], &pNew), priceScale)
	newD, err := newton_D(A, gamma, new(uint256.Int), xp, e.UseCustomMath)
	if err != nil {
		return err
	}
	newXcp := xcpV21d(newD, &pNew)
	scaledXcp := new(uint256.Int).Mul(U_1e18, newXcp)
	var newVirtualPrice, burn uint256.Int
	newVirtualPrice.Div(scaledXcp, totalSupply)

	goal := threshold
	if virtualPrice.Gt(goal) {
		goal = virtualPrice
	}
	if newVirtualPrice.Lt(goal) {
		tweakedSupply := new(uint256.Int).Div(scaledXcp, goal)
		if !tweakedSupply.Lt(totalSupply) {
			return ErrTweakedSupply
		}
		burn.Sub(totalSupply, tweakedSupply)
		if donationShares.Lt(&burn) {
			burn.Set(donationShares)
		}
		newVirtualPrice.Div(scaledXcp, tmp.Sub(totalSupply, &burn))
	}
	if !newVirtualPrice.Gt(U_1e18) || newVirtualPrice.Lt(threshold) {
		return nil
	}

	s.PriceScale[0].Set(&pNew)
	s.D.Set(newD)
	s.VirtualPrice.Set(&newVirtualPrice)
	if !burn.IsZero() {
		t.burnDonationShares(&burn, donationShares, now, s)
	}
	return nil
}

// feeBalanceTermV21d is the balance term of v2.1.0d's _fee: B = 4e18 * x0 / S * x1 / S, then
// fee_gamma * B / (fee_gamma * B / 1e18 + 1e18 - B). v2.0.0 computes fee_gamma * 1e18 / (fee_gamma + 1e18 - B).
func feeBalanceTermV21d(xp []uint256.Int, feeGamma *uint256.Int, B *uint256.Int) error {
	var S, tmp uint256.Int
	S.Add(&xp[0], &xp[1])
	if S.IsZero() {
		return ErrZero
	}
	B.Mul(U_1e18, uint256.NewInt(NumTokens*NumTokens))
	B.Div(B.Mul(B, &xp[0]), &S)
	B.Div(B.Mul(B, &xp[1]), &S)
	tmp.Mul(feeGamma, B)
	var den uint256.Int
	den.Div(&tmp, U_1e18)
	den.Add(&den, U_1e18)
	den.Sub(&den, B)
	B.Div(&tmp, &den)
	return nil
}
