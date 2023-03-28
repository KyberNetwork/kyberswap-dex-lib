package dodo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/machinebox/graphql"
	cmap "github.com/orcaman/concurrent-map"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/pkg/logger"

	"context"

	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/core"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/uniswap"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
)

type Dodo struct {
	scanDexCfg  *config.ScanDex
	scanService *service.ScanService
	properties  Properties
}

const dodov1Pool = "CLASSICAL"
const vendingMachinePool = "DVM"
const stablePool = "DSP"
const privatePool = "DPP"

var blackList = cmap.New()
var isFirstTime = true

func New(scanDexCfg *config.ScanDex, scanService *service.ScanService) (core.IScanDex, error) {
	properties, err := NewProperties(scanDexCfg.Properties)

	if err != nil {
		return nil, err
	}
	return &Dodo{
		scanDexCfg:  scanDexCfg,
		scanService: scanService,
		properties:  properties,
	}, nil
}

func (d *Dodo) InitPool(ctx context.Context) error {
	return nil
}

func (d *Dodo) UpdateNewPools(ctx context.Context) {
	for {
		err := d.updateNewPoolsFunc(ctx)
		if err != nil {
			logger.Errorf("can not update new pool %v", err)
		}

		time.Sleep(time.Duration(d.properties.NewPoolJobIntervalSec) * time.Second)
	}
}

func (d *Dodo) UpdateReserves(ctx context.Context) {
	d.initBlackList()

	uniswap.UpdateReserveJob(
		ctx,
		d.scanDexCfg,
		d.scanService,
		d.updateReservesFunc,
		d.properties.ReserveJobInterval,
		d.properties.UpdateReserveBulk,
		d.properties.ConcurrentBatches,
	)
}

func (d *Dodo) UpdateTotalSupply(ctx context.Context) {
	for {
		d.updateTotalSupplyFunc(
			ctx, d.scanDexCfg, d.scanService, d.properties.TotalSupplyJobIntervalSec, d.properties.UpdateReserveBulk,
		)
		time.Sleep(time.Duration(d.properties.TotalSupplyJobIntervalSec) * time.Second)
	}
}

func (d *Dodo) getPairList(ctx context.Context, first, skip int, lastCreatedAtTimestamp *big.Int) (
	[]SubgraphPair, error,
) {
	client := graphql.NewClient(d.properties.SubgraphAPI)
	// 'CLASSICAL', 'DVM', 'DSP', 'DPP' pools
	req := graphql.NewRequest(
		fmt.Sprintf(
			`{
		pairs(
				first: %v, 
				skip: %v, 
				where: {
						type_not_in: ["VIRTUAL"]
						createdAtTimestamp_gte: %v
				}, 
				orderBy: createdAtTimestamp, 
				orderDirection: asc
		){
			id
			baseToken {
			    id
			    name
			    symbol
			    decimals
			}
			quoteToken {
			    id
			    name
			    symbol
			    decimals
			}
			baseLpToken { # LP token
			    id
				name
				symbol
				decimals
			}
			i
			k
			mtFeeRate
			lpFeeRate
			baseReserve
			quoteReserve
			isTradeAllowed
			type
            createdAtTimestamp
		}
	}`, first, skip, lastCreatedAtTimestamp,
		),
	)
	var response struct {
		Pairs []SubgraphPair `json:"pairs"`
	}
	if err := client.Run(ctx, req, &response); err != nil {
		logger.Errorf("failed to query subgraph, err: %v", err)
		return nil, err
	}
	return response.Pairs, nil
}

func (d *Dodo) updateNewPoolsFunc(ctx context.Context) error {

	const limit = 1000
	const skip = 0
	offsetKey := utils.Join(d.scanDexCfg.Id, "offset")

	offset, err := d.scanService.GetLastDexOffset(ctx, offsetKey)
	if err != nil {
		logger.Errorf("failed to get config pair offset from database, err: %v", err)
		return err
	}

	if isFirstTime {
		offset = 0
		isFirstTime = false
	}

	lastCreatedAtTimestamp := big.NewInt(int64(offset))

	subgraphPairs, err := d.getPairList(ctx, limit, skip, lastCreatedAtTimestamp)
	if err != nil {
		return err
	}

	pairLength := len(subgraphPairs)

	logger.Infof("got %v subgraphPairs from subgraph of DODO v2", pairLength)

	for _, p := range subgraphPairs {
		if d.scanService.ExistPool(ctx, p.ID) {
			staticExtra := StaticExtra{
				PoolId:           p.ID,
				LpToken:          p.BaseLpToken.Address,
				Type:             p.Type,
				Tokens:           []string{p.BaseToken.Address, p.QuoteToken.Address},
				DodoV1SellHelper: d.properties.DodoV1SellHelper,
			}
			extraBytes, err := json.Marshal(staticExtra)
			if err != nil {
				logger.Errorf("failed to marshal static extra: %v err %v", p.ID, err)
			}

			err = d.scanService.UpdatePoolStaticExtra(ctx, p.ID, string(extraBytes))
			if err != nil {
				logger.Errorf("failed to save pool static extra: %v err %v", p.ID, err)
			}

			continue
		}

		var tokens = make([]*entity.PoolToken, 0)
		var reserves = make([]*big.Int, 0)
		var staticField = StaticExtra{
			PoolId:           p.ID,
			LpToken:          p.BaseLpToken.Address,
			Type:             p.Type,
			Tokens:           []string{p.BaseToken.Address, p.QuoteToken.Address},
			DodoV1SellHelper: d.properties.DodoV1SellHelper,
		}

		if p.BaseToken.Address != "" {
			baseTokenDecimals, err := strconv.Atoi(p.BaseToken.Decimals)
			if err != nil {
				baseTokenDecimals = 18
			}

			tokenModel := entity.PoolToken{
				Address:   p.BaseToken.Address,
				Name:      p.BaseToken.Name,
				Symbol:    p.BaseToken.Symbol,
				Decimals:  uint8(baseTokenDecimals),
				Weight:    50,
				Swappable: true,
			}

			if _, err := d.scanService.FetchOrGetToken(ctx, tokenModel.Address); err != nil {
				logger.Errorf("failed to fetch or get token %v, err: %+v", tokenModel.Address, err)
				return err
			}

			tokens = append(tokens, &tokenModel)
			reserveF, ok := new(big.Float).SetString(p.BaseReserve)
			if !ok {
				logger.Errorf(
					"failed to parse reserve from string to big.Float for token %v, reserve: %v, err: %+v",
					tokenModel.Address, p.BaseReserve, err,
				)
				reserveF = big.NewFloat(0)
			}
			reserve, _ := reserveF.Int(nil)
			reserves = append(reserves, reserve)
		}

		if p.QuoteToken.Address != "" {
			quoteTokenDecimals, err := strconv.Atoi(p.QuoteToken.Decimals)
			if err != nil {
				quoteTokenDecimals = 18
			}

			tokenModel := entity.PoolToken{
				Address:   p.QuoteToken.Address,
				Name:      p.QuoteToken.Name,
				Symbol:    p.QuoteToken.Symbol,
				Decimals:  uint8(quoteTokenDecimals),
				Weight:    50,
				Swappable: true,
			}

			if _, err := d.scanService.FetchOrGetToken(ctx, tokenModel.Address); err != nil {
				logger.Errorf("failed to fetch or get token %v, err: %+v", tokenModel.Address, err)
				return err
			}

			tokens = append(tokens, &tokenModel)

			reserveF, ok := new(big.Float).SetString(p.QuoteReserve)
			if !ok {
				logger.Errorf(
					"failed to parse reserve from string to big.Float for token %v, reserve: %v, err: %+v",
					tokenModel.Address, p.BaseReserve, err,
				)
				reserveF = big.NewFloat(0)
			}
			reserve, _ := reserveF.Int(nil)
			reserves = append(reserves, reserve)
		}

		staticBytes, _ := json.Marshal(staticField)

		createdAtTimestamp, err := strconv.Atoi(p.CreatedAtTimestamp)
		if err != nil {
			createdAtTimestamp = 0
		}

		lpFeeRate, err := strconv.ParseFloat(p.LpFeeRate, 64)
		if err != nil {
			lpFeeRate = 0
		}
		mtFeeRateBF, ok := new(big.Float).SetString(p.MtFeeRate)
		if !ok {
			mtFeeRateBF = big.NewFloat(0)
		}
		mtFeeRate, _ := mtFeeRateBF.Float64()
		swapFee := lpFeeRate + mtFeeRate

		i, ok := new(big.Int).SetString(p.I, 10)
		if !ok {
			i = constant.Zero
		}
		k, ok := new(big.Int).SetString(p.K, 10)
		if !ok {
			i = constant.Zero
		}

		extraBytes, _ := json.Marshal(
			Extra{
				I:              i,
				K:              k,
				MtFeeRate:      mtFeeRateBF,
				LpFeeRate:      big.NewFloat(lpFeeRate),
				Swappable:      p.IsTradeAllowed,
				Reserves:       reserves,
				TargetReserves: []*big.Int{constant.Zero, constant.Zero},
			},
		)

		var newPool = entity.Pool{
			Address:      p.ID,
			ReserveUsd:   0,
			AmplifiedTvl: 0,
			SwapFee:      swapFee,
			Exchange:     d.scanDexCfg.Id,
			Type:         constant.PoolTypes.Dodo,
			Timestamp:    int64(createdAtTimestamp),
			Reserves:     []string{reserves[0].String(), reserves[1].String()},
			Tokens:       tokens,
			StaticExtra:  string(staticBytes),
			Extra:        string(extraBytes),
			TotalSupply:  "",
		}

		err = d.scanService.SavePool(ctx, newPool)
		if err != nil {
			logger.Errorf("can not save pair address=%v err=%+v", p.ID, err)
			return err
		}

		err = d.scanService.SetLastDexOffset(ctx, offsetKey, p.CreatedAtTimestamp)
		if err != nil {
			logger.Errorf("can not save config pair offset to database err %v", err)
			return err
		}
	}

	return err
}

func (d *Dodo) updateReservesFunc(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {
	var dodov1Pools, dodov2Pools []entity.Pool

	for _, pool := range pools {
		var staticExtra StaticExtra
		err := json.Unmarshal([]byte(pool.StaticExtra), &staticExtra)
		if err != nil {
			continue
		}
		if staticExtra.Type == dodov1Pool {
			dodov1Pools = append(dodov1Pools, pool)
		} else {
			dodov2Pools = append(dodov2Pools, pool)
		}
	}

	updated := 0
	updated += d.updateV1ReservesFunc(ctx, scanService, dodov1Pools)
	updated += d.updateV2ReservesFunc(ctx, scanService, dodov2Pools)

	return updated
}

func (d *Dodo) updateV1ReservesFunc(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {
	var calls = make([]*repository.TryCallParams, 0)

	type TargetReserve struct {
		BaseTarget  *big.Int `json:"baseTarget"`
		QuoteTarget *big.Int `json:"quoteTarget"`
	}

	targetReserveList := make([]TargetReserve, len(pools))
	kList := make([]*big.Int, len(pools))
	rStatusList := make([]uint8, len(pools))
	iList := make([]*big.Int, len(pools))
	lpFeeRates := make([]*big.Int, len(pools))
	mtFeeRates := make([]*big.Int, len(pools))
	baseReserves := make([]*big.Int, len(pools))
	quoteReserves := make([]*big.Int, len(pools))
	tradeAllows := make([]bool, len(pools))

	for i, pool := range pools {
		calls = append(
			calls, &repository.TryCallParams{
				ABI:    abis.DodoV1,
				Target: pool.Address,
				Method: "getExpectedTarget",
				Params: nil,
				Output: &targetReserveList[i],
			},
		)
		calls = append(
			calls, &repository.TryCallParams{
				ABI:    abis.DodoV1,
				Target: pool.Address,
				Method: "_K_",
				Params: nil,
				Output: &kList[i],
			},
		)
		calls = append(
			calls, &repository.TryCallParams{
				ABI:    abis.DodoV1,
				Target: pool.Address,
				Method: "_R_STATUS_",
				Params: nil,
				Output: &rStatusList[i],
			},
		)
		calls = append(
			calls, &repository.TryCallParams{
				ABI:    abis.DodoV1,
				Target: pool.Address,
				Method: "getOraclePrice",
				Params: nil,
				Output: &iList[i],
			},
		)
		calls = append(
			calls, &repository.TryCallParams{
				ABI:    abis.DodoV1,
				Target: pool.Address,
				Method: "_LP_FEE_RATE_",
				Params: nil,
				Output: &lpFeeRates[i],
			},
		)
		calls = append(
			calls, &repository.TryCallParams{
				ABI:    abis.DodoV1,
				Target: pool.Address,
				Method: "_MT_FEE_RATE_",
				Params: nil,
				Output: &mtFeeRates[i],
			},
		)
		calls = append(
			calls, &repository.TryCallParams{
				ABI:    abis.DodoV1,
				Target: pool.Address,
				Method: "_BASE_BALANCE_",
				Params: nil,
				Output: &baseReserves[i],
			},
		)
		calls = append(
			calls, &repository.TryCallParams{
				ABI:    abis.DodoV1,
				Target: pool.Address,
				Method: "_QUOTE_BALANCE_",
				Params: nil,
				Output: &quoteReserves[i],
			},
		)
		calls = append(
			calls, &repository.TryCallParams{
				ABI:    abis.DodoV1,
				Target: pool.Address,
				Method: "_TRADE_ALLOWED_",
				Params: nil,
				Output: &tradeAllows[i],
			},
		)
	}

	if err := scanService.TryAggregate(ctx, true, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return 0
	}

	var updated = 0
	for i, pool := range pools {
		targetReserves := []*big.Int{targetReserveList[i].BaseTarget, targetReserveList[i].QuoteTarget}
		reserves := []*big.Int{baseReserves[i], quoteReserves[i]}
		rStatus := rStatusList[i]
		mtFeeRate := new(big.Float).Quo(new(big.Float).SetInt64(mtFeeRates[i].Int64()), constant.BoneFloat)
		lpFeeRate := new(big.Float).Quo(new(big.Float).SetInt64(lpFeeRates[i].Int64()), constant.BoneFloat)

		extra := Extra{
			I:              iList[i],
			K:              kList[i],
			RStatus:        int(rStatus),
			MtFeeRate:      mtFeeRate,
			LpFeeRate:      lpFeeRate,
			Swappable:      true,
			Reserves:       reserves,
			TargetReserves: targetReserves,
		}

		extraBytes, _ := json.Marshal(extra)

		err := scanService.UpdatePoolExtra(ctx, pool.Address, string(extraBytes))
		if err != nil {
			logger.Errorf("failed to save pool extra: %v err %v", pool.Address, err)
			continue
		}

		err = scanService.UpdatePoolReserve(ctx,
			pool.Address, time.Now().Unix(), []string{reserves[0].String(), reserves[1].String()},
		)
		if err != nil {
			logger.Errorf("failed to save pool: %v err %v", pool.Address, err)
			continue
		}

		updated++
	}

	return updated
}

func (d *Dodo) updateV2ReservesFunc(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {
	var calls = make([]*repository.TryCallParams, 0)
	var feeRatesCalls = make([]*repository.TryCallParams, 0)

	states := make([]PoolState, len(pools))
	lpFeeRates := make([]*big.Int, len(pools))
	feeRates := make([]FeeRate, len(pools))

	for i, pool := range pools {
		calls = append(
			calls, &repository.TryCallParams{
				ABI:    abis.DodoV2,
				Target: pool.Address,
				Method: "getPMMStateForCall",
				Params: nil,
				Output: &states[i],
			},
		)
		calls = append(
			calls, &repository.TryCallParams{
				ABI:    abis.DodoV2,
				Target: pool.Address,
				Method: "_LP_FEE_RATE_",
				Params: nil,
				Output: &lpFeeRates[i],
			},
		)
	}

	if err := scanService.TryAggregate(ctx, false, calls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		return 0
	}

	// Some DPP pools have an issue with `getUserFeeRate` function, so we need to separately call
	for i, pool := range pools {
		if banned, _ := blackList.Get(pool.Address); banned == nil {
			feeRatesCalls = append(
				feeRatesCalls, &repository.TryCallParams{
					ABI:    abis.DodoV2,
					Target: pool.Address,
					Method: "getUserFeeRate",
					Params: []interface{}{common.HexToAddress(pool.Address)},
					Output: &feeRates[i],
				},
			)
		}
	}

	if err := scanService.TryAggregate(ctx, true, feeRatesCalls); err != nil {
		logger.Errorf("failed to process multicall, err: %v", err)
		for _, call := range feeRatesCalls {
			retryCall := []*repository.TryCallParams{call}
			if err := scanService.TryAggregate(ctx, true, retryCall); err != nil {
				blackList.Set(call.Target, true)
			}
		}
		logger.Warnf("DODO blacklist length: %d", len(blackList.Keys()))
	}

	var updated = 0
	for i, pool := range pools {
		var extra Extra
		err := json.Unmarshal([]byte(pool.Extra), &extra)
		if err != nil {
			extra = Extra{
				I:              big.NewInt(0),
				K:              big.NewInt(0),
				RStatus:        0,
				MtFeeRate:      big.NewFloat(0),
				LpFeeRate:      big.NewFloat(0),
				Swappable:      true,
				Reserves:       []*big.Int{big.NewInt(0), big.NewInt(0)},
				TargetReserves: []*big.Int{big.NewInt(0), big.NewInt(0)},
			}
		}

		if states[i].B != nil && states[i].Q != nil && states[i].B0 != nil && states[i].Q0 != nil && states[i].I != nil && states[i].K != nil && states[i].R != nil {
			extra.I = states[i].I
			extra.K = states[i].K
			extra.RStatus = int(states[i].R.Int64())
			extra.Reserves = []*big.Int{states[i].B, states[i].Q}
			extra.TargetReserves = []*big.Int{states[i].B0, states[i].Q0}
		} else {
			logger.Errorf("get pool states failed, pool address: %s", pool.Address)
		}

		if feeRates[i].MtFeeRate != nil {
			mtFeeRate := new(big.Float).Quo(
				new(big.Float).SetInt64(feeRates[i].MtFeeRate.Int64()),
				constant.BoneFloat,
			)
			extra.MtFeeRate = mtFeeRate
		} else {
			if banned, _ := blackList.Get(pool.Address); banned == nil {
				logger.Errorf("get pool feeRates failed, pool address: %s", pool.Address)
			}
		}

		if lpFeeRates[i] != nil {
			lpFeeRate := new(big.Float).Quo(new(big.Float).SetInt64(lpFeeRates[i].Int64()), constant.BoneFloat)
			extra.LpFeeRate = lpFeeRate
		} else {
			logger.Errorf("get pool lpFeeRates failed, pool address: %s", pool.Address)
		}

		extraBytes, _ := json.Marshal(extra)

		extraStr := string(extraBytes)
		err = scanService.UpdatePoolExtra(ctx, pool.Address, extraStr)
		if err != nil {
			logger.Errorf("failed to save pool extra: %v err %v", pool.Address, err)
			continue
		}

		err = scanService.UpdatePoolReserve(ctx,
			pool.Address, time.Now().Unix(), []string{extra.Reserves[0].String(), extra.Reserves[1].String()},
		)
		if err != nil {
			logger.Errorf("failed to save pool: %v err %v", pool.Address, err)
			continue
		}

		updated++
	}
	return updated
}

func (d *Dodo) updateTotalSupplyFunc(
	ctx context.Context, scanDexCfg *config.ScanDex, scanService *service.ScanService, intervalSec int64, bulk int,
) {
	startTime := time.Now()
	pools := scanService.GetPoolIdsByExchange(ctx, scanDexCfg.Id)
	sum := 0

	for i := 0; i < len(pools); i += bulk {
		end := i + bulk
		if end > len(pools) {
			end = len(pools)
		}

		pools, err := scanService.GetPoolsByAddresses(ctx, pools[i:end])
		if err != nil {
			logger.Errorf(err.Error())
			continue
		}

		var calls = make([]*repository.CallParams, 0)
		var totalSupplies = make([]*big.Int, len(pools))
		for i, pool := range pools {
			lpToken := pool.GetLpToken()
			var staticExtra StaticExtra
			err := json.Unmarshal([]byte(pool.StaticExtra), &staticExtra)
			if err != nil {
				continue
			}
			if staticExtra.Type != dodov1Pool && staticExtra.Type != privatePool {
				_, err := scanService.FetchOrGetTokenType(ctx, lpToken, pool.Exchange, pool.Address)
				if err != nil {
					continue
				}
			}
			if staticExtra.Type != vendingMachinePool && staticExtra.Type != stablePool {
				continue
			}
			calls = append(
				calls, &repository.CallParams{
					ABI:    abis.ERC20,
					Target: lpToken,
					Method: "totalSupply",
					Params: nil,
					Output: &totalSupplies[i],
				},
			)
		}
		if err := scanService.MultiCall(ctx, calls); err != nil {
			logger.Errorf("failed to process multicall, err: %v", err)
			continue
		}
		var ret = 0
		for i, pool := range pools {
			err := scanService.UpdatePoolSupply(ctx, pool.Address, totalSupplies[i].String())
			if err != nil {
				logger.Errorf("failed to save pool: %v err %v", pool.Address, err)
			} else {
				ret++
			}
		}

		sum += ret
	}
	logger.Infof("update total supply %v pairs in %v", sum, time.Since(startTime))
}

func (d *Dodo) initBlackList() {
	file, err := os.Open(path.Join(d.scanService.Config().DataFolder, "./dodo/blackList.txt"))
	if err != nil {
		logger.Errorf("Initialize DODO black list failed")
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		pool := scanner.Text()
		if pool != "" {
			blackList.Set(pool, true)
		}
	}
}
