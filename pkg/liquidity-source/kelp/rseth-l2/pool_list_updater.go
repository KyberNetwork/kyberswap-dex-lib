package rsethl2

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}
func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if d.hasInitialized {
		return nil, nil, nil
	}
	d.hasInitialized = true
	pool := entity.Pool{
		Address:   strings.ToLower(d.config.LRTDepositPool),
		Exchange:  d.config.DexID,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Extra:     "{}",
	}
	trackedPool, err := TrackPool(ctx, &pool, d.ethrpcClient, d.config)
	if err != nil {
		return nil, nil, err
	}
	return []entity.Pool{*trackedPool}, nil, nil
}

func TrackPool(ctx context.Context, pool *entity.Pool, rpcClient *ethrpc.Client, cfg *Config) (*entity.Pool, error) {
	var supportedTokens []common.Address
	var supportedTokenOracles []common.Address
	var wrsETH common.Address
	var rates []*big.Int
	var feeBps *big.Int
	var rseTHRate *big.Int
	nativeEnabled := true
	req := rpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    LRTDepositPoolABI,
		Target: cfg.LRTDepositPool,
		Method: "wrsETH",
		Params: nil,
	}, []any{&wrsETH})
	req.AddCall(&ethrpc.Call{
		ABI:    LRTDepositPoolABI,
		Target: cfg.LRTDepositPool,
		Method: "getSupportedTokens",
		Params: nil,
	}, []any{&supportedTokens})
	req.AddCall(&ethrpc.Call{
		ABI:    LRTDepositPoolABI,
		Target: cfg.LRTDepositPool,
		Method: "feeBps",
		Params: nil,
	}, []any{&feeBps})
	req.AddCall(&ethrpc.Call{
		ABI:    LRTDepositPoolABI,
		Target: cfg.LRTDepositPool,
		Method: "getRate",
		Params: nil,
	}, []any{&rseTHRate})
	if cfg.CheckNative {
		req.AddCall(&ethrpc.Call{
			ABI:    LRTDepositPoolABI,
			Target: cfg.LRTDepositPool,
			Method: "isEthDepositEnabled",
			Params: nil,
		}, []any{&nativeEnabled})
	}
	if len(pool.Tokens) >= 2 {
		supportedTokens := pool.Tokens[:len(pool.Tokens)-2]
		supportedTokenOracles = make([]common.Address, len(supportedTokens))
		ReqOracles(req,
			lo.Map(supportedTokens, func(token *entity.PoolToken, _ int) common.Address { return common.HexToAddress(token.Address) }),
			supportedTokenOracles, cfg,
		)
	}

	var extra Extra
	err := json.Unmarshal([]byte(pool.Extra), &extra)
	if err != nil {
		return nil, err
	}
	if len(extra.SupportedTokenOracles) > 0 {
		rates = make([]*big.Int, len(extra.SupportedTokenOracles))
		ReqRates(req, lo.Map(extra.SupportedTokenOracles, func(oracle string, _ int) common.Address { return common.HexToAddress(oracle) }), rates)
	}

	_, err = req.Aggregate()
	if err != nil {
		return nil, err
	}

	newSupportedTokens := len(pool.Tokens) < 2 || len(supportedTokens) != len(pool.Tokens)-2 || !lo.Every(
		supportedTokens,
		lo.Map(pool.Tokens[:len(pool.Tokens)-2], func(token *entity.PoolToken, _ int) common.Address { return common.HexToAddress(token.Address) }),
	)
	newOracles := len(extra.SupportedTokenOracles) != len(supportedTokenOracles) || !lo.Every(
		extra.SupportedTokenOracles,
		lo.Map(supportedTokenOracles, func(oracle common.Address, _ int) string { return strings.ToLower(oracle.Hex()) }),
	)

	if newSupportedTokens {
		// supportedTokens changed
		req := rpcClient.NewRequest().SetContext(ctx)
		supportedTokenOracles = make([]common.Address, len(supportedTokens))
		ReqOracles(req, supportedTokens, supportedTokenOracles, cfg)
		_, err = req.Aggregate()
		if err != nil {
			return nil, err
		}
		rates = make([]*big.Int, len(supportedTokenOracles))
		ReqRates(req, supportedTokenOracles, rates)
		_, err = req.Aggregate()
		if err != nil {
			return nil, err
		}
	} else if newOracles {
		// oracles changed
		req := rpcClient.NewRequest().SetContext(ctx)
		rates = make([]*big.Int, len(supportedTokenOracles))
		ReqRates(req, supportedTokenOracles, rates)
		_, err = req.Aggregate()
		if err != nil {
			return nil, err
		}
	}

	extra.SupportedTokenOracles = lo.Map(supportedTokenOracles, func(oracle common.Address, _ int) string { return strings.ToLower(oracle.Hex()) })
	extra.SupportedTokenRates = rates
	extra.RSETHRate = rseTHRate
	extra.Fee = feeBps
	extra.NativeEnabled = nativeEnabled
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, err
	}
	pool.Extra = string(extraBytes)
	pool.Tokens = lo.Map(
		append(supportedTokens, common.HexToAddress(cfg.WNative), wrsETH),
		func(token common.Address, _ int) *entity.PoolToken {
			return &entity.PoolToken{
				Address:   strings.ToLower(token.Hex()),
				Swappable: true,
			}
		})
	pool.Reserves = lo.Map(pool.Tokens, func(token *entity.PoolToken, _ int) string { return defaultReserve })
	pool.Timestamp = time.Now().Unix()
	return pool, nil
}

func ReqOracles(req *ethrpc.Request, supportedTokens []common.Address, supportedTokenOracles []common.Address, cfg *Config) {
	for i, token := range supportedTokens {
		req.AddCall(&ethrpc.Call{
			ABI:    LRTDepositPoolABI,
			Target: cfg.LRTDepositPool,
			Method: "supportedTokenOracle",
			Params: []any{token},
		}, []any{&supportedTokenOracles[i]})
	}
}

func ReqRates(req *ethrpc.Request, supportedTokenOracles []common.Address, rates []*big.Int) {
	for i, oracle := range supportedTokenOracles {
		req.AddCall(&ethrpc.Call{
			ABI:    LRTOracleABI,
			Target: oracle.Hex(),
			Method: "getRate",
			Params: nil,
		}, []any{&rates[i]})
	}
}
