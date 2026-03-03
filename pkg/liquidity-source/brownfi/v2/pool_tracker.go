package brownfiv2

import (
	"context"
	"encoding/hex"
	"math/big"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	pythClients  []*resty.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
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
	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	var staticExtra StaticExtra
	_ = json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	if time.Since(time.Unix(staticExtra.LastUpdated, 0)) > ttlStatic {
		var priceOracle common.Address
		if _, err := d.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
			ABI:    brownFiV2FactoryABI,
			Target: d.config.FactoryAddress,
			Method: factoryMethodPriceFeedIds,
			Params: []any{common.HexToAddress(p.Tokens[0].Address)},
		}, []any{&staticExtra.PriceFeedIds[0]}).AddCall(&ethrpc.Call{
			ABI:    brownFiV2FactoryABI,
			Target: d.config.FactoryAddress,
			Method: factoryMethodPriceFeedIds,
			Params: []any{common.HexToAddress(p.Tokens[1].Address)},
		}, []any{&staticExtra.PriceFeedIds[1]}).AddCall(&ethrpc.Call{
			ABI:    brownFiV2FactoryABI,
			Target: d.config.FactoryAddress,
			Method: factoryMethodPriceOracle,
		}, []any{&priceOracle}).AddCall(&ethrpc.Call{
			ABI:    brownFiV2FactoryABI,
			Target: d.config.FactoryAddress,
			Method: factoryMethodPriceOracle,
		}, []any{&priceOracle}).Aggregate(); err != nil {
			return p, errors.WithMessage(err, "fail to fetch price feed ids")
		} else {
			staticExtra.PriceOracle = hexutil.Encode(priceOracle[:])
			staticExtra.LastUpdated = time.Now().Unix()
			staticExtraBytes, _ := json.Marshal(staticExtra)
			p.StaticExtra = string(staticExtraBytes)
		}
	}

	pythUpdateDataCh := lo.Async(func() *PythUpdateData {
		now := time.Now()
		if now.Sub(time.Unix(p.Timestamp, 0)) < 5*time.Second {
			return nil // don't need to fetch this too often
		}
		permu := rand.Perm(len(d.pythClients))[:min(2, len(d.pythClients))]
		pythUpdateDataCh := make(chan *PythUpdateData)
		wg := &sync.WaitGroup{}
		ctx, cancel := context.WithCancelCause(ctx)
		for _, i := range permu { // do response racing amongst different urls
			wg.Go(func() {
				var pythUpdateData PythUpdateData
				if resp, err := d.pythClients[i].R().SetContext(ctx).
					SetQueryString("ids[]=" + hexutil.Encode(staticExtra.PriceFeedIds[0][:]) +
						"&ids[]=" + hexutil.Encode(staticExtra.PriceFeedIds[1][:])).
					SetResult(&pythUpdateData).
					Get(""); err != nil || !resp.IsSuccess() {
					if !errors.Is(context.Cause(ctx), ErrResponseRaced) {
						logger.WithFields(logger.Fields{"pool_id": p.Address, "err": err, "resp": resp,
							"url": d.pythClients[i].BaseURL}).Error("fail to fetch price feeds")
					}
					return
				}
				for _, price := range pythUpdateData.Parsed {
					if now.Sub(time.Unix(price.Price.PublishTime, 0)) > maxAge {
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
		case pythUpdateData := <-pythUpdateDataCh:
			return pythUpdateData
		case <-ctx.Done():
			return nil
		}
	})

	var extra Extra
	_ = json.Unmarshal([]byte(p.Extra), &extra)
	var reserveData GetReservesResult
	var kappa, updateFee, routerBalance *big.Int
	var brownfiPrices [2]*big.Int
	var pythPrices [2]PriceResult
	resp, err := d.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    brownFiV2PairABI,
		Target: p.Address,
		Method: pairMethodGetReserves,
	}, []any{&reserveData}).AddCall(&ethrpc.Call{
		ABI:    brownFiV2PairABI,
		Target: p.Address,
		Method: pairMethodFee,
	}, []any{&extra.Fee}).AddCall(&ethrpc.Call{
		ABI:    brownFiV2PairABI,
		Target: p.Address,
		Method: pairMethodLambda,
	}, []any{&extra.Lambda}).AddCall(&ethrpc.Call{
		ABI:    brownFiV2PairABI,
		Target: p.Address,
		Method: pairMethodKappa,
	}, []any{&kappa}).AddCall(&ethrpc.Call{
		ABI:    brownFiV2OracleABI,
		Target: staticExtra.PriceOracle,
		Method: oracleMethodGetPrice,
		Params: []any{staticExtra.PriceFeedIds[0], dummyMaxAge},
	}, []any{&brownfiPrices[0]}).AddCall(&ethrpc.Call{
		ABI:    brownFiV2OracleABI,
		Target: staticExtra.PriceOracle,
		Method: oracleMethodGetPrice,
		Params: []any{staticExtra.PriceFeedIds[1], dummyMaxAge},
	}, []any{&brownfiPrices[1]}).AddCall(&ethrpc.Call{
		ABI:    brownFiV2OracleABI,
		Target: d.config.Pyth.Address,
		Method: oracleMethodGetPriceUnsafe,
		Params: []any{staticExtra.PriceFeedIds[0]},
	}, []any{&pythPrices[0]}).AddCall(&ethrpc.Call{
		ABI:    brownFiV2OracleABI,
		Target: d.config.Pyth.Address,
		Method: oracleMethodGetPriceUnsafe,
		Params: []any{staticExtra.PriceFeedIds[1]},
	}, []any{&pythPrices[1]}).AddCall(&ethrpc.Call{
		ABI:    brownFiV2OracleABI,
		Target: d.config.Pyth.Address,
		Method: oracleMethodGetUpdateFee,
		Params: []any{[][]byte{extra.PriceUpdateData}},
	}, []any{&updateFee}).AddCall(&ethrpc.Call{
		ABI:    abi.Multicall3ABI,
		Target: d.config.Multicall3,
		Method: abi.Multicall3GetEthBalance,
		Params: []any{Router[d.config.ChainID]},
	}, []any{&routerBalance}).TryBlockAndAggregate()
	if err != nil {
		return p, err
	}
	if extra.Kappa == nil {
		extra.Kappa = new(uint256.Int)
	}
	extra.Kappa.SetFromBig(kappa)

	if pythUpdateData := <-pythUpdateDataCh; pythUpdateData != nil {
		var pythPrice, brownfiPrice uint256.Int
		for i, parsed := range pythUpdateData.Parsed {
			if extra.OPrices[i] == nil {
				extra.OPrices[i] = new(uint256.Int)
			}
			_ = extra.OPrices[i].SetFromDecimal(parsed.Price.Price)
			extra.OPrices[i].MulDivOverflow(extra.OPrices[i], q64, big256.TenPow(-parsed.Price.Expo))
			// brownfiPrice = max(pythPrice, uniV3Price)
			pythPrice.MulDivOverflow(pythPrice.SetUint64(pythPrices[i].Price), q64, big256.TenPow(-pythPrices[i].Expo))
			brownfiPrice.SetFromBig(brownfiPrices[i])
			if pythPrice.Lt(&brownfiPrice) && // brownfiPrice == uniV3Price
				brownfiPrice.Gt(extra.OPrices[i]) {
				extra.OPrices[i].Set(&brownfiPrice)
			}
		}
		extra.PriceUpdateData, _ = hex.DecodeString(pythUpdateData.Binary.Data[0])
		p.Timestamp = time.Now().Unix()
	} else {
		p.Timestamp = min(p.Timestamp+1, time.Now().Unix()) // minimal increment for lower save priority
	}

	routerEnoughBalance := routerBalance == nil || updateFee == nil || updateFee.Sign() <= 0 ||
		routerBalance.Div(routerBalance, updateFee).Cmp(bignumber.Ten) > 0
	logger.
		WithFields(
			logger.Fields{
				"pool_id":               p.Address,
				"old_reserve":           p.Reserves,
				"new_reserve":           reserveData,
				"router_enough_balance": routerEnoughBalance,
				"old_block_number":      p.BlockNumber,
				"new_block_number":      resp.BlockNumber.Uint64(),
				"duration_ms":           time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pool state")

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	if routerEnoughBalance {
		p.Reserves = entity.PoolReserves{reserveData.Reserve0.String(), reserveData.Reserve1.String()}
	} else {
		p.Reserves = entity.PoolReserves{"0", "0"}
	}
	p.Extra = string(extraBytes)
	p.BlockNumber = resp.BlockNumber.Uint64()

	return p, nil
}
