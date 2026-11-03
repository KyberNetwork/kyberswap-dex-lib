package dexv2

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/logger"
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	coreEntities "github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	V3Pool *UniV3FluidV2Pool
	pool.Pool
	Gas     Gas
	tickMin int
	tickMax int

	poolAccountingFlag bool
	tokenReserves      [2]*big.Int

	extra                 Extra
	staticExtra           StaticExtra
	unsupportedDynamicFee bool
	liquidityOut          [2]uint256.Int
	liquidityDebt         [2]uint256.Int
}

var _ = pool.RegisterFactory1(DexType, NewPoolSimulator)
var _ = pool.RegisterUseSwapLimit(DexType)

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	if len(entityPool.Tokens) != 2 || len(entityPool.Reserves) != 2 {
		return nil, fmt.Errorf("expected two tokens and reserves")
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	if extra.Liquidity == nil || extra.SqrtPriceX96 == nil || extra.Tick == nil || extra.DexVariables2 == nil || extra.TokenReserves == nil || extra.Token0ExchangePricesAndConfig == nil || extra.Token1ExchangePricesAndConfig == nil || extra.SnapshotTimestamp == 0 {
		return nil, fmt.Errorf("incomplete pool state")
	}

	extraTickU256 := uniswapv3.ExtraTickU256{
		Liquidity:    new(uint256.Int),
		SqrtPriceX96: new(uint256.Int),
		TickSpacing:  uint64(staticExtra.TickSpacing),
	}
	if extraTickU256.Liquidity.SetFromBig(extra.Liquidity) || extraTickU256.SqrtPriceX96.SetFromBig(extra.SqrtPriceX96) {
		return nil, ErrOverflow
	}
	currentTick := int(extra.Tick.Int64())
	extraTickU256.Tick = &currentTick

	token0 := coreEntities.NewToken(uint(chainID), common.HexToAddress(entityPool.Tokens[0].Address),
		uint(entityPool.Tokens[0].Decimals), entityPool.Tokens[0].Symbol, "")
	token1 := coreEntities.NewToken(uint(chainID), common.HexToAddress(entityPool.Tokens[1].Address),
		uint(entityPool.Tokens[1].Decimals), entityPool.Tokens[1].Symbol, "")

	v3Ticks := make([]v3Entities.Tick, 0, len(extra.Ticks))

	// Ticks are sorted from the pool service, so we don't have to do it again here
	// Purpose: to improve the latency
	for _, t := range extra.Ticks {
		liquidityGross := new(uint256.Int)
		liquidityGross.SetFromBig(t.LiquidityGross)
		// LiquidityGross = 0 means that the tick is uninitialized
		if liquidityGross.IsZero() {
			continue
		}
		liquidityNet := new(int256.Int)
		liquidityNet.SetFromBig(t.LiquidityNet)

		v3Ticks = append(v3Ticks, v3Entities.Tick{
			Index:          t.Index,
			LiquidityGross: liquidityGross,
			LiquidityNet:   liquidityNet,
		})
	}

	if len(v3Ticks) == 0 {
		return nil, ErrV3TicksEmpty
	}

	entityPool.SwapFee = float64(staticExtra.Fee)

	ticks, err := v3Entities.NewTickListDataProvider(v3Ticks, int(extraTickU256.TickSpacing))
	if err != nil {
		return nil, err
	}

	v3Pool, err := NewUniV3FluidV2Pool(
		token0,
		token1,
		constants.FeeAmount(entityPool.SwapFee),
		extraTickU256.SqrtPriceX96,
		extraTickU256.Liquidity,
		*extraTickU256.Tick,
		ticks,
		staticExtra.TickSpacing,
		extra.DexVariables2,
	)
	if err != nil {
		return nil, err
	}

	tickMin, tickMax := MIN_TICK, MAX_TICK
	if len(v3Ticks) > 0 {
		tickMin = v3Ticks[0].Index
		tickMax = v3Ticks[len(v3Ticks)-1].Index
	}

	var poolAccounting big.Int
	poolAccounting.Set(extra.DexVariables2).
		Rsh(&poolAccounting, BITS_DEX_V2_VARIABLES2_POOL_ACCOUNTING_FLAG).
		And(&poolAccounting, bignumber.One)

	token0Reserves, token1Reserves := extractTokenReserves(extra.TokenReserves)

	swapFee := big.NewInt(int64(entityPool.SwapFee))
	tokens := make([]string, 2)
	reserves := make([]*big.Int, 2)
	if len(entityPool.Reserves) == 2 && len(entityPool.Tokens) == 2 {
		tokens[0] = entityPool.Tokens[0].Address
		reserves[0] = bignumber.NewBig10(entityPool.Reserves[0])
		tokens[1] = entityPool.Tokens[1].Address
		reserves[1] = bignumber.NewBig10(entityPool.Reserves[1])
	}

	var info = pool.PoolInfo{
		Address:     strings.ToLower(entityPool.Address),
		SwapFee:     swapFee,
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      tokens,
		Reserves:    reserves,
		BlockNumber: entityPool.BlockNumber,
	}

	var feeVersion big.Int
	feeVersion.Rsh(extra.DexVariables2, 152).And(&feeVersion, X4)
	simulator := &PoolSimulator{
		Info:    info,
		V3Pool:  v3Pool,
		Gas:     defaultGas,
		tickMin: tickMin,
		tickMax: tickMax,

		poolAccountingFlag:    poolAccounting.Cmp(bignumber.ZeroBI) == 0,
		tokenReserves:         [2]*big.Int{token0Reserves, token1Reserves},
		extra:                 extra,
		staticExtra:           staticExtra,
		unsupportedDynamicFee: feeVersion.Sign() != 0 || extra.DexVariables2.Bit(141) != 0 || staticExtra.Fee == 0xffffff,
	}

	if !staticExtra.TrackLiquidityLimits {
		return simulator, nil
	}
	for i := range 2 {
		// This implementation only quotes direct Liquidity settlements. Float-backed
		// settlements can rebalance and require additional shared state tracking.
		if extra.FloatBalance[i] == nil || extra.FloatBalance[i].Sign() != 0 || extra.SwapOutWithdraw[i] == nil || extra.SwapOutBorrow[i] == nil || extra.LiquidityDebt[i] == nil {
			return nil, ErrUnsupportedLiquidityState
		}
		budget := extra.SwapOutWithdraw[i]
		if extra.SwapOutBorrow[i].Cmp(budget) < 0 {
			budget = extra.SwapOutBorrow[i]
		}
		if simulator.liquidityOut[i].SetFromBig(budget) || simulator.liquidityDebt[i].SetFromBig(extra.LiquidityDebt[i]) {
			return nil, ErrOverflow
		}
	}
	return simulator, nil
}

func (p *PoolSimulator) GetSqrtPriceLimit(zeroForOne bool, result *v3Utils.Uint160) error {
	tickLimit := lo.Ternary(zeroForOne, p.tickMin, p.tickMax)
	if err := v3Utils.GetSqrtRatioAtTickV2(tickLimit, result); err != nil {
		return err
	}
	lo.Ternary(zeroForOne, result.AddUint64, result.SubUint64)(result, 1)
	return nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if p.staticExtra.Controller != "" && common.HexToAddress(p.staticExtra.Controller) != (common.Address{}) {
		return nil, ErrUnsupportedController
	}

	if p.unsupportedDynamicFee {
		return nil, ErrUnsupportedDynamicFee
	}

	tokenIn, tokenOut := param.TokenAmountIn.Token, param.TokenOut
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 || tokenInIndex == tokenOutIndex {
		return nil, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	amountInBI := param.TokenAmountIn.Amount
	if err := _verifyAmountLimits(amountInBI); err != nil {
		return nil, err
	}

	zeroForOne := tokenInIndex == 0
	_, dexType := parseFluidDexV2PoolAddress(p.GetAddress())
	if (dexType == D3_MODULE && (p.extra.SupplyPaused[tokenInIndex] || p.extra.WithdrawPaused[tokenOutIndex])) || (dexType != D3_MODULE && (p.extra.PaybackPaused[tokenInIndex] || p.extra.BorrowPaused[tokenOutIndex] || p.extra.SupplyPaused[tokenOutIndex])) {
		return nil, ErrUnsupportedLiquidityState
	}
	if dexType != D3_MODULE && !p.checkLiquidityLimit(param.Limit, tokenInIndex, true, amountInBI) {
		return nil, ErrUnsupportedLiquidityState
	}

	// Adjust amountIn
	c, err := _calculateVars(p.extra.DexVariables2, p.extra.Token0ExchangePricesAndConfig, p.extra.Token1ExchangePricesAndConfig, p.extra.SnapshotTimestamp)
	if err != nil {
		return nil, err
	}

	var amountInRawAdjusted, tmp big.Int
	if zeroForOne {
		amountInRawAdjusted = *amountToAdjusted(amountInBI, c.Token0NumeratorPrecision, c.Token0DenominatorPrecision,
			lo.Ternary(dexType == D3_MODULE, c.Token0SupplyExchangePrice, c.Token0BorrowExchangePrice))
	} else {
		amountInRawAdjusted = *amountToAdjusted(amountInBI, c.Token1NumeratorPrecision, c.Token1DenominatorPrecision,
			lo.Ternary(dexType == D3_MODULE, c.Token1SupplyExchangePrice, c.Token1BorrowExchangePrice))
	}

	// The deployed module rounds normalized input down before the swap.
	var roundedInput uint256.Int
	if roundedInput.SetFromBig(&amountInRawAdjusted) {
		return nil, ErrOverflow
	}
	roundNormalizedDown(&roundedInput)
	amountInRawAdjusted.Set(roundedInput.ToBig())

	if err := _verifyAdjustedAmountLimits(&amountInRawAdjusted); err != nil {
		return nil, err
	}

	var amountIn v3Utils.Int256
	if overflow := amountIn.SetFromBig(&amountInRawAdjusted); overflow {
		return nil, ErrOverflow
	}

	var priceLimit v3Utils.Uint160
	if err := p.GetSqrtPriceLimit(zeroForOne, &priceLimit); err != nil {
		return nil, fmt.Errorf("can not GetSqrtPriceLimit, err: %+v", err)
	}
	// Each swap reads the 64-bit coefficient stored on chain. The upward direction
	// increments it when an exponent is present.
	quotePool := *p.V3Pool
	var startPrice uint256.Int
	packSqrtPrice(&startPrice, p.V3Pool.SqrtRatioX96, !zeroForOne)
	quotePool.SqrtRatioX96 = &startPrice
	amountOutResult, err := quotePool.GetOutputAmountV2(&amountIn, zeroForOne, &priceLimit)
	if err != nil {
		return nil, err
	}

	err = _verifySqrtPriceX96ChangeLimits(startPrice.ToBig(), amountOutResult.SqrtRatioX96.ToBig())
	if err != nil {
		return nil, err
	}

	amountOutRawAdjusted := amountOutResult.ReturnedAmount.ToBig()

	if p.poolAccountingFlag {
		if tmp.Add(p.tokenReserves[tokenInIndex], &amountInRawAdjusted).Cmp(X128) > 0 {
			return nil, ErrTokenReservesOverflow
		}
		if tmp.Sub(p.tokenReserves[tokenOutIndex], amountOutRawAdjusted).Sign() < 0 {
			return nil, ErrTokenReservesUnderflow
		}
	}

	if amountOutResult.RemainingAmountIn != nil && amountOutResult.RemainingAmountIn.Sign() > 0 {
		return nil, ErrNextTickOutOfBounds
	}

	// The reserve counter uses the engine output before the module rounds it down.
	var reserveOutputRaw uint256.Int
	if reserveOutputRaw.SetFromBig(amountOutRawAdjusted) {
		return nil, ErrOverflow
	}
	roundedOutput := reserveOutputRaw
	roundNormalizedDown(&roundedOutput)
	amountOutRawAdjusted.Set(roundedOutput.ToBig())

	// Adjust amountOut
	var amountOut big.Int
	if err := _verifyAdjustedAmountLimits(amountOutRawAdjusted); err != nil {
		return nil, err
	}

	if zeroForOne {
		amountOut = *adjustedToAmount(amountOutRawAdjusted, c.Token1NumeratorPrecision, c.Token1DenominatorPrecision,
			lo.Ternary(dexType == D3_MODULE, c.Token1SupplyExchangePrice, c.Token1BorrowExchangePrice))
	} else {
		amountOut = *adjustedToAmount(amountOutRawAdjusted, c.Token0NumeratorPrecision, c.Token0DenominatorPrecision,
			lo.Ternary(dexType == D3_MODULE, c.Token0SupplyExchangePrice, c.Token0BorrowExchangePrice))
	}

	amountOut.Sub(&amountOut, bignumber.One)
	if err := _verifyAmountLimits(&amountOut); err != nil {
		return nil, err
	}

	var liquidityOut uint256.Int
	if liquidityOut.SetFromBig(&amountOut) {
		return nil, ErrOverflow
	}
	if dexType != D3_MODULE {
		// A D4 output borrows net output plus both fee legs. The gross normalized
		// engine output bounds all three; round up the token conversion separately.
		numerator, denominator, price := c.Token0NumeratorPrecision, c.Token0DenominatorPrecision, c.Token0BorrowExchangePrice
		if zeroForOne {
			numerator, denominator, price = c.Token1NumeratorPrecision, c.Token1DenominatorPrecision, c.Token1BorrowExchangePrice
		}
		var n, d, e, scale uint256.Int
		if n.SetFromBig(numerator) || d.SetFromBig(denominator) || e.SetFromBig(price) {
			return nil, ErrOverflow
		}
		if _, overflow := d.MulOverflow(&d, &e); overflow {
			return nil, ErrOverflow
		}
		scale.SetUint64(1000000000000)
		if _, overflow := n.MulOverflow(&n, &scale); overflow {
			return nil, ErrOverflow
		}
		if _, overflow := liquidityOut.MulDivOverflow(&amountOutResult.grossOut, &d, &n); overflow {
			return nil, ErrOverflow
		}
		if _, overflow := liquidityOut.AddOverflow(&liquidityOut, uint256.NewInt(3)); overflow {
			return nil, ErrOverflow
		}
	}
	if !p.checkLiquidityLimit(param.Limit, tokenOutIndex, false, liquidityOut.ToBig()) {
		return nil, ErrUnsupportedLiquidityState
	}
	var liquidityDebtIn uint256.Int
	if dexType != D3_MODULE {
		if liquidityDebtIn.SetFromBig(amountInBI) {
			return nil, ErrOverflow
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: &amountOut,
		},
		Fee: &pool.TokenAmount{
			Token: tokenIn,
		},
		Gas: p.Gas.BaseGas + p.Gas.CrossInitTickGas*int64(amountOutResult.CrossInitTickLoops),
		SwapInfo: SwapInfo{
			RemainingAmountIn:     amountOutResult.RemainingAmountIn,
			NextStateSqrtRatioX96: amountOutResult.SqrtRatioX96,
			NextStateTickCurrent:  amountOutResult.CurrentTick,
			nextStateLiquidity:    amountOutResult.Liquidity,
			amountInRawAdjusted:   &amountInRawAdjusted,
			amountOutRawAdjusted:  reserveOutputRaw,
			liquidityOut:          liquidityOut, liquidityDebtIn: liquidityDebtIn,
		},
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	v3Pool := *p.V3Pool
	cloned.V3Pool = &v3Pool
	cloned.tokenReserves = [2]*big.Int{
		new(big.Int).Set(p.tokenReserves[0]),
		new(big.Int).Set(p.tokenReserves[1]),
	}
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalance for FluidDexV2 pool, wrong swapInfo type")
		return
	}
	p.V3Pool.SqrtRatioX96 = new(uint256.Int)
	packSqrtPrice(p.V3Pool.SqrtRatioX96, si.NextStateSqrtRatioX96, false)
	p.V3Pool.TickCurrent = si.NextStateTickCurrent
	p.V3Pool.Liquidity = new(uint256.Int).Set(si.nextStateLiquidity)

	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenAmountOut.Token
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)

	p.tokenReserves[tokenInIndex].Add(p.tokenReserves[tokenInIndex], si.amountInRawAdjusted)
	p.tokenReserves[tokenOutIndex].Sub(p.tokenReserves[tokenOutIndex], si.amountOutRawAdjusted.ToBig())
	if !p.staticExtra.TrackLiquidityLimits {
		return
	}
	p.liquidityOut[tokenOutIndex].Sub(&p.liquidityOut[tokenOutIndex], &si.liquidityOut)
	p.liquidityDebt[tokenInIndex].Sub(&p.liquidityDebt[tokenInIndex], &si.liquidityDebtIn)
	if params.SwapLimit != nil {
		_, _, err := params.SwapLimit.UpdateLimit(p.limitKey(tokenOutIndex, false), p.limitKey(tokenInIndex, false), si.liquidityOut.ToBig(), big.NewInt(0))
		if err != nil {
			logger.Warnf("failed to update Fluid output limit: %v", err)
		}
		if !si.liquidityDebtIn.IsZero() {
			_, _, err = params.SwapLimit.UpdateLimit(p.limitKey(tokenInIndex, true), p.limitKey(tokenOutIndex, true), si.liquidityDebtIn.ToBig(), big.NewInt(0))
			if err != nil {
				logger.Warnf("failed to update Fluid debt limit: %v", err)
			}
		}
	}

}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)

	return PoolMeta{
		BlockNumber: p.Info.BlockNumber,
		Dex:         p.staticExtra.Dex,
		ZeroForOne:  tokenInIndex == 0,
		DexType:     p.staticExtra.DexType,
		Fee:         p.staticExtra.Fee,
		TickSpacing: p.staticExtra.TickSpacing,
		Controller:  p.staticExtra.Controller,

		IsNativeIn:  p.staticExtra.IsNative[tokenInIndex],
		IsNativeOut: p.staticExtra.IsNative[tokenOutIndex],
	}
}

func (s *PoolSimulator) SwapReceiveNativeIn(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(PoolMeta)
	return meta.IsNativeIn
}

func (s *PoolSimulator) SwapReturnNativeOut(tokenIn, tokenOut string, _ valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(PoolMeta)
	return meta.IsNativeOut
}

// packSqrtPrice reproduces the 64-bit coefficient and 8-bit exponent storage encoding.
func packSqrtPrice(dst, price *uint256.Int, roundUp bool) {
	shift := max(price.BitLen()-64, 0)
	dst.Rsh(price, uint(shift))
	if roundUp && shift > 0 {
		dst.AddUint64(dst, 1)
	}
	dst.Lsh(dst, uint(shift))
}

// roundNormalizedDown applies the two explicit protocol-favoring rounding steps.
// Input is at most 128 bits times two 1e12 precision factors; multiplying by 1e9 fits.
func roundNormalizedDown(amount *uint256.Int) {
	var factor, denominator uint256.Int
	factor.SetUint64(999999999)
	denominator.SetUint64(1000000000)
	amount.Mul(amount, &factor).Div(amount, &denominator)
	if !amount.IsZero() {
		amount.SubUint64(amount, 1)
	}
}

func (p *PoolSimulator) limitKey(i int, debt bool) string {
	token := p.Info.Tokens[i]
	if p.staticExtra.IsNative[i] {
		token = valueobject.NativeAddress
	}
	kind := ":out:"
	if debt {
		kind = ":debt:"
	}
	return strings.ToLower(p.staticExtra.Dex + kind + token)
}
func (p *PoolSimulator) CalculateLimit() map[string]*big.Int {
	if !p.staticExtra.TrackLiquidityLimits {
		return nil
	}
	m := make(map[string]*big.Int, 4)
	for i := range 2 {
		m[p.limitKey(i, false)] = p.liquidityOut[i].ToBig()
		m[p.limitKey(i, true)] = p.liquidityDebt[i].ToBig()
	}
	return m
}
func (p *PoolSimulator) checkLiquidityLimit(limit pool.SwapLimit, i int, debt bool, amount *big.Int) bool {
	if !p.staticExtra.TrackLiquidityLimits {
		return true
	}
	local := &p.liquidityOut[i]
	if debt {
		local = &p.liquidityDebt[i]
	}
	var x uint256.Int
	if x.SetFromBig(amount) || x.Cmp(local) > 0 {
		return false
	}
	if limit != nil {
		v := limit.GetLimit(p.limitKey(i, debt))
		if v == nil || amount.Cmp(v) > 0 {
			return false
		}
	}
	return true
}
