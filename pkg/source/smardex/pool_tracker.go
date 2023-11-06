package smardex

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
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
		reserve        Reserve
		feeToAmount    FeeToAmount
		fictiveReserve FictiveReserve
		pairFee        PairFee
		priceAverage   PriceAverage
		totalSupply    *big.Int
	)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairGetReservesMethod,
		Params: nil,
	}, []interface{}{&reserve})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairGetFictiveReservesMethod,
		Params: nil,
	}, []interface{}{&fictiveReserve})

	/*
	 * On ethereum: feesPool and feesLP are hardcode in sc
	 * Other chains: feesPool and feesLP are gotten by rpc method
	 * https://github.com/SmarDex-Dev/smart-contracts/pull/7/files
	 */
	if u.config.ChainID == 1 {
		pairFee = PairFee{
			FeesPool: FEES_POOL_DEFAULT_ETHEREUM,
			FeesLP:   FEES_LP_DEFAULT_ETHEREUM,
			FeesBase: FEES_BASE_ETHEREUM,
		}
	} else {
		pairFee = PairFee{
			FeesBase: FEES_BASE,
		}
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: p.Address,
			Method: pairGetPairFeesMethod,
			Params: nil,
		}, []interface{}{&pairFee})
	}

	/*
	 * On ethereum: feeToAmount will be gotten by getFees, in other chains getFeeToAmounts will be used instead
	 */
	getFeeMethodName := pairGetFeeToAmountsMethod
	if u.config.ChainID == 1 {
		getFeeMethodName = pairGetFeesMethod

	}
	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: getFeeMethodName,
		Params: nil,
	}, []interface{}{&feeToAmount})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairGetPriceAverageMethod,
		Params: nil,
	}, []interface{}{&priceAverage})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairTotalSupplyMethod,
		Params: nil,
	}, []interface{}{&totalSupply})

	_, err := rpcRequest.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("%s: failed to process tryAggregate for pool", u.config.DexID)
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(SmardexPair{
		PairFee:        pairFee,
		FictiveReserve: fictiveReserve,
		PriceAverage:   priceAverage,
		FeeToAmount:    feeToAmount,
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
			"dex":         u.config.DexID}).Errorf(
			"failed to marshal pool extra")
		return entity.Pool{}, err
	}

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
	p.TotalSupply = totalSupply.String()
	p.Reserves = entity.PoolReserves([]string{reserve.Reserve0.String(), reserve.Reserve1.String()})

	return p, nil

}
