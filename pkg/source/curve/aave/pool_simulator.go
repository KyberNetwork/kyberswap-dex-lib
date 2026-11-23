package aave

import (
	"slices"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type PoolSimulator struct {
	Info             base.PoolInfo
	LpToken          string
	UnderlyingTokens []string
	Multipliers      []*uint256.Int
	// extra fields
	InitialA            *uint256.Int
	FutureA             *uint256.Int
	InitialATime        int64
	FutureATime         int64
	AdminFee            *uint256.Int
	OffpegFeeMultiplier *uint256.Int
	gas                 curve.Gas

	LpSupply *uint256.Int
}

var (
	_ pool.IPoolSim = (*PoolSimulator)(nil)
	_               = pool.RegisterFactory0(curve.PoolTypeAave, NewPoolSimulator)
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra curve.PoolAaveStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra curve.PoolAaveExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var numTokens = len(entityPool.Tokens)
	var tokens = make([]string, numTokens)
	var reserves = make([]*uint256.Int, numTokens)
	var multipliers = make([]*uint256.Int, numTokens)
	for i := range numTokens {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = big256.New(entityPool.Reserves[i])
		multipliers[i] = big256.New(staticExtra.PrecisionMultipliers[i])
	}

	lpSupply := big256.U1
	if len(entityPool.Reserves) > numTokens {
		lpSupply = big256.New(entityPool.Reserves[numTokens])
	}

	return &PoolSimulator{
		Info: base.PoolInfo{
			Address:     strings.ToLower(entityPool.Address),
			SwapFee:     big256.New(extra.SwapFee),
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
		},
		LpToken:             staticExtra.LpToken,
		UnderlyingTokens:    staticExtra.UnderlyingTokens,
		Multipliers:         multipliers,
		InitialA:            big256.New(extra.InitialA),
		FutureA:             big256.New(extra.FutureA),
		InitialATime:        extra.InitialATime,
		FutureATime:         extra.FutureATime,
		AdminFee:            big256.New(extra.AdminFee),
		OffpegFeeMultiplier: big256.New(extra.OffpegFeeMultiplier),
		gas:                 DefaultGas,
		LpSupply:            lpSupply,
	}, nil
}

func (t *PoolSimulator) GetInfo() base.PoolInfo {
	return t.Info
}

func (t *PoolSimulator) GetTokenIndex(address string) int {
	return t.Info.GetTokenIndex(address)
}

func (t *PoolSimulator) GetAddress() string {
	return t.Info.Address
}

func (t *PoolSimulator) GetExchange() string {
	return t.Info.Exchange
}

func (t *PoolSimulator) GetType() string {
	return t.Info.Type
}

func (t *PoolSimulator) GetReserves() []uint256.Int {
	out := make([]uint256.Int, len(t.Info.Reserves))
	for i, r := range t.Info.Reserves {
		out[i].Set(r)
	}
	return out
}

// CalcLimit returns nil: aave pools have no shared inventory limit.
func (t *PoolSimulator) CalcLimit() map[string]*uint256.Int {
	return nil
}

// GetApprovalAddress returns the empty string: aave pools need no approval address.
func (t *PoolSimulator) GetApprovalAddress(tokenIn, tokenOut string) string {
	return ""
}

func (t *PoolSimulator) CloneState() pool.IPoolSim {
	cloned := *t
	cloned.Info.Reserves = cloneU256s(t.Info.Reserves)
	return &cloned
}

func (t *PoolSimulator) CalcAmtOut(param pool.CalcAmtOutParams) (*pool.CalcAmtOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	idxIn, idxOut := t.GetTokenIndex(tokenAmountIn.Token), t.GetTokenIndex(tokenOut)
	if idxIn >= 0 && idxOut >= 0 {
		A := _getAPrecise(t.FutureATime, t.FutureA, t.InitialATime, t.InitialA)
		D, err := t.getDPrecision(t.Info.Reserves, A)
		if err != nil {
			return nil, err
		}
		amountOut, fee, err := t.GetDy(idxIn, idxOut, &tokenAmountIn.Amount, D)
		if err != nil {
			return nil, err
		}
		if amountOut.Sign() > 0 {
			return &pool.CalcAmtOutResult{
				TokenAmountOut: &pool.TokenAmt{Token: tokenOut, Amount: *amountOut},
				Fee:            &pool.TokenAmt{Token: tokenOut, Amount: *fee},
				Gas:            t.gas.Exchange,
			}, nil
		}
		return nil, ErrInvalidAmountOut
	} else if DepositFrozen {
		return nil, ErrTokenNotFound
	}

	idxIn, idxOut = slices.Index(t.UnderlyingTokens, tokenAmountIn.Token), slices.Index(t.UnderlyingTokens, tokenOut)
	if idxIn < 0 || idxOut < 0 {
		return nil, ErrTokenNotFound
	}
	amountOut, fee, err := GetDyUnderlying(
		t.Info.Reserves,
		t.Multipliers,
		t.FutureATime,
		t.FutureA,
		t.InitialATime,
		t.InitialA,
		t.Info.SwapFee,
		t.OffpegFeeMultiplier,
		idxIn,
		idxOut,
		&tokenAmountIn.Amount,
	)
	if err != nil {
		return nil, err
	}
	if amountOut.Sign() > 0 {
		return &pool.CalcAmtOutResult{
			TokenAmountOut: &pool.TokenAmt{Token: tokenOut, Amount: *amountOut},
			Fee:            &pool.TokenAmt{Token: tokenOut, Amount: *fee},
			Gas:            t.gas.Exchange,
		}, nil
	}
	return nil, ErrInvalidAmountOut
}

// UpdateState mirrors the legacy UpdateBalance: the output also loses its admin share of the swap fee.
func (t *PoolSimulator) UpdateState(params pool.UpdateStateParams) {
	var outputAmount, adminShare uint256.Int
	outputAmount.Set(&params.TokenAmountOut.Amount)
	adminShare.Mul(&outputAmount, t.Info.SwapFee)
	adminShare.Div(&adminShare, FeeDenominator)
	adminShare.Mul(&adminShare, t.AdminFee)
	adminShare.Div(&adminShare, FeeDenominator)
	outputAmount.Add(&outputAmount, &adminShare)
	for i := range t.Info.Tokens {
		if t.Info.Tokens[i] == params.TokenAmountIn.Token {
			t.Info.Reserves[i] = new(uint256.Int).Add(t.Info.Reserves[i], &params.TokenAmountIn.Amount)
		}
		if t.Info.Tokens[i] == params.TokenAmountOut.Token {
			t.Info.Reserves[i] = new(uint256.Int).Sub(t.Info.Reserves[i], &outputAmount)
		}
	}
}

func (t *PoolSimulator) CanSwapTo(address string) []string {
	if t.GetTokenIndex(address) != -1 {
		return lo.Filter(t.Info.Tokens, func(item string, _ int) bool { return item != address })
	}
	// check from underlying
	if DepositFrozen || slices.Index(t.UnderlyingTokens, address) < 0 {
		return nil
	}
	// exchange_underlying can swap underlying token to other underlying tokens
	return lo.Filter(t.UnderlyingTokens, func(item string, _ int) bool {
		return item != address
	})
}

func (t *PoolSimulator) CanSwapFrom(address string) []string { return t.CanSwapTo(address) }

func (t *PoolSimulator) GetTokens() []string {
	if DepositFrozen {
		return t.Info.Tokens
	}
	result := make([]string, 0, 2*len(t.UnderlyingTokens))
	result = append(result, t.Info.Tokens...)
	result = append(result, t.UnderlyingTokens...)
	return result
}

func (t *PoolSimulator) GetLpToken() string {
	return t.LpToken
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	return curve.Meta{
		TokenInIndex:  t.GetTokenIndex(tokenIn),
		TokenOutIndex: t.GetTokenIndex(tokenOut),
		Underlying:    false,
		BlockNumber:   t.Info.BlockNumber,
	}
}

func cloneU256s(xs []*uint256.Int) []*uint256.Int {
	out := make([]*uint256.Int, len(xs))
	for i, x := range xs {
		out[i] = new(uint256.Int).Set(x)
	}
	return out
}

func (t *PoolSimulator) getDPrecision(xp []*uint256.Int, a *uint256.Int) (*uint256.Int, error) {
	var _xp = make([]*uint256.Int, len(xp))
	for i := range xp {
		_xp[i] = new(uint256.Int).Mul(t.Multipliers[i], xp[i])
	}
	return getD(_xp, a)
}

func (t *PoolSimulator) GetVirtualPrice() (*uint256.Int, *uint256.Int, error) {
	A := _getAPrecise(t.FutureATime, t.FutureA, t.InitialATime, t.InitialA)
	D, err := t.getDPrecision(t.Info.Reserves, A)
	if err != nil {
		return nil, nil, err
	}
	if t.LpSupply.IsZero() {
		return nil, nil, ErrDenominatorZero
	}
	var vPrice uint256.Int
	vPrice.Mul(D, Precision)
	vPrice.Div(&vPrice, t.LpSupply)
	return &vPrice, D, nil
}

func (t *PoolSimulator) GetDy(i int, j int, dx *uint256.Int, dCached *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	var nTokens = len(t.Info.Tokens)
	xp := make([]*uint256.Int, nTokens)
	for k := range nTokens {
		xp[k] = new(uint256.Int).Mul(t.Multipliers[k], t.Info.Reserves[k])
	}

	// x: uint256 = xp[i] + dx * precisions[i]
	var x uint256.Int
	x.Mul(dx, t.Multipliers[i])
	x.Add(&x, xp[i])

	// y: uint256 = self.get_y(i, j, x, xp)
	y, err := getY(t.FutureATime, t.FutureA, t.InitialATime, t.InitialA, i, j, &x, xp, dCached)
	if err != nil {
		return nil, nil, err
	}

	// dy: uint256 = (xp[j] - y) / precisions[j]; legacy goes negative when y > xp[j]
	if xp[j].Cmp(y) < 0 {
		return nil, nil, ErrNegativeResult
	}
	var dy uint256.Int
	dy.Sub(xp[j], y)
	dy.Div(&dy, t.Multipliers[j])

	// _fee: uint256 = self._dynamic_fee((xp[i] + x) / 2, (xp[j] + y) / 2, self.fee, self.offpeg_fee_multiplier) * dy / FEE_DENOMINATOR
	var xi, xj uint256.Int
	xi.Add(xp[i], &x)
	xi.Div(&xi, big256.U2)
	xj.Add(xp[j], y)
	xj.Div(&xj, big256.U2)
	dynFee, err := _dynamicFee(&xi, &xj, t.Info.SwapFee, t.OffpegFeeMultiplier)
	if err != nil {
		return nil, nil, err
	}
	var fee uint256.Int
	fee.Mul(dynFee, &dy)
	fee.Div(&fee, FeeDenominator)

	// return dy - _fee; legacy goes negative when the fee exceeds dy
	if dy.Cmp(&fee) < 0 {
		return nil, nil, ErrNegativeResult
	}
	dy.Sub(&dy, &fee)
	return &dy, &fee, nil
}
