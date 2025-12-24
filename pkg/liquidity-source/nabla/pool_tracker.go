package nabla

import (
	"context"
	"encoding/hex"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/go-resty/resty/v2"
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
	pythClient   *resty.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	pythCfg := config.Pyth
	if len(pythCfg.URL) == 0 {
		pythCfg.URL = nablaPriceAPI
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
		priceFeedIs := make([][32]byte, len(assets))

		req := t.ethrpcClient.R().SetContext(ctx)
		for i, asset := range assets {
			req.AddCall(&ethrpc.Call{
				ABI:    RouterABI,
				Target: p.Address,
				Method: "poolByAsset",
				Params: []any{asset},
			}, []any{&poolByAssets[i]}).AddCall(&ethrpc.Call{
				ABI:    pythAdapterV2ABI,
				Target: t.config.PythAdapterV2,
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

		extra.PriceFeedIds = lo.Map(priceFeedIs, func(id [32]byte, _ int) string {
			return hexutil.Encode(id[:])
		})
		extra.Pools = lo.Map(poolByAssets, func(poolByAsset common.Address, i int) NablaPool {
			return NablaPool{
				Address: poolByAsset,
				Curve:   curveAddresses[i],
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
					Price:               nil,
				},
			}
		})

		extra.DependenciesStored = false

		p.BlockNumber = resp.BlockNumber.Uint64()

		logger.Infof("finished refreshing pool %v after asset changes", p.Address)
	} else if len(params.Logs) > 0 {
		t.handleEvents(&extra, params.Logs, p.BlockNumber)

		extra.DependenciesStored = true

		p.BlockNumber = eth.GetLatestBlockNumberFromLogs(params.Logs)
	}

	extra.PriceFeedData = nil

	if !t.config.SkipPriceUpdate {
		queryString := lo.Reduce(extra.PriceFeedIds, func(acc string, feedId string, _ int) string {
			if acc != "" {
				acc += "&"
			}
			return acc + "ids[]=" + strings.TrimPrefix(feedId, "0x")
		}, "")

		var priceUpdateData PriceUpdateData
		if resp, err := t.pythClient.R().SetContext(ctx).
			SetQueryString(queryString).
			SetResult(&priceUpdateData).
			Get(""); err != nil || !resp.IsSuccess() {
			logger.WithFields(logger.Fields{
				"pool_id": p.Address, "err": err, "resp": resp,
			}).Errorf("failed to fetch price feed data from antenna")
			return p, err
		}
		extra.PriceFeedData, _ = hex.DecodeString(priceUpdateData.Binary.Data[0])

		for _, parsed := range priceUpdateData.Parsed {
			parsedId := "0x" + parsed.Id
			idx := lo.IndexOf(extra.PriceFeedIds, parsedId)
			if idx >= 0 {
				extra.Pools[idx].State.Price = new(int256.Int).SetInt64(parsed.Price.Price)
			}
		}
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

func (t *PoolTracker) handleEvents(extra *Extra, events []types.Log, blockNumber uint64) {
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

		address := hexutil.Encode(event.Address[:])

		switch event.Topics[0] {
		case curveABI.Events["PriceFeedUpdate"].ID:
			if !strings.EqualFold(address, t.config.Oracle) {
				continue
			}

			data, err := oracleFilterer.ParsePriceFeedUpdate(event)
			if err != nil {
				continue
			}

			idx := lo.IndexOf(extra.PriceFeedIds, hexutil.Encode(data.Id[:]))
			if idx < 0 {
				continue
			}

			extra.Pools[idx].State.Price = new(int256.Int).SetInt64(data.Price)

		case swapPoolABI.Events["ReserveUpdated"].ID:
			data, err := swapPoolFilterer.ParseReserveUpdated(event)
			if err != nil {
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
