package clear

import (
	"context"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	*sync.RWMutex
	pool.Pool
	staticExtra  StaticExtra
	extra        Extra
	gas          Gas
	ethrpcClient *ethrpc.Client
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	if len(entityPool.StaticExtra) == 0 {
		return nil, ErrStaticExtraEmpty
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if len(entityPool.Extra) > 0 {
		if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
			return nil, err
		}
	}

	// Parse reserves from entity
	reserves := make([]*big.Int, len(entityPool.Tokens))
	for i, r := range entityPool.Reserves {
		reserve, ok := new(big.Int).SetString(r, 10)
		if !ok {
			reserve = big.NewInt(0)
		}
		reserves[i] = reserve
	}

	info := pool.PoolInfo{
		Address:  strings.ToLower(entityPool.Address),
		Exchange: entityPool.Exchange,
		Type:     entityPool.Type,
		Tokens:   lo.Map(entityPool.Tokens, func(e *entity.PoolToken, _ int) string { return e.Address }),
		Reserves: reserves,
	}

	return &PoolSimulator{
		RWMutex:     &sync.RWMutex{},
		Pool:        pool.Pool{Info: info},
		extra:       extra,
		staticExtra: staticExtra,
		gas:         DefaultGas,
	}, nil
}

// SetEthrpcClient sets the RPC client for on-chain pricing calls
func (p *PoolSimulator) SetEthrpcClient(client *ethrpc.Client) {
	p.ethrpcClient = client
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut

	// Validate tokens
	tokenInIndex := p.GetTokenIndex(tokenAmountIn.Token)
	tokenOutIndex := p.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, ErrInvalidToken
	}

	if tokenAmountIn.Amount == nil || tokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	// For Clear, we need to call previewSwap on-chain to get the exact output
	// Since we can't make RPC calls during simulation, we use the cached rate
	// The actual rate will be verified during execution

	// Estimate output based on cached reserves ratio
	// This is an approximation - actual output comes from previewSwap
	amountOut := p.estimateAmountOut(tokenInIndex, tokenOutIndex, tokenAmountIn.Amount)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		RemainingTokenAmountIn: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: integer.Zero(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: integer.Zero(), // Clear handles fees internally
		},
		Gas: p.gas.Swap,
	}, nil
}

// estimateAmountOut estimates the output amount based on cached data
// For Clear protocol, this is an approximation since actual pricing requires RPC
func (p *PoolSimulator) estimateAmountOut(tokenInIndex, tokenOutIndex int, amountIn *big.Int) *big.Int {
	index0, index1 := tokenInIndex, tokenOutIndex
	if tokenInIndex > tokenOutIndex {
		index0, index1 = tokenOutIndex, tokenInIndex
	}
	if p.extra.Reserves == nil || !lo.HasKey(p.extra.Reserves, index0) || !lo.HasKey(p.extra.Reserves[index0], index1) {
		return big.NewInt(0)
	}
	reserves := p.extra.Reserves[index0][index1]
	var reserveIn, reserveOut *big.Int
	if tokenInIndex < tokenOutIndex {
		reserveIn, reserveOut = reserves.AmountIn, reserves.AmountOut
	} else {
		reserveIn, reserveOut = reserves.AmountOut, reserves.AmountIn
	}

	if reserveIn == nil || reserveOut == nil || reserveIn.Sign() == 0 || reserveOut.Sign() == 0 {
		return big.NewInt(0)
	}

	// Simple ratio calculation: amountOut = amountIn * reserveOut / reserveIn

	return bignumber.MulDivDown(new(big.Int), amountIn, reserveOut, reserveIn)
}

// CalcAmountOutWithRPC calculates output using actual RPC call to previewSwap
// This should be called when accurate pricing is needed
func (p *PoolSimulator) CalcAmountOutWithRPC(
	ctx context.Context,
	tokenIn, tokenOut string,
	amountIn *big.Int,
	ethrpcClient *ethrpc.Client,
) (*big.Int, *big.Int, error) {
	if ethrpcClient == nil {
		return nil, nil, ErrPoolNotFound
	}

	// Create timeout context before building the request
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var amountOut, ious *big.Int

	calls := ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    clearSwapABI,
		Target: p.staticExtra.SwapAddress,
		Method: methodPreviewSwap,
		Params: []any{
			common.HexToAddress(p.Info.Address),
			common.HexToAddress(tokenIn),
			common.HexToAddress(tokenOut),
			amountIn,
		},
	}, []any{&amountOut, &ious})

	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"pool":  p.Info.Address,
			"error": err,
		}).Errorf("[Clear] failed to call previewSwap")
		return nil, nil, err
	}

	return amountOut, ious, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	return p
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return PoolMeta{
		SwapAddress: p.staticExtra.SwapAddress,
	}
}

func (p *PoolSimulator) GetLpToken() string {
	return p.Info.Address
}
