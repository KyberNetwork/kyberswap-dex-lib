package brownfiv3

import (
	"context"
	"encoding/hex"
	"math/big"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	pythClients  []*resty.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	pythCfg := config.Pyth
	if len(pythCfg.Urls) == 0 {
		pythCfg.Urls = []string{pythDefaultUrl}
	}
	if pythCfg.Timeout == 0 {
		pythCfg.Timeout = 10 * time.Second
	}
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		pythClients: lo.Map(pythCfg.Urls, func(url string, _ int) *resty.Client {
			pythCfg.BaseUrl = url
			return pythCfg.NewRestyClient()
		}),
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	if len(p.Tokens) != 2 {
		return p, ErrInvalidToken
	}
	startTime := time.Now()
	l := log.Ctx(ctx).With().Str("dex", DexType).Str("pool", p.Address).Logger()
	l.Info().Msg("Started getting new pool state")

	// ── Static data (hourly) ───────────────────────────────────────────────
	var staticExtra StaticExtra
	_ = json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	if time.Since(time.Unix(staticExtra.LastUpdated, 0)) > ttlStatic {
		var priceOracle, pairConfigAddr common.Address
		if _, err := d.ethrpcClient.NewRequest().SetContext(ctx).
			AddCall(&ethrpc.Call{
				ABI: brownFiV3FactoryABI, Target: d.config.FactoryAddress,
				Method: factoryMethodPriceFeedIds,
				Params: []any{common.HexToAddress(p.Tokens[0].Address)},
			}, []any{&staticExtra.PriceFeedIds[0]}).
			AddCall(&ethrpc.Call{
				ABI: brownFiV3FactoryABI, Target: d.config.FactoryAddress,
				Method: factoryMethodPriceFeedIds,
				Params: []any{common.HexToAddress(p.Tokens[1].Address)},
			}, []any{&staticExtra.PriceFeedIds[1]}).
			AddCall(&ethrpc.Call{
				ABI: brownFiV3FactoryABI, Target: d.config.FactoryAddress,
				Method: factoryMethodPriceOracle,
			}, []any{&priceOracle}).
			AddCall(&ethrpc.Call{
				ABI: brownFiV3FactoryABI, Target: d.config.FactoryAddress,
				Method: factoryMethodPairConfig,
			}, []any{&pairConfigAddr}).
			AddCall(&ethrpc.Call{
				ABI: brownFiV3PairABI, Target: p.Address,
				Method: pairMethodQuoteTokenIndex,
			}, []any{&staticExtra.QuoteTokenIndex}).
			Aggregate(); err != nil {
			return p, errors.WithMessage(err, "fail to fetch static data")
		}
		staticExtra.PriceOracle = hexutil.Encode(priceOracle[:])
		staticExtra.PairConfig = hexutil.Encode(pairConfigAddr[:])
		staticExtra.LastUpdated = startTime.Unix()
		staticExtraBytes, _ := json.Marshal(staticExtra)
		p.StaticExtra = string(staticExtraBytes)
	}

	var extra Extra
	_ = json.Unmarshal([]byte(p.Extra), &extra)

	// ── Async Pyth fetch ───────────────────────────────────────────────────
	pythUpdateDataCh := lo.Async(func() *PythUpdateData {
		if startTime.Sub(time.Unix(extra.PythTimestamp, 0)) < 5*time.Second {
			return nil
		}
		permu := rand.Perm(len(d.pythClients))[:min(2, len(d.pythClients))]
		pythUpdateDataCh := make(chan *PythUpdateData)
		wg := &sync.WaitGroup{}
		ctx, cancel := context.WithCancelCause(ctx)
		for _, i := range permu {
			wg.Go(func() {
				var pythUpdateData PythUpdateData
				if resp, err := d.pythClients[i].R().SetContext(ctx).
					SetQueryString("ids[]=" + hexutil.Encode(staticExtra.PriceFeedIds[0][:]) +
						"&ids[]=" + hexutil.Encode(staticExtra.PriceFeedIds[1][:])).
					SetResult(&pythUpdateData).
					Get(""); err != nil || !resp.IsSuccess() {
					if !errors.Is(context.Cause(ctx), ErrResponseRaced) {
						l.Error().Err(err).Interface("resp", resp).
							Str("url", d.pythClients[i].BaseURL).Msg("fail to fetch price feeds")
					}
					return
				}
				for _, price := range pythUpdateData.Parsed {
					if startTime.Sub(time.Unix(price.Price.PublishTime, 0)) > maxAge {
						return
					}
				}
				select {
				case pythUpdateDataCh <- &pythUpdateData:
					cancel(ErrResponseRaced)
				case <-ctx.Done():
				}
			})
		}
		go func() {
			wg.Wait()
			cancel(ErrFailToFetchPriceFeeds)
		}()
		select {
		case data := <-pythUpdateDataCh:
			return data
		case <-ctx.Done():
			return nil
		}
	})

	// ── Batch A: reserves, full pairConfig, ammPrice, updateFee, router balance, pause state ─
	var reserveData GetReservesResult
	var configResult PairConfigResult
	var ammPrice, updateFee, routerBalance *big.Int
	var isPaused bool
	resp, err := d.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI: brownFiV3PairABI, Target: p.Address,
			Method: pairMethodGetReserves,
		}, []any{&reserveData}).
		AddCall(&ethrpc.Call{
			ABI: brownFiV3PairConfigABI, Target: staticExtra.PairConfig,
			Method: pairConfigMethodGetConfig,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&struct{*PairConfigResult}{&configResult}}).
		AddCall(&ethrpc.Call{
			ABI: brownFiV3FactoryABI, Target: d.config.FactoryAddress,
			Method: factoryMethodGetAmmPrice,
			Params: []any{common.HexToAddress(p.Address)},
		}, []any{&ammPrice}).
		AddCall(&ethrpc.Call{
			ABI: brownFiV3OracleABI, Target: staticExtra.PriceOracle,
			Method: oracleMethodGetUpdateFee,
			Params: []any{[][]byte{extra.PriceUpdateData}},
		}, []any{&updateFee}).
		AddCall(&ethrpc.Call{
			ABI: abi.Multicall3ABI, Target: d.config.Multicall3,
			Method: abi.Multicall3GetEthBalance,
			Params: []any{Router[d.config.ChainID]},
		}, []any{&routerBalance}).
		AddCall(&ethrpc.Call{
			ABI: brownFiV3FactoryABI, Target: d.config.FactoryAddress,
			Method: factoryMethodIsPaused,
		}, []any{&isPaused}).
		TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

	// AMM parameters
	if extra.KB == nil {
		extra.KB = new(uint256.Int)
	}
	if extra.KQ == nil {
		extra.KQ = new(uint256.Int)
	}
	extra.KB.SetFromBig(configResult.KB)
	extra.KQ.SetFromBig(configResult.KQ)
	extra.Fee = configResult.Fee
	extra.Gamma = configResult.Gamma

	// Spread / skew parameters (full pairConfig)
	extra.Lambda = configResult.Lambda
	extra.SSell = configResult.SSell
	extra.SBuy = configResult.SBuy
	extra.FixS = configResult.FixS
	extra.Compress = configResult.Compress
	extra.SBound = configResult.SBound
	extra.PythWeight = configResult.PythWeight
	extra.DisThreshold = configResult.DisThreshold

	// AMM price for off-chain getSwapPrices
	if extra.AmmPrice == nil {
		extra.AmmPrice = new(uint256.Int)
	}
	if ammPrice != nil {
		extra.AmmPrice.SetFromBig(ammPrice)
	} else {
		extra.AmmPrice.Clear()
	}

	// ── Pyth prices ────────────────────────────────────────────────────────
	// getSwapPrice always calls getPriceNoOlderThan regardless of ammPrice,
	// so we must always submit fresh PriceUpdateData to avoid on-chain revert.
	if pythUpdateData := <-pythUpdateDataCh; pythUpdateData != nil && len(pythUpdateData.Parsed) >= 2 {
		// Store raw Q64 prices for off-chain computeSwapPrices
		expo := pythUpdateData.Parsed[0].Price.Expo
		if extra.Price0 == nil {
			extra.Price0 = new(uint256.Int)
		}
		if extra.Conf0 == nil {
			extra.Conf0 = new(uint256.Int)
		}
		extra.Price0.Set(pythToQ64(pythUpdateData.Parsed[0].Price.Price, expo))
		extra.Conf0.Set(pythToQ64(pythUpdateData.Parsed[0].Price.Conf, expo))

		expo = pythUpdateData.Parsed[1].Price.Expo
		if extra.Price1 == nil {
			extra.Price1 = new(uint256.Int)
		}
		if extra.Conf1 == nil {
			extra.Conf1 = new(uint256.Int)
		}
		extra.Price1.Set(pythToQ64(pythUpdateData.Parsed[1].Price.Price, expo))
		extra.Conf1.Set(pythToQ64(pythUpdateData.Parsed[1].Price.Conf, expo))

		extra.PriceUpdateData, _ = hex.DecodeString(pythUpdateData.Binary.Data[0])
		extra.PythTimestamp = pythUpdateData.Parsed[0].Price.PublishTime
		p.Timestamp = startTime.Unix()
	} else if time.Since(time.Unix(extra.PythTimestamp, 0)) >= 5*time.Second {
		// Pyth fetch was attempted (guard did not fire) but failed — stale the pool.
		p.Timestamp = min(p.Timestamp+1, startTime.Unix())
	} else {
		// Throttled — Pyth data is still fresh; update pool timestamp normally.
		p.Timestamp = startTime.Unix()
	}

	// ── Router balance check ────────────────────────────────────────────────
	poolActive := !isPaused &&
		(routerBalance == nil || updateFee == nil || updateFee.Sign() <= 0 ||
			routerBalance.Div(routerBalance, updateFee).Cmp(bignumber.Ten) > 0)

	r0 := reserveData.Reserve0
	r1 := reserveData.Reserve1
	if r0 == nil {
		r0 = bignumber.ZeroBI
	}
	if r1 == nil {
		r1 = bignumber.ZeroBI
	}

	l.Info().Bool("is_paused", isPaused).Bool("pool_active", poolActive).
		Uint64("old_block_number", p.BlockNumber).Uint64("new_block_number", resp.BlockNumber.Uint64()).
		Int64("duration_ms", time.Since(startTime).Milliseconds()).Msg("Finished getting new pool state")

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	if poolActive {
		p.Reserves = entity.PoolReserves{r0.String(), r1.String()}
	} else {
		p.Reserves = entity.PoolReserves{"0", "0"}
	}
	p.Extra = string(extraBytes)
	p.BlockNumber = resp.BlockNumber.Uint64()
	return p, nil
}
