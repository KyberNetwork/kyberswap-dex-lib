package clear

import (
	"context"
	"math/big"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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
		staticExtra: staticExtra,
		extra:       extra,
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

	// Check if pool is paused
	if p.extra.Paused {
		return nil, ErrInsufficientOutput
	}

	// For Clear, we need to call previewSwap on-chain to get the exact output
	// Since we can't make RPC calls during simulation, we use the cached rate
	// The actual rate will be verified during execution

	// Estimate output based on cached reserves ratio
	// This is an approximation - actual output comes from previewSwap
	amountOut := p.estimateAmountOut(tokenAmountIn.Token, tokenOut, tokenAmountIn.Amount)

	if amountOut == nil || amountOut.Sign() <= 0 {
		return nil, ErrInsufficientOutput
	}

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
func (p *PoolSimulator) estimateAmountOut(tokenIn, tokenOut string, amountIn *big.Int) *big.Int {
	tokenIn = strings.ToLower(tokenIn)
	tokenOut = strings.ToLower(tokenOut)

	// Check if we have cached reserves
	if p.extra.Reserves == nil {
		// Fall back to 1:1 ratio for stablecoins (Clear's primary use case)
		return new(big.Int).Set(amountIn)
	}

	reserveIn, hasIn := p.extra.Reserves[tokenIn]
	reserveOut, hasOut := p.extra.Reserves[tokenOut]

	if !hasIn || !hasOut || reserveIn == nil || reserveOut == nil {
		// Fall back to 1:1 ratio
		return new(big.Int).Set(amountIn)
	}

	if reserveIn.IsZero() {
		return big.NewInt(0)
	}

	// Simple ratio calculation: amountOut = amountIn * reserveOut / reserveIn
	amountInU256 := new(uint256.Int).SetBytes(amountIn.Bytes())
	result := new(uint256.Int).Mul(amountInU256, reserveOut)
	result = result.Div(result, reserveIn)

	return result.ToBig()
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
			common.HexToAddress(p.staticExtra.VaultAddress),
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
	cloned := *p
	cloned.RWMutex = &sync.RWMutex{}
	cloned.Info.Reserves = slices.Clone(p.Info.Reserves)

	// Deep copy extra reserves map
	if p.extra.Reserves != nil {
		cloned.extra.Reserves = make(map[string]*uint256.Int)
		for k, v := range p.extra.Reserves {
			if v != nil {
				cloned.extra.Reserves[k] = new(uint256.Int).Set(v)
			}
		}
	}

	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut

	tokenInIndex := p.GetTokenIndex(input.Token)
	tokenOutIndex := p.GetTokenIndex(output.Token)

	if tokenInIndex >= 0 && tokenInIndex < len(p.Info.Reserves) {
		p.Info.Reserves[tokenInIndex] = new(big.Int).Add(p.Info.Reserves[tokenInIndex], input.Amount)
	}

	if tokenOutIndex >= 0 && tokenOutIndex < len(p.Info.Reserves) {
		p.Info.Reserves[tokenOutIndex] = new(big.Int).Sub(p.Info.Reserves[tokenOutIndex], output.Amount)
	}
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return PoolMeta{
		VaultAddress: p.staticExtra.VaultAddress,
		SwapAddress:  p.staticExtra.SwapAddress,
	}
}

func (p *PoolSimulator) GetLpToken() string {
	return p.Info.Address
}
