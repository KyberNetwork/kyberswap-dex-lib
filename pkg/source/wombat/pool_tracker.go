package wombat

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/dsmath"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	graphqlClient *graphqlpkg.Client
}

var _ = pooltrack.RegisterFactoryCEG0(DexTypeWombat, NewPoolTracker)

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client,
) *PoolTracker {
	return &PoolTracker{
		config:        cfg,
		ethrpcClient:  ethrpcClient,
		graphqlClient: graphqlClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (d *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new states of pool", p.Type)

	var ampFactor, haircutRate, startCovRatio, endcovRatio *big.Int
	var paused bool
	var assetAddresses = make([]common.Address, len(p.Tokens))

	// We use the asset's underlying token as the key to check whether an asset is paused.
	var isAssetPaused = make([]bool, len(p.Tokens))

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    PoolV2ABI,
		Target: p.Address,
		Method: poolMethodAmpFactor,
		Params: nil,
	}, []interface{}{&ampFactor})
	calls.AddCall(&ethrpc.Call{
		ABI:    PoolV2ABI,
		Target: p.Address,
		Method: poolMethodHaircutRate,
		Params: nil,
	}, []interface{}{&haircutRate})
	calls.AddCall(&ethrpc.Call{
		ABI:    PoolV2ABI,
		Target: p.Address,
		Method: poolMethodStartCovRatio,
		Params: nil,
	}, []interface{}{&startCovRatio})
	calls.AddCall(&ethrpc.Call{
		ABI:    PoolV2ABI,
		Target: p.Address,
		Method: poolMethodEndCovRatio,
		Params: nil,
	}, []interface{}{&endcovRatio})
	calls.AddCall(&ethrpc.Call{
		ABI:    PoolV2ABI,
		Target: p.Address,
		Method: poolMethodPaused,
		Params: nil,
	}, []interface{}{&paused})
	for i, token := range p.Tokens {
		calls.AddCall(&ethrpc.Call{
			ABI:    PoolV2ABI,
			Target: p.Address,
			Method: poolMethodAddressOfAsset,
			Params: []interface{}{common.HexToAddress(token.Address)},
		}, []interface{}{&assetAddresses[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    PoolV2ABI,
			Target: p.Address,
			Method: poolMethodIsPaused,
			Params: []interface{}{common.HexToAddress(token.Address)},
		}, []interface{}{&isAssetPaused[i]})
	}
	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"type":    p.Type,
			"address": p.Address,
		}).Errorf("failed to aggregate call")
		return entity.Pool{}, err
	}

	var (
		cashes         = make([]*big.Int, len(assetAddresses))
		liabilities    = make([]*big.Int, len(assetAddresses))
		relativePrices = make([]*big.Int, len(assetAddresses))
	)

	assetCalls := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		assetCalls.SetOverrides(overrides)
	}

	for i, assetAddress := range assetAddresses {
		assetCalls.AddCall(&ethrpc.Call{
			ABI:    DynamicAssetABI,
			Target: assetAddress.Hex(),
			Method: assetMethodCash,
			Params: nil,
		}, []interface{}{&cashes[i]})
		assetCalls.AddCall(&ethrpc.Call{
			ABI:    DynamicAssetABI,
			Target: assetAddress.Hex(),
			Method: assetMethodLiability,
			Params: nil,
		}, []interface{}{&liabilities[i]})
		assetCalls.AddCall(&ethrpc.Call{
			ABI:    DynamicAssetABI,
			Target: assetAddress.Hex(),
			Method: assetMethodGetRelativePrice,
			Params: nil,
		}, []interface{}{&relativePrices[i]})
	}
	if _, err := assetCalls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"type":    p.Type,
			"address": p.Address,
		}).Errorf("failed to try aggregate call")

		return entity.Pool{}, err
	}

	var assetMap = make(map[string]Asset)
	var reserves = make([]string, len(p.Tokens))
	for i, token := range p.Tokens {
		// This pool has token in subgraph but not in contract: https://bscscan.com/address/0x2ea772346486972e7690219c190dadda40ac5da4#readProxyContract
		if eth.IsZeroAddress(assetAddresses[i]) {
			continue
		}

		assetMap[token.Address] = Asset{
			IsPause:                 isAssetPaused[i],
			Address:                 assetAddresses[i].Hex(),
			UnderlyingTokenDecimals: p.Tokens[i].Decimals,
			Cash:                    cashes[i],
			Liability:               liabilities[i],
			RelativePrice:           relativePrices[i],
		}
		if cashes[i] != nil {
			underlyingReserves := dsmath.FromWAD(cashes[i], p.Tokens[i].Decimals)
			reserves[i] = underlyingReserves.String()
		}
	}

	extraByte, err := json.Marshal(Extra{
		Paused:             paused,
		HaircutRate:        haircutRate,
		AmpFactor:          ampFactor,
		StartCovRatio:      startCovRatio,
		EndCovRatio:        endcovRatio,
		AssetMap:           assetMap,
		DependenciesStored: true,
	})
	if err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"type":    p.Type,
			"error":   err,
		}).Errorf("failed to marshal extra data")

		return entity.Pool{}, err
	}

	p.Reserves = reserves
	p.Extra = string(extraByte)
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"address": p.Address,
		"type":    p.Type,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}

func (d *PoolTracker) GetDependencies(ctx context.Context, p entity.Pool) ([]string, bool, error) {
	var extra Extra
	err := json.Unmarshal([]byte(p.Extra), &extra)
	if err != nil {
		return nil, false, err
	}

	return lo.MapToSlice(extra.AssetMap, func(_ string, asset Asset) string {
		return asset.Address
	}), extra.DependenciesStored, nil
}
