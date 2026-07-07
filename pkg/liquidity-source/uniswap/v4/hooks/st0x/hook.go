package st0x

import (
	"context"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Hook struct {
	uniswapv4.Hook `json:"-"`

	Price     uint64 `json:"p"`
	UpdatedAt uint64 `json:"u"`
	SpreadBps uint64 `json:"s"`

	MaxStaleness uint64 `json:"ms"`

	Reserve0 *uint256.Int `json:"r0"`
	Reserve1 *uint256.Int `json:"r1"`

	TrackedAt uint64 `json:"t"`
}

type SwapInfo struct {
	NewReserve0 *uint256.Int
	NewReserve1 *uint256.Int
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4ST0x},
	}
	_ = param.HookExtra.Unmarshal(hook)
	return hook
}, HookAddresses...)

func (h *Hook) AllowEmptyTicks() bool { return true }

func (h *Hook) GetReserves(ctx context.Context, param *uniswapv4.HookParam) (entity.PoolReserves, error) {
	hookAddr := param.HookAddress.Hex()
	token0 := common.HexToAddress(param.Pool.Tokens[0].Address)
	token1 := common.HexToAddress(param.Pool.Tokens[1].Address)

	var reserve0, reserve1 *big.Int
	req := param.RpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    propAMMHookABI,
		Target: hookAddr,
		Method: "getBalance",
		Params: []any{token0},
	}, []any{&reserve0}).AddCall(&ethrpc.Call{
		ABI:    propAMMHookABI,
		Target: hookAddr,
		Method: "getBalance",
		Params: []any{token1},
	}, []any{&reserve1})
	if _, err := req.Aggregate(); err != nil {
		return nil, fmt.Errorf("failed to fetch st0x reserves: %w", err)
	}

	h.Reserve0, _ = uint256.FromBig(reserve0)
	h.Reserve1, _ = uint256.FromBig(reserve1)

	return entity.PoolReserves{h.Reserve0.Dec(), h.Reserve1.Dec()}, nil
}

func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	poolId := common.HexToHash(param.Pool.Address)
	hookAddr := param.HookAddress
	oracle := oracleAddress

	var (
		price struct {
			Data struct {
				Price, UpdatedAt, SpreadBps *big.Int
			}
		}
		maxStaleness *big.Int
	)

	req := param.RpcClient.NewRequest().SetContext(ctx)
	if param.BlockNumber != nil {
		req.SetBlockNumber(param.BlockNumber)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    priceOracleABI,
		Target: oracle.Hex(),
		Method: "getPrice",
		Params: []any{poolId},
	}, []any{&price}).AddCall(&ethrpc.Call{
		ABI:    propAMMHookABI,
		Target: hookAddr.Hex(),
		Method: "maxStaleness",
	}, []any{&maxStaleness})

	if _, err := req.Aggregate(); err != nil {
		return nil, fmt.Errorf("failed to track st0x hook: %w", err)
	}

	h.Price = price.Data.Price.Uint64()
	h.UpdatedAt = price.Data.UpdatedAt.Uint64()
	h.SpreadBps = price.Data.SpreadBps.Uint64()
	h.MaxStaleness = maxStaleness.Uint64()
	if param.BlockNumber != nil {
		h.TrackedAt = param.BlockNumber.Uint64()
	}

	return json.Marshal(h)
}

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if h.Price == 0 {
		return nil, ErrNoPriceSet
	}
	if h.MaxStaleness > 0 && h.TrackedAt > h.UpdatedAt+h.MaxStaleness {
		return nil, ErrStalePrice
	}

	amt := params.AmountSpecified
	if amt == nil || amt.Sign() <= 0 {
		return &uniswapv4.BeforeSwapResult{
			DeltaSpecified:   bignumber.ZeroBI,
			DeltaUnspecified: bignumber.ZeroBI,
			Gas:              gasBeforeSwap,
		}, nil
	}

	amtU, overflow := uint256.FromBig(amt)
	if overflow {
		return nil, ErrInvalidSpread
	}

	reserveOut := h.Reserve1
	if !params.ZeroForOne {
		reserveOut = h.Reserve0
	}

	var (
		amountIn, amountOut              *uint256.Int
		deltaSpecified, deltaUnspecified *big.Int
	)
	if params.CalcOut {
		out, _, err := calcAmountOut(h.Price, h.SpreadBps, params.ZeroForOne, amtU)
		if err != nil {
			return nil, err
		}
		if reserveOut != nil && out.Cmp(reserveOut) > 0 {
			fmt.Println("reserve out:", out.String())
			return nil, ErrInsufficientRsv
		}
		amountIn, amountOut = amtU, out
		deltaSpecified = new(big.Int).Set(amt)
		deltaUnspecified = new(big.Int).Neg(out.ToBig())
	} else {
		if reserveOut != nil && amtU.Cmp(reserveOut) > 0 {
			fmt.Println("reserve out:", amtU.String())
			return nil, ErrInsufficientRsv
		}
		in, _, err := calcAmountIn(h.Price, h.SpreadBps, params.ZeroForOne, amtU)
		if err != nil {
			return nil, err
		}
		amountIn, amountOut = in, amtU
		deltaSpecified = new(big.Int).Neg(amt)
		deltaUnspecified = new(big.Int).Set(in.ToBig())
	}

	newReserve0, newReserve1 := h.applyDelta(params.ZeroForOne, amountIn, amountOut)

	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   deltaSpecified,
		DeltaUnspecified: deltaUnspecified,
		Gas:              gasBeforeSwap,
		SwapInfo:         &SwapInfo{NewReserve0: newReserve0, NewReserve1: newReserve1},
	}, nil
}

func (h *Hook) applyDelta(zeroForOne bool, amountIn, amountOut *uint256.Int) (*uint256.Int, *uint256.Int) {
	r0 := new(uint256.Int).Set(h.Reserve0)
	r1 := new(uint256.Int).Set(h.Reserve1)
	if zeroForOne {
		r0.Add(r0, amountIn)
		r1.Sub(r1, amountOut)
	} else {
		r1.Add(r1, amountIn)
		r0.Sub(r0, amountOut)
	}
	return r0, r1
}

func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h
	if h.Reserve0 != nil {
		cloned.Reserve0 = h.Reserve0.Clone()
	}
	if h.Reserve1 != nil {
		cloned.Reserve1 = h.Reserve1.Clone()
	}
	return &cloned
}

func (h *Hook) UpdateBalance(swapInfo any) {
	info, ok := swapInfo.(*SwapInfo)
	if !ok || info == nil {
		return
	}
	if info.NewReserve0 != nil {
		h.Reserve0 = info.NewReserve0
	}
	if info.NewReserve1 != nil {
		h.Reserve1 = info.NewReserve1
	}
}

var _ uniswapv4.Hook = (*Hook)(nil)
