package meta

import (
	"fmt"
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// ICurveBasePool is the interface for curve base pool inside a meta pool
// It can be:
// 1. base/plain pool
// 2. plain oracle pool
// 3. lending pool
// 4. or even meta pool
// At the moment, our code can only support base/plain pool and plain oracle pool
type ICurveBasePool interface {
	pool.IPoolSimulator
	GetInfo() pool.PoolInfo

	// GetVirtualPrice returns both vPrice and D
	GetVirtualPrice() (vPrice *big.Int, D *big.Int, err error)
	// GetDy recalculates `dCached` if it is nil
	GetDy(i int, j int, dx *big.Int, dCached *big.Int) (*big.Int, *big.Int, error)
	CalculateTokenAmount(amounts []*big.Int, deposit bool) (*big.Int, error)
	CalculateWithdrawOneCoin(tokenAmount *big.Int, i int) (*big.Int, *big.Int, error)
	AddLiquidity(amounts []*big.Int) (*big.Int, error)
	RemoveLiquidityOneCoin(tokenAmount *big.Int, i int) (*big.Int, error)
}

type PoolSimulator struct {
	pool.Pool
	basePool       ICurveBasePool
	Reserves       []*uint256.Int
	SwapFee        *uint256.Int
	RateMultiplier *uint256.Int
	InitialA       *uint256.Int
	FutureA        *uint256.Int
	InitialATime   int64
	FutureATime    int64
	AdminFee       *uint256.Int
	LpToken        string
	LpSupply       *uint256.Int
	APrecision     *uint256.Int
	gas            curve.Gas
}

var _ = pool.RegisterFactoryMeta(curve.PoolTypeMeta, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, basePoolMap map[string]pool.IPoolSimulator) (*PoolSimulator, error) {
	var staticExtra curve.PoolMetaStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	basePool, ok := basePoolMap[staticExtra.BasePool].(ICurveBasePool)
	if !ok {
		return nil, ErrInvalidBasePool
	}

	var extraStr curve.PoolMetaExtra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extraStr); err != nil {
		return nil, err
	}

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)
	uReserves := make([]*uint256.Int, numTokens)
	multipliers := make([]*big.Int, numTokens)
	rates := make([]*big.Int, numTokens)
	for i := range numTokens {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
		uReserves[i] = big256.New(entityPool.Reserves[i])
		multipliers[i] = bignumber.NewBig10(staticExtra.PrecisionMultipliers[i])
		rates[i] = bignumber.NewBig10(staticExtra.Rates[i])
	}

	aPrecision := big256.U1
	if len(staticExtra.APrecision) > 0 {
		aPrecision = big256.New(staticExtra.APrecision)
	}

	rateMultiplier := big256.New(staticExtra.RateMultiplier)
	// Handle a specific case for the RAI Curve-Meta pool,
	// since this pool uses a different contract version, leading the "rates"
	// is calculated using contract data.
	if entityPool.Address == curve.RAIMetaPool {
		rateMultiplier.Div(extraStr.SnappedRedemptionPrice, big256.TenPow(9))
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				SwapFee:  bignumber.NewBig10(extraStr.SwapFee),
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Reserves: reserves,
			},
		},
		basePool:       basePool,
		Reserves:       uReserves,
		SwapFee:        big256.New(extraStr.SwapFee),
		RateMultiplier: rateMultiplier,
		InitialA:       big256.New(extraStr.InitialA),
		FutureA:        big256.New(extraStr.FutureA),
		InitialATime:   extraStr.InitialATime,
		FutureATime:    extraStr.FutureATime,
		AdminFee:       big256.New(extraStr.AdminFee),
		LpToken:        staticExtra.LpToken,
		LpSupply:       big256.New(entityPool.Reserves[numTokens]),
		APrecision:     aPrecision,
		gas:            DefaultGas,
	}, nil
}

func (t *PoolSimulator) GetBasePools() []pool.IPoolSimulator {
	return []pool.IPoolSimulator{t.basePool}
}

func (t *PoolSimulator) SetBasePool(basePool pool.IPoolSimulator) {
	if curveBasePool, ok := basePool.(ICurveBasePool); ok {
		t.basePool = curveBasePool
	}
}

func (t *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	idxIn, idxOut := t.Info.GetTokenIndex(tokenAmountIn.Token), t.Info.GetTokenIndex(tokenOut)

	if idxIn == len(t.Info.Tokens)-1 && idxOut < 0 || idxOut == len(t.Info.Tokens)-1 && idxIn < 0 {
		return nil, ErrTokenToUnderLyingNotSupported
	}

	amtIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrAmountInOverflow
	}
	if idxIn >= 0 && idxOut >= 0 {
		amountOut, fee, err := t.GetDy(
			idxIn,
			idxOut,
			amtIn,
		)
		if err != nil {
			return nil, err
		}
		if amountOut.Sign() > 0 {
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
	// check exchange_underlying
	var baseInputIndex = t.basePool.GetTokenIndex(tokenAmountIn.Token)
	var baseOutputIndex = t.basePool.GetTokenIndex(tokenOut)
	var maxCoin = len(t.Info.Tokens) - 1
	if idxIn < 0 && baseInputIndex >= 0 {
		idxIn = maxCoin + baseInputIndex
	}
	if idxOut < 0 && baseOutputIndex >= 0 {
		idxOut = maxCoin + baseOutputIndex
	}
	if idxIn >= 0 && idxOut >= 0 {
		// get_dy_underlying
		amountOut, fee, err := t.GetDyUnderlying(
			idxIn,
			idxOut,
			amtIn)
		if err != nil {
			return nil, err
		}
		if amountOut.Sign() > 0 {
			return &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: amountOut.ToBig(),
				},
				Fee: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: fee.ToBig(),
				},
				Gas: t.gas.ExchangeUnderlying,
			}, nil

		}
	}
	return &pool.CalcAmountOutResult{
		Gas: t.gas.ExchangeUnderlying,
	}, fmt.Errorf("idxIn %v or idxOut %v is not correct", idxIn, idxOut)
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	inputAmount, _ := uint256.FromBig(input.Amount)
	idxIn, idxOut := t.GetTokenIndex(input.Token), t.GetTokenIndex(output.Token)
	if idxIn >= 0 && idxOut >= 0 {
		// exchange
		_, _ = t.Exchange(idxIn, idxOut, inputAmount)
		return
	}
	// check exchange_underlying
	var baseInputIndex = t.basePool.GetTokenIndex(input.Token)
	var baseOutputIndex = t.basePool.GetTokenIndex(output.Token)
	var maxCoin = len(t.Info.Tokens) - 1
	if idxIn < 0 && baseInputIndex >= 0 {
		idxIn = maxCoin + baseInputIndex
	}
	if idxOut < 0 && baseOutputIndex >= 0 {
		idxOut = maxCoin + baseOutputIndex
	}
	if idxIn >= 0 && idxOut >= 0 {
		// exchange_underlying
		_, _ = t.ExchangeUnderlying(idxIn, idxOut, inputAmount)
	}
}

func (t *PoolSimulator) CanSwapTo(address string) []string {
	var ret = make([]string, 0)
	var tokenIndex = t.GetTokenIndex(address)
	if tokenIndex < 0 {
		// check from underlying
		tokenIndex = t.basePool.GetTokenIndex(address)
		if tokenIndex >= 0 {
			// base token can be swapped to anything other than the last meta token
			for i := 0; i < len(t.Info.Tokens)-1; i += 1 {
				ret = append(ret, t.Info.Tokens[i])
			}

			// We don't allow swapping between underlying tokens here.
			// Swap between underlying tokens must go directly through the base pool.
		}
		return ret
	}
	// exchange
	for i := 0; i < len(t.Info.Tokens); i += 1 {
		if i != tokenIndex {
			ret = append(ret, t.Info.Tokens[i])
		}
	}
	// exchange_underlying
	// last meta token can't be swapped with underlying tokens
	if tokenIndex != len(t.Info.Tokens)-1 {
		ret = append(ret, t.basePool.GetInfo().Tokens...)
	}
	return ret
}

func (t *PoolSimulator) CanSwapFrom(address string) []string { return t.CanSwapTo(address) }

func (t *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	fromId, toId := t.GetTokenIndex(tokenIn), t.GetTokenIndex(tokenOut)
	if fromId >= 0 && toId >= 0 {
		return curve.Meta{
			TokenInIndex:  fromId,
			TokenOutIndex: toId,
			Underlying:    false,
		}
	}
	baseFromId, baseToId := t.getUnderlyingIndex(tokenIn), t.getUnderlyingIndex(tokenOut)
	return curve.Meta{
		TokenInIndex:  baseFromId,
		TokenOutIndex: baseToId,
		Underlying:    true,
	}
}

func (t *PoolSimulator) GetTokens() []string {
	var result []string
	result = append(result, t.GetInfo().Tokens...)
	result = append(result, t.basePool.GetInfo().Tokens...)
	return result
}

func (t *PoolSimulator) getUnderlyingIndex(token string) int {
	tokenIndex := t.GetTokenIndex(token)
	if tokenIndex >= 0 {
		return tokenIndex
	}
	baseIndex := t.basePool.GetTokenIndex(token)
	maxCoin := len(t.Info.Tokens) - 1
	if baseIndex >= 0 {
		tokenIndex = maxCoin + baseIndex
	}
	return tokenIndex
}
