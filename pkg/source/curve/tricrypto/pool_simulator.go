package tricrypto

import (
	"fmt"
	"math/big"
	"slices"
	"strings"
	"sync/atomic"
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

	// Unpacked price_scale / price_oracle / last_prices (the contract packs them into one uint256 each).
	LastPricesTimestamp int64
	PriceScale          [nCoins - 1]uint256.Int
	PriceOracle         [nCoins - 1]uint256.Int
	LastPrices          [nCoins - 1]uint256.Int

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

var _ = pool.RegisterFactory0(curve.PoolTypeTricrypto, NewPoolSimulator)

// fixedNow pins the block timestamp used by UpdateBalance in tests; 0 means wall-clock time.
var fixedNow atomic.Int64

func nowUnix() int64 {
	if now := fixedNow.Load(); now != 0 {
		return now
	}
	return time.Now().Unix()
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
	var staticExtra curve.PoolTricryptoStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extraStr curve.PoolTricryptoExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extraStr); err != nil {
		return nil, err
	}

	if len(entityPool.Tokens) != nCoins || len(entityPool.Reserves) < nCoins ||
		len(staticExtra.PrecisionMultipliers) < nCoins || len(extraStr.PriceScale) < nCoins-1 ||
		len(extraStr.LastPrices) < nCoins-1 || len(extraStr.PriceOracle) < nCoins-1 {
		return nil, errNotThreeCoins
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

	type field struct {
		dst *uint256.Int
		src string
	}
	fields := []field{
		{&sim.A, extraStr.A}, {&sim.D, extraStr.D}, {&sim.Gamma, extraStr.Gamma},
		{&sim.FeeGamma, extraStr.FeeGamma}, {&sim.MidFee, extraStr.MidFee}, {&sim.OutFee, extraStr.OutFee},
		{&sim.FutureAGamma, extraStr.FutureAGamma}, {&sim.InitialAGamma, extraStr.InitialAGamma},
		{&sim.LpSupply, extraStr.LpSupply}, {&sim.XcpProfit, extraStr.XcpProfit},
		{&sim.VirtualPrice, extraStr.VirtualPrice}, {&sim.AllowedExtraProfit, extraStr.AllowedExtraProfit},
		{&sim.AdjustmentStep, extraStr.AdjustmentStep}, {&sim.MaHalfTime, extraStr.MaHalfTime},
	}
	for i := range nCoins {
		fields = append(fields, field{&sim.Precisions[i], staticExtra.PrecisionMultipliers[i]})
	}
	for k := range nCoins - 1 {
		fields = append(fields, field{&sim.PriceScale[k], extraStr.PriceScale[k]},
			field{&sim.LastPrices[k], extraStr.LastPrices[k]}, field{&sim.PriceOracle[k], extraStr.PriceOracle[k]})
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
		var dx uint256.Int
		if tokenAmountIn.Amount.Sign() < 0 || dx.SetFromBig(tokenAmountIn.Amount) {
			return nil, ErrOverflow
		}
		amountOut, fee, err := t.getDy(tokenIndexFrom, tokenIndexTo, dx)
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
	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenIndexFrom %v or tokenIndexTo %v is not correct", tokenIndexFrom, tokenIndexTo)
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var dx uint256.Int
	if input.Amount.Sign() < 0 || dx.SetFromBig(input.Amount) {
		return
	}
	_, _ = t.exchange(t.GetTokenIndex(input.Token), t.GetTokenIndex(output.Token), dx, nowUnix())
}

// GetDy returns the amount out (after fee) and the fee for swapping dx of coin i to coin j.
func (t *PoolSimulator) GetDy(i int, j int, dx *big.Int) (*big.Int, *big.Int, error) {
	var u uint256.Int
	if dx.Sign() < 0 || u.SetFromBig(dx) {
		return nil, nil, ErrOverflow
	}
	dy, fee, err := t.getDy(i, j, u)
	if err != nil {
		return nil, nil, err
	}
	return dy.ToBig(), fee.ToBig(), nil
}

// Exchange applies a swap of dx of coin i to coin j to the pool state and returns the amount out.
func (t *PoolSimulator) Exchange(i int, j int, dx *big.Int) (*big.Int, error) {
	var u uint256.Int
	if dx.Sign() < 0 || u.SetFromBig(dx) {
		return nil, ErrOverflow
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
