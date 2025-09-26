package brownfiv2

import (
	"context"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	pythClient   *resty.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	pythCfg := config.Pyth
	if pythCfg.BaseUrl == "" {
		pythCfg.BaseUrl = pythDefaultBaseUrl
	}
	if pythCfg.Timeout == 0 {
		pythCfg.Timeout = 10 * time.Second
	}
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		pythClient:   pythCfg.NewRestyClient(),
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
	if staticExtra.PriceFeedIds[0] == "" {
		var priceFeedIds [2]common.Hash
		if _, err := d.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
			ABI:    brownFiV2FactoryABI,
			Target: d.config.FactoryAddress,
			Method: factoryMethodPriceFeedIds,
			Params: []any{common.HexToAddress(p.Tokens[0].Address)},
		}, []any{&priceFeedIds[0]}).AddCall(&ethrpc.Call{
			ABI:    brownFiV2FactoryABI,
			Target: d.config.FactoryAddress,
			Method: factoryMethodPriceFeedIds,
			Params: []any{common.HexToAddress(p.Tokens[1].Address)},
		}, []any{&priceFeedIds[1]}).Aggregate(); err != nil {
			return p, errors.WithMessage(err, "fail to fetch price feed ids")
		} else {
			staticExtra.PriceFeedIds[0] = hexutil.Encode(priceFeedIds[0][:])
			staticExtra.PriceFeedIds[1] = hexutil.Encode(priceFeedIds[1][:])
			staticExtraBytes, _ := json.Marshal(staticExtra)
			p.StaticExtra = string(staticExtraBytes)
		}
	}

	pythUpdateDataCh := lo.Async(func() *PythUpdateData {
		if time.Since(time.Unix(p.Timestamp, 0)) < 12*time.Second {
			return nil // don't need to fetch this too often
		}
		var pythUpdateData PythUpdateData
		if resp, err := d.pythClient.R().SetContext(ctx).
			SetQueryString("ids[]=" + staticExtra.PriceFeedIds[0] + "&ids[]=" + staticExtra.PriceFeedIds[1]).
			SetResult(&pythUpdateData).
			Get(pythPathUpdatesPriceLatest); err != nil || !resp.IsSuccess() {
			logger.WithFields(logger.Fields{"pool_id": p.Address, "err": err, "resp": resp}).
				Error("fail to fetch price feeds")
			return nil
		} else {
			return &pythUpdateData
		}
	})

	var extra Extra
	_ = json.Unmarshal([]byte(p.Extra), &extra)
	var reserveData GetReservesResult
	var kappa *big.Int
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
	}, []any{&kappa}).TryBlockAndAggregate()
	if err != nil {
		return p, err
	}
	extra.Kappa.SetFromBig(kappa)

	if pythUpdateData := <-pythUpdateDataCh; pythUpdateData != nil {
		for i, parsed := range pythUpdateData.Parsed {
			_ = extra.OPrices[i].SetFromDecimal(parsed.Price.Price)
			extra.OPrices[i].MulDivOverflow(extra.OPrices[i], q64, big256.TenPow(-parsed.Price.Expo))
		}
		extra.PriceUpdateData, _ = hex.DecodeString(pythUpdateData.Binary.Data[0])
		p.Timestamp = time.Now().Unix()
	} else {
		p.Timestamp = min(p.Timestamp+1, time.Now().Unix()) // minimal increment for lower save priority
	}

	logger.
		WithFields(
			logger.Fields{
				"pool_id":          p.Address,
				"old_reserve":      p.Reserves,
				"new_reserve":      reserveData,
				"old_block_number": p.BlockNumber,
				"new_block_number": resp.BlockNumber.Uint64(),
				"duration_ms":      time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pool state")

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Reserves = entity.PoolReserves{
		reserveData.Reserve0.String(),
		reserveData.Reserve1.String(),
	}
	p.Extra = string(extraBytes)
	p.BlockNumber = resp.BlockNumber.Uint64()

	return p, nil
}
