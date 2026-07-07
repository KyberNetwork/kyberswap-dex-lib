package unipool

import (
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var bpsDivisorU = uint256.NewInt(bpsDivisor)

// nowUnix returns the current wall-clock time used to project the virtual
// reserves forward. It is a package var so tests can pin it to a specific block
// timestamp and compare against the on-chain quoter deterministically.
var nowUnix = func() int64 { return time.Now().Unix() }

// PoolSimulator replays UniPool's swap math fully off-chain.
//
// State stored: real reserves at lastUpdateTimestamp, the 4 virtual reserves at
// the same instant, plus priceDecay so we can interpolate forward at quote time
// (mirroring UniPoolPairGetters.previewVirtualReservesElapsed). Liquidations are
// NOT replayed (would require porting the tick state).
type PoolSimulator struct {
	pool.Pool

	reserve0 *uint256.Int
	reserve1 *uint256.Int

	vr0In  *uint256.Int
	vr0Out *uint256.Int
	vr1In  *uint256.Int
	vr1Out *uint256.Int

	lastUpdateTimestamp uint64
	priceDecay          uint64

	feeLpBps   *uint256.Int
	feePoolBps *uint256.Int

	totalBorrowed0 *uint256.Int
	totalBorrowed1 *uint256.Int

	swapPriceToleranceBps uint16

	gas int64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	reserve0, err := fromBig(extra.Reserve0)
	if err != nil {
		return nil, err
	}
	reserve1, err := fromBig(extra.Reserve1)
	if err != nil {
		return nil, err
	}
	vr0In, err := fromBig(extra.VirtualReserve0In)
	if err != nil {
		return nil, err
	}
	vr0Out, err := fromBig(extra.VirtualReserve0Out)
	if err != nil {
		return nil, err
	}
	vr1In, err := fromBig(extra.VirtualReserve1In)
	if err != nil {
		return nil, err
	}
	vr1Out, err := fromBig(extra.VirtualReserve1Out)
	if err != nil {
		return nil, err
	}
	tb0, err := fromBig(extra.TotalBorrowed0)
	if err != nil {
		return nil, err
	}
	tb1, err := fromBig(extra.TotalBorrowed1)
	if err != nil {
		return nil, err
	}

	tokens := lo.Map(entityPool.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address })
	reserves := []*big.Int{reserve0.ToBig(), reserve1.ToBig()}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
		}},
		reserve0:              reserve0,
		reserve1:              reserve1,
		vr0In:                 vr0In,
		vr0Out:                vr0Out,
		vr1In:                 vr1In,
		vr1Out:                vr1Out,
		lastUpdateTimestamp:   extra.LastUpdateTimestamp,
		priceDecay:            extra.PriceDecay,
		feeLpBps:              uint256.NewInt(uint64(extra.FeeLpBps)),
		feePoolBps:            uint256.NewInt(uint64(extra.FeePoolBps)),
		totalBorrowed0:        tb0,
		totalBorrowed1:        tb1,
		swapPriceToleranceBps: extra.SwapPriceToleranceBps,
		gas:                   defaultGas,
	}, nil
}

func fromBig(v *big.Int) (*uint256.Int, error) {
	if v == nil {
		return new(uint256.Int), nil
	}
	out, overflow := uint256.FromBig(v)
	if overflow {
		return nil, ErrInvalidReserve
	}
	return out, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := params.TokenAmountIn, params.TokenOut
	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	totalFee := new(uint256.Int).Add(s.feeLpBps, s.feePoolBps)
	if totalFee.Cmp(bpsDivisorU) >= 0 {
		return nil, ErrFeeExceedsMax
	}

	// Project the 4 VRs forward in time (port of previewVirtualReservesElapsed).
	now := uint64(nowUnix())
	vr0In, vr0Out, vr1In, vr1Out := s.projectVirtualReserves(now)

	// Pick the effective reserves used by the AMM (port of getSwapInfo).
	//
	// isToken0Out is mapped from the route direction:
	//   if indexOut == 0 -> token0 is output -> isToken0Out = true
	isToken0Out := indexOut == 0
	var effIn, effOut, realIn, realOut, totalBorrowedOut *uint256.Int
	if isToken0Out {
		// token1 in, token0 out
		realIn, realOut = s.reserve1, s.reserve0
		effIn = max256(vr1In, realIn)
		effOut = min256(vr0Out, realOut)
		totalBorrowedOut = s.totalBorrowed0
	} else {
		// token0 in, token1 out
		realIn, realOut = s.reserve0, s.reserve1
		effIn = max256(vr0In, realIn)
		effOut = min256(vr1Out, realOut)
		totalBorrowedOut = s.totalBorrowed1
	}

	// Compute amountOut on the EFFECTIVE reserves (the curve the contract uses).
	amountOut := getAmountOut(amountIn, effIn, effOut, totalFee)
	if amountOut.Sign() <= 0 {
		return nil, ErrInsufficientOutputAmount
	}
	if amountOut.Cmp(effOut) >= 0 {
		return nil, ErrInsufficientSwapLiquidity
	}

	// Available liquidity cap: amountOut < realReserveOut - totalBorrowedOut.
	// Borrowed tokens are physically lent out and cannot be transferred.
	cap := new(uint256.Int)
	if realOut.Cmp(totalBorrowedOut) <= 0 {
		return nil, ErrInsufficientLiquidity
	}
	cap.Sub(realOut, totalBorrowedOut)
	if amountOut.Cmp(cap) >= 0 {
		return nil, ErrInsufficientSwapLiquidity
	}

	// Spread validation (port of UniPoolPairSwap._validateSpreads). The contract
	// runs it with the post-swap reserves and the pre-swap projected VRs.
	netAmountIn := poolFeeNetIn(amountIn, s.feePoolBps)
	newReserveIn := new(uint256.Int).Add(realIn, netAmountIn)
	newReserveOut := new(uint256.Int).Sub(realOut, amountOut)
	if err := s.validateSpreads(
		isToken0Out, newReserveIn, newReserveOut,
		vr0In, vr0Out, vr1In, vr1Out,
	); err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: s.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: s.Info.Tokens[indexIn], Amount: big.NewInt(0)},
		Gas:            s.gas,
	}, nil
}

// poolFeeNetIn returns amountIn * (BPS - feePoolBps) / BPS, mirroring the on-chain
// `netAmountIn = amountIn - poolFeeAmount` computation.
func poolFeeNetIn(amountIn, feePoolBps *uint256.Int) *uint256.Int {
	one := new(uint256.Int).Sub(bpsDivisorU, feePoolBps)
	out := new(uint256.Int).Mul(amountIn, one)
	out.Div(out, bpsDivisorU)
	return out
}

// CalcAmountIn is the exact-out counterpart of CalcAmountOut: given a desired
// amountOut, compute the minimum amountIn required. Port of UniPoolPairSwap's
// `getAmountIn` (UniPoolPairSwap.sol) which rounds the division UP so
// the user always supplies at least enough input on-chain.
func (s *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenAmountOut, tokenIn := params.TokenAmountOut, params.TokenIn
	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow || amountOut.Sign() <= 0 {
		return nil, ErrInvalidAmountOut
	}

	totalFee := new(uint256.Int).Add(s.feeLpBps, s.feePoolBps)
	if totalFee.Cmp(bpsDivisorU) >= 0 {
		return nil, ErrFeeExceedsMax
	}

	now := uint64(nowUnix())
	vr0In, vr0Out, vr1In, vr1Out := s.projectVirtualReserves(now)

	isToken0Out := indexOut == 0
	var effIn, effOut, realOut, totalBorrowedOut *uint256.Int
	if isToken0Out {
		realOut = s.reserve0
		effIn = max256(vr1In, s.reserve1)
		effOut = min256(vr0Out, s.reserve0)
		totalBorrowedOut = s.totalBorrowed0
	} else {
		realOut = s.reserve1
		effIn = max256(vr0In, s.reserve0)
		effOut = min256(vr1Out, s.reserve1)
		totalBorrowedOut = s.totalBorrowed1
	}

	// Both bounds apply in exact-out: cannot consume more than the effective
	// curve allows AND cannot transfer out more than reserve - borrowed.
	if amountOut.Cmp(effOut) >= 0 {
		return nil, ErrInsufficientSwapLiquidity
	}
	if realOut.Cmp(totalBorrowedOut) <= 0 {
		return nil, ErrInsufficientLiquidity
	}
	cap := new(uint256.Int).Sub(realOut, totalBorrowedOut)
	if amountOut.Cmp(cap) >= 0 {
		return nil, ErrInsufficientSwapLiquidity
	}

	amountIn := getAmountInRoundUp(amountOut, effIn, effOut, totalFee)
	if amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: s.Info.Tokens[indexIn], Amount: amountIn.ToBig()},
		Fee:           &pool.TokenAmount{Token: s.Info.Tokens[indexIn], Amount: big.NewInt(0)},
		Gas:           s.gas,
	}, nil
}

// getAmountInRoundUp ports UniPoolPairSwap.getAmountIn (fullMulDivUpChecked):
//
//	amountIn = ⌈(reserveIn * amountOut * BPS) / ((reserveOut - amountOut) * (BPS - totalFee))⌉
//
// On-chain reserves and amountOut are uint128, so reserveIn * amountOut fits in
// uint256 (2 * 128 bits). Multiplying by BPS (~14 bits) can push the
// intermediate above 2^256 in adversarial inputs. We do the rounded-up
// 3-factor multiplication in math/big to stay precision-correct in all cases;
// the cost is negligible since this runs once per CalcAmountIn call.
func getAmountInRoundUp(amountOut, reserveIn, reserveOut, totalFee *uint256.Int) *uint256.Int {
	if reserveOut.Cmp(amountOut) <= 0 {
		// amountOut >= reserveOut: undefined output, return 0 (caller rejects).
		return new(uint256.Int)
	}

	rIn := reserveIn.ToBig()
	aOut := amountOut.ToBig()
	rOut := reserveOut.ToBig()
	fee := totalFee.ToBig()
	bps := big.NewInt(int64(bpsDivisor))

	numerator := new(big.Int).Mul(rIn, aOut)
	numerator.Mul(numerator, bps)

	denominator := new(big.Int).Sub(rOut, aOut)
	denominator.Mul(denominator, new(big.Int).Sub(bps, fee))
	if denominator.Sign() == 0 {
		return new(uint256.Int)
	}

	// ceil(num / den) = (num + den - 1) / den
	tmp := new(big.Int).Sub(denominator, big.NewInt(1))
	tmp.Add(tmp, numerator)
	tmp.Quo(tmp, denominator)

	out, overflow := uint256.FromBig(tmp)
	if overflow {
		// amountIn that overflows uint256 is unreachable on-chain anyway
		// (the contract caps amounts to uint128). Return 0 so the caller's
		// `Sign() <= 0` guard kicks in and rejects the request.
		return new(uint256.Int)
	}
	return out
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.reserve0 = new(uint256.Int).Set(s.reserve0)
	cloned.reserve1 = new(uint256.Int).Set(s.reserve1)
	cloned.vr0In = new(uint256.Int).Set(s.vr0In)
	cloned.vr0Out = new(uint256.Int).Set(s.vr0Out)
	cloned.vr1In = new(uint256.Int).Set(s.vr1In)
	cloned.vr1Out = new(uint256.Int).Set(s.vr1Out)
	cloned.totalBorrowed0 = new(uint256.Int).Set(s.totalBorrowed0)
	cloned.totalBorrowed1 = new(uint256.Int).Set(s.totalBorrowed1)

	// Deep-copy the embedded pool.Pool.Info.Reserves slice — `cloned := *s`
	// only shallow-copies the slice header, so the backing array would
	// otherwise be shared and UpdateBalance on the clone would mutate the
	// original.
	if s.Info.Reserves != nil {
		clonedReserves := make([]*big.Int, len(s.Info.Reserves))
		for i, r := range s.Info.Reserves {
			if r != nil {
				clonedReserves[i] = new(big.Int).Set(r)
			}
		}
		cloned.Info.Reserves = clonedReserves
	}
	return &cloned
}

// UpdateBalance applies a swap to local state so subsequent CalcAmountOut calls
// on the same simulator see the post-swap reserves.
//
// We mirror the on-chain UniPoolPairSwap.swap finalization:
//   - real reserves: in += netAmountIn (= amountIn * (BPS - feePoolBps)/BPS), out -= amountOut
//   - virtualReserveIn  of input token  set to effectiveIn + netAmountIn  (if it was clamped up)
//   - virtualReserveOut of output token set to effectiveOut - amountOut   (if it was clamped down)
//
// The "other" two VRs (in-side of output token, out-side of input token) are not
// touched by the swap; they continue to decay normally.
//
// We also reset lastUpdateTimestamp to the wall clock so subsequent calls use
// elapsed=0, matching the contract's State.updateState() behavior.
func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn := s.GetTokenIndex(params.TokenAmountIn.Token)
	indexOut := s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	amtIn, _ := uint256.FromBig(params.TokenAmountIn.Amount)
	amtOut, _ := uint256.FromBig(params.TokenAmountOut.Amount)
	if amtIn == nil || amtOut == nil {
		return
	}

	netAmountIn := poolFeeNetIn(amtIn, s.feePoolBps)

	now := uint64(nowUnix())
	vr0In, vr0Out, vr1In, vr1Out := s.projectVirtualReserves(now)

	var reserveIn, reserveOut, effIn, effOut, newReserveIn, newReserveOut *uint256.Int
	if indexOut == 0 {
		reserveIn, reserveOut = s.reserve1, s.reserve0
		effIn = max256(vr1In, reserveIn)
		effOut = min256(vr0Out, reserveOut)
	} else {
		reserveIn, reserveOut = s.reserve0, s.reserve1
		effIn = max256(vr0In, reserveIn)
		effOut = min256(vr1Out, reserveOut)
	}
	newReserveIn = new(uint256.Int).Add(reserveIn, netAmountIn)
	newReserveOut = new(uint256.Int).Sub(reserveOut, amtOut)

	// Write back real reserves.
	if indexOut == 0 {
		s.reserve1.Set(newReserveIn)
		s.reserve0.Set(newReserveOut)
	} else {
		s.reserve0.Set(newReserveIn)
		s.reserve1.Set(newReserveOut)
	}

	// Update VR_in of input token if MEV protection was active on the in side.
	if effIn.Cmp(reserveIn) != 0 {
		newVRIn := new(uint256.Int).Add(effIn, netAmountIn)
		if indexOut == 0 {
			s.vr1In.Set(newVRIn)
		} else {
			s.vr0In.Set(newVRIn)
		}
	}
	// Update VR_out of output token if MEV protection was active on the out side.
	if effOut.Cmp(reserveOut) != 0 {
		newVROut := new(uint256.Int).Sub(effOut, amtOut)
		if indexOut == 0 {
			s.vr0Out.Set(newVROut)
		} else {
			s.vr1Out.Set(newVROut)
		}
	}

	// On-chain, every state-changing call refreshes lastUpdateTimestamp to
	// block.timestamp. Mirror that so the next projection uses elapsed=0.
	s.lastUpdateTimestamp = now

	// Update the parent reserves slice used by some router code paths.
	s.Info.Reserves[0] = s.reserve0.ToBig()
	s.Info.Reserves[1] = s.reserve1.ToBig()
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any { return nil }

// projectVirtualReserves replays UniPoolPairGetters.previewVirtualReservesElapsed:
//
//	if elapsed >= decay : VR  = reserve
//	else                : VR  = (VR_stored*(decay-elapsed) + reserve*elapsed) / decay
//
// We use *uint256.Int to stay within the contract's domain (uint128 reserves,
// uint128 VRs, decay <= a few minutes — no overflow risk in the multiplications).
func (s *PoolSimulator) projectVirtualReserves(now uint64) (vr0In, vr0Out, vr1In, vr1Out *uint256.Int) {
	// Match the contract's three-way split in previewVirtualReservesElapsed:
	//   elapsed == 0          -> stored VRs (no time passed)
	//   elapsed >= priceDecay -> VR == reserve (fully converged; covers decay==0)
	//   else                  -> linear interpolation
	if now <= s.lastUpdateTimestamp {
		return new(uint256.Int).Set(s.vr0In), new(uint256.Int).Set(s.vr0Out),
			new(uint256.Int).Set(s.vr1In), new(uint256.Int).Set(s.vr1Out)
	}
	elapsed := now - s.lastUpdateTimestamp
	if s.priceDecay == 0 || elapsed >= s.priceDecay {
		return new(uint256.Int).Set(s.reserve0), new(uint256.Int).Set(s.reserve0),
			new(uint256.Int).Set(s.reserve1), new(uint256.Int).Set(s.reserve1)
	}

	decay := uint256.NewInt(s.priceDecay)
	elapsedU := uint256.NewInt(elapsed)
	diffU := new(uint256.Int).Sub(decay, elapsedU)

	r0Elapsed := new(uint256.Int).Mul(elapsedU, s.reserve0)
	r1Elapsed := new(uint256.Int).Mul(elapsedU, s.reserve1)

	vr0In = interpolate(s.vr0In, diffU, r0Elapsed, decay)
	vr0Out = interpolate(s.vr0Out, diffU, r0Elapsed, decay)
	vr1In = interpolate(s.vr1In, diffU, r1Elapsed, decay)
	vr1Out = interpolate(s.vr1Out, diffU, r1Elapsed, decay)
	return
}

func interpolate(vrStored, diff, reserveElapsed, decay *uint256.Int) *uint256.Int {
	tmp := new(uint256.Int).Mul(vrStored, diff)
	tmp.Add(tmp, reserveElapsed)
	return tmp.Div(tmp, decay)
}

// getAmountOut is the port of UniPoolPairSwap.getAmountOut (CPMM with bps fees):
//
//	amountInWithFee = amountIn * (BPS - totalFee)
//	amountOut       = (reserveOut * amountInWithFee) / (reserveIn * BPS + amountInWithFee)
func getAmountOut(amountIn, reserveIn, reserveOut, totalFee *uint256.Int) *uint256.Int {
	amountInWithFee := new(uint256.Int).Sub(bpsDivisorU, totalFee)
	amountInWithFee.Mul(amountIn, amountInWithFee)

	denominator := new(uint256.Int).Mul(reserveIn, bpsDivisorU)
	denominator.Add(denominator, amountInWithFee)
	if denominator.Sign() == 0 {
		return new(uint256.Int)
	}

	numerator := new(uint256.Int).Mul(reserveOut, amountInWithFee)
	return numerator.Div(numerator, denominator)
}

func max256(a, b *uint256.Int) *uint256.Int {
	if a.Cmp(b) >= 0 {
		return a
	}
	return b
}

func min256(a, b *uint256.Int) *uint256.Int {
	if a.Cmp(b) <= 0 {
		return a
	}
	return b
}

// validateSpreads is the port of UniPoolPairSwap._validateSpreads.
//
// It only checks the spread on the side OPPOSITE to the swap direction, using the
// pre-swap (projected) virtual reserves. The active side is intentionally NOT
// checked so that healing swaps (e.g. after a liquidation that left the active
// side above tolerance) are not blocked; MEV stays bounded by the opposite-side
// check. The contract reverts with UniPoolPairExcessiveSpread when:
//
//	|reserveIn*max - reserveOut*min| * BPS  >  tolerance * min(reserveOut*min, reserveIn*max)
//
// where (max, min) select the opposite-side virtual reserves. reserveIn/Out are
// the POST-swap reserves. We compute in math/big to avoid uint256 overflow on the
// `abs * BPS` and `tolerance * comp` products (worst case ~2^286).
func (s *PoolSimulator) validateSpreads(
	isToken0Out bool,
	newReserveIn, newReserveOut *uint256.Int,
	vr0In, vr0Out, vr1In, vr1Out *uint256.Int,
) error {
	if s.swapPriceToleranceBps == swapPriceToleranceDisabled {
		return nil
	}

	toBI := func(u *uint256.Int) *big.Int { return u.ToBig() }
	bigMax := func(a, b *big.Int) *big.Int {
		if a.Cmp(b) >= 0 {
			return a
		}
		return b
	}
	bigMin := func(a, b *big.Int) *big.Int {
		if a.Cmp(b) <= 0 {
			return a
		}
		return b
	}

	rIn := toBI(newReserveIn)
	rOut := toBI(newReserveOut)
	v0InB, v0OutB := toBI(vr0In), toBI(vr0Out)
	v1InB, v1OutB := toBI(vr1In), toBI(vr1Out)

	// Select the opposite-side virtual reserves.
	var maxVal, minVal *big.Int
	if isToken0Out {
		maxVal = bigMax(rOut, v0InB)
		minVal = bigMin(rIn, v1OutB)
	} else {
		maxVal = bigMax(rOut, v1InB)
		minVal = bigMin(rIn, v0OutB)
	}

	rLmax := new(big.Int).Mul(rIn, maxVal)
	rRmin := new(big.Int).Mul(rOut, minVal)
	abs := new(big.Int).Sub(rLmax, rRmin)
	abs.Abs(abs)
	comp := bigMin(rRmin, rLmax)

	lhs := new(big.Int).Mul(abs, big.NewInt(bpsDivisor))
	rhs := new(big.Int).Mul(big.NewInt(int64(s.swapPriceToleranceBps)), comp)
	if lhs.Cmp(rhs) > 0 {
		return ErrExcessiveSpread
	}
	return nil
}
