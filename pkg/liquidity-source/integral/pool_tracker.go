package integral

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (u *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	logger.WithFields(
		logger.Fields{"poolAddress": p.Address}).Infof(
		"%s: Start getting new state of pool", u.config.DexID)

	rpcRequest := u.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	var (
		reserves          [2]*big.Int
		swapFee           = big.NewInt(0)
		decimalsConverter = big.NewInt(0)

		token0 common.Address
		token1 common.Address

		oracle common.Address
	)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    reserveABI,
		Target: p.Address,
		Method: libraryGetReservesMethod,
		Params: nil,
	}, []interface{}{&reserves})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairSwapFeeMethod,
		Params: nil,
	}, []interface{}{&swapFee})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairOracleMethod,
		Params: nil,
	}, []interface{}{&oracle})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairToken0Method,
		Params: nil,
	}, []interface{}{&token0})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairToken1Method,
		Params: nil,
	}, []interface{}{&token1})

	_, err := rpcRequest.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("%s: failed to process tryAggregate for pool", u.config.DexID)
		return entity.Pool{}, err
	}

	rpcRequest = u.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    oracleABI,
		Target: oracle.Hex(),
		Method: oracleDecimalsConverterMethod,
		Params: nil,
	}, []interface{}{&decimalsConverter})

	_, err = rpcRequest.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("%s: failed to process tryAggregate for pool", u.config.DexID)
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(IntegralPair{
		Reserve:           reserves,
		SwapFee:           ToUint256(swapFee),
		DecimalsConverter: decimalsConverter,
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
			"dex":         u.config.DexID}).Errorf(
			"failed to marshal pool extra")
		return entity.Pool{}, err
	}

	p.Timestamp = time.Now().Unix()
	p.Extra = string(extraBytes)
	p.Tokens = []*entity.PoolToken{
		{Address: token0.Hex()},
		{Address: token1.Hex()},
	}
	p.Reserves = entity.PoolReserves([]string{reserves[0].String(), reserves[1].String()})
	p.SwapFee, _ = swapFee.Float64()

	return p, nil
}
