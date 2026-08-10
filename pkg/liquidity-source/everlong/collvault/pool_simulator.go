package everlongcollvault

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// PoolSimulator prices the Everlong CollVault settlement venue
// (CollateralRebalancerSwapper): a two-token venue between the stable (e.g. NECT) and
// the volatile (e.g. WBTC) leg whose price comes from the deployed CollateralRebalancer CR
// bonding curve, not an AMM invariant.
//
//   - volatile -> stable is LEVERAGE (swapVolatileForStable): the caller's volatile leg
//     mints CollVault shares, the position draws debt, the caller receives net stable.
//   - stable -> volatile is DELEVERAGE (swapStableForVolatile): the caller fronts net
//     stable to repay debt, CollVault shares burn and the freed volatile leg pays out.
//
// Both directions fill whenever the curve accepts them — the non-favored direction
// simply prices ~the spread worse (it does NOT revert on-chain). All math is a
// wei-exact port of the deployed contracts (see math.go).
type PoolSimulator struct {
	pool.Pool
	StaticExtra StaticExtra
	Extra       Extra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(p entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, err
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}
	if extra.CvDecimalsOffset == 0 {
		extra.CvDecimalsOffset = staticExtra.CvDecimalsOffset
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     p.Address,
			Exchange:    p.Exchange,
			Type:        p.Type,
			Tokens:      lo.Map(p.Tokens, func(e *entity.PoolToken, _ int) string { return e.Address }),
			Reserves:    lo.Map(p.Reserves, func(e string, _ int) *big.Int { return bignumber.NewBig(e) }),
			BlockNumber: p.BlockNumber,
		}},
		StaticExtra: staticExtra,
		Extra:       extra,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}
	amountIn := params.TokenAmountIn.Amount
	if amountIn == nil || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}
	if s.Extra.Collateral == nil {
		return nil, ErrNotPriceable
	}

	cp := &s.StaticExtra.CurveParams
	state := &s.Extra
	if cp.stateRegion(state.Collateral, state.Debt, state.PriceWad) == regionOut {
		// degenerate/unmarked states — refuse to quote rather than risk a wrong price.
		// (Recovery states past the CR wall quote deleverage-only; leverage quotes
		// reject there by construction.)
		return nil, ErrNotPriceable
	}

	if indexIn == 1 { // volatile -> stable: LEVERAGE
		return s.calcLeverage(params, amountIn)
	}
	return s.calcDeleverage(params, amountIn) // stable -> volatile: DELEVERAGE
}

func (s *PoolSimulator) calcLeverage(params pool.CalcAmountOutParams,
	amountIn *big.Int) (*pool.CalcAmountOutResult, error) {
	cp := &s.StaticExtra.CurveParams
	state := &s.Extra

	// Cap by the physical-CR floor (with a safety shave: the contract re-checks the
	// floor POST-fill, which can bind slightly before the PRE-fill prediction).
	maxShares := cp.maxLeverageShares(state)
	if maxShares.Sign() == 0 {
		return nil, ErrSwapRejected
	}
	var buffered big.Int
	buffered.Mul(maxShares, big.NewInt(10_000-leverageBufferBps))
	buffered.Quo(&buffered, bigBp)
	if buffered.Sign() == 0 {
		return nil, ErrSwapRejected
	}

	shares := state.sharesForVolatileIn(amountIn, &buffered)
	if shares.Sign() == 0 {
		return nil, ErrSwapRejected
	}
	stableLeg, volatileLeg, ok := state.previewTokenAmounts(shares, true)
	if !ok {
		return nil, ErrSwapRejected
	}
	netStableOut, ok := cp.quoteLeverageAt(state, shares)
	if !ok {
		return nil, ErrSwapRejected
	}
	if netStableOut.Sign() <= 0 {
		return nil, ErrZeroAmountOut
	}
	_, newColl, newDebt := cp.leverageQuoteChecked(state, shares)

	var remaining big.Int
	remaining.Sub(amountIn, volatileLeg)
	if remaining.Sign() < 0 {
		return nil, ErrSwapRejected
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: params.TokenOut, Amount: netStableOut},
		Fee:                    &pool.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		RemainingTokenAmountIn: &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: &remaining},
		Gas:                    s.gasSwap(),
		SwapInfo: SwapInfo{
			IsLeverage:      true,
			CollVaultShares: shares,
			StableLeg:       stableLeg,
			VolatileLeg:     volatileLeg,
			AlmShares:       state.cvConvertToAssets(shares, true),
			NewCollateral:   newColl,
			NewDebt:         newDebt,
		},
	}, nil
}

func (s *PoolSimulator) calcDeleverage(params pool.CalcAmountOutParams,
	amountIn *big.Int) (*pool.CalcAmountOutResult, error) {
	cp := &s.StaticExtra.CurveParams
	state := &s.Extra

	maxGross := cp.maxDeleverageIn(state)
	if maxGross.Sign() == 0 {
		return nil, ErrSwapRejected
	}

	gross := cp.grossForNetStableIn(state, amountIn, maxGross)
	if gross.Sign() == 0 {
		return nil, ErrSwapRejected
	}
	sharesOut, newColl, newDebt := cp.deleverageQuoteChecked(state, gross)
	if sharesOut.Sign() == 0 {
		return nil, ErrSwapRejected
	}
	stableOut, volatileOut, ok := state.previewTokenAmounts(sharesOut, false)
	if !ok {
		return nil, ErrSwapRejected
	}
	if volatileOut.Sign() <= 0 {
		return nil, ErrZeroAmountOut
	}

	// The forward net is what the swapper actually pulls: gross - freed stable.
	var net, remaining big.Int
	net.Sub(gross, stableOut)
	remaining.Sub(amountIn, &net)
	if remaining.Sign() < 0 {
		return nil, ErrSwapRejected
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: params.TokenOut, Amount: volatileOut},
		Fee:                    &pool.TokenAmount{Token: params.TokenOut, Amount: bignumber.ZeroBI},
		RemainingTokenAmountIn: &pool.TokenAmount{Token: params.TokenAmountIn.Token, Amount: &remaining},
		Gas:                    s.gasSwap(),
		SwapInfo: SwapInfo{
			IsLeverage:      false,
			CollVaultShares: sharesOut,
			GrossStableIn:   gross,
			StableLeg:       stableOut,
			VolatileLeg:     volatileOut,
			AlmShares:       s.deleverageAlmShares(sharesOut),
			NewCollateral:   newColl,
			NewDebt:         newDebt,
		},
	}, nil
}

// deleverageAlmShares mirrors previewTokenAmounts' redeem path share conversion.
func (s *PoolSimulator) deleverageAlmShares(collVaultShares *big.Int) *big.Int {
	shareFee := mulDivUp(collVaultShares, s.Extra.WithdrawFeeBp, bigBp)
	var net big.Int
	net.Sub(collVaultShares, shareFee)
	return s.Extra.cvConvertToAssets(&net, false)
}

// UpdateBalance replays the exact fill from SwapInfo — never recomputes swap results.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	e := &s.Extra
	e.Collateral = si.NewCollateral
	e.Debt = si.NewDebt
	if si.IsLeverage {
		e.AlmStableReserve = new(big.Int).Add(e.AlmStableReserve, si.StableLeg)
		e.AlmVolatileReserve = new(big.Int).Add(e.AlmVolatileReserve, si.VolatileLeg)
		e.AlmSupply = new(big.Int).Add(e.AlmSupply, si.AlmShares)
		e.CvTotalAssets = new(big.Int).Add(e.CvTotalAssets, si.AlmShares)
		e.CvTotalSupply = new(big.Int).Add(e.CvTotalSupply, si.CollVaultShares)
		// reference-marked reserves shift by (approximately) the same physical legs
		e.RefStableReserve = new(big.Int).Add(e.RefStableReserve, si.StableLeg)
		e.RefAssetReserve = new(big.Int).Add(e.RefAssetReserve, si.VolatileLeg)
	} else {
		e.AlmStableReserve = new(big.Int).Sub(e.AlmStableReserve, si.StableLeg)
		e.AlmVolatileReserve = new(big.Int).Sub(e.AlmVolatileReserve, si.VolatileLeg)
		e.AlmSupply = new(big.Int).Sub(e.AlmSupply, si.AlmShares)
		e.CvTotalAssets = new(big.Int).Sub(e.CvTotalAssets, si.AlmShares)
		e.CvTotalSupply = new(big.Int).Sub(e.CvTotalSupply, si.CollVaultShares)
		e.RefStableReserve = new(big.Int).Sub(e.RefStableReserve, si.StableLeg)
		e.RefAssetReserve = new(big.Int).Sub(e.RefAssetReserve, si.VolatileLeg)
	}
	s.Info.Reserves = []*big.Int{
		new(big.Int).Set(e.AlmStableReserve),
		new(big.Int).Set(e.AlmVolatileReserve),
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = lo.Map(s.Info.Reserves, func(r *big.Int, _ int) *big.Int {
		return new(big.Int).Set(r)
	})
	// UpdateBalance reassigns Extra's pointers wholesale (copy-on-write), so the value
	// copy of Extra above already insulates the clone.
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{
		Swapper:              s.StaticExtra.Swapper,
		Rebalancer:           s.StaticExtra.Rebalancer,
		UpfrontNetStablePull: true,
	}
}

// GetApprovalAddress: the swapper transferFroms both legs from the payer.
func (s *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return s.StaticExtra.Swapper
}

// gasSwap returns the configured per-deployment gas estimate, or the default.
func (s *PoolSimulator) gasSwap() int64 {
	if s.StaticExtra.GasSwap != 0 {
		return s.StaticExtra.GasSwap
	}
	return defaultGasSwap
}
