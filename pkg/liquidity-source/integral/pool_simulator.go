package integral

import (
	"errors"
	"math/big"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

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

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokens := p.GetTokens()
	if len(tokens) != 2 {
		return nil, errors.New("")
	}

	data := []byte{}

	switch param.TokenOut {
	case tokens[0]:
		amount1In := ToUint256(param.TokenAmountIn.Amount)
		amount0Out, err := p.getSwapAmount0Out(amount1In, data)
		if err != nil {
			return nil, err
		}
		return p.swap(amount0Out, uZero, data)

	case tokens[1]:
		amount0In := ToUint256(param.TokenAmountIn.Amount)
		amount1Out, err := p.getSwapAmount1Out(amount0In, data)
		if err != nil {
			return nil, err
		}
		return p.swap(uZero, amount1Out, data)

	default:
		return nil, errors.New("")
	}
}

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

func (p *PoolSimulator) swap(amount0Out *uint256.Int, amount1Out *uint256.Int, data []byte) (*pool.CalcAmountOutResult, error) {
	// Step 3: Validate output amounts
	if !(amount0Out.Cmp(uZero) > 0 && amount1Out.Cmp(uZero) == 0) && !(amount1Out.Cmp(uZero) > 0 && amount0Out.Cmp(uZero) == 0) {
		return nil, ErrTP31
	}

	// Step 4: Get reserves
	reserves := p.GetReserves()

	reserve0 := ToUint256(reserves[0])
	reserve1 := ToUint256(reserves[1])

	// Step 5: Check reserve limits
	if amount0Out.Cmp(reserve0) >= 0 || amount1Out.Cmp(reserve1) >= 0 {
		return nil, ErrTP07
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
	balance0, balance1 := reserve0, reserve1

	swapFee := ToUint256(p.GetInfo().SwapFee)

	var balance0After, balance1After *uint256.Int

	if amount0Out.Cmp(uZero) > 0 {

		if balance1.Cmp(reserve1) <= 0 {
			return nil, ErrTP08
		}

		amount1In := SubUint256(balance1, reserve1)

		fee1 := DivUint256(SubUint256(amount1In, swapFee), precison)

		balance1After := SubUint256(balance1, fee1)

		balance0After, err := tradeY(balance1After, reserve0, reserve1, data)
		if err != nil {
			return nil, err
		}

		if balance0.Cmp(balance0After) < 0 {
			return nil, ErrTP2E
		}

		// fee0 := SubUint256(balance0, balance0After)

	} else {
		// Trading token0 for token1
		if balance0.Cmp(reserve0) <= 0 {
			return nil, ErrTP08
		}

		amount0In := SubUint256(balance0, reserve0)

		fee0 := DivUint256(MulUint256(amount0In, swapFee), precison)
		balance0After := SubUint256(balance0, fee0)

		// Call tradeX function from Oracle
		var err error
		balance1After, err = tradeX(balance0After, reserve0, reserve1, data)
		if err != nil {
			return nil, err
		}

		if balance1.Cmp(balance1After) < 0 {
			return nil, ErrTP2E
		}

		// fee1 := SubUint256(balance1, balance1After)

	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token: p.GetTokens()[0],
			// Amount: amount0,
		},
		Fee: &pool.TokenAmount{
			Token: p.GetTokens()[0],
			// Amount: feeToAmount0,
		},
		Gas: p.gas.Swap,
		SwapInfo: SwapInfo{
			newReserveIn:  ToInt256(balance0After),
			newReserveOut: ToInt256(balance1After),
		},
	}, nil
}

func (p *PoolSimulator) getSwapAmount0Out(amount1In *uint256.Int, data []byte) (*uint256.Int, error) {
	reserves := p.GetReserves()

	reserve0 := ToUint256(reserves[0])
	reserve1 := ToUint256(reserves[1])

	swapFee := ToUint256(p.GetInfo().SwapFee)

	fee := DivUint256(MulUint256(amount1In, swapFee), precison)

	balanceAfter0, err := tradeY(
		SubUint256(AddUint256(reserve1, amount1In), fee),
		reserve0,
		reserve1,
		data,
	)
	if err != nil {
		return nil, err
	}

	return SubUint256(reserve0, balanceAfter0), nil
}

func (p *PoolSimulator) getSwapAmount1Out(amount0In *uint256.Int, data []byte) (*uint256.Int, error) {
	reserves := p.GetReserves()

	reserve0 := ToUint256(reserves[0])
	reserve1 := ToUint256(reserves[1])

	swapFee := ToUint256(p.GetInfo().SwapFee)

	fee := DivUint256(MulUint256(amount0In, swapFee), precison)

	balanceAfter1, err := tradeY(
		SubUint256(AddUint256(reserve0, amount0In), fee),
		reserve0,
		reserve1,
		data,
	)
	if err != nil {
		return nil, err
	}

	return SubUint256(reserve1, balanceAfter1), nil
}

func (p *PoolSimulator) getSwapAmount0In(amount1Out *uint256.Int, data []byte) (*uint256.Int, error) {
	reserves := p.GetReserves()

	reserve0 := ToUint256(reserves[0])
	reserve1 := ToUint256(reserves[1])

	swapFee := ToUint256(p.GetInfo().SwapFee)

	balance1After := SubUint256(reserve1, amount1Out)
	balance0After, err := tradeY(balance1After, reserve0, reserve1, data)
	if err != nil {
		return nil, err
	}

	return CeilDivUint256(MulUint256(SubUint256(balance0After, reserve0), precison), SubUint256(precison, swapFee)), nil
}

func (p *PoolSimulator) getSwapAmount1In(amount0Out *uint256.Int, data []byte) (*uint256.Int, error) {
	reserves := p.GetReserves()

	reserve0 := ToUint256(reserves[0])
	reserve1 := ToUint256(reserves[1])

	swapFee := ToUint256(p.GetInfo().SwapFee)

	balance0After := SubUint256(reserve0, amount0Out)
	balance1After, err := tradeY(balance0After, reserve0, reserve1, data)
	if err != nil {
		return nil, err
	}

	return CeilDivUint256(MulUint256(SubUint256(AddUint256(balance1After, uint256.NewInt(1)), reserve0), precison), SubUint256(precison, swapFee)), nil
}
