package limitorder

import (
	"context"
	"encoding/json"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var _ = cl.RegisterHooksFactory(func(param *cl.HookParam) cl.Hook {
	hook := &Hook{Hook: cl.NewBaseHook(valueobject.ExchangePancakeInfinityCLLO, param)}
	if len(param.HookExtra) > 0 {
		_ = json.Unmarshal(param.HookExtra, &hook.Extra)
	}
	return hook
},
	common.HexToAddress("0x6AdC560aF85377f9a73d17c658D798c9B39186e8"),
)

type Hook struct {
	cl.Hook
	Extra
}

type OrderId *big.Int

// OrderStatus
// Open: order is active and not yet executed.
// Pending: order is executed, but pending liquidity removal, and ready to withdraw.
// Filled: order is executed and liquidity removed, ready to withdraw.
type OrderStatus uint8

const (
	OrderStatusOpen    OrderStatus = 0
	OrderStatusPending OrderStatus = 1
	OrderStatusFilled  OrderStatus = 2
)

// OrderInfo
// status uint8,
// liquidityTotal uint128,
// tickLower int24,
// zeroForOne bool,
// accCurrency0PerLiquidity uint256,
// accCurrency1PerLiquidity uint256,
// poolId bytes32

type OrderInfo struct {
	Status                   OrderStatus  `json:"status"`
	LiquidityTotal           *uint256.Int `json:"liquidityTotal"`
	TickLower                *big.Int     `json:"tickLower"`
	ZeroForOne               bool         `json:"zeroForOne"`
	AccCurrency0PerLiquidity *big.Int     `json:"accCurrency0PerLiquidity"`
	AccCurrency1PerLiquidity *big.Int     `json:"accCurrency1PerLiquidity"`
	PoolId                   common.Hash  `json:"poolId"`
}

type Extra struct {
	PendingFillOrderList   []OrderId            `json:"pendingFillOrderList"`
	PendingFillOrderLength *uint256.Int         `json:"pendingFillOrderLength"`
	OrderInfos             map[string]OrderInfo `json:"orderInfos"`
}

func (h *Hook) Track(ctx context.Context, param *cl.HookParam) ([]byte, error) {
	var extra Extra
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    Abi,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "getPendingFillOrderList",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&extra.PendingFillOrderList}).AddCall(&ethrpc.Call{
		ABI:    Abi,
		Target: hexutil.Encode(param.HookAddress[:]),
		Method: "pendingFillOrderLength",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&extra.PendingFillOrderLength}).TryBlockAndAggregate(); err != nil {
		return nil, err
	}

	rpcRequest := param.RpcClient.NewRequest().SetContext(ctx)
	extra.OrderInfos = make(map[string]OrderInfo)
	for _, orderId := range extra.PendingFillOrderList {
		var orderInfo OrderInfo
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    Abi,
			Target: hexutil.Encode(param.HookAddress[:]),
			Method: "orderInfos",
			Params: []any{orderId},
		}, []any{&orderInfo})

		orderIdStr := (*big.Int)(orderId).Text(16)
		extra.OrderInfos[orderIdStr] = orderInfo
	}
	if _, err := rpcRequest.TryBlockAndAggregate(); err != nil {
		return nil, err
	}

	return json.Marshal(extra)
}

func (h *Hook) ModifyTicks(ctx context.Context, extraTickU256 *uniswapv3.ExtraTickU256) error {
	// ref: https://bscscan.com/address/0x6AdC560aF85377f9a73d17c658D798c9B39186e8#code
	// Contract: CLLimitOrderHook.sol
	/*
			/// @dev Processes any pending limit orders from previous swaps.
		    function _beforeSwap(address, PoolKey calldata key, ICLPoolManager.SwapParams calldata, bytes calldata)
		        internal
		        override
		        returns (bytes4, BeforeSwapDelta, uint24)
		    {
		        PoolId poolId = key.toId();
		        OrderIdLibrary.OrderId[] storage pendingFillOrderIds = pendingFillOrderList[poolId];
		        uint256 orderLength = pendingFillOrderIds.length;

		        if (orderLength > 0) {
		            uint256 remainingPendingFillOrderLength = pendingFillOrderLength[poolId];
		            if (remainingPendingFillOrderLength > 0) {
		                for (uint256 i = 0; i < orderLength; i++) {
		                    OrderIdLibrary.OrderId orderId = pendingFillOrderIds[i];
		                    OrderInfo storage orderInfo = orderInfos[orderId];
		                    if (orderInfo.status == OrderStatus.Pending) {
		                        _fillOrder(orderId, orderInfo, key);
		                        remainingPendingFillOrderLength -= 1;
		                    }
		                    if (remainingPendingFillOrderLength == 0) {
		                        break;
		                    }
		                }
		            }
		            // need to clear pendingFillOrder storage if it is not empty
		            delete pendingFillOrderList[poolId];
		            delete pendingFillOrderLength[poolId];
		        }
		        return (this.beforeSwap.selector, BeforeSwapDeltaLibrary.ZERO_DELTA, 0);
		    }
	*/
	// 1. Mapping BeforeSwap of CLHook to get list of orders to be filled
	toBeFilledOrderInfos := make([]OrderInfo, 0)
	orderLength := len(h.PendingFillOrderList)
	if orderLength > 0 {
		remainingPendingFillOrderLength := h.PendingFillOrderLength.Uint64()
		if remainingPendingFillOrderLength > 0 {
			for i := 0; i < orderLength; i++ {
				orderId := h.PendingFillOrderList[i]
				orderInfo := h.OrderInfos[(*big.Int)(orderId).Text(16)]
				if orderInfo.Status == OrderStatusPending {
					remainingPendingFillOrderLength -= 1
					toBeFilledOrderInfos = append(toBeFilledOrderInfos, orderInfo)
				}

				if remainingPendingFillOrderLength == 0 {
					break
				}
			}
		}
	}

	// 2. Remove liquidity from the pool by ModifyLiquidity, since LO is filled.
	tickSpacing := big.NewInt(int64(extraTickU256.TickSpacing))
	var liquidityDelta *int256.Int
	for _, orderInfo := range toBeFilledOrderInfos {
		tickLower := orderInfo.TickLower
		tickUpper := new(big.Int).Add(tickLower, tickSpacing)
		tickCurrent := big.NewInt(int64(*extraTickU256.Tick))

		// since we are removed liquidity from the pool (LO filled), so liquidityDelta is negative
		// liquidityDelta: -int256(uint256(totalLiquidity)),
		int256LiquidityTotal := int256.MustFromBig(orderInfo.LiquidityTotal.ToBig())
		liquidityDelta = int256LiquidityTotal.Mul(int256LiquidityTotal, int256.NewInt(-1))

		// Pools liquidity tracks the currently active liquidity given pools current tick.
		// We only want to update it on mint if the new position includes the current tick.
		if tickCurrent != nil &&
			tickLower.Cmp(tickCurrent) <= 0 &&
			tickUpper.Cmp(tickCurrent) > 0 {

			liquidity := orderInfo.LiquidityTotal
			// Note: In this LO hook, we are removing liquidity from the pool, because the order is going to be filled
			if extraTickU256.Liquidity.Cmp(liquidity) < 0 {
				return ErrorSubLiquidityUnderflow
			}
			extraTickU256.Liquidity.Sub(extraTickU256.Liquidity, liquidity)
		}

		lowerTickIdx := tickLower.Int64()
		upperTickIdx := tickUpper.Int64()

		// find lower and upper tick instances
		var lowerTickInstance, upperTickInstance *uniswapv3.TickU256
		for i := range extraTickU256.Ticks {
			if extraTickU256.Ticks[i].Index == int(lowerTickIdx) {
				lowerTickInstance = &extraTickU256.Ticks[i]
			}
			if extraTickU256.Ticks[i].Index == int(upperTickIdx) {
				upperTickInstance = &extraTickU256.Ticks[i]
			}

			if lowerTickInstance != nil && upperTickInstance != nil {
				break
			}
		}

		if lowerTickInstance == nil {
			lowerTickInstance = &uniswapv3.TickU256{
				Index: int(lowerTickIdx),
				// if tick not found, it means we are adding liq (not expected), so we can leave LiquidityGross = LiquidityDelta
				LiquidityGross: orderInfo.LiquidityTotal,
				LiquidityNet:   liquidityDelta,
			}
			extraTickU256.Ticks = append(extraTickU256.Ticks, *lowerTickInstance)
		} else {
			// since LiquidityGross is uint256, and we are removing liq, so we need check sign of liquidityDelta
			// abs(liquidityDelta) == liquidityTotal > 0
			if liquidityDelta.Sign() < 0 {
				lowerTickInstance.LiquidityGross.Sub(lowerTickInstance.LiquidityGross, orderInfo.LiquidityTotal)
			} else {
				lowerTickInstance.LiquidityGross.Add(lowerTickInstance.LiquidityGross, orderInfo.LiquidityTotal)
			}

			lowerTickInstance.LiquidityNet.Add(lowerTickInstance.LiquidityNet, liquidityDelta)
		}

		if upperTickInstance == nil {
			upperTickInstance = &uniswapv3.TickU256{
				Index: int(upperTickIdx),
				// if tick not found, it means we are adding liq (not expected), so we can leave LiquidityGross = LiquidityDelta
				LiquidityGross: orderInfo.LiquidityTotal,
				LiquidityNet:   liquidityDelta.Mul(liquidityDelta, int256.NewInt(-1)),
			}
			extraTickU256.Ticks = append(extraTickU256.Ticks, *upperTickInstance)
		} else {
			// since LiquidityGross is uint256, and we are removing liq, so we need check sign of liquidityDelta
			// abs(liquidityDelta) == liquidityTotal > 0
			if liquidityDelta.Sign() < 0 {
				upperTickInstance.LiquidityGross.Sub(upperTickInstance.LiquidityGross, orderInfo.LiquidityTotal)
			} else {
				upperTickInstance.LiquidityGross.Add(upperTickInstance.LiquidityGross, orderInfo.LiquidityTotal)
			}

			// remember this is Sub
			upperTickInstance.LiquidityNet.Sub(upperTickInstance.LiquidityNet, liquidityDelta)
		}
	}

	// resort ticks by index
	sort.Slice(extraTickU256.Ticks, func(i, j int) bool {
		return extraTickU256.Ticks[i].Index < extraTickU256.Ticks[j].Index
	})

	return nil
}
