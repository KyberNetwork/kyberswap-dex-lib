package core

import (
	"context"
	"errors"
	"math/big"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	poolPkg "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/pool"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"

	"github.com/oleiade/lane"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type NodeInfo struct {
	Amount poolPkg.TokenAmount
	Hops   uint32
}

type TraceInfo struct {
	PoolID string
	Token  string
}

func BestPathExactIn(
	ctx context.Context,
	pools map[string]poolPkg.IPool,
	tokens map[string]entity.Token,
	prices map[string]float64,
	tokenAmountIn poolPkg.TokenAmount,
	tokenOut string,
	options BestRouteOption,
	usedPaths []Path,
) (*Path, error) {
	span, _ := tracer.StartSpanFromContext(ctx, "BestPathExactIn")
	defer span.Finish()

	var err error
	if len(pools) == 0 {
		return nil, errors.New("pools is empty")
	}
	var tokenOutInfo = tokens[tokenOut]
	// check usedPaths with zero gas
	var bestPath *Path = nil
	if options.Gas.GasFeeInclude {
		var amountOut = poolPkg.TokenAmount{
			Token:  tokenOut,
			Amount: constant.Zero,
		}
		for _, path := range usedPaths {
			amount, _ := path.TrySwap(tokenAmountIn)
			if amountOut.CompareTo(&amount) < 0 {
				amountOut = amount
				bestPath = &path
			}
		}
		if bestPath != nil {
			bestPath, err = NewPath(
				bestPath.Pools,
				bestPath.Tokens,
				tokenAmountIn,
				tokenOut,
				prices[tokenOut],
				tokenOutInfo.Decimals,
				GasOption{
					GasFeeInclude: false,
					Price:         options.Gas.Price,
					TokenPrice:    0,
				},
			)
			if err == nil {
				bestPath.TotalGas = 0
			} else {
				bestPath = nil
			}
		}
	}

	var queue = lane.NewQueue()
	var dp = make(map[string]NodeInfo)
	var inQueue = make(map[string]bool)
	var trace = make(map[string]TraceInfo)

	hopsToTokenOut := minHopsToToken(
		pools,
		tokens,
		tokenOut)

	queue.Enqueue(tokenAmountIn.Token)

	dp[tokenAmountIn.Token] = NodeInfo{
		Amount: tokenAmountIn,
		Hops:   0,
	}

	checkSwap := func(p poolPkg.IPool, fromInfo NodeInfo, tokenTo string) {
		var tokenToInfo = tokens[tokenTo]
		var toInfo, ok = dp[tokenTo]
		if !ok {
			toInfo = NodeInfo{
				Amount: poolPkg.TokenAmount{
					Token:     tokenTo,
					Amount:    big.NewInt(0),
					AmountUsd: 0,
				},
				Hops: 0,
			}
		}
		calcAmountOutResult, err := p.CalcAmountOut(
			fromInfo.Amount,
			tokenTo)
		if err != nil {
			// fmt.Printf("PoolAddress: %v, PoolExchange: %v, CalcAmountOut[ tokenAmountIn: %v, tokenOut: %s, error: %v]\n",
			// 	(*p).GetAddress(), (*p).GetExchange(), fromInfo.Amount, tokenTo, err)
			return
		}
		tokenAmountOut, totalGas := calcAmountOutResult.TokenAmountOut, calcAmountOutResult.Gas

		if tokenAmountOut == nil {
			// fmt.Printf("PoolAddress: %v, PoolExchange: %v, CalcAmountOut[ tokenAmountIn: %v, tokenOut: %s, error: %s]\n",
			// 	(*p).GetAddress(), (*p).GetExchange(), fromInfo.Amount, tokenTo, "tokenAmountOut is nil")
			return
		}
		var amountCmp = tokenAmountOut.Amount.Cmp(toInfo.Amount.Amount)
		var outputPrice float64 = 0
		if price, hasPrice := prices[tokenTo]; hasPrice {
			outputPrice = price
		}
		tokenAmountOut.AmountUsd = utils.CalcTokenAmountUsd(tokenAmountOut.Amount, tokenToInfo.Decimals, outputPrice)
		var gasUsd = utils.CalcGasUsd(options.Gas.Price, totalGas, options.Gas.TokenPrice)
		// totalOutUsd = amountOut * outputPrice / 10^decimal
		tokenAmountOut.AmountUsd -= gasUsd

		if options.Gas.GasFeeInclude && outputPrice > 0 && toInfo.Amount.Amount.Cmp(constant.Zero) > 0 {
			if tokenAmountOut.AmountUsd > toInfo.Amount.AmountUsd {
				amountCmp = 1
			} else {
				amountCmp = -1
			}
		}
		var hopsToOut = options.MaxHops
		if hops, exist := hopsToTokenOut[tokenTo]; exist {
			hopsToOut = hops
		}
		if fromInfo.Hops+1+hopsToOut <= options.MaxHops && (amountCmp > 0 || amountCmp == 0 && toInfo.Hops > fromInfo.Hops+1) {
			toInfo = NodeInfo{
				Amount: *tokenAmountOut,
				Hops:   fromInfo.Hops + 1,
			}
			dp[tokenTo] = toInfo
			trace[tokenTo] = TraceInfo{
				PoolID: p.GetAddress(),
				Token:  fromInfo.Amount.Token,
			}
			if !inQueue[tokenTo] {
				inQueue[tokenTo] = true
				queue.Enqueue(tokenTo)
			}
		}
	}

	for !queue.Empty() {
		var tokenItf = queue.Dequeue()
		var token = tokenItf.(string)
		inQueue[token] = false
		var fromInfo, ok = dp[token]
		if !ok {
			continue
		}
		if token == tokenOut {
			// trace back
			var u = tokenOut
			var pathPools = make([]poolPkg.IPool, 0)
			var pathTokens = []entity.Token{tokens[tokenOut]}
			for u != tokenAmountIn.Token {
				var tr = trace[u]
				pathTokens = append(pathTokens, tokens[tr.Token])
				pathPools = append(pathPools, pools[tr.PoolID])
				u = tr.Token
			}
			for i, j := 0, len(pathTokens)-1; i < j; i, j = i+1, j-1 {
				pathTokens[i], pathTokens[j] = pathTokens[j], pathTokens[i]
			}
			for i, j := 0, len(pathPools)-1; i < j; i, j = i+1, j-1 {
				pathPools[i], pathPools[j] = pathPools[j], pathPools[i]
			}

			var path, err = NewPath(
				pathPools,
				pathTokens,
				tokenAmountIn,
				tokenOut,
				prices[tokenOut],
				tokenOutInfo.Decimals,
				options.Gas,
			)
			if err == nil {
				if path.CompareTo(bestPath, options.Gas.GasFeeInclude) < 0 {
					bestPath = path
				}
			}
			continue
		}
		if fromInfo.Hops == options.MaxHops {
			continue
		}
		for _, pool := range pools {
			var toTokens = pool.CanSwapTo(token)
			for _, toToken := range toTokens {
				if _, isWhitelist := tokens[toToken]; isWhitelist {
					checkSwap(pool, fromInfo, toToken)
				}
			}
		}
	}
	return bestPath, nil
}

func BestRouteExactIn(
	ctx context.Context,
	pools []poolPkg.IPool,
	originalPools []poolPkg.IPool,
	tokens map[string]entity.Token,
	prices map[string]float64,
	tokenIn string,
	tokenOut string,
	amountIn *big.Int,
	options BestRouteOption,
) (*Route, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "BestRouteExactIn")
	defer span.Finish()

	var tokenInInfo, okTokenInInfo = tokens[tokenIn]
	var amountInUsd float64 = 0
	if okTokenInInfo {
		amountInUsd = utils.CalcTokenAmountUsd(amountIn, tokenInInfo.Decimals, prices[tokenIn])
	}
	var tokenAmountIn = poolPkg.TokenAmount{
		Token:     tokenIn,
		Amount:    amountIn,
		AmountUsd: amountInUsd,
	}
	var emptyRoute = NewRoute(
		make([]poolPkg.IPool, 0),
		make([]poolPkg.IPool, 0),
		tokenAmountIn,
		tokenOut,
		make([]Path, 0),
	)
	var poolInfo = make(map[string]poolPkg.IPool)
	for i := range pools {
		poolInfo[pools[i].GetAddress()] = pools[i]
	}

	var route = NewRoute(
		pools,
		originalPools,
		poolPkg.TokenAmount{
			Token:  tokenIn,
			Amount: constant.Zero,
		},
		tokenOut,
		make([]Path, 0),
	)
	if options.SaveGas {
		if path, _ := BestPathExactIn(
			ctx,
			poolInfo,
			tokens,
			prices,
			tokenAmountIn,
			tokenOut,
			options,
			route.Paths,
		); path != nil {
			route.AddPath(path)
		}
		return route, nil
	}
	const partAmount = 20
	var frag = new(big.Int).Div(amountIn, big.NewInt(partAmount))
	var remainingAmount = new(big.Int).Set(amountIn)
	var totalAmountOut = big.NewInt(0)
	var fragAmountIn = big.NewInt(0)

	for i := 0; i < partAmount; i++ {
		if i == partAmount-1 {
			frag = remainingAmount
		}
		fragAmountIn = new(big.Int).Add(fragAmountIn, frag)
		var fragUsd float64 = 0
		var remainingUsd float64 = 0
		if okTokenInInfo {
			fragUsd = utils.CalcTokenAmountUsd(fragAmountIn, tokenInInfo.Decimals, prices[tokenIn])
			remainingUsd = utils.CalcTokenAmountUsd(new(big.Int).Sub(remainingAmount, fragAmountIn), tokenInInfo.Decimals, prices[tokenIn])
		}
		if remainingUsd < options.MinPartUsd {
			fragAmountIn = remainingAmount
			fragUsd += remainingUsd
		}
		if fragAmountIn.Cmp(constant.Zero) > 0 && (fragAmountIn.Cmp(remainingAmount) == 0 || fragUsd >= options.MinPartUsd) {
			var path, err = BestPathExactIn(
				ctx,
				poolInfo,
				tokens,
				prices,
				poolPkg.TokenAmount{
					Token:     tokenIn,
					Amount:    fragAmountIn,
					AmountUsd: fragUsd,
				},
				tokenOut,
				options,
				route.Paths,
			)
			if err != nil || path == nil {
				return emptyRoute, nil
			}
			route.AddPath(path)
			totalAmountOut = new(big.Int).Add(totalAmountOut, path.Output.Amount)
			remainingAmount = new(big.Int).Sub(remainingAmount, fragAmountIn)
			fragAmountIn = big.NewInt(0)
			if remainingAmount.Cmp(constant.Zero) == 0 {
				break
			}
		}
	}

	return route, nil
}
