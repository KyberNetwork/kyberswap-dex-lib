package levelfinance

import (
	"context"
	"encoding/json"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
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
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new states of pool", p.Type)

	var (
		oracle        common.Address
		trancheLength *big.Int
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    LiquidityPoolAbi,
		Target: p.Address,
		Method: liquidityPoolMethodOracle,
		Params: nil,
	}, []interface{}{&oracle})
	calls.AddCall(&ethrpc.Call{
		ABI:    LiquidityPoolAbi,
		Target: p.Address,
		Method: liquidityPoolMethodGetAllTranchesLength,
		Params: nil,
	}, []interface{}{&trancheLength})

	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"type":        p.Type,
			"err":         err,
		}).Errorf("failed to aggregate oracle and tranche length call")
		return entity.Pool{}, err
	}

	type FeeResponse struct {
		PositionFee             *big.Int
		LiquidationFee          *big.Int
		BaseSwapFee             *big.Int
		TaxBasisPoint           *big.Int
		StableCoinBaseSwapFee   *big.Int
		StableCoinTaxBasisPoint *big.Int
		DaoFee                  *big.Int
	}

	var (
		totalWeight, virtualPoolValue *big.Int
		fee                           FeeResponse
		isStableCoinList              = make([]bool, len(p.Tokens))
		targetWeightList              = make([]*big.Int, len(p.Tokens))
		totalRiskFactorList           = make([]*big.Int, len(p.Tokens))
		minPriceList                  = make([]*big.Int, len(p.Tokens))
		maxPriceList                  = make([]*big.Int, len(p.Tokens))
	)

	calls = d.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    LiquidityPoolAbi,
		Target: p.Address,
		Method: liquidityPoolMethodTotalWeight,
		Params: nil,
	}, []interface{}{&totalWeight})
	calls.AddCall(&ethrpc.Call{
		ABI:    LiquidityPoolAbi,
		Target: p.Address,
		Method: liquidityPoolMethodVirtualPoolValue,
		Params: nil,
	}, []interface{}{&virtualPoolValue})
	calls.AddCall(&ethrpc.Call{
		ABI:    LiquidityPoolAbi,
		Target: p.Address,
		Method: liquidityPoolMethodFee,
		Params: nil,
	}, []interface{}{&fee})
	for i, tokenAddress := range p.Tokens {
		calls.AddCall(&ethrpc.Call{
			ABI:    LiquidityPoolAbi,
			Target: p.Address,
			Method: liquidityPoolMethodIsStableCoin,
			Params: []interface{}{common.HexToAddress(tokenAddress.Address)},
		}, []interface{}{&isStableCoinList[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    LiquidityPoolAbi,
			Target: p.Address,
			Method: liquidityPoolMethodTargetWeights,
			Params: []interface{}{common.HexToAddress(tokenAddress.Address)},
		}, []interface{}{&targetWeightList[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    LiquidityPoolAbi,
			Target: p.Address,
			Method: liquidityPoolMethodTotalRiskFactor,
			Params: []interface{}{common.HexToAddress(tokenAddress.Address)},
		}, []interface{}{&totalRiskFactorList[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    LevelOracleABI,
			Target: oracle.Hex(),
			Method: oracleMethodGetPrice,
			Params: []interface{}{common.HexToAddress(tokenAddress.Address), true},
		}, []interface{}{&maxPriceList[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    LevelOracleABI,
			Target: oracle.Hex(),
			Method: oracleMethodGetPrice,
			Params: []interface{}{common.HexToAddress(tokenAddress.Address), false},
		}, []interface{}{&minPriceList[i]})
	}

	var tranches = make([]common.Address, 0)
	if d.config.ChainID == int(valueobject.ChainIDBSC) {
		for i := 0; i < int(trancheLength.Int64()); i++ {
			tranches = append(tranches, common.Address{})
			calls.AddCall(&ethrpc.Call{
				ABI:    LiquidityPoolAbi,
				Target: p.Address,
				Method: liquidityPoolMethodAllTranches,
				Params: []interface{}{big.NewInt(int64(i))},
			}, []interface{}{&tranches[i]})
		}
	} else {
		calls.AddCall(&ethrpc.Call{
			ABI:    LiquidityPoolAbi,
			Target: p.Address,
			Method: liquidityPoolMethodGetAllTranches,
			Params: nil,
		}, []interface{}{&tranches})
	}

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"type":        p.Type,
			"err":         err,
		}).Errorf("failed to aggregate call")
		return entity.Pool{}, err
	}

	type TrancheAssetsResponse struct {
		PoolAmount      *big.Int
		ReservedAmount  *big.Int
		GuaranteedValue *big.Int
		TotalShortSize  *big.Int
	}
	var (
		trancheAssets = make([][]TrancheAssetsResponse, len(p.Tokens))
		riskFactors   = make([][]*big.Int, len(p.Tokens))
	)

	calls = d.ethrpcClient.NewRequest().SetContext(ctx)
	for i, tokenAddress := range p.Tokens {
		trancheAssets[i] = make([]TrancheAssetsResponse, len(tranches))
		riskFactors[i] = make([]*big.Int, len(tranches))
		for j, tranche := range tranches {
			calls.AddCall(&ethrpc.Call{
				ABI:    LiquidityPoolAbi,
				Target: p.Address,
				Method: liquidityPoolMethodTrancheAssets,
				Params: []interface{}{tranche, common.HexToAddress(tokenAddress.Address)},
			}, []interface{}{&trancheAssets[i][j]})

			calls.AddCall(&ethrpc.Call{
				ABI:    LiquidityPoolAbi,
				Target: p.Address,
				Method: liquidityPoolMethodRiskFactor,
				Params: []interface{}{tranche, common.HexToAddress(tokenAddress.Address)},
			}, []interface{}{&riskFactors[i][j]})
		}
	}

	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"type":        p.Type,
			"err":         err,
		}).Errorf("failed to aggregate trancheAssets and riskFactor call")
		return entity.Pool{}, err
	}

	reserves := make(entity.PoolReserves, len(p.Tokens))
	var extra = Extra{
		Oracle:           oracle.Hex(),
		TotalWeight:      totalWeight,
		VirtualPoolValue: virtualPoolValue,

		StableCoinBaseSwapFee:   fee.StableCoinBaseSwapFee,
		StableCoinTaxBasisPoint: fee.StableCoinTaxBasisPoint,
		BaseSwapFee:             fee.BaseSwapFee,
		TaxBasisPoint:           fee.TaxBasisPoint,
		DaoFee:                  fee.DaoFee,

		TokenInfos: make(map[string]*TokenInfo),
	}

	for i, token := range p.Tokens {
		trancheAssetsMap := make(map[string]*AssetInfo, len(tranches))
		riskFactorMap := make(map[string]*big.Int, len(tranches))
		for j, trancheAddress := range tranches {
			trancheAssetsMap[trancheAddress.Hex()] = &AssetInfo{
				PoolAmount:    trancheAssets[i][j].PoolAmount,
				ReserveAmount: trancheAssets[i][j].ReservedAmount,
			}
			riskFactorMap[trancheAddress.Hex()] = riskFactors[i][j]
		}
		extra.TokenInfos[token.Address] = &TokenInfo{
			IsStableCoin:    isStableCoinList[i],
			TargetWeight:    targetWeightList[i],
			TrancheAssets:   trancheAssetsMap,
			RiskFactor:      riskFactorMap,
			TotalRiskFactor: totalRiskFactorList[i],

			MinPrice: minPriceList[i],
			MaxPrice: maxPriceList[i],
		}
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"err":         err,
		}).Errorf("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = reserves
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"address": p.Address,
		"type":    p.Type,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}
