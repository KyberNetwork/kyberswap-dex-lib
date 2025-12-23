package nabla

import (
	"context"
	"encoding/hex"
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
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	brownfiv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/brownfi/v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	pythClient   *resty.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	pythCfg := config.Pyth
	if len(pythCfg.BaseUrl) == 0 {
		pythCfg.URL = brownfiv2.PythDefaultUrl
	}
	if pythCfg.Timeout == 0 {
		pythCfg.Timeout = 10 * time.Second
	}
	pythCfg.BaseUrl = pythCfg.URL

	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		pythClient:   pythCfg.NewRestyClient(),
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

	router := p.Address
	pythAdapterV2 := t.config.PythAdapterV2

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
		priceFeedIs := make([][32]byte, len(assets))

		req := t.ethrpcClient.R().SetContext(ctx)
		for i, asset := range assets {
			req.AddCall(&ethrpc.Call{
				ABI:    routerABI,
				Target: router,
				Method: "poolByAsset",
				Params: []any{asset},
			}, []any{&poolByAssets[i]}).AddCall(&ethrpc.Call{
				ABI:    pythAdapterV2ABI,
				Target: pythAdapterV2,
				Method: "getPriceFeedIdByAsset",
				Params: []any{asset},
			}, []any{&priceFeedIs[i]})
		}
		resp, err := req.Aggregate()
		if err != nil {
			logger.Errorf("failed to aggregate pool by asset and price feed id")
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
			betaCParams          = make([]Params, n)
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

		req = t.ethrpcClient.R().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
		for i := 0; i < n; i++ {
			req.AddCall(&ethrpc.Call{
				ABI:    curveABI,
				Target: curveAddresses[i].String(),
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

		p.Reserves = lo.Map(reserves, func(r *big.Int, _ int) string {
			return r.String()
		})

		var newExtra = Extra{}
		newExtra.PriceFeedIds = lo.Map(priceFeedIs, func(id [32]byte, _ int) string {
			return hexutil.Encode(id[:])
		})
		newExtra.PoolByAssets = poolByAssets
		newExtra.Pools = make(map[common.Address]NablaPool)
		for i, poolByAsset := range poolByAssets {
			var price *int256.Int
			_, exists := extra.Pools[poolByAsset]
			if exists {
				price = extra.Pools[poolByAsset].State.Price
			}

			newExtra.Pools[poolByAsset] = NablaPool{
				Meta: NablaPoolMeta{
					CurveBeta:   int256.MustFromBig(betaCParams[i].Beta),
					CurveC:      int256.MustFromBig(betaCParams[i].C),
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
		extra = newExtra

		p.Timestamp = time.Now().Unix()

		p.BlockNumber = resp.BlockNumber.Uint64()

		logger.Infof("finished refreshing pool %v after asset changes", p.Address)
	} else if len(params.Logs) > 0 {
		t.handleEvents(extra.Pools, params.Logs, p.BlockNumber)

		p.BlockNumber = eth.GetLatestBlockNumberFromLogs(params.Logs)

		p.Timestamp = time.Now().Unix()
	}

	queryString := lo.Reduce(extra.PriceFeedIds, func(acc string, feedId string, _ int) string {
		if acc != "" {
			acc += "&"
		}
		return acc + "ids[]=" + feedId
	}, "")
	var priceUpdateData PriceUpdateData
	if resp, err := t.pythClient.R().SetContext(ctx).
		SetQueryString(queryString).
		SetResult(&priceUpdateData).
		Get(""); err != nil || !resp.IsSuccess() {
		logger.WithFields(logger.Fields{"pool_id": p.Address, "err": err, "resp": resp}).
			Errorf("failed to fetch price feed data from antenna")
		return p, err
	}
	extra.PriceFeedData, _ = hex.DecodeString(priceUpdateData.Binary.Data[0])

	priceFeedIdxMap := lo.SliceToMap(extra.PriceFeedIds, func(feedId string) (string, int) {
		return feedId, lo.IndexOf(extra.PriceFeedIds, feedId)
	})

	for _, parsed := range priceUpdateData.Parsed {
		parsedId := "0x" + parsed.Id
		if idx, ok := priceFeedIdxMap[parsedId]; ok && idx < len(extra.PoolByAssets) {
			poolAddr := extra.PoolByAssets[idx]
			if swapPool, exists := extra.Pools[poolAddr]; exists {
				swapPool.State.Price = int256.MustFromDec(parsed.Price.Price)
				extra.Pools[poolAddr] = swapPool
			}
		}
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)

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
