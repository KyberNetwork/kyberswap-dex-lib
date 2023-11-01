package smardex

import (
	"context"
	"encoding/json"
	"math/big"

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
		ethereumFees   EthereumFeeToAmount
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
	 * https://github.com/SmarDex-Dev/smart-contracts/commit/8d5a1e84123e07459b5dd6afbed615eacd633cc0#diff-26f384a3693d317994210fb738bb0ae5dfe485c46d867ed0b6bbf4fb7671b199
	 */
	if u.config.ChainID == 1 {
		pairFee = PairFee{
			FeesPool: big.NewInt(2),
			FeesLP:   big.NewInt(5),
			FeesBase: FEES_BASE_ETHEREUM,
		}
	} else {
		pairFee = PairFee{
			FeesBase: FEES_BASE_ETHEREUM,
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
	if u.config.ChainID == 1 {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: p.Address,
			Method: pairGetFeesMethod,
			Params: nil,
		}, []interface{}{&ethereumFees})
	} else {
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: p.Address,
			Method: pairGetFeeToAmountsMethod,
			Params: nil,
		}, []interface{}{&feeToAmount})
	}

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

	if u.config.ChainID == 1 {
		feeToAmount.FeeToAmount0 = ethereumFees.Fees0
		feeToAmount.FeeToAmount1 = ethereumFees.Fees1
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

	p.Extra = string(extraBytes)
	p.TotalSupply = totalSupply.String()
	p.Reserves = entity.PoolReserves([]string{reserve.Reserve0.String(), reserve.Reserve1.String()})

	return p, nil

}
