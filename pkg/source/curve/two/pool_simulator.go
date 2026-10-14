package two

import (
	"fmt"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	Precisions        [nCoins]uint256.Int
	A                 uint256.Int
	Gamma             uint256.Int
	D                 uint256.Int
	FeeGamma          uint256.Int
	MidFee            uint256.Int
	OutFee            uint256.Int
	FutureAGammaTime  int64
	FutureAGamma      uint256.Int
	InitialAGammaTime int64
	InitialAGamma     uint256.Int

	LastPricesTimestamp int64
	PriceScale          uint256.Int
	PriceOracle         uint256.Int
	LastPrices          uint256.Int

	LpToken            string
	LpSupply           uint256.Int
	XcpProfit          uint256.Int
	VirtualPrice       uint256.Int
	AllowedExtraProfit uint256.Int
	AdjustmentStep     uint256.Int
	MaHalfTime         uint256.Int
	NotAdjusted        bool
	gas                Gas
}

type Gas struct {
	Exchange int64
}

var _ = pool.RegisterFactory0(curve.PoolTypeTwo, NewPoolSimulator)

// nowUnix is the block timestamp used by quotes (A/gamma ramps) and swaps; tests replace it.
var nowUnix = func() int64 { return time.Now().Unix() }

// toU256 converts an amount or balance to uint256, rejecting negative and too large values.
func toU256(b *big.Int) (z uint256.Int, err error) {
	if b.Sign() < 0 {
		return z, errNegativeAmount
	}
	if z.SetFromBig(b) {
		return z, ErrOverflow
	}
	return z, nil
}

// parseU256 parses a decimal string; an empty string is treated as 0.
func parseU256(s string) (z uint256.Int, err error) {
	if s == "" {
		return z, nil
	}
	err = z.SetFromDecimal(s)
	return z, err
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra curve.PoolTwoStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extraStr curve.PoolTwoExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extraStr); err != nil {
		return nil, err
	}

	if len(entityPool.Tokens) != nCoins {
		return nil, errNotTwoCoins
	}
	if len(entityPool.Reserves) < nCoins || len(staticExtra.PrecisionMultipliers) < nCoins {
		return nil, errMissingPoolData
	}

	sim := &PoolSimulator{
		FutureAGammaTime:    extraStr.FutureAGammaTime,
		InitialAGammaTime:   extraStr.InitialAGammaTime,
		LastPricesTimestamp: extraStr.LastPricesTimestamp,
		LpToken:             staticExtra.LpToken,
		NotAdjusted:         false,
		gas:                 DefaultGas,
	}

	tokens := make([]string, nCoins)
	reserves := make([]*big.Int, nCoins)
	for i := range nCoins {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
		if reserves[i] == nil || reserves[i].Sign() < 0 {
			return nil, fmt.Errorf("invalid reserve %q", entityPool.Reserves[i])
		}
	}
	sim.Pool = pool.Pool{
		Info: pool.PoolInfo{
			Address:     strings.ToLower(entityPool.Address),
			SwapFee:     bignumber.ZeroBI,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
		},
	}

	fields := []struct {
		dst *uint256.Int
		src string
	}{
		{&sim.Precisions[0], staticExtra.PrecisionMultipliers[0]},
		{&sim.Precisions[1], staticExtra.PrecisionMultipliers[1]},
		{&sim.A, extraStr.A}, {&sim.D, extraStr.D}, {&sim.Gamma, extraStr.Gamma},
		{&sim.FeeGamma, extraStr.FeeGamma}, {&sim.MidFee, extraStr.MidFee}, {&sim.OutFee, extraStr.OutFee},
		{&sim.FutureAGamma, extraStr.FutureAGamma}, {&sim.InitialAGamma, extraStr.InitialAGamma},
		{&sim.PriceScale, extraStr.PriceScale}, {&sim.LastPrices, extraStr.LastPrices},
		{&sim.PriceOracle, extraStr.PriceOracle},
		{&sim.LpSupply, extraStr.LpSupply}, {&sim.XcpProfit, extraStr.XcpProfit},
		{&sim.VirtualPrice, extraStr.VirtualPrice}, {&sim.AllowedExtraProfit, extraStr.AllowedExtraProfit},
		{&sim.AdjustmentStep, extraStr.AdjustmentStep}, {&sim.MaHalfTime, extraStr.MaHalfTime},
	}
	for _, f := range fields {
		v, err := parseU256(f.src)
		if err != nil {
			return nil, fmt.Errorf("invalid value %q: %w", f.src, err)
		}
		*f.dst = v
	}
	return sim, nil
}

func (t *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *t // all state is held by value, except reserves which exchange replaces (never mutates)
	cloned.Info.Reserves = slices.Clone(t.Info.Reserves)
	return &cloned
}

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	// swap from token to token
	var tokenIndexFrom = t.Info.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.Info.GetTokenIndex(tokenOut)
	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		dx, err := toU256(tokenAmountIn.Amount)
		if err != nil {
			return nil, err
		}
		amountOut, fee, err := t.getDy(tokenIndexFrom, tokenIndexTo, dx, nowUnix())
		if err != nil {
			return nil, err
		}
		if !amountOut.IsZero() {
			return &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: amountOut.ToBig(),
				},
				Fee: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: fee.ToBig(),
				},
				Gas: t.gas.Exchange,
			}, nil
		}
	}
	return &pool.CalcAmountOutResult{}, fmt.Errorf(
		"tokenIndexFrom %v or tokenIndexTo %v is not correct", tokenIndexFrom, tokenIndexTo,
	)
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	_, _, _, _ = t.Swap(input, output.Token)
}

func (t *PoolSimulator) Swap(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.TokenAmount, *pool.TokenAmount, int64, error) {
	var inputIndex = t.GetTokenIndex(tokenAmountIn.Token)
	var outputIndex = t.GetTokenIndex(tokenOut)
	amountOut, err := t.Exchange(inputIndex, outputIndex, tokenAmountIn.Amount)
	if err != nil {
		return nil, nil, 0, err
	}
	return &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		}, &pool.TokenAmount{
			Token:  tokenOut,
			Amount: bignumber.ZeroBI,
		}, t.gas.Exchange, nil
}

// GetDy returns the amount out (after fee) for swapping dx of coin i to coin j, and the fee rate (10**10 = 100%).
func (t *PoolSimulator) GetDy(i int, j int, dx *big.Int) (*big.Int, *big.Int, error) {
	u, err := toU256(dx)
	if err != nil {
		return nil, nil, err
	}
	dy, fee, err := t.getDy(i, j, u, nowUnix())
	if err != nil {
		return nil, nil, err
	}
	return dy.ToBig(), fee.ToBig(), nil
}

// Exchange applies a swap of dx of coin i to coin j to the pool state and returns the amount out.
func (t *PoolSimulator) Exchange(i int, j int, dx *big.Int) (*big.Int, error) {
	u, err := toU256(dx)
	if err != nil {
		return nil, err
	}
	dy, err := t.exchange(i, j, u, nowUnix())
	if err != nil {
		return nil, err
	}
	return dy.ToBig(), nil
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	return curve.Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    false,
		BlockNumber:   t.Info.BlockNumber,
	}
}
