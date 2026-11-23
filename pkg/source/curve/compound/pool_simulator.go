package compound

import (
	"fmt"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type PoolSimulator struct {
	Info        base.PoolInfo
	A           *uint256.Int
	Multipliers []*uint256.Int
	AdminFee    *uint256.Int
	Rates       []*uint256.Int
	gas         Gas
}

type Gas struct {
	Exchange           int64
	ExchangeUnderlying int64
}

var (
	_ pool.IPoolSim = (*PoolSimulator)(nil)
	_               = pool.RegisterFactory0(curve.PoolTypeCompound, NewPoolSimulator)
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra curve.PoolCompoundStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra curve.PoolCompoundExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*uint256.Int, numTokens)
	multipliers := make([]*uint256.Int, numTokens)
	for i := range numTokens {
		tokens[i] = staticExtra.UnderlyingTokens[i]
		reserves[i] = big256.New(entityPool.Reserves[i])
		multipliers[i] = big256.New(staticExtra.PrecisionMultipliers[i])
	}

	rates := make([]*uint256.Int, 0, len(extra.Rates))
	for _, rateStr := range extra.Rates {
		rate, err := uint256.FromDecimal(rateStr)
		if err != nil {
			// legacy ignores the parse error and keeps a nil rate, which panics on use; keep the failure visible here
			return nil, err
		}
		rates = append(rates, rate)
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
		Multipliers: multipliers,
		A:           big256.New(extra.A),
		AdminFee:    big256.New(extra.AdminFee),
		Rates:       rates,
		gas:         DefaultGas,
	}, nil
}

func (t *PoolSimulator) GetInfo() base.PoolInfo {
	return t.Info
}

func (t *PoolSimulator) GetTokens() []string {
	return t.Info.Tokens
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

func (t *PoolSimulator) GetTokenIndex(address string) int {
	return t.Info.GetTokenIndex(address)
}

func (t *PoolSimulator) CanSwapTo(address string) []string {
	if t.GetTokenIndex(address) == -1 {
		return nil
	}
	result := make([]string, 0, len(t.Info.Tokens)-1)
	for _, token := range t.Info.Tokens {
		if token != address {
			result = append(result, token)
		}
	}
	return result
}

func (t *PoolSimulator) CanSwapFrom(address string) []string {
	return t.CanSwapTo(address)
}

// GetApprovalAddress returns the empty string: compound pools need no approval address.
func (t *PoolSimulator) GetApprovalAddress(tokenIn, tokenOut string) string {
	return ""
}

func (t *PoolSimulator) GetReserves() []uint256.Int {
	out := make([]uint256.Int, len(t.Info.Reserves))
	for i, r := range t.Info.Reserves {
		out[i].Set(r)
	}
	return out
}

// CalcLimit returns nil: compound pools have no shared inventory limit.
func (t *PoolSimulator) CalcLimit() map[string]*uint256.Int {
	return nil
}

func (t *PoolSimulator) CloneState() pool.IPoolSim {
	cloned := *t
	cloned.Info.Reserves = cloneU256s(t.Info.Reserves)
	return &cloned
}

func (t *PoolSimulator) CalcAmtOut(param pool.CalcAmtOutParams) (*pool.CalcAmtOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	var tokenIndexFrom = t.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.GetTokenIndex(tokenOut)

	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		amountOut, fee, err := GetDyUnderlying(
			t.Info.Reserves,
			t.Rates,
			t.Multipliers,
			t.A,
			t.Info.SwapFee,
			tokenIndexFrom,
			tokenIndexTo,
			&tokenAmountIn.Amount,
		)
		if err != nil {
			return nil, err
		}
		if !amountOut.IsZero() {
			// UpdateState subtracts amountOut plus its admin share from the output reserve; reject a quote that
			// would drive that reserve negative, before any state is written (tricrypto-ng pattern).
			if t.outputWouldGoNegative(tokenIndexTo, amountOut) {
				return nil, ErrNegativeResult
			}
			return &pool.CalcAmtOutResult{
				TokenAmountOut: &pool.TokenAmt{Token: tokenOut, Amount: *amountOut},
				Fee:            &pool.TokenAmt{Token: tokenOut, Amount: *fee},
				Gas:            t.gas.ExchangeUnderlying,
			}, nil
		}
	}
	return nil, fmt.Errorf("tokenIndexFrom or tokenIndexTo is not correct: tokenIndexFrom: %v, tokenIndexTo: %v", tokenIndexFrom, tokenIndexTo)
}

func (t *PoolSimulator) CalcAmtIn(param pool.CalcAmtInParams) (*pool.CalcAmtInResult, error) {
	tokenIn := strings.ToLower(param.TokenIn)
	tokenAmountOut := param.TokenAmountOut
	tokenOut := strings.ToLower(tokenAmountOut.Token)
	var tokenIndexFrom = t.GetTokenIndex(tokenIn)
	var tokenIndexTo = t.GetTokenIndex(tokenOut)

	if tokenIndexFrom < 0 || tokenIndexTo < 0 {
		return nil, fmt.Errorf("tokenIndexFrom or tokenIndexTo is not correct: tokenIndexFrom: %v, tokenIndexTo: %v", tokenIndexFrom, tokenIndexTo)
	}

	amountIn, fee, err := GetDxUnderlying(
		t.Info.Reserves,
		t.Rates,
		t.Multipliers,
		t.A,
		t.Info.SwapFee,
		tokenIndexFrom,
		tokenIndexTo,
		&tokenAmountOut.Amount,
	)
	if err != nil {
		return nil, err
	}
	if amountIn.IsZero() {
		return nil, ErrZero
	}
	return &pool.CalcAmtInResult{
		TokenAmountIn: &pool.TokenAmt{Token: tokenIn, Amount: *amountIn},
		Fee:           &pool.TokenAmt{Token: tokenOut, Amount: *fee},
		Gas:           t.gas.ExchangeUnderlying,
	}, nil
}

// outputWouldGoNegative reports whether the output reserve cannot cover amountOut plus its admin share.
func (t *PoolSimulator) outputWouldGoNegative(tokenIndexOut int, amountOut *uint256.Int) bool {
	var share uint256.Int
	share.Mul(amountOut, t.Info.SwapFee)
	share.Div(&share, FeeDenominator)
	share.Mul(&share, t.AdminFee)
	share.Div(&share, FeeDenominator)
	var need uint256.Int
	need.Add(amountOut, &share)
	return t.Info.Reserves[tokenIndexOut].Cmp(&need) < 0
}

// UpdateState mirrors the legacy UpdateBalance: the output also loses its admin share of the swap fee.
// A reserve that would go negative returns ErrNegativeResult before any write.
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

func (t *PoolSimulator) GetLpToken() string {
	return ""
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	return curve.Meta{
		TokenInIndex:  t.GetTokenIndex(tokenIn),
		TokenOutIndex: t.GetTokenIndex(tokenOut),
		Underlying:    true,
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
