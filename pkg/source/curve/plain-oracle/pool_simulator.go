package plainoracle

import (
	"errors"
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
	Multipliers []*uint256.Int
	// extra fields
	Rates        []*uint256.Int
	InitialA     *uint256.Int
	FutureA      *uint256.Int
	InitialATime int64
	FutureATime  int64
	AdminFee     *uint256.Int
	LpToken      string
	LpSupply     *uint256.Int
	APrecision   *uint256.Int
	gas          curve.Gas
}

var (
	_ pool.IPoolSim = (*PoolSimulator)(nil)
	_               = pool.RegisterFactory0(curve.PoolTypePlainOracle, NewPoolSimulator)
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra curve.PoolPlainOracleStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra curve.PoolPlainOracleExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var numTokens = len(entityPool.Tokens)
	var tokens = make([]string, numTokens)
	var reserves = make([]*uint256.Int, numTokens)
	var multipliers = make([]*uint256.Int, numTokens)
	var rates = make([]*uint256.Int, numTokens)

	for i := range numTokens {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = big256.New(entityPool.Reserves[i])
		multipliers[i] = big256.New(staticExtra.PrecisionMultipliers[i])
		rates[i] = big256.FromBig(extra.Rates[i])
	}

	var aPrecision = big256.U1
	if len(staticExtra.APrecision) > 0 {
		aPrecision = big256.New(staticExtra.APrecision)
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
		Multipliers:  multipliers,
		Rates:        rates,
		InitialA:     big256.New(extra.InitialA),
		FutureA:      big256.New(extra.FutureA),
		InitialATime: extra.InitialATime,
		FutureATime:  extra.FutureATime,
		AdminFee:     big256.New(extra.AdminFee),
		LpToken:      staticExtra.LpToken,
		LpSupply:     big256.New(entityPool.Reserves[numTokens]),
		APrecision:   aPrecision,
		gas:          DefaultGas,
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

// GetApprovalAddress returns the empty string: plain-oracle pools need no approval address.
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

// CalcLimit returns nil: plain-oracle pools have no shared inventory limit.
func (t *PoolSimulator) CalcLimit() map[string]*uint256.Int {
	return nil
}

func (t *PoolSimulator) CloneState() pool.IPoolSim {
	cloned := *t
	cloned.Info.Reserves = cloneU256s(t.Info.Reserves)
	return &cloned
}

func (t *PoolSimulator) CalcAmtOut(param pool.CalcAmtOutParams) (*pool.CalcAmtOutResult, error) {
	indexIn, indexOut := t.GetTokenIndex(param.TokenAmountIn.Token), t.GetTokenIndex(param.TokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, fmt.Errorf("tokenIndexFrom %v or TokenOutIndex %v is not correct", indexIn, indexOut)
	}
	amountOut, fee, err := t.GetDy(indexIn, indexOut, &param.TokenAmountIn.Amount, nil)
	if err != nil {
		return nil, err
	}
	if amountOut.IsZero() {
		return nil, errors.New("[core.CurvePlainOracle] - GetDy returns 0")
	}
	return &pool.CalcAmtOutResult{
		TokenAmountOut: &pool.TokenAmt{Token: param.TokenOut, Amount: *amountOut},
		Fee:            &pool.TokenAmt{Token: param.TokenOut, Amount: *fee},
		Gas:            t.gas.Exchange,
	}, nil
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
