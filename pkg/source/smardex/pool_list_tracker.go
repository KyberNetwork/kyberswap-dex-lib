package smardex

import (
	"context"
	"encoding/hex"
	"fmt"
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
		// reserve Reserve
		// feeToAmount FeeToAmount
		fictiveReserve FictiveReseerve
		// pairFee        PairFee
		// priceAverage PriceAverage
		totalSupply *big.Int
	)

	// rpcRequest.AddCall(&ethrpc.Call{
	// 	ABI:    pairABI,
	// 	Target: p.Address,
	// 	Method: pairGetReservesMethod,
	// 	Params: nil,
	// }, []interface{}{&reserve})

	// rpcRequest.AddCall(&ethrpc.Call{
	// 	ABI:    pairABI,
	// 	Target: p.Address,
	// 	Method: pairGetFeeToAmountsMethod,
	// 	Params: nil,
	// }, []interface{}{&feeToAmount})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairGetFictiveReservesMethod,
		Params: nil,
	}, []interface{}{&fictiveReserve})

	// rpcRequest.AddCall(&ethrpc.Call{
	// 	ABI:    pairABI,
	// 	Target: p.Address,
	// 	Method: pairGetPairFeesMethod,
	// 	Params: nil,
	// }, []interface{}{&pairFee})

	// rpcRequest.AddCall(&ethrpc.Call{
	// 	ABI:    pairABI,
	// 	Target: p.Address,
	// 	Method: pairGetPriceAverageMethod,
	// 	Params: nil,
	// }, []interface{}{&priceAverage})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairTotalSupplyMethod,
		Params: nil,
	}, []interface{}{&totalSupply})

	res, err := rpcRequest.TryAggregate()
	// var buff bytes.Buffer
	// buff.Write(res.RawResponse)
	fmt.Printf("cmnguyen res=%s\n", hex.EncodeToString(res.RawResponse))
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
			"response":    res,
		}).Errorf("%s: failed to process tryAggregate for pool %s", u.config.DexID, res)
		return entity.Pool{}, err
	}

	// extraBytes, err := json.Marshal(SmardexPair{
	// 	FictiveReserve: fictiveReserve,
	// 	PriceAverage:   priceAverage,
	// 	FeeToAmount:    feeToAmount,
	// 	Reserve:        reserve,
	// })
	// if err != nil {
	// 	logger.WithFields(logger.Fields{
	// 		"poolAddress": p.Address,
	// 		"error":       err,
	// 		"dex":         u.config.DexID}).Errorf(
	// 		"failed to marshal pool extra")
	// 	return entity.Pool{}, err
	// }

	// staticExtraBytes, err := json.Marshal(PairFee{
	// 	feesLP:   pairFee.feesLP,
	// 	feesPool: pairFee.feesPool,
	// })
	// if err != nil {
	// 	logger.WithFields(logger.Fields{
	// 		"poolAddress": p.Address,
	// 		"error":       err,
	// 		"dex":         u.config.DexID}).Errorf(
	// 		"failed to marshal pool extra")
	// 	return entity.Pool{}, err
	// }

	// p.Extra = string(extraBytes)
	// p.StaticExtra = string(staticExtraBytes)
	p.TotalSupply = totalSupply.String()
	// p.Reserves = entity.PoolReserves([]string{reserve.reserve0.String(), reserve.reserve1.String()})

	return p, nil

}
