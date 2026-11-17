package smardex

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var now = time.Now

type PoolSimulator struct {
	pool.Pool
	SmardexPair
	gas Gas
}

var _ = pool.RegisterFactory0(DexTypeSmardex, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var pair SmardexPair
	if err := json.Unmarshal([]byte(entityPool.Extra), &pair); err != nil {
		return nil, err
	}

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)
	for i := range numTokens {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig(entityPool.Reserves[i])
	}

	return &PoolSimulator{
		Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
		},
		PairFee:        pair.PairFee,
		FictiveReserve: pair.FictiveReserve,
		PriceAverage:   pair.PriceAverage,
		FeeToAmount:    pair.FeeToAmount,
		FeeOff:         pair.FeeOff,
		gas:            DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
		amountIn      = uint256.MustFromBig(tokenAmountIn.Amount)
	)
	if tokenAmountIn.Token == tokenOut {
		return nil, ErrSameAddress
	}

	if amountIn.IsZero() {
		return nil, ErrZeroAmount
	}

	var zeroForOne bool
	if tokenAmountIn.Token == p.GetTokens()[0] {
		zeroForOne = true
	}

	var (
		fictiveReserveIn   = p.FictiveReserve.FictiveReserve0
		fictiveReserveOut  = p.FictiveReserve.FictiveReserve1
		priceAverageIn     = p.PriceAverage.PriceAverage0
		priceAverageOut    = p.PriceAverage.PriceAverage1
		balanceIn          = p.GetReserves()[0]
		balanceOut         = p.GetReserves()[1]
		userTradeTimestamp = uint256.NewInt(uint64(now().Unix()))
	)
	if !zeroForOne {
		fictiveReserveIn = p.FictiveReserve.FictiveReserve1
		fictiveReserveOut = p.FictiveReserve.FictiveReserve0
		priceAverageIn = p.PriceAverage.PriceAverage1
		priceAverageOut = p.PriceAverage.PriceAverage0
		balanceIn = p.GetReserves()[1]
		balanceOut = p.GetReserves()[0]
	}

	var err error
	// compute new price average
	priceAverageIn, priceAverageOut, err = getUpdatedPriceAverage(fictiveReserveIn, fictiveReserveOut,
		p.PriceAverage.PriceAverageLastTimestamp, priceAverageIn, priceAverageOut, userTradeTimestamp)
	if err != nil {
		return nil, err
	}

	result, err := getAmountOut(
		GetAmountParameters{
			amount:            amountIn,
			reserveIn:         uint256.MustFromBig(balanceIn),
			reserveOut:        uint256.MustFromBig(balanceOut),
			fictiveReserveIn:  fictiveReserveIn,
			fictiveReserveOut: fictiveReserveOut,
			priceAverageIn:    priceAverageIn,
			priceAverageOut:   priceAverageOut,
			feesLP:            p.PairFee.FeesLP,
			feesPool:          p.PairFee.FeesPool,
			feesBase:          p.PairFee.FeesBase,
		})
	if err != nil {
		return nil, err
	}

	feeToAmount0, feeToAmount1 := new(uint256.Int).Set(p.FeeToAmount.Fees0), new(uint256.Int).Set(p.FeeToAmount.Fees1)
	// The pool's reserves are its balances less feeToAmount (getReserves), so the input side moves by the input less
	// the one pool fee the pair takes on it, whatever the legs' rounding in getAmountOut. The output side moves by
	// the output. feeToAmount is already left out. Without a feeTo the pair takes no pool fee: it stays in the
	// reserves.
	poolFee := new(uint256.Int)
	if !p.FeeOff {
		poolFee.Div(new(uint256.Int).Mul(amountIn, p.PairFee.FeesPool), p.PairFee.FeesBase)
	}
	newReserveIn := new(uint256.Int).Add(uint256.MustFromBig(balanceIn), amountIn)
	newReserveIn.Sub(newReserveIn, poolFee)
	newPriceAverageIn, newPriceAverageOut := priceAverageIn, priceAverageOut
	newFictiveReserveIn, newFictiveReserveOut := result.newFictiveReserveIn, result.newFictiveReserveOut
	newReserve0, newReserve1 := newReserveIn, result.newReserveOut
	outToken, feeOut := p.GetTokens()[1], feeToAmount1
	if zeroForOne {
		feeToAmount0.Add(feeToAmount0, poolFee)
	} else {
		outToken, feeOut = p.GetTokens()[0], feeToAmount0
		feeToAmount1.Add(feeToAmount1, poolFee)
		newPriceAverageIn, newPriceAverageOut = priceAverageOut, priceAverageIn
		newFictiveReserveIn, newFictiveReserveOut = result.newFictiveReserveOut, result.newFictiveReserveIn
		newReserve0, newReserve1 = result.newReserveOut, newReserveIn
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  outToken,
			Amount: result.amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  outToken,
			Amount: feeOut.ToBig(),
		},
		Gas: p.gas.Swap,
		SwapInfo: SwapInfo{
			newReserve0:               newReserve0,
			newReserve1:               newReserve1,
			newFictiveReserve0:        newFictiveReserveIn,
			newFictiveReserve1:        newFictiveReserveOut,
			newPriceAverage0:          newPriceAverageIn,
			newPriceAverage1:          newPriceAverageOut,
			priceAverageLastTimestamp: userTradeTimestamp,
			feeToAmount0:              feeToAmount0,
			feeToAmount1:              feeToAmount1,
		},
	}, nil
}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return pool.MetaInfo{BlockNumber: p.Info.BlockNumber}
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warnf("failed to UpdateBalance for Smardex %v %v pool, wrong swapInfo type",
			p.Info.Address, p.Info.Exchange)
		return
	}
	p.Info.Reserves = []*big.Int{si.newReserve0.ToBig(), si.newReserve1.ToBig()}
	p.FictiveReserve = FictiveReserve{
		si.newFictiveReserve0,
		si.newFictiveReserve1,
	}
	p.PriceAverage = PriceAverage{
		si.newPriceAverage0,
		si.newPriceAverage1,
		si.priceAverageLastTimestamp,
	}
	p.FeeToAmount = FeeToAmount{
		si.feeToAmount0,
		si.feeToAmount1,
	}
}
