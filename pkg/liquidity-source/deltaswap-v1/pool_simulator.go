package deltaswapv1

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/blockchain-toolkit/number"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v2"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type (
	PoolSimulator struct {
		poolpkg.Pool
		feePrecision *uint256.Int

		tradeLiquidityEMA        *uint256.Int
		liquidityEMA             *uint256.Int
		lastLiquidityBlockNumber uint64
		lastTradeLiquiditySum    *uint256.Int
		lastTradeBlockNumber     uint64
		dsFee                    *uint256.Int
		dsFeeThreshold           *uint256.Int

		gas uniswapv2.Gas
	}

	SwapInfo struct {
		Fee            uint32       `json:"fee"`
		FeePrecision   uint32       `json:"feePrecision"`
		TradeLiquidity *uint256.Int `json:"tradeLiquidity"`
	}
)

var (
	ErrZeroTradeLiquidity = errors.New("DeltaSwap: ZERO_TRADE_LIQUIDITY")
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: poolpkg.Pool{Info: poolpkg.PoolInfo{
			Address:     entityPool.Address,
			ReserveUsd:  entityPool.ReserveUsd,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, index int) string { return item.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return utils.NewBig(item) }),
			BlockNumber: entityPool.BlockNumber,
		}},
		dsFee:                    uint256.NewInt(uint64(extra.DsFee)),
		dsFeeThreshold:           uint256.NewInt(uint64(extra.DsFeeThreshold)),
		liquidityEMA:             extra.LiquidityEMA,
		lastLiquidityBlockNumber: extra.LastLiquidityBlockNumber,
		tradeLiquidityEMA:        extra.TradeLiquidityEMA,
		lastTradeLiquiditySum:    extra.LastTradeLiquiditySum,
		lastTradeBlockNumber:     extra.LastTradeBlockNumber,

		feePrecision: Number_1000,
		gas:          defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
	)
	indexIn, indexOut := s.GetTokenIndex(tokenAmountIn.Token), s.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, uniswapv2.ErrInvalidToken
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, uniswapv2.ErrInvalidAmountIn
	}

	if amountIn.Cmp(number.Zero) <= 0 {
		return nil, uniswapv2.ErrInsufficientInputAmount
	}

	reserveIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, uniswapv2.ErrInvalidReserve
	}

	reserveOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, uniswapv2.ErrInvalidReserve
	}

	if reserveIn.Cmp(number.Zero) <= 0 || reserveOut.Cmp(number.Zero) <= 0 {
		return nil, uniswapv2.ErrInsufficientLiquidity
	}

	tradeLiquidity, fee, err := s.calcPairTradingFee(amountIn, reserveIn, reserveOut)
	if err != nil {
		return nil, err
	}

	amountOut := s.getAmountOut(amountIn, reserveIn, reserveOut, fee)
	if amountOut.Cmp(reserveOut) > 0 {
		return nil, uniswapv2.ErrInsufficientLiquidity
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexOut], Amount: amountOut.ToBig()},
		Fee:            &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas:            s.gas.Swap,
		SwapInfo: SwapInfo{
			Fee:            uint32(fee.Uint64()),
			FeePrecision:   uint32(s.feePrecision.Uint64()),
			TradeLiquidity: tradeLiquidity,
		},
	}, nil
}

func (s *PoolSimulator) CalcAmountIn(param poolpkg.CalcAmountInParams) (*poolpkg.CalcAmountInResult, error) {
	var (
		tokenAmountOut = param.TokenAmountOut
		tokenIn        = param.TokenIn
	)
	indexIn, indexOut := s.GetTokenIndex(tokenIn), s.GetTokenIndex(tokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return nil, uniswapv2.ErrInvalidToken
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, uniswapv2.ErrInvalidAmountOut
	}

	if amountOut.Cmp(number.Zero) <= 0 {
		return nil, uniswapv2.ErrInsufficientOutputAmount
	}

	reserveIn, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexIn])
	if overflow {
		return nil, uniswapv2.ErrInvalidReserve
	}

	reserveOut, overflow := uint256.FromBig(s.Pool.Info.Reserves[indexOut])
	if overflow {
		return nil, uniswapv2.ErrInvalidReserve
	}

	if reserveIn.Cmp(number.Zero) <= 0 || reserveOut.Cmp(number.Zero) <= 0 {
		return nil, uniswapv2.ErrInsufficientLiquidity
	}

	var fee, amountIn, tradeLiquidity uint256.Int
	fee.Set(number.Number_3)
	for {
		newAmountIn, err := s.getAmountIn(amountOut, reserveIn, reserveOut, &fee)
		if err != nil {
			return nil, err
		}

		newTradeLiquidity, newFee, err := s.calcPairTradingFee(newAmountIn, reserveIn, reserveOut)
		if err != nil {
			return nil, err
		}

		tradeLiquidity.Set(newTradeLiquidity)
		if fee.Cmp(newFee) == 0 {
			amountIn.Set(newAmountIn)
			break
		}
		fee.Set(newFee)
	}

	if amountIn.Cmp(reserveIn) > 0 {
		return nil, uniswapv2.ErrInsufficientLiquidity
	}

	balanceIn := new(uint256.Int).Add(reserveIn, &amountIn)
	balanceOut := new(uint256.Int).Sub(reserveOut, amountOut)

	balanceInAdjusted := new(uint256.Int).Sub(
		new(uint256.Int).Mul(balanceIn, s.feePrecision),
		new(uint256.Int).Mul(&amountIn, &fee),
	)
	balanceOutAdjusted := new(uint256.Int).Mul(balanceOut, s.feePrecision)

	kBefore := new(uint256.Int).Mul(new(uint256.Int).Mul(reserveIn, reserveOut), new(uint256.Int).Mul(s.feePrecision, s.feePrecision))
	kAfter := new(uint256.Int).Mul(balanceInAdjusted, balanceOutAdjusted)

	if kAfter.Cmp(kBefore) < 0 {
		return nil, uniswapv2.ErrInvalidK
	}

	return &poolpkg.CalcAmountInResult{
		TokenAmountIn: &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: amountIn.ToBig()},
		Fee:           &poolpkg.TokenAmount{Token: s.Pool.Info.Tokens[indexIn], Amount: integer.Zero()},
		Gas:           s.gas.Swap,
		SwapInfo: SwapInfo{
			Fee:            uint32(fee.Uint64()),
			FeePrecision:   uint32(s.feePrecision.Uint64()),
			TradeLiquidity: &tradeLiquidity,
		},
	}, nil
}

func (s *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	s.Pool.Info.Reserves[indexIn] = new(big.Int).Add(s.Pool.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	s.Pool.Info.Reserves[indexOut] = new(big.Int).Sub(s.Pool.Info.Reserves[indexOut], params.TokenAmountOut.Amount)

	s._updateLiquidityTradedEMA(params.SwapInfo.(SwapInfo).TradeLiquidity)

	blockNumber := s.Pool.Info.BlockNumber
	if blockNumber != s.lastLiquidityBlockNumber {
		temp := new(big.Int).Mul(s.Pool.Info.Reserves[indexIn], s.Pool.Info.Reserves[indexOut])
		s.liquidityEMA = calcEMA(uint256.MustFromBig(temp), s.liquidityEMA, uint256.NewInt(max(blockNumber-s.lastLiquidityBlockNumber, 10)))
		s.lastLiquidityBlockNumber = blockNumber
	}
}

func (s *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (s *PoolSimulator) getAmountOut(amountIn, reserveIn, reserveOut, fee *uint256.Int) *uint256.Int {
	amountInWithFee := new(uint256.Int).Mul(amountIn, new(uint256.Int).Sub(s.feePrecision, fee))
	numerator := new(uint256.Int).Mul(amountInWithFee, reserveOut)
	denominator := new(uint256.Int).Add(new(uint256.Int).Mul(reserveIn, s.feePrecision), amountInWithFee)

	return new(uint256.Int).Div(numerator, denominator)
}

func (s *PoolSimulator) getAmountIn(amountOut, reserveIn, reserveOut, fee *uint256.Int) (amountIn *uint256.Int, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	numerator := uniswapv2.SafeMul(
		uniswapv2.SafeMul(reserveIn, amountOut),
		s.feePrecision,
	)
	denominator := uniswapv2.SafeMul(
		uniswapv2.SafeSub(reserveOut, amountOut),
		uniswapv2.SafeSub(s.feePrecision, fee),
	)

	return uniswapv2.SafeAdd(new(uint256.Int).Div(numerator, denominator), number.Number_1), nil
}

func (s *PoolSimulator) calcPairTradingFee(amountIn, reserveIn, reserveOut *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	tradeLiquidity := calcTradeLiquidity(amountIn, number.Zero, reserveIn, reserveOut)
	if !tradeLiquidity.Gt(number.Zero) {
		return nil, nil, ErrZeroTradeLiquidity
	}

	return tradeLiquidity, s.estimateTradingFee(tradeLiquidity), nil
}

func (s *PoolSimulator) estimateTradingFee(tradeLiquidity *uint256.Int) *uint256.Int {
	_tradeLiquidityEMA, _, _ := s._getTradeLiquidityEMA(tradeLiquidity, s.Pool.Info.BlockNumber)
	return s.calcTradingFee(tradeLiquidity, _tradeLiquidityEMA, s.liquidityEMA)
}

func (s *PoolSimulator) calcTradingFee(tradeLiquidity, lastLiquidityTradedEMA, lastLiquidityEMA *uint256.Int) *uint256.Int {
	var threshold uint256.Int
	threshold.Set(lastLiquidityEMA).Mul(&threshold, s.dsFeeThreshold).Div(&threshold, Number_1000)
	// if trade >= threshold, charge fee
	if (Max(tradeLiquidity, lastLiquidityTradedEMA)).Gt(&threshold) {
		return s.dsFee
	}

	return number.Zero
}

func (s *PoolSimulator) _getTradeLiquidityEMA(
	tradeLiquidity *uint256.Int, blockNumber uint64,
) (*uint256.Int, *uint256.Int, *uint256.Int) {
	blockDiff := blockNumber - s.lastTradeBlockNumber
	tradeLiquiditySum := s._getLastTradeLiquiditySum(tradeLiquidity, blockDiff)
	lastTradeLiquidityEMA := s._getLastTradeLiquidityEMA(blockDiff)

	var _tradeLiquidityEMA uint256.Int
	if tradeLiquidity.Gt(number.Zero) {
		_tradeLiquidityEMA.Set(calcEMA(tradeLiquiditySum, lastTradeLiquidityEMA, Number_20))
	} else {
		_tradeLiquidityEMA.Set(lastTradeLiquidityEMA)
	}

	return &_tradeLiquidityEMA, lastTradeLiquidityEMA, tradeLiquiditySum
}

func (s *PoolSimulator) _getLastTradeLiquiditySum(tradeLiquidity *uint256.Int, blockDiff uint64) *uint256.Int {
	if blockDiff > 0 {
		return tradeLiquidity
	}
	return new(uint256.Int).Add(s.lastTradeLiquiditySum, tradeLiquidity)
}

func (s *PoolSimulator) _getLastTradeLiquidityEMA(blockDiff uint64) *uint256.Int {
	if blockDiff > 50 {
		return number.Zero
	}
	return s.tradeLiquidityEMA
}

func (s *PoolSimulator) _updateLiquidityTradedEMA(tradeLiquidity *uint256.Int) {
	_tradeLiquidityEMA, _, tradeLiquiditySum := s._getTradeLiquidityEMA(tradeLiquidity, s.Pool.Info.BlockNumber)
	s.lastTradeLiquiditySum = tradeLiquiditySum
	s.tradeLiquidityEMA = _tradeLiquidityEMA

	if s.lastTradeBlockNumber != s.Pool.Info.BlockNumber {
		s.lastTradeBlockNumber = s.Pool.Info.BlockNumber
	}
}

func calcSingleSideLiquidity(amount, reserve0, reserve1 *uint256.Int) *uint256.Int {
	var amount0, amount1 uint256.Int
	amount0.Set(amount).Div(&amount0, number.Number_2)
	amount1.Set(&amount0).Mul(&amount1, reserve1).Div(&amount1, reserve0)
	// return Sqrt(new(uint256.Int).Mul(&amount0, &amount1))
	return amount0.Mul(&amount0, &amount1).Sqrt(&amount0)
}

func calcTradeLiquidity(amount0, amount1, reserve0, reserve1 *uint256.Int) *uint256.Int {
	return Max(
		calcSingleSideLiquidity(amount0, reserve0, reserve1),
		calcSingleSideLiquidity(amount1, reserve1, reserve0),
	)
}

func calcEMA(last, emaLast, emaWeight *uint256.Int) *uint256.Int {
	if emaLast.IsZero() {
		return last
	}

	var weight, result, temp uint256.Int
	weight.SetUint64(min(100, emaWeight.Uint64()))

	result.Mul(last, &weight).Div(&result, Number_100)
	temp.Sub(Number_100, &weight).Mul(emaLast, &temp).Div(&temp, Number_100)
	result.Add(&result, &temp)

	return &result
}
