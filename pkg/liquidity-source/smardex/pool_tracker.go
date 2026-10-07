package smardex

import (
	"context"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexTypeSmardex, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (u *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return u.getNewPoolState(ctx, p, params, nil)
}

func (u *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return u.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (u *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(
		logger.Fields{"poolAddress": p.Address}).Infof(
		"%s: Start getting new state of pool", u.config.DexID)

	rpcRequest := u.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)
	if overrides != nil {
		rpcRequest.SetOverrides(overrides)
	}
	var (
		reserve        Reserve
		feeToAmount    FeeToAmountResult
		fictiveReserve FictiveReserveResult
		pairFee        PairFeeResult
		priceAverage   PriceAverageResult
	)

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairGetReservesMethod,
	}, []any{&reserve})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairGetFictiveReservesMethod,
	}, []any{&fictiveReserve})

	/*
	 * On ethereum: feesPool and feesLP are hardcode in sc
	 * Other chains: feesPool and feesLP are gotten by rpc method
	 * https://github.com/SmarDex-Dev/smart-contracts/pull/7/files
	 */
	if u.config.ChainID == 1 {
		pairFee = PairFeeResult{
			FeesPool: uint256.MustFromBig(FEES_POOL_DEFAULT_ETHEREUM),
			FeesLP:   uint256.MustFromBig(FEES_LP_DEFAULT_ETHEREUM),
			FeesBase: uint256.MustFromBig(FEES_BASE_ETHEREUM),
		}
	} else {
		pairFee = PairFeeResult{
			FeesBase: uint256.MustFromBig(FEES_BASE),
		}
		rpcRequest.AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: p.Address,
			Method: pairGetPairFeesMethod,
		}, []any{&pairFee})
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
	}, []any{&feeToAmount})

	rpcRequest.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: p.Address,
		Method: pairGetPriceAverageMethod,
	}, []any{&priceAverage})

	resp, err := rpcRequest.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("%s: failed to process tryAggregate for pool", u.config.DexID)
		return entity.Pool{}, err
	}
	if resp != nil && resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}

	extraBytes, err := json.Marshal(SmardexPair{
		PairFee: PairFee{
			FeesLP:   pairFee.FeesLP,
			FeesPool: pairFee.FeesPool,
			FeesBase: pairFee.FeesBase,
		},
		FictiveReserve: FictiveReserve{
			FictiveReserve0: fictiveReserve.FictiveReserve0,
			FictiveReserve1: fictiveReserve.FictiveReserve1,
		},
		PriceAverage: PriceAverage{
			PriceAverage0:             priceAverage.PriceAverage0,
			PriceAverage1:             priceAverage.PriceAverage1,
			PriceAverageLastTimestamp: priceAverage.PriceAverageLastTimestamp,
		},
		FeeToAmount: FeeToAmount{
			Fees0: feeToAmount.Fees0,
			Fees1: feeToAmount.Fees1,
		},
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
			"dex":         u.config.DexID,
		}).Errorf("failed to marshal pool extra")
		return entity.Pool{}, err
	}

	p.Timestamp = time.Now().Unix()
	p.Extra = string(extraBytes)
	p.Reserves = []string{reserve.Reserve0.Dec(), reserve.Reserve1.Dec()}

	return p, nil

}
