package v2

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	abipkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, params)
}

func (d *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	var staticExtra StaticExtra
	err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	if err != nil {
		return p, err
	}

	reserves, pairReserves, gradThreshold, tokenInfo, isXLaunch, blockNumber, err := d.getBondingData(
		ctx, p.Address, p.Tokens, staticExtra.Bonding)
	if err != nil {
		return p, err
	}

	buyTax, sellTax, kLast, antiSniperBuyTaxStartValue, taxStartTime, startTime, err := d.getTax(ctx, p.Address)
	if err != nil {
		return p, err
	}

	graduated := !valueobject.IsZeroAddress(tokenInfo.AgentToken)

	var extra = Extra{
		Trading:                    tokenInfo.Trading,
		LaunchExecuted:             tokenInfo.LaunchExecuted,
		GradThreshold:              gradThreshold,
		SellTax:                    sellTax,
		BuyTax:                     buyTax,
		ReserveA:                   pairReserves[0],
		ReserveB:                   pairReserves[1],
		KLast:                      kLast,
		AntiSniperBuyTaxStartValue: antiSniperBuyTaxStartValue,
		TaxStartTime:               taxStartTime,
		StartTime:                  startTime,
		Graduated:                  graduated,
		IsXLaunch:                  isXLaunch,
	}

	newExtra, err := json.Marshal(&extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(newExtra)
	p.Reserves = lo.Map(reserves, func(r *big.Int, _ int) string { return r.String() })
	p.BlockNumber = blockNumber.Uint64()
	p.Timestamp = time.Now().Unix()

	if graduated {
		p.Timestamp = 0
	}

	return p, nil
}

func (d *PoolTracker) getBondingData(
	ctx context.Context,
	poolAddress string,
	tokens []*entity.PoolToken,
	bondingAddress string,
) ([]*big.Int, [2]*big.Int, *big.Int, *BondingTokenInfo, bool, *big.Int, error) {
	agentToken := common.HexToAddress(tokens[0].Address)
	var (
		tokenBalances = make([]*big.Int, len(tokens))
		reserves      [2]*big.Int
		gradThreshold *big.Int
		tokenInfo     BondingTokenInfo
		isXLaunch     bool
	)

	req := d.ethrpcClient.R().SetContext(ctx)
	for i, token := range tokens {
		req.AddCall(&ethrpc.Call{
			ABI:    abipkg.Erc20ABI,
			Target: token.Address,
			Method: "balanceOf",
			Params: []any{common.HexToAddress(poolAddress)},
		}, []any{&tokenBalances[i]})
	}

	req.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: poolAddress,
		Method: "getReserves",
	}, []any{&reserves}).AddCall(&ethrpc.Call{
		ABI:    bondingABI,
		Target: bondingAddress,
		Method: "gradThreshold",
	}, []any{&gradThreshold}).AddCall(&ethrpc.Call{
		ABI:    bondingABI,
		Target: bondingAddress,
		Method: "tokenInfo",
		Params: []any{agentToken},
	}, []any{&tokenInfo}).AddCall(&ethrpc.Call{
		ABI:    bondingABI,
		Target: bondingAddress,
		Method: "isProjectXLaunch",
		Params: []any{agentToken},
	}, []any{&isXLaunch})

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, [2]*big.Int{}, nil, nil, false, nil, err
	}

	return tokenBalances, reserves, gradThreshold, &tokenInfo, isXLaunch, resp.BlockNumber, nil
}

func (d *PoolTracker) getTax(
	ctx context.Context,
	poolAddress string,
) (*big.Int, *big.Int, *big.Int, *big.Int, *big.Int, *big.Int, error) {
	var buyTax, sellTax, kLast, antiSniperBuyTaxStartValue, taxStartTime, startTime *big.Int

	req := d.ethrpcClient.R().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: d.config.Factory,
		Method: "buyTax",
	}, []any{&buyTax}).AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: d.config.Factory,
		Method: "sellTax",
	}, []any{&sellTax}).AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: poolAddress,
		Method: "kLast",
	}, []any{&kLast}).AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: d.config.Factory,
		Method: "antiSniperBuyTaxStartValue",
	}, []any{&antiSniperBuyTaxStartValue}).AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: poolAddress,
		Method: "taxStartTime",
	}, []any{&taxStartTime}).AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: poolAddress,
		Method: "startTime",
	}, []any{&startTime})
	if _, err := req.TryAggregate(); err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	// Old pair contract doesn't have taxStartTime function
	// Use startTime for backward compatibility
	if taxStartTime == nil {
		if startTime != nil {
			taxStartTime = startTime
		} else {
			taxStartTime = new(big.Int)
		}
	}

	return buyTax, sellTax, kLast, antiSniperBuyTaxStartValue, taxStartTime, startTime, nil
}
