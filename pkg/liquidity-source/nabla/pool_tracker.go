package nabla

import (
	"context"
	"math/big"
	"strings"
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

	var assets []common.Address
	if _, err := t.ethrpcClient.R().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    portalABI,
			Target: t.config.Portal,
			Method: "getRouterAssets",
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&assets}).
		Call(); err != nil {
		logger.Errorf("failed to get router assets")
		return p, err
	}

	currentAssets := lo.Map(p.Tokens, func(t *entity.PoolToken, _ int) common.Address {
		return common.HexToAddress(t.Address)
	})

	removedAssets, addedAssets := lo.Difference(currentAssets, assets)
	if len(removedAssets) > 0 || len(addedAssets) > 0 || eth.HasRevertedLog(params.Logs) {
		logger.Infof("starting refresh of pool %v due to asset changes", p.Address)

		poolByAssets := make([]common.Address, len(assets))
		req := t.ethrpcClient.R().SetContext(ctx)
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
		resp, err = req.Aggregate()
		if err != nil {
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
					CurveBeta: int256.MustFromBig(betaCParams[i].Beta),
					CurveC:    int256.MustFromBig(betaCParams[i].C),
				},
			}
		})

		extra.DependenciesStored = false

		if err = t.getRPCState(ctx, &p, &extra); err != nil {
			return p, nil
		}

		logger.Infof("finished refreshing pool %v after asset changes", p.Address)
	}

	if len(params.Logs) > 0 {
		t.handleEvents(ctx, &p, &extra, params.Logs)
	} else if err := t.getRPCState(ctx, &p, &extra); err != nil {
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
		swapFees             = make([]SwapFees, n)
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
			Method: "swapFees",
		}, []any{&swapFees[i]})
	}
	resp, err := req.Aggregate()
	if err != nil {
		return err
	}

	assets := lo.Map(p.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address })
	prices, err := t.getAssetPrices(ctx, assets, resp.BlockNumber)
	if err != nil {
		return err
	}

	for i := range n {
		extra.Pools[i].Meta.BackstopFee = int256.MustFromBig(swapFees[i].BackstopFee)
		extra.Pools[i].Meta.ProtocolFee = int256.MustFromBig(swapFees[i].ProtocolFee)
		extra.Pools[i].Meta.LpFee = int256.MustFromBig(swapFees[i].LpFee)

		extra.Pools[i].State.Reserve = int256.MustFromBig(reserves[i])
		extra.Pools[i].State.ReserveWithSlippage = int256.MustFromBig(reservesWithSlippage[i])
		extra.Pools[i].State.TotalLiabilities = int256.MustFromBig(totalLiabilities[i])
		extra.Pools[i].State.Price = int256.MustFromBig(prices[i])
	}

	p.BlockNumber = resp.BlockNumber.Uint64()

	logger.Infof("finished getting state from RPC for %v", p.Address)

	return nil
}

func (t *PoolTracker) getAssetPrices(ctx context.Context, assets []string, blockNumber *big.Int) ([]*big.Int, error) {
	if len(assets) == 0 {
		return nil, nil
	}

	batch := make([]rpc.BatchElem, len(assets))
	results := make([]hexutil.Bytes, len(assets))

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
					"to":   t.config.Oracle,
					"data": hexutil.Encode(callData),
				},
				lo.Ternary(blockNumber.Sign() <= 0, "latest", hexutil.EncodeBig(blockNumber)),
			},
			Result: &results[i],
		}
	}
	if err := t.ethrpcClient.GetETHClient().Client().BatchCallContext(ctx, batch); err != nil {
		logger.Errorf("getAssetPrices batch call failed: %v", err)
		return nil, err
	}

	prices := make([]*big.Int, len(assets))
	for i, elem := range batch {
		if elem.Error != nil {
			return nil, elem.Error
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

		switch event.Topics[0] {
		case oracleABI.Events["PriceFeedUpdate"].ID:
			if !strings.EqualFold(address, t.config.Oracle) {
				continue
			}

			shouldGetAssetPrices = true

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

		default:
		}
	}

	if shouldGetAssetPrices {
		assets := lo.Map(p.Tokens, func(token *entity.PoolToken, index int) string { return token.Address })
		prices, err := t.getAssetPrices(ctx, assets, big.NewInt(int64(p.BlockNumber)))
		if err != nil {
			logger.Errorf("failed to get asset prices: %v", err)
			return
		}

		for i := range extra.Pools {
			extra.Pools[i].State.Price = int256.MustFromBig(prices[i])
		}
	}
}

func (t *PoolTracker) GetDependencies(_ context.Context, p entity.Pool) ([]string, bool, error) {
	var extra Extra
	err := json.Unmarshal([]byte(p.Extra), &extra)
	if err != nil {
		return nil, false, err
	}

	return append(lo.Map(extra.Pools, func(np NablaPool, _ int) string {
		return hexutil.Encode(np.Address[:])
	}), strings.ToLower(t.config.Oracle)), extra.DependenciesStored, nil
}
