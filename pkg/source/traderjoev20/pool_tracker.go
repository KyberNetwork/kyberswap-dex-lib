package traderjoev20

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/traderjoecommon"
)

type PoolTracker struct {
	EthrpcClient *ethrpc.Client
	Config       *traderjoecommon.Config
}

func NewPoolTracker(
	ethrpcClient *ethrpc.Client,
	config *traderjoecommon.Config,
) (*PoolTracker, error) {
	return &PoolTracker{
		EthrpcClient: ethrpcClient,
		Config:       config,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.Infof("[TraderJoe v2.0] Start getting new state of pool: %v", p.Address)

	rpcRequest := d.EthrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	var reserves ReservesAndID
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairGetReservesAndIDMethod,
	}, []interface{}{&reserves})

	_, err := rpcRequest.Call()
	if err != nil {
		logger.Errorf("failed to call pool: %v, err: %v", p.Address, err)
		return entity.Pool{}, err
	}

	rpcRequest = d.EthrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	var binReserves BinReserves
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairGetBinReservesMethod,
		Params: []interface{}{reserves.ActiveId},
	}, []interface{}{&binReserves})

	var priceX128 *big.Int
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    routerABI,
		Target: d.Config.RouterAddress,
		Method: routerGetPriceFromIDMethod,
		Params: []interface{}{
			common.HexToAddress(p.Address),
			reserves.ActiveId,
		},
	}, []interface{}{&priceX128})

	_, err = rpcRequest.TryAggregate()
	if err != nil {
		logger.Errorf("failed to call pool: %v, err: %v", p.Address, err)
		return entity.Pool{}, err
	}

	liquidity := traderjoecommon.CalculateLiquidity(priceX128, binReserves.ReserveX, binReserves.ReserveY)

	extraBytes, err := json.Marshal(Extra{
		Liquidity: liquidity,
		PriceX128: priceX128,
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves.GetPoolReserves()

	logger.Infof("[TraderJoe v2.0] Finish getting new state of pool: %v", p.Address)
	return p, nil
}
