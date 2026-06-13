package stable

import (
	"errors"
	"math/big"
	"slices"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	ErrInvalidValue    = errors.New("invalid value")
	ErrPriceDiffToken0 = errors.New("price difference for token0 exceeds threshold")
	ErrPriceDiffToken1 = errors.New("price difference for token1 exceeds threshold")
)

type PoolSimulator struct {
	pool.Pool
	baseSim            *base.PoolSimulator
	decimals           []uint8
	isNativeCoins      []bool
	oraclePrices       [2]*uint256.Int
	priceDiffThreshold [2]*uint256.Int
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	curveBaseSimulator, err := base.NewPoolSimulator(entityPool)
	if err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	decimals := make([]uint8, len(entityPool.Tokens))
	for i, token := range entityPool.Tokens {
		decimals[i] = token.Decimals
	}

	return &PoolSimulator{
		Pool:               curveBaseSimulator.Pool,
		baseSim:            curveBaseSimulator,
		decimals:           decimals,
		isNativeCoins:      staticExtra.IsNativeCoins,
		oraclePrices:       [2]*uint256.Int{uint256.MustFromBig(extra.OraclePrices[0]), uint256.MustFromBig(extra.OraclePrices[1])},
		priceDiffThreshold: [2]*uint256.Int{uint256.MustFromBig(extra.PriceDiffThreshold[0]), uint256.MustFromBig(extra.PriceDiffThreshold[1])},
	}, nil
}

func (t *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *t
	clonedBase := *t.baseSim
	clonedBase.Info.Reserves = slices.Clone(t.baseSim.Info.Reserves)
	cloned.baseSim = &clonedBase
	cloned.Info.Reserves = clonedBase.Info.Reserves
	return &cloned
}

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	res, err := t.baseSim.CalcAmountOut(param)
	if err != nil {
		return nil, err
	}

	updatedBalances := t.prepareUpdateBalance(param, res)

	if err := t.checkPriceDiff(updatedBalances); err != nil {
		return nil, err
	}

	return res, nil
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	t.baseSim.UpdateBalance(params)
}

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	var fromId = t.baseSim.GetTokenIndex(tokenIn)
	var toId = t.baseSim.GetTokenIndex(tokenOut)

	meta := Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    false,
	}
	if len(t.isNativeCoins) == len(t.Info.Tokens) {
		meta.TokenInIsNative = t.isNativeCoins[fromId]
		meta.TokenOutIsNative = t.isNativeCoins[toId]
	}

	return meta
}

func (s *PoolSimulator) SwapReceiveNativeIn(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(Meta)
	return meta.TokenInIsNative
}

func (s *PoolSimulator) SwapReturnNativeOut(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(Meta)
	return meta.TokenOutIsNative
}

// prepareUpdateBalance mirrors the on-chain balance update after exchange,
// returning the post-swap stored balances that checkPriceDiff operates on.
func (t *PoolSimulator) prepareUpdateBalance(
	param pool.CalcAmountOutParams,
	res *pool.CalcAmountOutResult,
) []*big.Int {
	outAmt := uint256.MustFromBig(res.TokenAmountOut.Amount)
	fd := big256.TenPow(10)

	// admin_fee = outAmt * swapFee / fd * adminFee / fd
	out := big256.MulDivDown(new(uint256.Int), outAmt, uint256.MustFromBig(t.Info.SwapFee), fd)
	big256.MulDivDown(out, out, uint256.MustFromBig(t.baseSim.AdminFee), fd)
	out.Add(outAmt, out) // balance decrease = user output + admin fee portion

	updatedBalances := make([]*big.Int, len(t.Info.Reserves))
	for i, tok := range t.Info.Tokens {
		if tok == param.TokenAmountIn.Token {
			r := uint256.MustFromBig(t.Info.Reserves[i])
			updatedBalances[i] = r.Add(r, uint256.MustFromBig(param.TokenAmountIn.Amount)).ToBig()
		}
		if tok == param.TokenOut {
			r := uint256.MustFromBig(t.Info.Reserves[i])
			updatedBalances[i] = r.Sub(r, out).ToBig()
		}
	}

	return updatedBalances
}

func (t *PoolSimulator) checkPriceDiff(updatedBalances []*big.Int) error {
	if len(updatedBalances) != 2 ||
		len(t.decimals) != 2 ||
		len(t.baseSim.Multipliers) != 2 {
		return ErrInvalidValue
	}

	// dx = $100 worth of each token in raw units: 100 * 10^(18+decimals) / oraclePrice
	hundred := uint256.NewInt(100)
	dx0 := big256.MulDivDown(new(uint256.Int), hundred, big256.TenPow(int(t.decimals[0])+18), t.oraclePrices[0])
	dx1 := big256.MulDivDown(hundred, hundred, big256.TenPow(int(t.decimals[1])+18), t.oraclePrices[1])

	dy1, err := t.getDyWithoutFee(updatedBalances, 0, 1, dx0.ToBig())
	if err != nil {
		return err
	}
	dy0, err := t.getDyWithoutFee(updatedBalances, 1, 0, dx1.ToBig())
	if err != nil {
		return err
	}

	mul0 := uint256.MustFromBig(t.baseSim.Multipliers[0])
	mul1 := uint256.MustFromBig(t.baseSim.Multipliers[1])

	// price0 = (dx1 * mul1 * oracle1) / (dy0 * mul0)  — implied price of token0 in oracle units
	dx1.Mul(dx1, mul1)                                                  // dx1 = dx1_xp
	dy0u := uint256.MustFromBig(dy0)
	dy0u.Mul(dy0u, mul0)                                                // dy0u = dy0_xp
	price0 := big256.MulDivDown(dx1, dx1, t.oraclePrices[1], dy0u)    // reuse dx1 as price0

	// price1 = (dx0 * mul0 * oracle0) / (dy1 * mul1)  — implied price of token1 in oracle units
	dx0.Mul(dx0, mul0)                                                  // dx0 = dx0_xp
	dy1u := uint256.MustFromBig(dy1)
	dy1u.Mul(dy1u, mul1)                                                // dy1u = dy1_xp
	price1 := big256.MulDivDown(dx0, dx0, t.oraclePrices[0], dy1u)    // reuse dx0 as price1

	// check: |price - oracle| * 1e18 <= oracle * threshold
	var diff, rhs uint256.Int
	if price0.Gt(t.oraclePrices[0]) {
		diff.Sub(price0, t.oraclePrices[0])
	} else {
		diff.Sub(t.oraclePrices[0], price0)
	}
	diff.Mul(&diff, big256.BONE)
	rhs.Mul(t.oraclePrices[0], t.priceDiffThreshold[0])
	if diff.Gt(&rhs) {
		return ErrPriceDiffToken0
	}

	if price1.Gt(t.oraclePrices[1]) {
		diff.Sub(price1, t.oraclePrices[1])
	} else {
		diff.Sub(t.oraclePrices[1], price1)
	}
	diff.Mul(&diff, big256.BONE)
	rhs.Mul(t.oraclePrices[1], t.priceDiffThreshold[1])
	if diff.Gt(&rhs) {
		return ErrPriceDiffToken1
	}

	return nil
}
