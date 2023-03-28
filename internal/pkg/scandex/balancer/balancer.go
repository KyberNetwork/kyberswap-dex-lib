package balancer

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/machinebox/graphql"

	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/core"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/uniswap"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

type Balancer struct {
	scanDexCfg  *config.ScanDex
	scanService *service.ScanService
	properties  Properties
}

func (t *Balancer) InitPool(ctx context.Context) error {
	return nil
}

func New(scanDexCfg *config.ScanDex, scanService *service.ScanService) (core.IScanDex, error) {
	properties, err := NewProperties(scanDexCfg.Properties)

	if err != nil {
		return nil, err
	}
	return &Balancer{
		scanDexCfg:  scanDexCfg,
		scanService: scanService,
		properties:  properties,
	}, nil
}

func (t *Balancer) getPairList(ctx context.Context, first, skip int) ([]SubgraphPair, error) {
	client := graphql.NewClient(t.properties.SubgraphAPI)
	//'Weighted','Stable','MetaStable';
	req := graphql.NewRequest(fmt.Sprintf(`{
		pools(where : {totalShares_gt: 0.01, swapEnabled: true}, first: %v, skip: %v) {
			id
			address
			poolType
			swapFee
			tokens {
			  address
			  decimals	
			  weight
			}
		}
	}`, first, skip),
	)
	var response struct {
		Pairs []SubgraphPair `json:"pools"`
	}
	if err := client.Run(ctx, req, &response); err != nil {
		logger.Errorf("failed to query subgraph, err: %v", err)
		return nil, err
	}
	return response.Pairs, nil
}

func (t *Balancer) UpdateNewPools(ctx context.Context) {

	const limit = 1000
	for {
		var subgraphPairs, err = t.getPairList(ctx, limit, 0)
		if err != nil {
			continue
		}
		logger.Infof("got %v subgraphPairs from subgraph", len(subgraphPairs))

		var calls = make([]*repository.CallParams, 0)

		var vaultAddresses = make([]common.Address, len(subgraphPairs))
		for i, subgraphPair := range subgraphPairs {
			calls = append(calls, &repository.CallParams{
				ABI:    abis.BalancerPool,
				Target: subgraphPair.Address,
				Method: "getVault",
				Params: nil,
				Output: &vaultAddresses[i],
			})
		}
		if err := t.scanService.MultiCall(ctx, calls); err != nil {
			logger.Errorf("failed to process multicall, err: %v", err)
			return
		}
		for i, p := range subgraphPairs {
			if t.scanService.ExistPool(ctx, p.Address) {
				continue
			}
			var tokens = make([]*entity.PoolToken, 0)
			var reserves = make([]string, 0)
			var staticField = StaticExtra{
				VaultAddress: strings.ToLower(vaultAddresses[i].Hex()),
				PoolId:       p.ID,
			}
			for _, item := range p.Tokens {
				weight, _ := strconv.ParseFloat(item.Weight, 64)
				tokenModel := entity.PoolToken{
					Address:   item.Address,
					Weight:    uint(weight * 1e18),
					Swappable: true,
				}
				staticField.TokenDecimals = append(staticField.TokenDecimals, item.Decimals)
				if tokenModel.Weight == 0 {
					tokenModel.Weight = uint(1e18 / len(p.Tokens))
				}
				if _, err := t.scanService.FetchOrGetToken(ctx, tokenModel.Address); err != nil {
					logger.Errorf("failed to fetch or get token %va, err: %v", tokenModel.Address, err)
					return
				}
				tokens = append(tokens, &tokenModel)
				reserves = append(reserves, "0")
			}
			var swapFee, _ = strconv.ParseFloat(p.SwapFee, 64)

			staticBytes, _ := json.Marshal(staticField)
			var newPool = entity.Pool{
				Address:     p.Address,
				ReserveUsd:  0,
				SwapFee:     swapFee,
				Exchange:    t.scanDexCfg.Id,
				Timestamp:   0,
				Reserves:    reserves,
				Tokens:      tokens,
				StaticExtra: string(staticBytes),
			}
			if p.PoolType == "Weighted" {
				newPool.Type = constant.PoolTypes.BalancerWeighted
			} else if p.PoolType == "Stable" {
				newPool.Type = constant.PoolTypes.BalancerStable
			} else if p.PoolType == "MetaStable" {
				newPool.Type = constant.PoolTypes.BalancerMetaStable
			} else {
				logger.Warnf("can not handler pool type %v", p.PoolType)
				continue
			}
			t.scanService.SavePool(ctx, newPool)
		}
		time.Sleep(time.Duration(t.properties.NewPoolJobIntervalSec) * time.Second)
	}
}

func (t *Balancer) UpdateReserves(ctx context.Context) {
	f := func(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {
		var calls = make([]*repository.TryCallParams, 0)

		poolTokens := make([]PoolToken, len(pools))
		swapFees := make([]*big.Int, len(pools))
		amplificationParameters := make([]AmplificationParameter, len(pools))
		scalingFactors := make([][]*big.Int, len(pools))
		for i, pool := range pools {
			var staticExtra = StaticExtra{}
			_ = json.Unmarshal([]byte(pool.StaticExtra), &staticExtra)
			param := common.HexToHash(staticExtra.PoolId)
			calls = append(calls, &repository.TryCallParams{
				ABI:    abis.BalancerVault,
				Target: staticExtra.VaultAddress,
				Method: "getPoolTokens",
				Params: []interface{}{param},
				Output: &poolTokens[i],
			})
			calls = append(calls, &repository.TryCallParams{
				ABI:    abis.BalancerPool,
				Target: pool.Address,
				Method: "getSwapFeePercentage",
				Params: nil,
				Output: &swapFees[i],
			})
			if pool.Type == constant.PoolTypes.BalancerStable || pool.Type == constant.PoolTypes.BalancerMetaStable {
				calls = append(calls, &repository.TryCallParams{
					ABI:    abis.BalancerPool,
					Target: pool.Address,
					Method: "getAmplificationParameter",
					Params: nil,
					Output: &amplificationParameters[i],
				})
			}
			if pool.Type == constant.PoolTypes.BalancerMetaStable {
				calls = append(calls, &repository.TryCallParams{
					ABI:    abis.BalancerMetaStablePool,
					Target: pool.Address,
					Method: "getScalingFactors",
					Params: nil,
					Output: &scalingFactors[i],
				})
			}
		}
		if err := t.scanService.TryAggregate(ctx, true, calls); err != nil {
			logger.Errorf("failed to process multicall, err: %v", err)
			return 0
		}
		var ret = 0
		for i, pool := range pools {
			if swapFees[i] != nil {
				swapFee, _ := new(big.Float).Quo(new(big.Float).SetInt(swapFees[i]), constant.BoneFloat).Float64()
				if err := scanService.UpdatePoolSwapFee(ctx, pool.Address, swapFee); err != nil {
					logger.Errorf("failed to update pool swap fee, pool: %s, err: %v", pool.Address, err)
				}
			}

			reserves := make([]string, len(pool.Tokens))
			for j, token := range pool.Tokens {
				for k, p := range poolTokens[i].Tokens {
					if strings.ToLower(p.Hex()) == token.Address {
						reserves[j] = poolTokens[i].Balances[k].String()
						break
					}
					if k == len(poolTokens[i].Tokens)-1 {
						logger.Errorf("can not get reserve for pool %v", pool.Address)
						return 0
					}
				}
			}
			if pool.Type == constant.PoolTypes.BalancerStable || pool.Type == constant.PoolTypes.BalancerMetaStable {
				extraBytes, _ := json.Marshal(Extra{
					AmplificationParameter: amplificationParameters[i],
					ScalingFactors:         scalingFactors[i],
				})
				extra := string(extraBytes)
				_ = scanService.UpdatePoolExtra(ctx, pool.Address, extra)
			}
			err := scanService.UpdatePoolReserve(ctx, pool.Address, poolTokens[i].LastChangeBlock.Int64(), reserves)
			if err != nil {
				logger.Errorf("failed to save pool: %v err %v", pool.Address, err)
			} else {
				ret++
			}

		}
		return ret
	}
	uniswap.UpdateReserveJob(
		ctx,
		t.scanDexCfg,
		t.scanService,
		f,
		t.properties.ReserveJobInterval,
		t.properties.UpdateReserveBulk,
		t.properties.ConcurrentBatches,
	)
}

func (t *Balancer) UpdateTotalSupply(ctx context.Context) {
	uniswap.UpdateTotalSupplyJob(ctx,
		t.scanDexCfg,
		t.scanService,
		uniswap.UpdateTotalSupplyHandler,
		t.properties.TotalSupplyJobIntervalSec,
		t.properties.UpdateReserveBulk)
}
