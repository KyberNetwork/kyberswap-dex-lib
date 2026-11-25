package base

import (
	"errors"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type PoolSimulator struct {
	Info        PoolInfo
	Multipliers []*uint256.Int
	Rates       []*uint256.Int
	// extra fields
	InitialA     *uint256.Int
	FutureA      *uint256.Int
	InitialATime int64
	FutureATime  int64
	AdminFee     *uint256.Int
	LpToken      string
	LpSupply     *uint256.Int
	APrecision   *uint256.Int
	gas          curve.Gas
	numTokensBI  *uint256.Int
}

// PoolInfo is the uint256 pool state of a curve base pool.
type PoolInfo struct {
	Address     string
	Exchange    string
	Type        string
	Tokens      []string
	Reserves    []*uint256.Int
	SwapFee     *uint256.Int
	BlockNumber uint64
}

func (i *PoolInfo) GetTokenIndex(address string) int {
	for idx, token := range i.Tokens {
		if token == address {
			return idx
		}
	}
	return -1
}

var (
	_ pool.IPoolSim = (*PoolSimulator)(nil)
	_               = pool.RegisterFactory0(curve.PoolTypeBase, NewPoolSimulator)
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra curve.PoolBaseStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra curve.PoolBaseExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var numTokens = len(entityPool.Tokens)
	if entityPool.Reserves == nil || len(entityPool.Reserves) < numTokens {
		return nil, errors.New("empty reserve")
	}

	var tokens = make([]string, numTokens)
	var reserves = make([]*uint256.Int, numTokens)
	var multipliers = make([]*uint256.Int, numTokens)
	var rates = make([]*uint256.Int, numTokens)

	for i := range numTokens {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = big256.New(entityPool.Reserves[i])
		multipliers[i] = big256.New(staticExtra.PrecisionMultipliers[i])
		rates[i] = big256.New(staticExtra.Rates[i])
	}

	var aPrecision = big256.U1
	if len(staticExtra.APrecision) > 0 {
		aPrecision = big256.New(staticExtra.APrecision)
	}

	return &PoolSimulator{
		Info: PoolInfo{
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
		numTokensBI:  uint256.NewInt(uint64(numTokens)),
	}, nil
}

func (t *PoolSimulator) GetInfo() PoolInfo {
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

// GetApprovalAddress returns the empty string: curve base pools need no approval address.
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

// CalcLimit returns nil: curve base pools have no shared inventory limit.
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
		return nil, errors.New("token index is not correct")
	}
	amountOut, _, err := t.GetDy(indexIn, indexOut, &param.TokenAmountIn.Amount, nil)
	if err != nil {
		return nil, err
	}
	if amountOut.IsZero() {
		return nil, errors.New("zero output")
	}
	return &pool.CalcAmtOutResult{
		TokenAmountOut: &pool.TokenAmt{Token: param.TokenOut, Amount: *amountOut},
		Gas:            t.gas.Exchange,
	}, nil
}

// UpdateState mirrors the big.Int UpdateBalance: the output also loses its admin share of the swap fee.
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
