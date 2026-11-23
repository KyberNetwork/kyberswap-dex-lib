package stableng

import (
	"fmt"
	"slices"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Info is the pool identity and token list. Reserves live in PoolSimulator.Reserves, the uint256 source of truth.
type Info struct {
	Address     string
	Exchange    string
	Type        string
	Tokens      []string
	SwapFee     *uint256.Int
	BlockNumber uint64
}

func (i *Info) GetTokenIndex(address string) int {
	for idx, token := range i.Tokens {
		if token == address {
			return idx
		}
	}
	return -1
}

type PoolSimulator struct {
	Info Info

	precisionMultipliers []uint256.Int
	Reserves             []uint256.Int

	LpSupply uint256.Int
	gas      Gas

	NumTokens     int
	NumTokensU256 uint256.Int

	Extra       Extra
	StaticExtra StaticExtra

	dCache dCache       `msgpack:"-"`
	now    func() int64 `msgpack:"-"` // test clock; nil means time.Now
}

type Gas struct {
	Exchange int64
}

var (
	_ pool.IPoolSim = (*PoolSimulator)(nil)
	_               = pool.RegisterFactory0(DexType, NewPoolSimulator)
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	sim := &PoolSimulator{}

	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &sim.StaticExtra); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(entityPool.Extra), &sim.Extra); err != nil {
		return nil, err
	}

	var numTokens = len(entityPool.Tokens)
	// Reserves: N tokens & lpSupply
	if entityPool.Reserves == nil || len(entityPool.Reserves) != numTokens+1 {
		return nil, ErrInvalidReserve
	}

	if numTokens > shared.MaxTokenCount {
		return nil, ErrInvalidNumToken
	}

	var tokens = make([]string, numTokens)

	sim.Reserves = make([]uint256.Int, numTokens)
	sim.precisionMultipliers = make([]uint256.Int, numTokens)

	for i := range numTokens {
		tokens[i] = entityPool.Tokens[i].Address

		if err := sim.Reserves[i].SetFromDecimal(entityPool.Reserves[i]); err != nil {
			return nil, err
		}

		sim.precisionMultipliers[i].Exp(
			uint256.NewInt(10),
			uint256.NewInt(uint64(18-entityPool.Tokens[i].Decimals)),
		)
	}

	sim.Info = Info{
		Address:     strings.ToLower(entityPool.Address),
		SwapFee:     sim.Extra.SwapFee.Clone(),
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      tokens,
		BlockNumber: entityPool.BlockNumber,
	}

	sim.gas = DefaultGas

	if err := sim.LpSupply.SetFromDecimal(entityPool.Reserves[numTokens]); err != nil {
		return nil, err
	}

	sim.NumTokens = numTokens
	sim.NumTokensU256.SetUint64(uint64(numTokens))
	sim.cacheD()
	return sim, nil
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

// GetApprovalAddress returns the empty string: stable-ng pools need no approval address.
func (t *PoolSimulator) GetApprovalAddress(tokenIn, tokenOut string) string {
	return ""
}

func (t *PoolSimulator) GetReserves() []uint256.Int {
	return slices.Clone(t.Reserves)
}

// CalcLimit returns nil: stable-ng pools have no shared inventory limit.
func (t *PoolSimulator) CalcLimit() map[string]*uint256.Int {
	return nil
}

func (t *PoolSimulator) CalcAmtOut(param pool.CalcAmtOutParams) (*pool.CalcAmtOutResult, error) {
	tokenIndexFrom := t.Info.GetTokenIndex(param.TokenAmountIn.Token)
	tokenIndexTo := t.Info.GetTokenIndex(param.TokenOut)
	if tokenIndexFrom < 0 || tokenIndexTo < 0 {
		return nil, fmt.Errorf("tokenIndexFrom %v or TokenOutIndex %v is not correct", tokenIndexFrom, tokenIndexTo)
	}

	var amountOut, adminFee uint256.Int
	amount := param.TokenAmountIn.Amount
	if err := t.GetDy(tokenIndexFrom, tokenIndexTo, &amount, nil, &amountOut, &adminFee); err != nil {
		return nil, err
	}
	if amountOut.IsZero() {
		return nil, ErrZero
	}
	// UpdateState subtracts amountOut plus its admin fee from the output reserve. Reject a quote that would drive
	// that reserve negative, before any state is written (tricrypto-ng and compound pattern).
	var need uint256.Int
	need.Add(&amountOut, &adminFee)
	if t.Reserves[tokenIndexTo].Lt(&need) {
		return nil, ErrNegativeResult
	}

	return &pool.CalcAmtOutResult{
		TokenAmountOut: &pool.TokenAmt{Token: param.TokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmt{Token: param.TokenOut, Amount: adminFee},
		Gas:            t.gas.Exchange,
	}, nil
}

func (t *PoolSimulator) CalcAmtIn(param pool.CalcAmtInParams) (*pool.CalcAmtInResult, error) {
	tokenIndexFrom := t.Info.GetTokenIndex(param.TokenIn)
	tokenIndexTo := t.Info.GetTokenIndex(param.TokenAmountOut.Token)
	if tokenIndexFrom < 0 || tokenIndexTo < 0 {
		return nil, fmt.Errorf("tokenIndexFrom %v or TokenOutIndex %v is not correct", tokenIndexFrom, tokenIndexTo)
	}

	var amountIn, adminFee uint256.Int
	amountOut := param.TokenAmountOut.Amount
	if err := t.GetDx(tokenIndexFrom, tokenIndexTo, &amountOut, nil, &amountIn, &adminFee); err != nil {
		return nil, err
	}
	if amountIn.IsZero() {
		return nil, ErrZero
	}

	return &pool.CalcAmtInResult{
		TokenAmountIn: &pool.TokenAmt{Token: param.TokenIn, Amount: amountIn},
		Fee:           &pool.TokenAmt{Token: param.TokenAmountOut.Token, Amount: adminFee},
		Gas:           t.gas.Exchange,
	}, nil
}

func (t *PoolSimulator) CloneState() pool.IPoolSim {
	cloned := *t
	cloned.Info.Tokens = slices.Clone(t.Info.Tokens)
	cloned.Reserves = slices.Clone(t.Reserves)
	return &cloned
}

// UpdateState applies a swap. CalcAmtOut has already rejected any write that would drive a reserve negative.
func (t *PoolSimulator) UpdateState(params pool.UpdateStateParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var outputAmount uint256.Int
	outputAmount.Add(&output.Amount, &params.Fee.Amount)
	for i := range t.Info.Tokens {
		if t.Info.Tokens[i] == input.Token {
			t.Reserves[i].Add(&t.Reserves[i], &input.Amount)
		}
		if t.Info.Tokens[i] == output.Token {
			t.Reserves[i].Sub(&t.Reserves[i], &outputAmount)
		}
	}
	t.cacheD()
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	meta := curve.Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    false,
		BlockNumber:   t.Info.BlockNumber,
	}
	if len(t.StaticExtra.IsNativeCoins) == t.NumTokens {
		if fromId >= 0 && fromId < t.NumTokens {
			meta.TokenInIsNative = &t.StaticExtra.IsNativeCoins[fromId]
		}
		if toId >= 0 && toId < t.NumTokens {
			meta.TokenOutIsNative = &t.StaticExtra.IsNativeCoins[toId]
		}
	}
	return meta
}

func (s *PoolSimulator) SwapReceiveNativeIn(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(curve.Meta)
	return meta.TokenInIsNative != nil && *meta.TokenInIsNative
}

func (s *PoolSimulator) SwapReturnNativeOut(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(curve.Meta)
	return meta.TokenOutIsNative != nil && *meta.TokenOutIsNative
}
