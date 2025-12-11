package limitorder

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl"
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
	Status                   OrderStatus `json:"status"`
	LiquidityTotal           *big.Int    `json:"liquidityTotal"`
	TickLower                *big.Int    `json:"tickLower"`
	ZeroForOne               bool        `json:"zeroForOne"`
	AccCurrency0PerLiquidity *big.Int    `json:"accCurrency0PerLiquidity"`
	AccCurrency1PerLiquidity *big.Int    `json:"accCurrency1PerLiquidity"`
	PoolId                   common.Hash `json:"poolId"`
}

type Extra struct {
	PendingFillOrderList   []OrderId             `json:"pendingFillOrderList"`
	PendingFillOrderLength *uint256.Int          `json:"pendingFillOrderLength"`
	OrderInfos             map[OrderId]OrderInfo `json:"orderInfos"`
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
	extra.OrderInfos = make(map[OrderId]OrderInfo)
	for _, orderId := range extra.PendingFillOrderList {
		var orderInfo OrderInfo
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    Abi,
			Target: hexutil.Encode(param.HookAddress[:]),
			Method: "orderInfos",
			Params: []any{orderId},
		}, []any{&orderInfo})

		extra.OrderInfos[orderId] = orderInfo
	}
	if _, err := rpcRequest.TryBlockAndAggregate(); err != nil {
		return nil, err
	}

	return json.Marshal(extra)
}

// BeforeSwap might change pool state and affect the swap result
func (h *Hook) BeforeSwap(swapHookParams *cl.BeforeSwapParams) (*cl.BeforeSwapResult, error) {
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
	toBeFilledOrderInfos := make([]OrderInfo, 0)
	orderLength := len(h.Extra.PendingFillOrderList)
	if orderLength > 0 {
		remainingPendingFillOrderLength := h.Extra.PendingFillOrderLength.Uint64()
		if remainingPendingFillOrderLength > 0 {
			for i := 0; i < orderLength; i++ {
				orderId := h.Extra.PendingFillOrderList[i]
				orderInfo := h.OrderInfos[orderId]
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

	// TODO: update V3Pool inner state

	return &cl.BeforeSwapResult{
		DeltaSpecified:   big.NewInt(0),
		DeltaUnspecified: big.NewInt(0),
		SwapFee:          0,
		Gas:              0,
		SwapInfo:         nil,
	}, nil
}
