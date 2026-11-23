package plain

import (
	"fmt"
	"strings"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	Info base.PoolInfo

	precisionMultipliers []uint256.Int

	LpSupply uint256.Int
	gas      Gas

	numTokens     int
	numTokensU256 uint256.Int

	extra       Extra
	staticExtra StaticExtra
}

var _ pool.IPoolSim = (*PoolSimulator)(nil)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	/*
		Curve StableSwap Plain pools are the most basic Curve pools that implement StableSwap invariant
		(see the whitepaper for more details https://docs.curve.fi/references/whitepapers/stableswap/#constructing-the-stableswap-invariant )
		A*n^n * sum(x_i) + D = A*D*n^n + (D^{n+1}) / (n^n * prod(x_i))

		There are many variants of Curve StableSwap Plain pool:

		The first group are pools created and owned by Curve themselves
		the deployed pools will be put to https://github.com/curvefi/curve-contract/tree/master/contracts/pools
		(they are based on these templates but might come with some modifications https://github.com/curvefi/curve-contract/tree/master/contracts/pool-templates)

		The second group are pools created from various Factories (permissionless deployment)
			Plain2Basic.vy: standard
			Plain2Optimized.vy: same as Plain2Basic, but optimized for case when all coins have 18 decimals
			Plain2Balances.vy: support positive-rebasing & FOT tokens (call to coin to get pool's balance, instead of storing/calculating in pool itself)
				(correct balances should have been filled in by pool-tracker already)
			Plain2BasicEMA.vy: support EMA (exponential moving average) price (seems not affecting CalcAmountOut/UpdateBalance)
			Plain2ETH.vy: Uses (and optimized for) native Ether as coins[0]
			Plain2ETHEMA.vy: Plain2ETH with moving average price
			Plain2Price.vy: call to external contract to get rate multipliers (instead of hardcoding in contract)
				(should have been filled in by pool-tracker already)

			Plain3-xxx, Plain4-xxx: same as Plain2 but for 3/4 coins (Plain2 is more optimized)
	*/

	sim := &PoolSimulator{}

	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &sim.staticExtra); err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(entityPool.Extra), &sim.extra); err != nil {
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
	var reserves = make([]*uint256.Int, numTokens)

	sim.precisionMultipliers = make([]uint256.Int, numTokens)

	/*
		most Plain pools use standard rate 10^(36 - token_decimal)
		- Factory pools: https://github.com/curvefi/curve-factory/blob/master/contracts/Factory.vy#L558
		- Curve-owned pools: see explanation of ___RATES___ in https://github.com/curvefi/curve-contract/blob/master/contracts/pool-templates/README.md
		some pools (ETH/rETH and ETH/aETH) have rates changed overtime, they have been filled in by pool-tracker already
	*/
	useStandardRate := false
	if len(sim.extra.RateMultipliers) == 0 {
		sim.extra.RateMultipliers = make([]uint256.Int, numTokens)
		useStandardRate = true
	}

	for i := range numTokens {
		tokens[i] = entityPool.Tokens[i].Address

		reserve, err := uint256.FromDecimal(entityPool.Reserves[i])
		if err != nil {
			return nil, err
		}
		reserves[i] = reserve

		if useStandardRate {
			sim.extra.RateMultipliers[i].Exp(
				uint256.NewInt(10),
				uint256.NewInt(uint64(36-entityPool.Tokens[i].Decimals)),
			)
		}

		/*
			different Plain variants have slightly different way to deal with this,
			but they can all be expressed as 10^(18 - token_decimal)
			- Curve-owned pools: see explanation of ___PRECISION_MUL___ in https://github.com/curvefi/curve-contract/blob/master/contracts/pool-templates/README.md
			- Factory pools: see code, for example Plain3Basic _calc_withdraw_one_coin function:
					dy_0: uint256 = (xp[i] - new_y) * PRECISION / rates[i]  # w/o fees
					dy = (dy - 1) * PRECISION / rates[i]  # Withdraw less to account for rounding errors
				(something * 10^18 / 10^(36-decimal) --> something / 10^(18-decimal))
		*/
		sim.precisionMultipliers[i].Exp(
			uint256.NewInt(10),
			uint256.NewInt(uint64(18-entityPool.Tokens[i].Decimals)),
		)
	}

	sim.Info = base.PoolInfo{
		Address:     strings.ToLower(entityPool.Address),
		SwapFee:     sim.extra.SwapFee,
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      tokens,
		Reserves:    reserves,
		BlockNumber: entityPool.BlockNumber,
	}

	sim.gas = DefaultGas

	if err := sim.LpSupply.SetFromDecimal(entityPool.Reserves[numTokens]); err != nil {
		return nil, err
	}

	sim.numTokens = numTokens
	sim.numTokensU256.SetUint64(uint64(numTokens))
	return sim, nil
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

// GetApprovalAddress returns the empty string: plain pools need no approval address.
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

// CalcLimit returns nil: plain pools have no shared inventory limit.
func (t *PoolSimulator) CalcLimit() map[string]*uint256.Int {
	return nil
}

func (t *PoolSimulator) CloneState() pool.IPoolSim {
	cloned := *t
	cloned.Info.Reserves = cloneU256s(t.Info.Reserves)
	return &cloned
}

func (t *PoolSimulator) CalcAmtOut(param pool.CalcAmtOutParams) (*pool.CalcAmtOutResult, error) {
	tokenIndexFrom := t.Info.GetTokenIndex(param.TokenAmountIn.Token)
	tokenIndexTo := t.Info.GetTokenIndex(param.TokenOut)
	if tokenIndexFrom < 0 || tokenIndexTo < 0 {
		return nil, fmt.Errorf("tokenIndexFrom %v or TokenOutIndex %v is not correct", tokenIndexFrom, tokenIndexTo)
	}
	var amountOut, fee uint256.Int
	err := t.GetDyU256(tokenIndexFrom, tokenIndexTo, &param.TokenAmountIn.Amount, nil, &amountOut, &fee)
	if err != nil {
		return nil, err
	}
	if amountOut.IsZero() {
		return &pool.CalcAmtOutResult{}, ErrZero
	}
	return &pool.CalcAmtOutResult{
		TokenAmountOut: &pool.TokenAmt{Token: param.TokenOut, Amount: amountOut},
		Fee:            &pool.TokenAmt{Token: param.TokenOut, Amount: fee},
		Gas:            t.gas.Exchange,
	}, nil
}

func (t *PoolSimulator) CalcAmtIn(param pool.CalcAmtInParams) (*pool.CalcAmtInResult, error) {
	tokenIndexFrom := t.Info.GetTokenIndex(param.TokenIn)
	tokenIndexTo := t.Info.GetTokenIndex(param.TokenAmountOut.Token)
	if tokenIndexFrom < 0 || tokenIndexTo < 0 {
		return nil, fmt.Errorf("tokenIndexFrom %v or TokenOutIndex %v is not correct", tokenIndexFrom, tokenIndexTo)
	}
	var amountIn, fee uint256.Int
	err := t.GetDxU256(tokenIndexFrom, tokenIndexTo, &param.TokenAmountOut.Amount, nil, &amountIn, &fee)
	if err != nil {
		return nil, err
	}
	if amountIn.IsZero() {
		return &pool.CalcAmtInResult{}, ErrZero
	}
	return &pool.CalcAmtInResult{
		TokenAmountIn: &pool.TokenAmt{Token: param.TokenIn, Amount: amountIn},
		Fee:           &pool.TokenAmt{Token: param.TokenAmountOut.Token, Amount: fee},
		Gas:           t.gas.Exchange,
	}, nil
}

// UpdateState mirrors the legacy UpdateBalance: the output also loses its admin share of the swap fee.
func (t *PoolSimulator) UpdateState(params pool.UpdateStateParams) {
	var outputAmount, adminShare uint256.Int
	outputAmount.Set(&params.TokenAmountOut.Amount)
	adminShare.Mul(&outputAmount, t.Info.SwapFee)
	adminShare.Div(&adminShare, FeeDenominator)
	adminShare.Mul(&adminShare, t.extra.AdminFee)
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

func cloneU256s(xs []*uint256.Int) []*uint256.Int {
	out := make([]*uint256.Int, len(xs))
	for i, x := range xs {
		out[i] = new(uint256.Int).Set(x)
	}
	return out
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
	if len(t.staticExtra.IsNativeCoin) == t.numTokens {
		if fromId >= 0 {
			meta.TokenInIsNative = &t.staticExtra.IsNativeCoin[fromId]
		}
		if toId >= 0 {
			meta.TokenOutIsNative = &t.staticExtra.IsNativeCoin[toId]
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
