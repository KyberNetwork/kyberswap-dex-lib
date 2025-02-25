package overnightusdp

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, _ []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}

	startTime := time.Now()
	u.hasInitialized = true

	logger.WithFields(logger.Fields{"dex_id": u.config.DexID}).Debug("Start getting new pools")

	usdcAddress, usdPlusAddress, usdcDecimals, usdPlusDecimals, buyFee, reedeemFee, isPaused, blockNumber, err := u.queryPoolInfo(ctx)
	if err != nil {
		return nil, nil, err
	}

	extraBytes, err := json.Marshal(&Extra{
		IsPaused:  isPaused,
		BuyFee:    buyFee,
		RedeemFee: reedeemFee,
	})
	if err != nil {
		return nil, nil, err
	}

	staticExtraBytes, err := json.Marshal(&StaticExtra{
		AssetDecimals:   int64(usdcDecimals),
		UsdPlusDecimals: int64(usdPlusDecimals),
	})
	if err != nil {
		return nil, nil, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      DexType,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return []entity.Pool{
		{
			Address:   u.config.Exchange,
			Reserves:  []string{defaultReserves, defaultReserves},
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Tokens: []*entity.PoolToken{
				{
					Address:   strings.ToLower(usdcAddress.Hex()),
					Symbol:    "USD",
					Decimals:  usdcDecimals,
					Name:      "USD Coin",
					Swappable: true,
				},
				{
					Address:   strings.ToLower(usdPlusAddress.Hex()),
					Symbol:    "USD+",
					Decimals:  usdPlusDecimals,
					Name:      "USD+",
					Swappable: true,
				},
			},
			BlockNumber: uint64(blockNumber),
			Extra:       string(extraBytes),
			StaticExtra: string(staticExtraBytes),
		},
	}, nil, nil
}

func (u *PoolsListUpdater) queryPoolInfo(ctx context.Context) (
	usdcAddress, usdPlusAddress common.Address,
	usdcDecimals, usdPlusDecimals uint8,
	buyFee, redeemFee *big.Int,
	isPaused bool,
	blockNumber uint64,
	err error,
) {
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    exchangeABI,
		Target: u.config.Exchange,
		Method: exchangeMethodUsdc,
		Params: nil,
	}, []interface{}{&usdcAddress})
	req.AddCall(&ethrpc.Call{
		ABI:    exchangeABI,
		Target: u.config.Exchange,
		Method: exchangeMethodUsdPlus,
		Params: nil,
	}, []interface{}{&usdPlusAddress})
	req.AddCall(&ethrpc.Call{
		ABI:    exchangeABI,
		Target: u.config.Exchange,
		Method: exchangeMethodPaused,
		Params: nil,
	}, []interface{}{&isPaused})
	req.AddCall(&ethrpc.Call{
		ABI:    exchangeABI,
		Target: u.config.Exchange,
		Method: exchangeMethodBuyFee,
		Params: nil,
	}, []interface{}{&buyFee})
	req.AddCall(&ethrpc.Call{
		ABI:    exchangeABI,
		Target: u.config.Exchange,
		Method: exchangeMethodRedeemFee,
		Params: nil,
	}, []interface{}{&redeemFee})

	var resp *ethrpc.Response
	resp, err = req.Aggregate()
	if err != nil {
		return
	}

	blockNumber = resp.BlockNumber.Uint64()

	req = u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: usdcAddress.Hex(),
		Method: erc20MethodDecimals,
		Params: nil,
	}, []interface{}{&usdcDecimals})
	req.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: usdPlusAddress.Hex(),
		Method: erc20MethodDecimals,
		Params: nil,
	}, []interface{}{&usdPlusDecimals})

	_, err = req.Aggregate()

	return
}
