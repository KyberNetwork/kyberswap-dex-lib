package iziswap

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
	"github.com/KyberNetwork/logger"
	"github.com/izumiFinance/iZiSwap-SDK-go/swap"
)

func getPointDelta(fee int) int {
	return pointDeltas[fee]
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func (d *PoolTracker) getLiquiditySnapshot(ctx context.Context, pool entity.Pool, poolInfo swap.PoolInfo) ([]swap.LiquidityPoint, error) {
	ptRange := d.config.PointRange
	if ptRange <= 0 {
		ptRange = DEFAULT_PT_RANGE
	}
	pointDelta := poolInfo.PointDelta
	leftPoint := poolInfo.CurrentPoint - ptRange
	modl := (leftPoint%pointDelta + pointDelta) % pointDelta
	if modl != 0 {
		leftPoint = leftPoint - modl
	}
	if leftPoint < poolInfo.LeftMostPt {
		leftPoint = poolInfo.LeftMostPt
	}
	rightPoint := poolInfo.CurrentPoint + ptRange
	modr := (rightPoint%pointDelta + pointDelta) % pointDelta
	if modr != 0 {
		rightPoint = rightPoint + pointDelta - modr
	}
	if rightPoint > poolInfo.RightMostPt {
		rightPoint = poolInfo.RightMostPt
	}
	batchLen := SNAPSHOT_BATCH * poolInfo.PointDelta
	deltaLiquidities := make([]*big.Int, SNAPSHOT_BATCH)
	liqudityPointLen := (rightPoint - leftPoint) / pointDelta
	liquidityPointData := make([]swap.LiquidityPoint, 0, liqudityPointLen)
	for start := leftPoint; start < rightPoint; start += batchLen {
		end := minInt(start+batchLen, rightPoint)
		rpcRequest := d.ethrpcClient.NewRequest()
		rpcRequest.SetContext(util.NewContextWithTimestamp(ctx))
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    iZiSwapPoolABI,
			Target: pool.Address,
			Method: methodGetLiquiditySnapshot,
			Params: []interface{}{big.NewInt(int64(start)), big.NewInt(int64(end))},
		}, []interface{}{&deltaLiquidities})
		resp, err := rpcRequest.TryAggregate()
		if err != nil {
			return nil, err
		}
		if !resp.Result[0] {
			logger.Errorf("failed to try multicall with param: %v", resp.Request.Calls[0].Params)
			continue
		}
		for idx, delta := range deltaLiquidities {
			if delta.Cmp(zeroBI) == 0 {
				continue
			}
			// todo:
			// we now simply fill liquidity-point-data with non-zero
			//     delta liqudity, which may cause some numerical
			//     error when calling pool_simulator,
			//     due to the fact that 0-deltaLiquidity may also
			//     be an endpoint of a liquidity segment
			// in our future work, we will try a finer way to fill
			// liquidity-point-data which will not cause numerical error
			liquidityPointData = append(
				liquidityPointData,
				swap.LiquidityPoint{
					LiqudityDelta: delta,
					Point:         start + idx*pointDelta,
				},
			)
		}
	}
	return liquidityPointData, nil
}

func (d *PoolTracker) getLimitOrderSnapshot(ctx context.Context, pool entity.Pool, poolInfo swap.PoolInfo) ([]swap.LimitOrderPoint, error) {
	ptRange := d.config.PointRange
	if ptRange <= 0 {
		ptRange = DEFAULT_PT_RANGE
	}
	pointDelta := poolInfo.PointDelta
	leftPoint := poolInfo.CurrentPoint - ptRange
	modl := (leftPoint%pointDelta + pointDelta) % pointDelta
	if modl != 0 {
		leftPoint = leftPoint - modl
	}
	if leftPoint < poolInfo.LeftMostPt {
		leftPoint = poolInfo.LeftMostPt
	}
	rightPoint := poolInfo.CurrentPoint + ptRange
	modr := (rightPoint%pointDelta + pointDelta) % pointDelta
	if modr != 0 {
		rightPoint = rightPoint + pointDelta - modr
	}
	if rightPoint > poolInfo.RightMostPt {
		rightPoint = poolInfo.RightMostPt
	}
	batchLen := SNAPSHOT_BATCH * poolInfo.PointDelta
	limitOrderDataRaw := make([]LimitOrder, SNAPSHOT_BATCH)
	limitOrderPointLen := (rightPoint - leftPoint) / pointDelta
	limitOrderPointData := make([]swap.LimitOrderPoint, 0, limitOrderPointLen)
	for start := leftPoint; start < rightPoint; start += batchLen {
		end := minInt(start+batchLen, rightPoint)
		rpcRequest := d.ethrpcClient.NewRequest()
		rpcRequest.SetContext(util.NewContextWithTimestamp(ctx))
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    iZiSwapPoolABI,
			Target: pool.Address,
			Method: methodGetLimitOrderSnapshot,
			Params: []interface{}{big.NewInt(int64(start)), big.NewInt(int64(end))},
		}, []interface{}{&limitOrderDataRaw})
		resp, err := rpcRequest.TryAggregate()
		if err != nil {
			return nil, err
		}
		if !resp.Result[0] {
			logger.Errorf("failed to try multicall with param: %v", resp.Request.Calls[0].Params)
			continue
		}
		for idx, limitOrder := range limitOrderDataRaw {
			if limitOrder.SellingX.Cmp(zeroBI) == 0 && limitOrder.SellingY.Cmp(zeroBI) == 0 {
				continue
			}
			limitOrderPointData = append(
				limitOrderPointData,
				swap.LimitOrderPoint{
					SellingX: limitOrder.SellingX,
					SellingY: limitOrder.SellingY,
					Point:    start + idx*pointDelta,
				},
			)
		}
	}
	return limitOrderPointData, nil
}
