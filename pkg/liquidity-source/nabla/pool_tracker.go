package nabla

import (
	"context"
	"math/big"
	"slices"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
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

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	params pool.GetNewPoolStateParams) (entity.Pool, error) {

	logger.Infof("getting new pool state for %v", p.Address)
	defer logger.Infof("finished getting pool state for %v", p.Address)

	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return p, err
	}

	extra.DependenciesStored = true

	var (
		assets       []common.Address
		oracle       common.Address
		currentPools = make([]common.Address, len(p.Tokens))
	)
	req := t.ethrpcClient.R().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    portalABI,
			Target: t.config.Portal,
			Method: "getRouterAssets",
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&assets}).
		AddCall(&ethrpc.Call{
			ABI:    RouterABI,
			Target: p.Address,
			Method: "oracleAdapter",
		}, []any{&oracle})
	for i, token := range p.Tokens {
		req.AddCall(&ethrpc.Call{
			ABI:    RouterABI,
			Target: p.Address,
			Method: "poolByAsset",
			Params: []any{common.HexToAddress(token.Address)},
		}, []any{&currentPools[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		logger.Errorf("failed to get router assets")
		return p, err
	}

	currentAssets := lo.Map(p.Tokens, func(t *entity.PoolToken, _ int) common.Address {
		return common.HexToAddress(t.Address)
	})

	// Swap pools can be replaced behind an unchanged asset list, and the router's oracle can change.
	removedAssets, addedAssets := lo.Difference(currentAssets, assets)
	poolsChanged := !slices.EqualFunc(extra.Pools, currentPools, func(np NablaPool, sp common.Address) bool {
		return np.Address == sp
	})
	if len(removedAssets) > 0 || len(addedAssets) > 0 || poolsChanged || oracle != extra.Oracle ||
		eth.HasRevertedLog(params.Logs) {
		logger.Infof("starting refresh of pool %v due to asset, swap pool or oracle changes", p.Address)

		poolByAssets := make([]common.Address, len(assets))
		req = t.ethrpcClient.R().SetContext(ctx)
		for i, asset := range assets {
			req.AddCall(&ethrpc.Call{
				ABI:    RouterABI,
				Target: p.Address,
				Method: "poolByAsset",
				Params: []any{asset},
			}, []any{&poolByAssets[i]})
		}
		_, err := req.Aggregate()
		if err != nil {
			logger.Errorf("failed to aggregate pool by asset")
			return p, err
		}

		// Skip unregistered assets: getRouterAssets still lists them but poolByAsset is zero.
		n := 0
		for i, sp := range poolByAssets {
			if sp != (common.Address{}) {
				assets[n], poolByAssets[n] = assets[i], sp
				n++
			}
		}
		assets, poolByAssets = assets[:n], poolByAssets[:n]

		curves := make([]common.Address, len(assets))
		req = t.ethrpcClient.R().SetContext(ctx)
		for i, sp := range poolByAssets {
			req.AddCall(&ethrpc.Call{
				ABI:    swapPoolABI,
				Target: sp.String(),
				Method: "slippageCurve",
			}, []any{&curves[i]})
		}
		resp, err := req.Aggregate()
		if err != nil {
			return p, err
		}

		betaCParams := make([]Params, len(assets))
		req = t.ethrpcClient.R().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
		for i := 0; i < len(assets); i++ {
			req.AddCall(&ethrpc.Call{
				ABI:    curveABI,
				Target: curves[i].String(),
				Method: "params",
			}, []any{&betaCParams[i]})
		}
		_, err = req.Aggregate()
		if err != nil {
			return p, err
		}

		// Older SwapPool versions lack dynamicFeeProvider; their reverts leave the zero address.
		feeProviders := make([]common.Address, len(assets))
		req = t.ethrpcClient.R().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
		for i, sp := range poolByAssets {
			req.AddCall(&ethrpc.Call{
				ABI:    swapPoolABI,
				Target: sp.String(),
				Method: "dynamicFeeProvider",
			}, []any{&feeProviders[i]})
		}
		if _, err = req.TryAggregate(); err != nil {
			return p, err
		}

		p.Tokens = lo.Map(assets, func(asset common.Address, _ int) *entity.PoolToken {
			return &entity.PoolToken{
				Address:   hexutil.Encode(asset[:]),
				Swappable: true,
			}
		})

		extra.Pools = lo.Map(poolByAssets, func(poolByAsset common.Address, i int) NablaPool {
			return NablaPool{
				Address: poolByAsset,
				Curve:   curves[i],
				Meta: NablaPoolMeta{
					CurveBeta:   int256.MustFromBig(betaCParams[i].Beta),
					CurveC:      int256.MustFromBig(betaCParams[i].C),
					FeeProvider: feeProviders[i],
				},
			}
		})

		extra.Oracle = oracle
		extra.DependenciesStored = false

		if err = t.getRPCState(ctx, &p, &extra); err != nil {
			logger.Errorf("failed to get state from RPC error %v", err)
			return p, err
		}

		logger.Infof("finished refreshing pool %v", p.Address)
	}

	if len(params.Logs) > 0 {
		t.handleEvents(ctx, &p, &extra, params.Logs)
	} else if err := t.getRPCState(ctx, &p, &extra); err != nil {
		logger.Errorf("failed to get state from RPC error %v", err)
		return p, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)

	p.Reserves = lo.Map(extra.Pools, func(np NablaPool, _ int) string { return np.State.Reserve.Dec() })

	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (t *PoolTracker) getRPCState(ctx context.Context, p *entity.Pool, extra *Extra) error {
	n := len(p.Tokens)
	var (
		reserves             = make([]*big.Int, n)
		reservesWithSlippage = make([]*big.Int, n)
		totalLiabilities     = make([]*big.Int, n)
		maxCoverageRatios    = make([]*big.Int, n)
		swapFees             = make([]SwapFees, n)
		prices               = make([]*big.Int, n)
	)
	req := t.ethrpcClient.R().SetContext(ctx)
	for i, sp := range extra.Pools {
		req.AddCall(&ethrpc.Call{
			ABI:    swapPoolABI,
			Target: sp.Address.String(),
			Method: "reserve",
		}, []any{&reserves[i]}).AddCall(&ethrpc.Call{
			ABI:    swapPoolABI,
			Target: sp.Address.String(),
			Method: "reserveWithSlippage",
		}, []any{&reservesWithSlippage[i]}).AddCall(&ethrpc.Call{
			ABI:    swapPoolABI,
			Target: sp.Address.String(),
			Method: "totalLiabilities",
		}, []any{&totalLiabilities[i]}).AddCall(&ethrpc.Call{
			ABI:    swapPoolABI,
			Target: sp.Address.String(),
			Method: "maxCoverageRatioForSwapIn",
		}, []any{&maxCoverageRatios[i]}).AddCall(&ethrpc.Call{
			ABI:    swapPoolABI,
			Target: sp.Address.String(),
			Method: "swapFees",
		}, []any{&swapFees[i]})

		if len(t.config.Whitelisted) == 0 {
			req.AddCall(&ethrpc.Call{
				ABI:    oracleABI,
				Target: hexutil.Encode(extra.Oracle[:]),
				Method: "getAssetPrice",
				Params: []any{common.HexToAddress(p.Tokens[i].Address)},
			}, []any{&prices[i]})
		}
	}
	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return err
	}

	if len(t.config.Whitelisted) > 0 {
		assets := lo.Map(p.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address })
		prices, err = t.getAssetPrices(ctx, extra.Oracle, assets, resp.BlockNumber)
		if err != nil {
			return err
		}
	}

	for i := range n {
		extra.Pools[i].Meta.BackstopFee = int256.MustFromBig(swapFees[i].BackstopFee)
		extra.Pools[i].Meta.ProtocolFee = int256.MustFromBig(swapFees[i].ProtocolFee)
		extra.Pools[i].Meta.LpFee = int256.MustFromBig(swapFees[i].LpFee)
		extra.Pools[i].Meta.MaxCoverageRatioForSwapIn = int256.MustFromBig(maxCoverageRatios[i])

		extra.Pools[i].State.Reserve = int256.MustFromBig(reserves[i])
		extra.Pools[i].State.ReserveWithSlippage = int256.MustFromBig(reservesWithSlippage[i])
		extra.Pools[i].State.TotalLiabilities = int256.MustFromBig(totalLiabilities[i])

		extra.Pools[i].State.Price = nil
		if prices[i] != nil {
			extra.Pools[i].State.Price = int256.MustFromBig(prices[i])
		}
	}
	if err = t.getSurgeFees(ctx, p, extra, resp.BlockNumber); err != nil {
		return err
	}
	extra.PriceTimestamp = time.Now().Unix()

	p.BlockNumber = resp.BlockNumber.Uint64()

	logger.Infof("finished getting state from RPC for %v", p.Address)

	return nil
}

// getSurgeFees reads DynamicFeeProvider.getFee for pools that have a provider. getFee resolves asset class
// and expiry on-chain, so it is re-read with every price refresh to drop expired fees.
func (t *PoolTracker) getSurgeFees(ctx context.Context, p *entity.Pool, extra *Extra, blockNumber *big.Int) error {
	fees := make([]SurgeFees, len(extra.Pools))
	req := t.ethrpcClient.R().SetContext(ctx).SetBlockNumber(blockNumber)
	for i, np := range extra.Pools {
		if np.Meta.FeeProvider == (common.Address{}) {
			continue
		}
		req.AddCall(&ethrpc.Call{
			ABI:    feeProvABI,
			Target: hexutil.Encode(np.Meta.FeeProvider[:]),
			Method: "getFee",
			Params: []any{common.HexToAddress(p.Tokens[i].Address)},
		}, []any{&fees[i]})
	}
	if len(req.Calls) > 0 {
		if _, err := req.Aggregate(); err != nil {
			return err
		}
	}

	for i := range extra.Pools {
		extra.Pools[i].Meta.SwapInFee, extra.Pools[i].Meta.SwapOutFee = 0, 0
		if fees[i].SwapInFee != nil {
			extra.Pools[i].Meta.SwapInFee = fees[i].SwapInFee.Uint64()
			extra.Pools[i].Meta.SwapOutFee = fees[i].SwapOutFee.Uint64()
		}
	}
	return nil
}

func (t *PoolTracker) getAssetPrices(ctx context.Context, oracle common.Address, assets []string,
	blockNumber *big.Int) ([]*big.Int, error) {
	if len(assets) == 0 {
		return nil, nil
	}

	oracleAddr := hexutil.Encode(oracle[:])

	prices := make([]*big.Int, len(assets))

	if len(t.config.Whitelisted) == 0 {
		req := t.ethrpcClient.R().SetContext(ctx).SetBlockNumber(blockNumber)
		for i, asset := range assets {
			req.AddCall(&ethrpc.Call{
				ABI:    oracleABI,
				Target: oracleAddr,
				Method: "getAssetPrice",
				Params: []any{common.HexToAddress(asset)},
			}, []any{&prices[i]})
		}
		if _, err := req.TryBlockAndAggregate(); err != nil {
			return nil, err
		}

		return prices, nil
	}

	batch := make([]rpc.BatchElem, len(assets))
	results := make([]hexutil.Bytes, len(assets))
	blockNumberHex := lo.Ternary(blockNumber.Sign() <= 0, "latest", hexutil.EncodeBig(blockNumber))

	for i, asset := range assets {
		callData, err := oracleABI.Pack("getAssetPrice", common.HexToAddress(asset))
		if err != nil {
			logger.Errorf("failed to pack get asset price")
			return nil, err
		}

		batch[i] = rpc.BatchElem{
			Method: "eth_call",
			Args: []any{
				map[string]any{
					"from": lo.Ternary(len(t.config.Whitelisted) > 0, t.config.Whitelisted, valueobject.ZeroAddress),
					"to":   oracleAddr,
					"data": hexutil.Encode(callData),
				},
				blockNumberHex,
			},
			Result: &results[i],
		}
	}
	if err := t.ethrpcClient.GetETHClient().Client().BatchCallContext(ctx, batch); err != nil {
		logger.Errorf("getAssetPrice batch call failed: %v", err)
		return nil, err
	}

	for i, elem := range batch {
		if elem.Error != nil {
			logger.Warnf("getAssetPrice(%v) failed: %v", assets[i], elem.Error)
			continue
		}
		unpacked, err := oracleABI.Unpack("getAssetPrice", results[i])
		if err != nil {
			return nil, err
		}
		prices[i] = unpacked[0].(*big.Int)
	}

	return prices, nil
}

func (t *PoolTracker) handleEvents(ctx context.Context, p *entity.Pool, extra *Extra, events []types.Log) {
	eth.SortLogs(events)

	p.BlockNumber = eth.GetBlockNumberFromLogs(events)

	shouldGetAssetPrices := false
	for _, event := range events {
		if len(event.Topics) == 0 {
			continue
		}

		address := hexutil.Encode(event.Address[:])

		// Surge fees are re-read together with prices; FeesUpdated is a global provider event.
		if event.Address == extra.Oracle || slices.ContainsFunc(extra.Pools, func(np NablaPool) bool {
			return np.Meta.FeeProvider == event.Address
		}) {
			shouldGetAssetPrices = true
		}

		switch event.Topics[0] {
		case swapPoolABI.Events["ReserveUpdated"].ID:
			data, err := swapPoolFilterer.ParseReserveUpdated(event)
			if err != nil {
				logger.Errorf("failed to parse ReserveUpdated event, error %v", err)
				continue
			}

			_, idx, _ := lo.FindIndexOf(extra.Pools, func(np NablaPool) bool {
				return hexutil.Encode(np.Address[:]) == address
			})
			if idx < 0 {
				continue
			}

			extra.Pools[idx].State.Reserve = int256.MustFromBig(data.NewReserve)
			extra.Pools[idx].State.ReserveWithSlippage = int256.MustFromBig(data.NewReserveWithSlippage)
			extra.Pools[idx].State.TotalLiabilities = int256.MustFromBig(data.NewTotalLiabilities)

		case swapPoolABI.Events["SwapFeesSet"].ID:
			data, err := swapPoolFilterer.ParseSwapFeesSet(event)
			if err != nil {
				logger.Errorf("failed to parse swap SwapFeesSet event, error %v", err)
				continue
			}

			_, idx, _ := lo.FindIndexOf(extra.Pools, func(np NablaPool) bool {
				return hexutil.Encode(np.Address[:]) == address
			})
			if idx < 0 {
				continue
			}

			extra.Pools[idx].Meta.LpFee = int256.MustFromBig(data.LpFee)
			extra.Pools[idx].Meta.ProtocolFee = int256.MustFromBig(data.ProtocolFee)
			extra.Pools[idx].Meta.BackstopFee = int256.MustFromBig(data.BackstopFee)

		case swapPoolABI.Events["DynamicFeeProviderSet"].ID:
			data, err := swapPoolABI.Unpack("DynamicFeeProviderSet", event.Data)
			if err != nil || len(data) != 2 {
				logger.Errorf("failed to parse DynamicFeeProviderSet event, error %v", err)
				continue
			}

			_, idx, _ := lo.FindIndexOf(extra.Pools, func(np NablaPool) bool {
				return hexutil.Encode(np.Address[:]) == address
			})
			if idx < 0 {
				continue
			}

			extra.Pools[idx].Meta.FeeProvider = data[1].(common.Address)
			extra.DependenciesStored = false
			shouldGetAssetPrices = true

		default:
		}
	}

	if shouldGetAssetPrices {
		assets := lo.Map(p.Tokens, func(token *entity.PoolToken, index int) string { return token.Address })
		prices, err := t.getAssetPrices(ctx, extra.Oracle, assets, big.NewInt(int64(p.BlockNumber)))
		if err != nil {
			logger.Errorf("failed to get asset prices: %v", err)
			return
		}

		for i := range extra.Pools {
			extra.Pools[i].State.Price = nil
			if prices[i] != nil {
				extra.Pools[i].State.Price = int256.MustFromBig(prices[i])
			}
		}
		if err = t.getSurgeFees(ctx, p, extra, big.NewInt(int64(p.BlockNumber))); err != nil {
			logger.Errorf("failed to get surge fees: %v", err)
			return
		}
		extra.PriceTimestamp = time.Now().Unix()
	}
}

func (t *PoolTracker) GetDependencies(_ context.Context, p entity.Pool) ([]string, bool, error) {
	var extra Extra
	err := json.Unmarshal([]byte(p.Extra), &extra)
	if err != nil {
		return nil, false, err
	}

	deps := lo.Map(extra.Pools, func(np NablaPool, _ int) string {
		return hexutil.Encode(np.Address[:])
	})
	if extra.Oracle != (common.Address{}) {
		deps = append(deps, hexutil.Encode(extra.Oracle[:]))
	}
	for _, np := range extra.Pools {
		if provider := hexutil.Encode(np.Meta.FeeProvider[:]); np.Meta.FeeProvider != (common.Address{}) &&
			!slices.Contains(deps, provider) {
			deps = append(deps, provider)
		}
	}
	return deps, extra.DependenciesStored, nil
}

func (t *PoolTracker) SetDependenciesStored(p *entity.Pool, isStored bool) error {
	var extra Extra
	err := json.Unmarshal([]byte(p.Extra), &extra)
	if err != nil {
		return err
	}
	extra.DependenciesStored = isStored
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return err
	}
	p.Extra = string(extraBytes)

	return err
}
