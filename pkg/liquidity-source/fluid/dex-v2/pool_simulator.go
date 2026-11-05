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
	tokenReserves      [2]uint256.Int
	vars               [2]tokenVars

	extra                 Extra
	staticExtra           StaticExtra
	unsupportedDynamicFee bool
	liquidityLimits       [2]LiquidityLimits
	liquidityKeys         [2]string
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
	var reserves256 [2]uint256.Int
	if reserves256[0].SetFromBig(token0Reserves) || reserves256[1].SetFromBig(token1Reserves) {
		return nil, ErrOverflow
	}
	// Conversion factors only depend on the snapshot, so derive them once with the
	// verified big.Int formulas; quotes then use fixed-width integers.
	calc, err := _calculateVars(extra.DexVariables2, extra.Token0ExchangePricesAndConfig, extra.Token1ExchangePricesAndConfig, extra.SnapshotTimestamp)
	if err != nil {
		return nil, err
	}
	var vars [2]tokenVars
	var overflow0, overflow1 bool
	vars[0], overflow0 = newTokenVars(calc.Token0NumeratorPrecision, calc.Token0DenominatorPrecision, calc.Token0SupplyExchangePrice, calc.Token0BorrowExchangePrice)
	vars[1], overflow1 = newTokenVars(calc.Token1NumeratorPrecision, calc.Token1DenominatorPrecision, calc.Token1SupplyExchangePrice, calc.Token1BorrowExchangePrice)
	if overflow0 || overflow1 {
		return nil, ErrOverflow
	}

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
		tokenReserves:         reserves256,
		vars:                  vars,
		extra:                 extra,
		staticExtra:           staticExtra,
		unsupportedDynamicFee: feeVersion.Uint64() > 1 || extra.DexVariables2.Bit(141) != 0 || (feeVersion.Sign() == 0 && staticExtra.Fee == 0xffffff),
	}

	if feeVersion.Uint64() == 1 {
		simulator.V3Pool.dynamicVersion = true
		simulator.V3Pool.snapshotTimestamp = extra.SnapshotTimestamp
		if simulator.V3Pool.dynamicPacked.SetFromBig(extra.DexVariables2) {
			return nil, ErrOverflow
		}
	}
	if !staticExtra.TrackLiquidityLimits {
		return simulator, nil
	}
	for i := range 2 {
		token := simulator.Info.Tokens[i]
		if staticExtra.IsNative[i] {
			token = valueobject.NativeAddress
		}
		simulator.liquidityKeys[i] = strings.ToLower(staticExtra.Dex + ":" + token + ":")
		// Aggregate-only snapshots cannot prove the individual settlement legs.
		// Wait for a tracker refresh rather than infer an execution mode.
		if extra.LiquidityLimits[i] == nil {
			return nil, ErrUnsupportedLiquidityState
		}
		simulator.liquidityLimits[i] = *extra.LiquidityLimits[i]
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

	var amountIn uint256.Int
	if amt := param.TokenAmountIn.Amount; amt.Sign() < 0 || amountIn.SetFromBig(amt) {
		return nil, ErrAmountOutOfLimits
	}
	if err := _verifyAmountLimits(&amountIn); err != nil {
		return nil, err
	}

	zeroForOne := tokenInIndex == 0
	_, dexType := parseFluidDexV2PoolAddress(p.GetAddress())
	d3 := dexType == D3_MODULE
	inVars, outVars := &p.vars[tokenInIndex], &p.vars[tokenOutIndex]

	// Adjust amountIn; the deployed module rounds normalized input down before the swap.
	var amountInRawAdjusted uint256.Int
	inVars.toAdjusted(&amountInRawAdjusted, &amountIn, inVars.price(d3))
	roundNormalizedDown(&amountInRawAdjusted)
	if err := _verifyAdjustedAmountLimits(&amountInRawAdjusted); err != nil {
		return nil, err
	}
	// Bounded to 86 bits above, so the signed view is the same number.
	signedIn := v3Utils.Int256(amountInRawAdjusted)

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
	amountOutResult, err := quotePool.GetOutputAmountV2(&signedIn, zeroForOne, &priceLimit)
	if err != nil {
		return nil, err
	}

	if err = _verifySqrtPriceX96ChangeLimits(&amountOutResult.sqrtPriceReference, amountOutResult.SqrtRatioX96); err != nil {
		return nil, err
	}

	if amountOutResult.ReturnedAmount.IsNegative() {
		return nil, ErrOverflow
	}
	// The reserve counter uses the engine output before the module rounds it down.
	reserveOutputRaw := uint256.Int(*amountOutResult.ReturnedAmount)

	if p.poolAccountingFlag {
		var total uint256.Int
		if _, overflow := total.AddOverflow(&p.tokenReserves[tokenInIndex], &amountInRawAdjusted); overflow || total.BitLen() > 128 {
			return nil, ErrTokenReservesOverflow
		}
		if p.tokenReserves[tokenOutIndex].Lt(&reserveOutputRaw) {
			return nil, ErrTokenReservesUnderflow
		}
	}

	if amountOutResult.RemainingAmountIn != nil && amountOutResult.RemainingAmountIn.Sign() > 0 {
		return nil, ErrNextTickOutOfBounds
	}

	roundedOutput := reserveOutputRaw
	roundNormalizedDown(&roundedOutput)
	if err = _verifyAdjustedAmountLimits(&roundedOutput); err != nil {
		return nil, err
	}

	// Adjust amountOut; the module subtracts one wei in the protocol's favor.
	var amountOut uint256.Int
	outVars.toAmount(&amountOut, &roundedOutput, outVars.price(d3))
	if !amountOut.IsZero() {
		amountOut.SubUint64(&amountOut, 1)
	}
	if err = _verifyAmountLimits(&amountOut); err != nil {
		return nil, err
	}

	// D3 keeps fees inside the supply position; only D4 pays them out of borrowed output.
	var fees uint256.Int
	if !d3 {
		var protocolFee, lpFee uint256.Int
		// Adjusted output is bounded to 86 bits, so each conversion product fits.
		outVars.toAmount(&protocolFee, &amountOutResult.protocolFeeOut, &outVars.borrowPrice)
		outVars.toAmount(&lpFee, &amountOutResult.lpFeeOut, &outVars.borrowPrice)
		if _, overflow := fees.AddOverflow(&protocolFee, &lpFee); overflow {
			return nil, ErrOverflow
		}
	}
	usage, err := p.quoteLiquidity(param.Limit, tokenInIndex, tokenOutIndex, dexType, &amountIn, &amountOut, &fees)
	if err != nil {
		return nil, err
	}
	gas := p.Gas.BaseGas + p.Gas.CrossInitTickGas*int64(amountOutResult.CrossInitTickLoops)
	if p.V3Pool.dynamicVersion {
		gas += 10000
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token: tokenIn,
		},
		Gas: gas,
		SwapInfo: SwapInfo{
			RemainingAmountIn:     amountOutResult.RemainingAmountIn,
			NextStateSqrtRatioX96: amountOutResult.SqrtRatioX96,
			NextStateTickCurrent:  amountOutResult.CurrentTick,
			nextStateLiquidity:    amountOutResult.Liquidity,
			amountInRawAdjusted:   amountInRawAdjusted,
			amountOutRawAdjusted:  reserveOutputRaw,
			liquidityUsage:        usage,
			nextDynamicPacked:     amountOutResult.dynamicPacked,
		},
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	v3Pool := *p.V3Pool
	cloned.V3Pool = &v3Pool
	// tokenReserves, vars and liquidityLimits are value arrays, so the copy is already deep.
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
	if p.V3Pool.dynamicVersion {
		p.V3Pool.dynamicPacked.Set(&si.nextDynamicPacked)
	}

	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenAmountOut.Token
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)

	p.tokenReserves[tokenInIndex].Add(&p.tokenReserves[tokenInIndex], &si.amountInRawAdjusted)
	p.tokenReserves[tokenOutIndex].Sub(&p.tokenReserves[tokenOutIndex], &si.amountOutRawAdjusted)
	p.applyLiquidity(si.liquidityUsage, params.SwapLimit)

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
