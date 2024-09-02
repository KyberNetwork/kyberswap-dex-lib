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
		reserve [2]*big.Int
		pairFee [2]*big.Int

		mintFee *big.Int
		burnFee *big.Int
		swapFee *big.Int

		// balances [2]*big.Int

		oracle common.Address
	)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    reserveABI,
		Target: p.Address,
		Method: libraryGetReservesMethod,
		Params: nil,
	}, []interface{}{&reserve})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    reserveABI,
		Target: p.Address,
		Method: libraryGetFeesMethod,
		Params: nil,
	}, []interface{}{&pairFee})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairMintFeeMethod,
		Params: nil,
	}, []interface{}{&mintFee})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairBurnFeeMethod,
		Params: nil,
	}, []interface{}{&burnFee})

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

	// rpcRequest.AddCall(&ethrpc.Call{
	// 	ABI:    pairABI,
	// 	Target: p.Address,
	// 	Method: libraryGetBalancesMethod,
	// 	Params: []string{},
	// }, []interface{}{&oracle})

	_, err := rpcRequest.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("%s: failed to process tryAggregate for pool", u.config.DexID)
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(IntegralPair{
		// PairFee: ToUint256(pairFee),
		MintFee: ToUint256(mintFee),
		BurnFee: ToUint256(burnFee),
		SwapFee: ToUint256(swapFee),
		Oracle:  oracle,
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
	p.Reserves = entity.PoolReserves([]string{reserve[0].String(), reserve[1].String()})

	return p, nil
}
