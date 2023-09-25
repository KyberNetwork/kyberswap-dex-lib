package balancercomposablestable

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	}).Infof("[Balancer-Composable-Stable] Start updating state ...")

	var staticExtra = StaticExtra{}
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
		}).Errorf("failed to unmarshal pool static extra")

		return entity.Pool{}, err
	}

	poolIdParam := common.HexToHash(staticExtra.PoolId)

	var (
		poolTokens                          PoolTokens
		amplificationParameter              AmplificationParameter
		scalingFactors                      []*big.Int
		swapFeePercentage                   *big.Int
		bptIndex                            *big.Int
		totalSupply                         *big.Int
		lastJoinExit                        LastJoinExitData
		protocolFeePercentageCacheSwapType  *big.Int
		protocolFeePercentageCacheYieldType *big.Int
	)
	tokensExemptFromYieldProtocolFee := make([]bool, len(p.Tokens))
	tokenRateCaches := make([]TokenRateCache, len(p.Tokens))
	rateProviders := make([]common.Address, len(p.Tokens))

	calls := d.ethrpcClient.NewRequest()
	calls.SetContext(ctx)

	calls.AddCall(&ethrpc.Call{
		ABI:    vaultABI,
		Target: staticExtra.VaultAddress,
		Method: vaultMethodGetPoolTokens,
		Params: []interface{}{poolIdParam},
	}, []interface{}{&poolTokens})

	calls.AddCall(&ethrpc.Call{
		ABI:    balancerPoolABI,
		Target: p.Address,
		Method: poolMethodGetSwapFeePercentage,
		Params: nil,
	}, []interface{}{&swapFeePercentage})

	if DexType(p.Type) == DexTypeBalancerComposableStable {
		calls.AddCall(&ethrpc.Call{
			ABI:    composableStablePoolABI,
			Target: p.Address,
			Method: composableStablePoolMethodGetBptIndex,
			Params: nil,
		}, []interface{}{&bptIndex})

		calls.AddCall(&ethrpc.Call{
			ABI:    composableStablePoolABI,
			Target: p.Address,
			Method: poolMethodGetAmplificationParameter,
			Params: nil,
		}, []interface{}{&amplificationParameter})

		calls.AddCall(&ethrpc.Call{
			ABI:    composableStablePoolABI,
			Target: p.Address,
			Method: metaStablePoolMethodGetScalingFactors,
			Params: nil,
		}, []interface{}{&scalingFactors})

		calls.AddCall(&ethrpc.Call{
			ABI:    composableStablePoolABI,
			Target: p.Address,
			Method: composableStablePoolMethodGetLastJoinExitData,
			Params: nil,
		}, []interface{}{&lastJoinExit})

		calls.AddCall(&ethrpc.Call{
			ABI:    composableStablePoolABI,
			Target: p.Address,
			Method: composableStablePoolMethodGetTotalSupply,
			Params: nil,
		}, []interface{}{&totalSupply})

		calls.AddCall(&ethrpc.Call{
			ABI:    composableStablePoolABI,
			Target: p.Address,
			Method: composableStablePoolMethodGetRateProviders,
			Params: nil,
		}, []interface{}{&rateProviders})

		calls.AddCall(&ethrpc.Call{
			ABI:    composableStablePoolABI,
			Target: p.Address,
			Method: composableStablePoolMethodGetProtocolFeePercentageCache,
			Params: []interface{}{ProtocolFeeTypeSwap},
		}, []interface{}{&protocolFeePercentageCacheSwapType})

		calls.AddCall(&ethrpc.Call{
			ABI:    composableStablePoolABI,
			Target: p.Address,
			Method: composableStablePoolMethodGetProtocolFeePercentageCache,
			Params: []interface{}{ProtocolFeeTypeYield},
		}, []interface{}{&protocolFeePercentageCacheYieldType})

		for i, token := range p.Tokens {
			address := token.Address
			calls.AddCall(&ethrpc.Call{
				ABI:    composableStablePoolABI,
				Target: p.Address,
				Method: composableStablePoolMethodIsTokenExemptFromYieldProtocolFee,
				Params: []interface{}{common.HexToAddress(address)},
			}, []interface{}{&tokensExemptFromYieldProtocolFee[i]})
		}
	}
	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("[Balancer-Composable-Stable] failed to aggregate for pool data")
		return entity.Pool{}, err
	}

	/* DexTypeBalancerComposableStable
	This DexTypeBalancerComposableStable function must be called separately, after all above functions call successfully.
	Because this depends on  getRateProviders data.
	We only should call getTokenRateCache of a token only if this token has rateProvider.
	*/
	if DexType(p.Type) == DexTypeBalancerComposableStable {
		callsRPCForRateCache := d.ethrpcClient.NewRequest()
		callsRPCForRateCache.SetContext(ctx)
		for i, token := range p.Tokens {
			address := token.Address
			rateProvider := strings.ToLower(rateProviders[i].Hex())
			// Only get rate cache if this token has rate provider
			if address != p.Address && rateProvider != "" && rateProvider != valueobject.ZeroAddress {
				callsRPCForRateCache.AddCall(&ethrpc.Call{
					ABI:    composableStablePoolABI,
					Target: p.Address,
					Method: composableStablePoolMethodGetTokenRateCache,
					Params: []interface{}{common.HexToAddress(address)},
				}, []interface{}{&tokenRateCaches[i]})
			}
		}

		if _, err := callsRPCForRateCache.Aggregate(); err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
				"error":       err,
			}).Errorf("[Balancer-Composable-Stable] failed to aggregate for pool data in Second Call: RPC get rate cache")
			return entity.Pool{}, err
		}
	}

	if swapFeePercentage != nil {
		swapFee, _ := new(big.Float).Quo(new(big.Float).SetInt(swapFeePercentage), bOneFloat).Float64()
		p.SwapFee = swapFee
	}

	reserves := make([]string, len(p.Tokens))
	for i, token := range p.Tokens {
		for j, t := range poolTokens.Tokens {
			if strings.EqualFold(t.Hex(), token.Address) {
				reserves[i] = poolTokens.Balances[j].String()
				break
			}
		}

		if reserves[i] == emptyString {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
			}).Errorf("can not get reserve for pool")
			return entity.Pool{}, fmt.Errorf("can not get reserve for pool %v", p.Address)
		}
	}

	var extra string

	stringRateProviders := make([]string, len(rateProviders))
	for i, rateProvider := range rateProviders {
		stringRateProviders[i] = strings.ToLower(rateProvider.Hex())
	}

	if DexType(p.Type) == DexTypeBalancerComposableStable {
		extraBytes, err := json.Marshal(Extra{
			AmplificationParameter:              amplificationParameter,
			ScalingFactors:                      scalingFactors,
			BptIndex:                            bptIndex,
			LastJoinExit:                        &lastJoinExit,
			RateProviders:                       stringRateProviders,
			TokensExemptFromYieldProtocolFee:    tokensExemptFromYieldProtocolFee,
			TokenRateCaches:                     tokenRateCaches,
			ProtocolFeePercentageCacheSwapType:  protocolFeePercentageCacheSwapType,
			ProtocolFeePercentageCacheYieldType: protocolFeePercentageCacheYieldType,
		})
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
				"error":       err,
			}).Errorf("failed to marshal pool extra")
			return entity.Pool{}, err
		}

		extra = string(extraBytes)
		p.TotalSupply = totalSupply.String()
	}

	p.Extra = extra
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves

	logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	}).Infof("[Balancer-Composable-Stable] Finish getting new state of pool")

	return p, nil
}
