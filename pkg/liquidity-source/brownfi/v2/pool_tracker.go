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
	"github.com/holiman/uint256"
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
		pythCfg.Timeout = 3 * time.Second
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
	if p.StaticExtra != "" {
		_ = json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	}
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

	var reserveData GetReservesResult
	var fee, lambda uint64
	var kappa *big.Int
	resp, err := d.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    brownFiV2PairABI,
		Target: p.Address,
		Method: pairMethodGetReserves,
	}, []any{&reserveData}).AddCall(&ethrpc.Call{
		ABI:    brownFiV2PairABI,
		Target: p.Address,
		Method: pairMethodFee,
	}, []any{&fee}).AddCall(&ethrpc.Call{
		ABI:    brownFiV2PairABI,
		Target: p.Address,
		Method: pairMethodLambda,
	}, []any{&lambda}).AddCall(&ethrpc.Call{
		ABI:    brownFiV2PairABI,
		Target: p.Address,
		Method: pairMethodKappa,
	}, []any{&kappa}).TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

	pythUpdateData := <-pythUpdateDataCh
	if pythUpdateData == nil {
		return p, ErrFailToFetchPriceFeeds
	}
	oPrices := make([]*uint256.Int, 2)
	for i, parsed := range pythUpdateData.Parsed {
		oPrice, _ := uint256.FromDecimal(parsed.Price.Price)
		oPrices[i], _ = oPrice.MulDivOverflow(oPrice, q64, big256.TenPow(-parsed.Price.Expo))
	}
	priceUpdateData, _ := hex.DecodeString(pythUpdateData.Binary.Data[0])

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

	extra := Extra{
		Fee:             fee,
		Lambda:          lambda,
		Kappa:           uint256.MustFromBig(kappa),
		OPrices:         [2]*uint256.Int(oPrices),
		PriceUpdateData: priceUpdateData,
	}

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
	p.Timestamp = time.Now().Unix()

	return p, nil
}
