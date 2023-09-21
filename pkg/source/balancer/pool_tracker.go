package balancer

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
	}).Infof("[Balancer] Start updating state ...")

	var staticExtra = StaticExtra{}
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
		}).Errorf("failed to unmarshal pool static extra")

		return entity.Pool{}, err
	}

	poolIdParam := common.HexToHash(staticExtra.PoolId)

	var (
		poolTokens             PoolTokens
		amplificationParameter AmplificationParameter
		scalingFactors         []*big.Int
		swapFeePercentage      *big.Int
	)

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

	if DexType(p.Type) == DexTypeBalancerStable {
		calls.AddCall(&ethrpc.Call{
			ABI:    stablePoolABI,
			Target: p.Address,
			Method: poolMethodGetAmplificationParameter,
			Params: nil,
		}, []interface{}{&amplificationParameter})
	}

	if DexType(p.Type) == DexTypeBalancerMetaStable {
		calls.AddCall(&ethrpc.Call{
			ABI:    metaStablePoolABI,
			Target: p.Address,
			Method: poolMethodGetAmplificationParameter,
			Params: nil,
		}, []interface{}{&amplificationParameter})

		calls.AddCall(&ethrpc.Call{
			ABI:    metaStablePoolABI,
			Target: p.Address,
			Method: metaStablePoolMethodGetScalingFactors,
			Params: nil,
		}, []interface{}{&scalingFactors})
	}

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("[Balancer] failed to aggregate for pool data")
		return entity.Pool{}, err
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
	if DexType(p.Type) == DexTypeBalancerStable {
		extraBytes, err := json.Marshal(Extra{
			AmplificationParameter: amplificationParameter,
		})
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
				"error":       err,
			}).Errorf("failed to marshal pool extra")
			return entity.Pool{}, err
		}

		extra = string(extraBytes)
	}

	if DexType(p.Type) == DexTypeBalancerMetaStable {
		extraBytes, err := json.Marshal(Extra{
			AmplificationParameter: amplificationParameter,
			ScalingFactors:         scalingFactors,
		})
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
				"error":       err,
			}).Errorf("failed to marshal pool extra")
			return entity.Pool{}, err
		}

		extra = string(extraBytes)
	}

	p.Extra = extra
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves

	logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
	}).Infof("[Balancer] Finish getting new state of pool")

	return p, nil
}
