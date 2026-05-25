package fermi

import (
	"math/big"
	"slices"
	"strings"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	swapperAddress string
	token0         string
	token1         string
	curve          *CurveData
	blockNumber    uint64
	stateOverrides *StateOverrides
}

var (
	_ = pool.RegisterFactory0(DexType, NewPoolSimulator)
	_ = pool.RegisterUseSwapLimit(DexType)
)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(ep.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
			BlockNumber: ep.BlockNumber,
		}},
		swapperAddress: staticExtra.FermiSwapper,
		token0:         ep.Tokens[0].Address,
		token1:         ep.Tokens[1].Address,
		curve:          extra.Curve,
		blockNumber:    extra.BlockNumber,
		stateOverrides: extra.StateOverrides,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn := param.TokenAmountIn.Token
	amountIn := param.TokenAmountIn.Amount

	if amountIn == nil || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	if s.curve == nil {
		return nil, ErrCurveNotAvailable
	}

	idxIn := s.GetTokenIndex(tokenIn)
	idxOut := s.GetTokenIndex(param.TokenOut)
	if idxIn < 0 || idxOut < 0 || idxIn == idxOut {
		return nil, ErrInvalidToken
	}

	return s.evalCurveAmountOut(tokenIn, amountIn, param.Limit)
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if s.curve != nil {
		isBid := strings.EqualFold(params.TokenAmountIn.Token, s.token0)
		vr0 := bignumber.NewBig(s.curve.VaultReserve0)
		vr1 := bignumber.NewBig(s.curve.VaultReserve1)
		if vr0 == nil {
			vr0 = new(big.Int)
		}
		if vr1 == nil {
			vr1 = new(big.Int)
		}
		if isBid {
			vr0.Add(vr0, params.TokenAmountIn.Amount)
			vr1.Sub(vr1, params.TokenAmountOut.Amount)
		} else {
			vr1.Add(vr1, params.TokenAmountIn.Amount)
			vr0.Sub(vr0, params.TokenAmountOut.Amount)
		}
		if vr0.Sign() < 0 {
			vr0.SetInt64(0)
		}
		if vr1.Sign() < 0 {
			vr1.SetInt64(0)
		}
		// CloneState deep-copies Curve, so mutating the shared pointer is safe.
		curveCopy := *s.curve
		curveCopy.VaultReserve0 = vr0.String()
		curveCopy.VaultReserve1 = vr1.String()
		s.curve = &curveCopy
	}

	if limit := params.SwapLimit; limit != nil {
		_, _, _ = limit.UpdateLimit(
			params.TokenAmountOut.Token,
			params.TokenAmountIn.Token,
			params.TokenAmountOut.Amount,
			params.TokenAmountIn.Amount,
		)
	}
}

func (s *PoolSimulator) CalculateLimit() map[string]*big.Int {
	tokens, reserves := s.GetTokens(), s.GetReserves()
	out := make(map[string]*big.Int, len(tokens))
	for i, t := range tokens {
		if i < len(reserves) && reserves[i] != nil {
			out[t] = new(big.Int).Set(reserves[i])
		}
	}
	return out
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.Info.Reserves = slices.Clone(s.Info.Reserves)
	if s.curve != nil {
		curveCopy := *s.curve
		cloned.curve = &curveCopy
	}
	return &cloned
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{
		BlockNumber:    s.blockNumber,
		FermiSwapper:   s.swapperAddress,
		StateOverrides: s.stateOverrides,
	}
}

func (s *PoolSimulator) CanSwapTo(address string) []string {
	result := make([]string, 0, 1)
	for _, t := range s.Info.Tokens {
		if t != address {
			result = append(result, t)
		}
	}
	return result
}

func (s *PoolSimulator) evalCurveAmountOut(
	tokenIn string,
	amountIn *big.Int,
	limit pool.SwapLimit,
) (*pool.CalcAmountOutResult, error) {
	curve := s.curve
	if curve == nil {
		return nil, ErrCurveNotAvailable
	}
	if amountIn == nil || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	mid := bignumber.NewBig(curve.MidPrice)
	scalingDenom := bignumber.NewBig(curve.ScalingDenominator)
	maxIn := bignumber.NewBig(curve.MaxAmountIn)
	ds0 := bignumber.NewBig(curve.TokenInDecScale)
	ds1 := bignumber.NewBig(curve.TokenOutDecScale)
	vr0 := bignumber.NewBig(curve.VaultReserve0)
	vr1 := bignumber.NewBig(curve.VaultReserve1)
	if mid == nil || scalingDenom == nil || maxIn == nil || ds0 == nil ||
		ds1 == nil || vr0 == nil || vr1 == nil {
		return nil, ErrInvalidCurveData
	}
	if scalingDenom.Sign() <= 0 {
		return nil, ErrInvalidCurveData
	}

	isBid := tokenIn == s.token0
	var tokenOut string
	if isBid {
		tokenOut = s.token1
	} else {
		tokenOut = s.token0
	}

	vaultIn, vaultOut := vr0, vr1
	if limit != nil {
		if lim0 := limit.GetLimit(s.token0); lim0 != nil {
			vaultIn = new(big.Int).Set(lim0)
		}
		if lim1 := limit.GetLimit(s.token1); lim1 != nil {
			vaultOut = new(big.Int).Set(lim1)
		}
	}

	// Inventory adjustment (0x29ba lines 299-325).
	// The engine always converts vaultBal0 to token1 units using midPrice,
	// regardless of swap direction. ds0/ds1 are canonical scales.
	invRatio, err := inventoryRatio(vaultIn, vaultOut, mid, ds0, ds1, curve.SafetyFeeBps)
	if err != nil {
		return nil, err
	}
	invAdj, err := evalSpline(curve.InventorySpline, invRatio)
	if err != nil {
		return nil, err
	}

	// Inventory-adjusted price (engine line 353): v54 = priceFactor(midPrice, inv_adj).
	withInv := priceFactor(mid, invAdj)
	if withInv == nil {
		return nil, ErrZeroEffectivePrice
	}

	// Size adjustment.
	// Bid (token0→token1): convert amountIn to token1 units first via 0x36cb,
	// then normalise by scalingDenominator.
	// Ask (token1→token0): amountIn is already in token1 units.
	var sizeInput *big.Int
	if isBid {
		sizeInput = new(big.Int).Mul(amountIn, mid)
		sizeInput.Quo(sizeInput, oneE8)
		sizeInput.Mul(sizeInput, ds1)
		sizeInput.Quo(sizeInput, ds0)
	} else {
		sizeInput = new(big.Int).Set(amountIn)
	}

	// MaxAmountIn cap (0 means uncapped). Denominated in token1 raw units.
	if maxIn.Sign() > 0 && sizeInput.Cmp(maxIn) > 0 {
		return nil, ErrInsufficientLiquidity
	}
	sizeNorm := new(big.Int).Mul(sizeInput, oneE18)
	sizeNorm.Quo(sizeNorm, scalingDenom)
	sizeAdj, err := evalSpline(curve.SizeSpline, sizeNorm)
	if err != nil {
		return nil, err
	}
	// Engine clamps negative size_adj to zero (line 350-352).
	if sizeAdj.Sign() < 0 {
		sizeAdj = new(big.Int)
	}

	// Combined adjustment (engine lines 355-359):
	//   v56 = feeBaseBps * 1e18 + size_adj
	//   if (ask) v56 = -v56   ← negation reduces output symmetrically
	v56 := new(big.Int).SetInt64(int64(curve.FeeBaseBps))
	v56.Mul(v56, oneE18)
	v56.Add(v56, sizeAdj)
	if !isBid {
		v56.Neg(v56)
	}

	eff := priceFactor(withInv, v56)
	if eff == nil || eff.Sign() <= 0 {
		return nil, ErrZeroEffectivePrice
	}

	var amountOut *big.Int
	if isBid {
		amountOut = convertOutForward(amountIn, eff, ds0, ds1)
	} else {
		amountOut = convertOutReverse(amountIn, eff, ds0, ds1)
	}
	if amountOut.Sign() <= 0 {
		return nil, ErrZeroAmountOut
	}

	vaultPayout := vaultOut
	if !isBid {
		vaultPayout = vaultIn
	}
	if vaultPayout.Sign() == 0 {
		return nil, ErrInsufficientLiquidity
	}
	if limit != nil && amountOut.Cmp(vaultPayout) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmount{Token: tokenIn, Amount: bignumber.ZeroBI},
		Gas:            defaultGas,
	}, nil
}
