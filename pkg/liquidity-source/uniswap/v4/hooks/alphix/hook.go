package alphix

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook struct {
	uniswapv4.Hook
	hook common.Address

	// Cached state from Track
	swapFee          uniswapv4.FeeAmount
	tickLower        int
	tickUpper        int
	amount0Available *uint256.Int // yield source reserves for token0
	amount1Available *uint256.Int // yield source reserves for token1
	sqrtPriceX96     *uint256.Int // current pool price
}

// AlphixExtra is the JSON-serialized hook state stored between Track calls.
type AlphixExtra struct {
	Fee              uint64 `json:"f"`
	TickLower        int    `json:"tL"`
	TickUpper        int    `json:"tU"`
	Amount0Available string `json:"a0"`
	Amount1Available string `json:"a1"`
}

// reHypothecationConfigRPC wraps the on-chain tuple returned by getReHypothecationConfig().
type reHypothecationConfigRPC struct {
	Config struct {
		TickLower *big.Int
		TickUpper *big.Int
	}
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook:             &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Alphix},
		hook:             param.HookAddress,
		amount0Available: uint256.NewInt(0),
		amount1Available: uint256.NewInt(0),
		sqrtPriceX96:     uint256.NewInt(0),
	}

	if param.HookExtra != "" {
		var extra AlphixExtra
		if err := json.Unmarshal([]byte(param.HookExtra), &extra); err == nil {
			hook.swapFee = uniswapv4.FeeAmount(extra.Fee)
			hook.tickLower = extra.TickLower
			hook.tickUpper = extra.TickUpper
			a0 := new(uint256.Int)
			if err := a0.SetFromDecimal(extra.Amount0Available); err == nil {
				hook.amount0Available = a0
			}
			a1 := new(uint256.Int)
			if err := a1.SetFromDecimal(extra.Amount1Available); err == nil {
				hook.amount1Available = a1
			}
		}
	}

	// Extract current sqrtPriceX96 from pool extra
	if param.Pool != nil && param.Pool.Extra != "" {
		var poolExtra uniswapv4.ExtraU256
		if err := json.Unmarshal([]byte(param.Pool.Extra), &poolExtra); err == nil {
			if poolExtra.ExtraTickU256 != nil && poolExtra.SqrtPriceX96 != nil {
				hook.sqrtPriceX96 = poolExtra.SqrtPriceX96
			}
		}
	}

	return hook
}, HookAddresses...)

// GetReserves returns the available liquidity from yield sources (Aave/Sky vaults).
// This is the rehypothecated liquidity that will be JIT-minted on every swap.
// Non-rehypothecated liquidity (regular LPs) is tracked separately via the standard
// PoolManager tick system.
func (h *Hook) GetReserves(ctx context.Context, param *uniswapv4.HookParam) (entity.PoolReserves, error) {
	if param.Pool == nil || len(param.Pool.Tokens) < 2 {
		return nil, nil
	}

	token0 := common.HexToAddress(param.Pool.Tokens[0].Address)
	token1 := common.HexToAddress(param.Pool.Tokens[1].Address)

	var amount0, amount1 *big.Int
	req := param.RpcClient.NewRequest().SetContext(ctx)
	if param.BlockNumber != nil {
		req.SetBlockNumber(param.BlockNumber)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    alphixHookABI,
		Target: h.hook.Hex(),
		Method: "getAmountInYieldSource",
		Params: []any{token0},
	}, []any{&amount0})
	req.AddCall(&ethrpc.Call{
		ABI:    alphixHookABI,
		Target: h.hook.Hex(),
		Method: "getAmountInYieldSource",
		Params: []any{token1},
	}, []any{&amount1})

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	return entity.PoolReserves{amount0.String(), amount1.String()}, nil
}

// Track fetches the current dynamic fee, JIT tick range, and yield source amounts.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (string, error) {
	if param.Pool == nil || len(param.Pool.Tokens) < 2 {
		return "", nil
	}

	token0 := common.HexToAddress(param.Pool.Tokens[0].Address)
	token1 := common.HexToAddress(param.Pool.Tokens[1].Address)

	var fee *big.Int
	var rhConfig reHypothecationConfigRPC
	var amount0, amount1 *big.Int

	req := param.RpcClient.NewRequest().SetContext(ctx)
	if param.BlockNumber != nil {
		req.SetBlockNumber(param.BlockNumber)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    alphixHookABI,
		Target: h.hook.Hex(),
		Method: "getFee",
	}, []any{&fee})
	req.AddCall(&ethrpc.Call{
		ABI:    alphixHookABI,
		Target: h.hook.Hex(),
		Method: "getReHypothecationConfig",
	}, []any{&rhConfig})
	req.AddCall(&ethrpc.Call{
		ABI:    alphixHookABI,
		Target: h.hook.Hex(),
		Method: "getAmountInYieldSource",
		Params: []any{token0},
	}, []any{&amount0})
	req.AddCall(&ethrpc.Call{
		ABI:    alphixHookABI,
		Target: h.hook.Hex(),
		Method: "getAmountInYieldSource",
		Params: []any{token1},
	}, []any{&amount1})

	if _, err := req.Aggregate(); err != nil {
		return "", err
	}

	extra := AlphixExtra{
		Fee:              fee.Uint64(),
		TickLower:        int(rhConfig.Config.TickLower.Int64()),
		TickUpper:        int(rhConfig.Config.TickUpper.Int64()),
		Amount0Available: amount0.String(),
		Amount1Available: amount1.String(),
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return "", err
	}
	return string(extraBytes), nil
}

// BeforeSwap computes the JIT liquidity contribution from rehypothecated yield sources.
//
// Alphix mints concentrated liquidity at [tickLower, tickUpper] before every swap using
// funds from ERC-4626 vaults. This method simulates that JIT position and returns the
// swap output it would produce, so the V3 simulator handles only the remaining amount
// against non-rehypothecated (regular LP) tick liquidity.
func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	// If no JIT range configured or no reserves, just return the dynamic fee
	if h.tickLower >= h.tickUpper || h.sqrtPriceX96.IsZero() ||
		(h.amount0Available.IsZero() && h.amount1Available.IsZero()) {
		return &uniswapv4.BeforeSwapResult{
			SwapFee:          h.swapFee,
			DeltaSpecified:   bignumber.ZeroBI,
			DeltaUnspecified: bignumber.ZeroBI,
		}, nil
	}

	// Compute sqrtPrice at tick boundaries
	var sqrtPriceLowerX96, sqrtPriceUpperX96 uint256.Int
	if err := v3Utils.GetSqrtRatioAtTickV2(h.tickLower, &sqrtPriceLowerX96); err != nil {
		return nil, err
	}
	if err := v3Utils.GetSqrtRatioAtTickV2(h.tickUpper, &sqrtPriceUpperX96); err != nil {
		return nil, err
	}

	// Compute JIT liquidity from available amounts (mirrors getLiquidityForAmounts on-chain)
	jitLiquidity := getLiquidityForAmounts(
		h.sqrtPriceX96, &sqrtPriceLowerX96, &sqrtPriceUpperX96,
		h.amount0Available, h.amount1Available,
	)
	if jitLiquidity.IsZero() {
		return &uniswapv4.BeforeSwapResult{
			SwapFee:          h.swapFee,
			DeltaSpecified:   bignumber.ZeroBI,
			DeltaUnspecified: bignumber.ZeroBI,
		}, nil
	}

	// Simulate the swap against the JIT position
	// Compute how much input the JIT position can absorb and what output it produces
	deltaSpecified, deltaUnspecified := computeJitSwap(
		params.ZeroForOne, params.ExactIn,
		params.AmountSpecified,
		h.sqrtPriceX96, &sqrtPriceLowerX96, &sqrtPriceUpperX96,
		jitLiquidity, h.swapFee,
	)

	return &uniswapv4.BeforeSwapResult{
		SwapFee:          h.swapFee,
		DeltaSpecified:   deltaSpecified,
		DeltaUnspecified: deltaUnspecified,
		Gas:              jitBeforeSwapGas,
	}, nil
}

// AfterSwap is a no-op for Alphix — no additional fees are taken after the swap.
func (h *Hook) AfterSwap(_ *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	return &uniswapv4.AfterSwapResult{
		HookFee: bignumber.ZeroBI,
	}, nil
}

// CloneState returns a deep copy of the hook for concurrent simulation.
func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h
	cloned.amount0Available = new(uint256.Int).Set(h.amount0Available)
	cloned.amount1Available = new(uint256.Int).Set(h.amount1Available)
	cloned.sqrtPriceX96 = new(uint256.Int).Set(h.sqrtPriceX96)
	return &cloned
}
