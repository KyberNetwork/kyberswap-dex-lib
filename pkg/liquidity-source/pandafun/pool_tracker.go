package pandafun

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryE0(DexType, NewPoolTracker)

func NewPoolTracker(
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		ethrpcClient: ethrpcClient,
	}
}
func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Debugf("[%s] Start getting new state of pool", p.Type)

	var (
		minTradeSize               *big.Int
		amountInBuyRemainingTokens *big.Int
		liquidity                  *big.Int
		poolFees                   PoolFees
		sqrtPa                     *big.Int
		sqrtPb                     *big.Int
		baseReserve                *big.Int
		pandaReserve               *big.Int
	)

	poolABI, _ := PoolContractMetaData.GetAbi()
	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    *poolABI,
		Target: p.Address,
		Method: "minTradeSize",
		Params: nil,
	}, []any{&minTradeSize})
	calls.AddCall(&ethrpc.Call{
		ABI:    *poolABI,
		Target: p.Address,
		Method: "getAmountInBuyRemainingTokens",
		Params: nil,
	}, []any{&amountInBuyRemainingTokens})
	calls.AddCall(&ethrpc.Call{
		ABI:    *poolABI,
		Target: p.Address,
		Method: "liquidity",
		Params: nil,
	}, []any{&liquidity})
	calls.AddCall(&ethrpc.Call{
		ABI:    *poolABI,
		Target: p.Address,
		Method: "poolFees",
		Params: nil,
	}, []any{&poolFees})
	calls.AddCall(&ethrpc.Call{
		ABI:    *poolABI,
		Target: p.Address,
		Method: "sqrtPa",
		Params: nil,
	}, []any{&sqrtPa})
	calls.AddCall(&ethrpc.Call{
		ABI:    *poolABI,
		Target: p.Address,
		Method: "sqrtPb",
		Params: nil,
	}, []any{&sqrtPb})
	calls.AddCall(&ethrpc.Call{
		ABI:    *poolABI,
		Target: p.Address,
		Method: "baseReserve",
		Params: nil,
	}, []any{&baseReserve})
	calls.AddCall(&ethrpc.Call{
		ABI:    *poolABI,
		Target: p.Address,
		Method: "pandaReserve",
		Params: nil,
	}, []any{&pandaReserve})

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to get state of the pool")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(Extra{
		MinTradeSize:               minTradeSize,
		AmountInBuyRemainingTokens: amountInBuyRemainingTokens,
		Liquidity:                  liquidity,
		BuyFee:                     big.NewInt(int64(poolFees.BuyFee)),
		SellFee:                    big.NewInt(int64(poolFees.SellFee)),
		SqrtPa:                     sqrtPa,
		SqrtPb:                     sqrtPb,
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("failed to marshal extra data")

		return entity.Pool{}, err
	}

	p.Reserves = entity.PoolReserves{baseReserve.String(), pandaReserve.String()}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Debugf("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}
