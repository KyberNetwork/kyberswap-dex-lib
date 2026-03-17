package alphix

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook struct {
	uniswapv4.Hook          `json:"-"`
	uniswapv3.ExtraTickU256 `json:"-"`
	IsNative                [2]bool             `json:"-"`
	SwapFee                 uniswapv4.FeeAmount `json:"f"`
	TickLower               int                 `json:"l"`
	TickUpper               int                 `json:"u"`
	Amount0Available        *uint256.Int        `json:"0"` // yield source reserves for token0
	Amount1Available        *uint256.Int        `json:"1"` // yield source reserves for token1
	PoolManagerBalances     [2]*uint256.Int     `json:"b"`
	jitLiquidity            *uint256.Int
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
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Alphix},
	}
	var staticExtra uniswapv4.StaticExtra
	if param.HookExtra != "" {
		_ = json.Unmarshal([]byte(param.HookExtra), &hook)
	}
	if param.Pool != nil {
		if param.Pool.Extra != "" {
			_ = json.Unmarshal([]byte(param.Pool.Extra), &hook.ExtraTickU256)
		}
		if param.Pool.StaticExtra != "" {
			_ = json.Unmarshal([]byte(param.Pool.StaticExtra), &staticExtra)
			hook.IsNative = staticExtra.IsNative
		}
	}
	return hook
}, HookAddresses...)

func (h *Hook) AllowEmptyTicks() bool {
	return true
}

// GetReserves returns the available liquidity from yield sources (Aave/Sky vaults).
// This is the rehypothecated liquidity that will be JIT-minted on every swap.
// Non-rehypothecated liquidity (regular LPs) is tracked separately via the standard
// PoolManager tick system.
// It also fetches together the current dynamic fee and JIT tick range for Track.
func (h *Hook) GetReserves(ctx context.Context, param *uniswapv4.HookParam) (entity.PoolReserves, error) {
	if param.Pool == nil || len(param.Pool.Tokens) < 2 || h.SqrtPriceX96 == nil {
		return nil, nil
	}

	hook := hexutil.Encode(param.HookAddress[:])

	var amountsAvailable [2]*big.Int
	var poolManagerBalances [2]*big.Int
	var rhConfig reHypothecationConfigRPC
	req := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber)
	var tokens [2]common.Address
	for i, isNative := range h.IsNative {
		if isNative {
			req.AddCall(&ethrpc.Call{
				ABI:    abi.Multicall3ABI,
				Target: param.Cfg.Multicall3Address,
				Method: abi.Multicall3GetEthBalance,
				Params: []any{uniswapv4.PoolManager[param.Cfg.ChainID]},
			}, []any{&poolManagerBalances[i]})
		} else {
			token := param.Pool.Tokens[i].Address
			tokens[i] = common.HexToAddress(token)
			req.AddCall(&ethrpc.Call{
				ABI:    abi.Erc20ABI,
				Target: token,
				Method: abi.Erc20BalanceOfMethod,
				Params: []any{uniswapv4.PoolManager[param.Cfg.ChainID]},
			}, []any{&poolManagerBalances[i]})
		}
		req.AddCall(&ethrpc.Call{
			ABI:    alphixHookABI,
			Target: hook,
			Method: "getAmountInYieldSource",
			Params: []any{tokens[i]},
		}, []any{&amountsAvailable[i]})
	}
	if _, err := req.AddCall(&ethrpc.Call{
		ABI:    alphixHookABI,
		Target: hook,
		Method: "getFee",
	}, []any{(*uint64)(&h.SwapFee)}).AddCall(&ethrpc.Call{
		ABI:    alphixHookABI,
		Target: hook,
		Method: "getReHypothecationConfig",
	}, []any{&rhConfig}).Aggregate(); err != nil {
		return nil, err
	}

	h.TickLower = int(rhConfig.Config.TickLower.Int64())
	h.TickUpper = int(rhConfig.Config.TickUpper.Int64())
	h.Amount0Available = uint256.MustFromBig(amountsAvailable[0])
	h.Amount1Available = uint256.MustFromBig(amountsAvailable[1])
	h.PoolManagerBalances[0] = uint256.MustFromBig(poolManagerBalances[0])
	h.PoolManagerBalances[1] = uint256.MustFromBig(poolManagerBalances[1])

	reserve0, reserve1 := uniswapv4.EstimateReservesFromTicksU256(h.SqrtPriceX96, h.Ticks)
	return entity.PoolReserves{
		reserve0.Add(reserve0, big256.Min(h.Amount0Available, h.PoolManagerBalances[0])).String(),
		reserve1.Add(reserve1, big256.Min(h.Amount1Available, h.PoolManagerBalances[1])).String(),
	}, nil
}

// Track just encodes the current dynamic fee and JIT tick range fetched in GetReserves
func (h *Hook) Track(_ context.Context, _ *uniswapv4.HookParam) (string, error) {
	extraBytes, err := json.Marshal(h)
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
	if h.TickLower >= h.TickUpper || h.SqrtPriceX96 == nil || h.SqrtPriceX96.IsZero() ||
		(h.Amount0Available.IsZero() && h.Amount1Available.IsZero()) {
		return &uniswapv4.BeforeSwapResult{
			SwapFee:          h.SwapFee,
			DeltaSpecified:   bignumber.ZeroBI,
			DeltaUnspecified: bignumber.ZeroBI,
		}, nil
	}

	// Compute sqrtPrice at tick boundaries
	var sqrtPriceLowerX96, sqrtPriceUpperX96 uint256.Int
	if err := v3Utils.GetSqrtRatioAtTickV2(h.TickLower, &sqrtPriceLowerX96); err != nil {
		return nil, err
	}
	if err := v3Utils.GetSqrtRatioAtTickV2(h.TickUpper, &sqrtPriceUpperX96); err != nil {
		return nil, err
	}

	// Compute JIT liquidity from available amounts (mirrors getLiquidityForAmounts on-chain)
	if h.jitLiquidity == nil {
		h.jitLiquidity = getLiquidityForAmounts(
			h.SqrtPriceX96, &sqrtPriceLowerX96, &sqrtPriceUpperX96,
			h.Amount0Available, h.Amount1Available,
		)
	}
	if h.jitLiquidity.IsZero() {
		return &uniswapv4.BeforeSwapResult{
			SwapFee:          h.SwapFee,
			DeltaSpecified:   bignumber.ZeroBI,
			DeltaUnspecified: bignumber.ZeroBI,
		}, nil
	}

	// Simulate the swap against the JIT position
	// Compute how much input the JIT position can absorb and what output it produces
	deltaSpecified, deltaUnspecified, nextSqrtPriceX96 := computeJitSwap(
		params.ZeroForOne, params.ExactIn,
		params.AmountSpecified,
		h.SqrtPriceX96, &sqrtPriceLowerX96, &sqrtPriceUpperX96,
		h.jitLiquidity, h.SwapFee,
	)
	inputBalance := h.PoolManagerBalances[lo.Ternary(params.ZeroForOne, 0, 1)]
	if deltaSpecified.Gt(inputBalance) { // the hook transfers out tokenIn first before withdrawing from yield source
		return nil, uniswapv4.ErrInvalidAmountOut
	}

	unspecified := deltaUnspecified.ToBig()
	return &uniswapv4.BeforeSwapResult{
		SwapFee:          h.SwapFee,
		DeltaSpecified:   deltaSpecified.ToBig(),
		DeltaUnspecified: unspecified.Neg(unspecified), // negate to add to output
		Gas:              gasJitBeforeSwap,
		SwapInfo:         nextSqrtPriceX96,
	}, nil
}

// AfterSwap is a no-op for Alphix — no additional fees are taken after the swap.
func (h *Hook) AfterSwap(_ *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	return &uniswapv4.AfterSwapResult{
		HookFee: bignumber.ZeroBI,
		Gas:     gasJitAfterSwap,
	}, nil
}

func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h
	return &cloned
}

func (h *Hook) UpdateBalance(swapInfo any) {
	if nextSqrtPriceX96, ok := swapInfo.(*uint256.Int); ok {
		h.SqrtPriceX96 = nextSqrtPriceX96
	}
}
