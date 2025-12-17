package nabla

import (
	"context"
	"math/big"
	"slices"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
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

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return p, err
	}

	router := p.Address

	var assets []common.Address
	if _, err := t.ethrpcClient.R().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    portalABI,
			Target: t.config.Portal,
			Method: "getRouterAssets",
			Params: []any{common.HexToAddress(router)},
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
				ABI:    routerABI,
				Target: router,
				Method: "poolByAsset",
				Params: []any{asset},
			}, []any{&poolByAssets[i]})
		}
		resp, err := req.Aggregate()
		if err != nil {
			logger.Errorf("failed to aggregate pool by asset")
			return p, err
		}

		n := len(poolByAssets)
		var (
			reserves             = make([]*big.Int, n)
			reservesWithSlippage = make([]*big.Int, n)
			totalLiabilities     = make([]*big.Int, n)
			swapFees             = make([]SwapFees, n)
			curveAddresses       = make([]common.Address, n)
			assetAddresses       = make([]common.Address, n)
		)

		req = t.ethrpcClient.R().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
		for i, pAddress := range poolByAssets {
			req.AddCall(&ethrpc.Call{
				ABI:    swapPoolABI,
				Target: pAddress.String(),
				Method: "reserve",
			}, []any{&reserves[i]}).AddCall(&ethrpc.Call{
				ABI:    swapPoolABI,
				Target: pAddress.String(),
				Method: "reserveWithSlippage",
			}, []any{&reservesWithSlippage[i]}).AddCall(&ethrpc.Call{
				ABI:    swapPoolABI,
				Target: pAddress.String(),
				Method: "totalLiabilities",
			}, []any{&totalLiabilities[i]}).AddCall(&ethrpc.Call{
				ABI:    swapPoolABI,
				Target: pAddress.String(),
				Method: "swapFees",
			}, []any{&swapFees[i]}).AddCall(&ethrpc.Call{
				ABI:    swapPoolABI,
				Target: pAddress.String(),
				Method: "slippageCurve",
			}, []any{&curveAddresses[i]}).AddCall(&ethrpc.Call{
				ABI:    swapPoolABI,
				Target: pAddress.String(),
				Method: "asset",
			}, []any{&assetAddresses[i]})
		}
		resp, err = req.Aggregate()
		if err != nil {
			return p, err
		}

		var (
			curveBetas = make([]*big.Int, n)
			curveCs    = make([]*big.Int, n)
		)
		for i := 0; i < n; i++ {
			betaResp, err := t.ethrpcClient.NewRequest().SetContext(ctx).GetStorageAt(
				curveAddresses[i], slot0, curveStorageABI,
			)
			if err != nil || len(betaResp) == 0 {
				return p, err
			}

			cResp, err := t.ethrpcClient.NewRequest().SetContext(ctx).GetStorageAt(
				curveAddresses[i], slot1, curveStorageABI,
			)
			if err != nil || len(cResp) == 0 {
				return p, err
			}

			curveBetas[i] = betaResp[0].(*big.Int)
			curveCs[i] = cResp[0].(*big.Int)
		}

		p.Tokens = lo.Map(assets, func(asset common.Address, _ int) *entity.PoolToken {
			return &entity.PoolToken{
				Address:   hexutil.Encode(asset[:]),
				Swappable: true,
			}
		})

		p.Reserves = lo.Map(reserves, func(r *big.Int, _ int) string {
			return r.String()
		})

		var newExtra = Extra{}
		newExtra.PoolByAssets = assets
		newExtra.Pools = make(map[common.Address]NablaPool)
		for i, poolByAsset := range poolByAssets {
			var price *int256.Int
			_, exists := extra.Pools[poolByAsset]
			if exists {
				price = extra.Pools[poolByAsset].State.Price
			}

			newExtra.Pools[poolByAsset] = NablaPool{
				Meta: NablaPoolMeta{
					CurveBeta:   int256.MustFromBig(curveBetas[i]),
					CurveC:      int256.MustFromBig(curveCs[i]),
					BackstopFee: int256.MustFromBig(swapFees[i].BackstopFee),
					ProtocolFee: int256.MustFromBig(swapFees[i].ProtocolFee),
					LpFee:       int256.MustFromBig(swapFees[i].LpFee),
				},
				State: NablaPoolState{
					Reserve:             int256.MustFromBig(reserves[i]),
					ReserveWithSlippage: int256.MustFromBig(reservesWithSlippage[i]),
					TotalLiabilities:    int256.MustFromBig(totalLiabilities[i]),
					Price:               price, // keep old price
				},
			}
		}
		newExtraBytes, err := json.Marshal(newExtra)
		if err != nil {
			return p, err
		}
		p.Extra = string(newExtraBytes)

		p.Timestamp = time.Now().Unix()

		p.BlockNumber = resp.BlockNumber.Uint64()

		logger.Infof("finished refreshing pool %v after asset changes", p.Address)
	} else if len(params.Logs) > 0 {
		t.handleEvents(extra.Pools, params.Logs, p.BlockNumber)

		newExtraBytes, err := json.Marshal(extra)
		if err != nil {
			return p, err
		}
		p.Extra = string(newExtraBytes)

		p.BlockNumber = eth.GetLatestBlockNumberFromLogs(params.Logs)

		p.Timestamp = time.Now().Unix()
	}

	return p, nil
}

func (d *PoolTracker) handleEvents(pools map[common.Address]NablaPool, events []types.Log, blockNumber uint64) {
	slices.SortFunc(events, func(l, r types.Log) int {
		if l.BlockNumber == r.BlockNumber {
			return int(l.Index - r.Index)
		}
		return int(l.BlockNumber - r.BlockNumber)
	})

	for _, event := range events {
		if event.BlockNumber < blockNumber {
			continue
		}

		if len(event.Topics) == 0 {
			continue
		}

		switch event.Topics[0] {
		case curveABI.Events["PriceFeedUpdate"].ID:
			data, err := oracleFilterer.ParsePriceFeedUpdate(event)
			if err != nil {
				continue
			}

			asset, ok := priceFeedIdToAsset[data.Id]
			if !ok {
				logger.Infof("no price feed for asset %s", data.Id)
				continue
			} else if p, exists := pools[asset]; exists {
				p.State.Price = int256.MustFromDec(kutils.Itoa(data.Price))
				pools[asset] = p
			}

		case swapPoolABI.Events["ReserveUpdated"].ID:
			data, err := swapPoolFilterer.ParseReserveUpdated(event)
			if err != nil {
				continue
			}

			p, exists := pools[event.Address]
			if !exists {
				continue
			}

			p.State.Reserve = int256.MustFromBig(data.NewReserve)
			p.State.ReserveWithSlippage = int256.MustFromBig(data.NewReserveWithSlippage)
			p.State.TotalLiabilities = int256.MustFromBig(data.NewTotalLiabilities)
			pools[event.Address] = p

		case swapPoolABI.Events["SwapFeesSet"].ID:
			data, err := swapPoolFilterer.ParseSwapFeesSet(event)
			if err != nil {
				continue
			}

			p, exists := pools[event.Address]
			if !exists {
				continue
			}

			p.Meta.LpFee = int256.MustFromBig(data.LpFee)
			p.Meta.ProtocolFee = int256.MustFromBig(data.ProtocolFee)
			p.Meta.BackstopFee = int256.MustFromBig(data.BackstopFee)
			pools[event.Address] = p

		default:
		}
	}
}
