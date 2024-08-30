package integral

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/logger"
)

var (
	ErrEmptyPriceLevels      = errors.New("empty price levels")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrPoolSwapped           = errors.New("pool swapped")
	ErrOutOfLiquidity        = errors.New("out of liquidity")
	ErrOverflow              = errors.New("overflow")
)

type (
	PoolSimulator struct {
		pool.Pool
		IntegralPair
		gas Gas
	}

	MetaInfo struct {
		Timestamp int64 `json:"timestamp"`
	}

	PriceLevel struct {
		Price float64 `json:"price"`
		Level float64 `json:"level"`
	}

	Gas struct {
		Swap int64
	}

	PoolExtra struct {
		BaseToQuotePriceLevels []PriceLevel `json:"baseToQuotePriceLevels"`
		QuoteToBasePriceLevels []PriceLevel `json:"quoteToBasePriceLevels"`
		PriceTolerance         uint         `json:"priceTolerance"`
	}
)

var now = time.Now

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var pair IntegralPair
	if err := json.Unmarshal([]byte(entityPool.Extra), &pair); err != nil {
		return nil, err
	}

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i++ {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig(entityPool.Reserves[i])
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    entityPool.Address,
				ReserveUsd: entityPool.ReserveUsd,
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		IntegralPair: IntegralPair{
			PairFee: pair.PairFee,
		},
		gas: defaultGas,
	}, nil
}

// func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
// 	var (
// 		tokenAmountIn = param.TokenAmountIn
// 		tokenOut      = param.TokenOut
// 	)
// 	if tokenAmountIn.Token == tokenOut {
// 		return nil, ErrSameAddress
// 	}

// 	if isZero(tokenAmountIn.Amount) {
// 		return nil, ErrZeroAmount
// 	}

// 	var zeroForOne bool
// 	if tokenAmountIn.Token == p.GetTokens()[0] {
// 		zeroForOne = true
// 	}

// 	var (
// 		fictiveReserveIn   *big.Int = p.FictiveReserve.FictiveReserve0
// 		fictiveReserveOut  *big.Int = p.FictiveReserve.FictiveReserve1
// 		priceAverageIn     *big.Int = p.PriceAverage.PriceAverage0
// 		priceAverageOut    *big.Int = p.PriceAverage.PriceAverage1
// 		balanceIn          *big.Int = p.GetReserves()[0]
// 		balanceOut         *big.Int = p.GetReserves()[1]
// 		userTradeTimestamp          = now().Unix()
// 	)
// 	if !zeroForOne {
// 		balanceIn = p.GetReserves()[1]
// 		balanceOut = p.GetReserves()[0]
// 	}

// 	result, err := getAmountOut(
// 		GetAmountParameters{
// 			amount:          tokenAmountIn.Amount,
// 			reserveIn:       balanceIn,
// 			reserveOut:      balanceOut,
// 			priceAverageIn:  priceAverageIn,
// 			priceAverageOut: priceAverageOut,
// 			// feesLP:            p.PairFee.FeesLP,
// 			// feesPool:          p.PairFee.FeesPool,
// 			// feesBase:          p.PairFee.FeesBase,
// 		})
// 	if err != nil {
// 		return nil, err
// 	}

// 	amount0, amount1 := tokenAmountIn.Amount, result.amountOut
// 	// feeToAmount0, feeToAmount1 := new(big.Int).Set(p.FeeToAmount.Fees0), new(big.Int).Set(p.FeeToAmount.Fees1)
// 	// newPriceAverageIn, newPriceAverageOut := priceAverageIn, priceAverageOut
// 	// newFictiveReserveIn, newFictiveReserveOut := result.newFictiveReserveIn, result.newFictiveReserveOut
// 	// if zeroForOne {
// 	// 	feeToAmount0 = feeToAmount0.Add(
// 	// 		feeToAmount0,
// 	// 		new(big.Int).Div(new(big.Int).Mul(amount0, p.PairFee.FeesPool), p.PairFee.FeesBase))
// 	// } else {
// 	// 	amount0, amount1 = result.amountOut, tokenAmountIn.Amount
// 	// 	feeToAmount1 = feeToAmount1.Add(
// 	// 		feeToAmount1,
// 	// 		new(big.Int).Div(new(big.Int).Mul(amount1, p.PairFee.FeesPool), p.PairFee.FeesBase))
// 	// 	newPriceAverageIn, newPriceAverageOut = priceAverageOut, priceAverageIn
// 	// 	newFictiveReserveIn, newFictiveReserveOut = result.newFictiveReserveOut, result.newFictiveReserveIn
// 	// }

// 	if zeroForOne {
// 		return &pool.CalcAmountOutResult{
// 			TokenAmountOut: &pool.TokenAmount{
// 				Token:  p.GetTokens()[1],
// 				Amount: amount1,
// 			},
// 			Fee: &pool.TokenAmount{
// 				Token:  p.GetTokens()[1],
// 				Amount: feeToAmount1,
// 			},
// 			Gas: p.gas.Swap,
// 			SwapInfo: SwapInfo{
// 				newReserveIn:  new(big.Int).Sub(result.newReserveIn, feeToAmount0),
// 				newReserveOut: new(big.Int).Sub(result.newReserveOut, feeToAmount1),
// 			},
// 		}, nil
// 	}

// 	return &pool.CalcAmountOutResult{
// 		TokenAmountOut: &pool.TokenAmount{
// 			Token:  p.GetTokens()[0],
// 			Amount: amount0,
// 		},
// 		Fee: &pool.TokenAmount{
// 			Token:  p.GetTokens()[0],
// 			Amount: feeToAmount0,
// 		},
// 		Gas: p.gas.Swap,
// 		SwapInfo: SwapInfo{
// 			newReserveIn:  new(big.Int).Sub(result.newReserveIn, feeToAmount0),
// 			newReserveOut: new(big.Int).Sub(result.newReserveOut, feeToAmount1),
// 		},
// 	}, nil

// }

func (t *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warnf("failed to UpdateBalance for Smardex %v %v pool, wrong swapInfo type", p.Info.Address, p.Info.Exchange)
		return
	}

	p.Info.Reserves = []*big.Int{si.newReserveIn, si.newReserveOut}
}

// func (p *PoolSimulator) getSwapAmount0In(amount1Out *uint256.Int, data []byte) *big.Int {
// 	reserve0, reserve1 := getReserves()
// 	balance1After := new(uint256.Int).Sub(reserve1, amount1Out)
// 	balance0After := tradeY(balance1After, reserve0, reserve1, data)
// 	swapAmount0In := new(uint256.Int).Sub(balance0After, reserve0)
// 	swapAmount0In.Mul(swapAmount0In, precison)
// 	return CeilDiv(swapAmount0In, new(big.Int).Sub(precison, p.Info.SwapFee))
// }

// func (p *PoolSimulator) getSwapAmount1In(amount0Out *uint256.Int, data []byte) *big.Int {
// 	reserve0, reserve1 := getReserves()
// 	balance0After := new(big.Int).Sub(reserve0, amount0Out)
// 	balance1After := tradeX(balance0After, reserve0, reserve1, data)
// 	swapAmount1In := new(big.Int).Add(balance1After, big.NewInt(1))
// 	swapAmount1In.Sub(swapAmount1In, reserve1)
// 	swapAmount1In.Mul(swapAmount1In, precison)
// 	return CeilDiv(swapAmount1In, new(big.Int).Sub(precison, p.Info.SwapFee))
// }

var (
	decimalsConverterUint256 = uint256.NewInt(0)
	decimalsConverterInt256  = int256.NewInt(0)
)

func tradeY(yAfter, xBefore, yBefore *big.Int, data []byte) *big.Int {
	// yAfterInt := new(big.Int).SetBytes(yAfter.Bytes())
	// xBeforeInt := new(big.Int).SetBytes(xBefore.Bytes())
	// yBeforeInt := new(big.Int).SetBytes(yBefore.Bytes())
	// averagePriceInt := decodePriceInfo(data)

	// xTradedInt := MulInt256(SubInt256(yAfterInt, yBeforeInt), decimalsConverterInt256)

	// xAfterInt := SubInt256(xBeforeInt, NegFloorDiv(xTradedInt, averagePriceInt))

	// return new(big.Int).SetBytes(xAfterInt.ToBig().Bytes())

	return new(big.Int).SetUint64(0)
}

func tradeX(xAfter, xBefore, yBefore *big.Int, data []byte) *uint256.Int {
	// xAfterInt := new(big.Int).SetBytes(xAfter.Bytes())
	// xBeforeInt := new(big.Int).SetBytes(xBefore.Bytes())
	// yBeforeInt := new(big.Int).SetBytes(yBefore.Bytes())
	// averagePriceInt := decodePriceInfo(data)

	// xTradedInt := MulInt256(SubInt256(xAfterInt, xBeforeInt), decimalsConverterInt256)

	// yAfterInt := SubInt256(yBeforeInt, NegFloorDiv(xTradedInt, averagePriceInt))

	// if yAfterInt.Cmp(new(int256.Int)) < 0 {
	// 	panic(ErrT027)
	// }

	// return new(uint256.Int).SetBytes(yAfterInt.ToBig().Bytes())

	return new(uint256.Int).SetUint64(0)
}

func decodePriceInfo(data []byte) *big.Int {
	return new(big.Int).SetBytes(data)
}

func (p *PoolSimulator) Swap(amount0Out *big.Int, amount1Out *big.Int, to string, data []byte) (*pool.CalcAmountOutResult, error) {

	// Step 2: Validate 'to' address
	if to == "0x0000000000000000000000000000000000000000" {
		return nil, fmt.Errorf("TP02: Invalid 'to' address")
	}

	// Step 3: Validate output amounts
	if !(amount0Out.Cmp(big.NewInt(0)) > 0 && amount1Out.Cmp(big.NewInt(0)) == 0) && !(amount1Out.Cmp(big.NewInt(0)) > 0 && amount0Out.Cmp(big.NewInt(0)) == 0) {
		return nil, fmt.Errorf("TP31: Invalid output amounts")
	}

	// Step 4: Get reserves
	reserves := p.GetReserves()

	reserve0 := reserves[0]
	reserve1 := reserves[1]

	// Step 5: Check reserve limits
	if amount0Out.Cmp(reserve0) >= 0 || amount1Out.Cmp(reserve1) >= 0 {
		return nil, fmt.Errorf("TP07: Amount exceeds reserves")
	}

	// Step 6: Perform the swap logic
	tokens := p.GetTokens()

	token0 := tokens[0]
	token1 := tokens[1]

	if to == token0 || to == token1 {
		return nil, fmt.Errorf("TP2D: Invalid recipient address")
	}

	// if amount0Out.Cmp(big.NewInt(0)) > 0 {
	// 	// Transfer token0
	// 	err := contract.SafeTransfer(auth, token0, to, amount0Out)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("Failed to transfer token0: %v", err)
	// 	}
	// }

	// if amount1Out.Cmp(big.NewInt(0)) > 0 {
	// 	// Transfer token1
	// 	err := contract.SafeTransfer(auth, token1, to, amount1Out)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("Failed to transfer token1: %v", err)
	// 	}
	// }

	// Step 7: Get balances after swap
	balance0, balance1, err := GetBalances(token0, token1)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch balances: %v", err)
	}

	if amount0Out.Cmp(big.NewInt(0)) > 0 {

		if balance1.Cmp(reserve1) <= 0 {
			return nil, fmt.Errorf("TP08: Balance1 is too low")
		}

		amount1In := new(big.Int).Sub(balance1, reserve1)

		swapFee := p.GetInfo().SwapFee

		fee1 := new(big.Int).Div(new(big.Int).Sub(amount1In, swapFee), precison)

		balance1After := new(big.Int).Sub(balance1, fee1)

		balance0After := tradeY(balance1After, reserve0, reserve1, data)

		if balance0.Cmp(balance0After) < 0 {
			return nil, fmt.Errorf("TP2E: Invalid balance after swap")
		}

		fee0 := new(big.Int).Sub(balance0, balance0After)

	} else {
		// Trading token0 for token1
		if balance0.Cmp(reserve0) <= 0 {
			return nil, fmt.Errorf("TP08: Balance0 is too low")
		}

		amount0In := new(big.Int).Sub(balance0, reserve0)

		fee0 := calculateFee(amount0In)
		balance0After := new(big.Int).Sub(balance0, fee0)

		// Call tradeX function from Oracle
		balance1After := tradeX(balance0After, reserve0, reserve1, data)

		if balance1.Cmp(balance1After) < 0 {
			return nil, fmt.Errorf("TP2E: Invalid balance after swap")
		}

		fee1 := new(big.Int).Sub(balance1, balance1After)

	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  p.GetTokens()[0],
			Amount: amount0,
		},
		Fee: &pool.TokenAmount{
			Token:  p.GetTokens()[0],
			Amount: feeToAmount0,
		},
		Gas: p.gas.Swap,
		SwapInfo: SwapInfo{
			newReserveIn:  balance0After,
			newReserveOut: balance1After,
		},
	}, nil
}
