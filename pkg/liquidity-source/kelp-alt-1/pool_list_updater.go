package rsethalt1

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"

	gethcommon "github.com/ethereum/go-ethereum/common"
)

type PoolListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client

	hasInitialized bool
}

func NewPoolListUpdater(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
	return &PoolListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}

	startTime := time.Now()
	u.hasInitialized = true

	extra, blockNumber, err := getExtra(ctx, u.ethrpcClient, u.config)
	if err != nil {
		return nil, nil, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, nil, err
	}

	reserves := make([]string, len(extra.supportedTokens))
	for i := 0; i < len(reserves); i++ {
		reserves[i] = defaultReserves
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
			Address:     strings.ToLower(u.config.RsethPool),
			Exchange:    string(valueobject.ExchangeKelpRSETH),
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      extra.supportedTokens,
			BlockNumber: blockNumber,
			Extra:       string(extraBytes),
		},
	}, nil, nil
}

type LatestRoundData struct {
	RoundId         *big.Int
	Answer          *big.Int
	StartedAt       *big.Int
	UpdatedAt       *big.Int
	AnsweredInRound *big.Int
}

func getExtra(ctx context.Context, ethrpcClient *ethrpc.Client, config *Config) (PoolExtra, uint64, error) {
	wstEthSupported := false
	assetsLen := 2
	assets := []gethcommon.Address{gethcommon.HexToAddress(config.Rseth), gethcommon.HexToAddress(config.Weth)}
	if config.Wsteth != "" {
		wstEthSupported = true
		assetsLen = 3
		assets = append(assets, gethcommon.HexToAddress(config.Wsteth))
	}
	var (
		priceByAsset    = make([]*big.Int, assetsLen)
		latestRoundData LatestRoundData
		feeBps          *big.Int
	)
	tokenDecimals := []uint8{18, 18, 18}

	getPoolStateRequest := ethrpcClient.NewRequest().SetContext(ctx)

	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    RsETHPool,
		Target: config.RsethPool,
		Method: methodGetRateRSETH,
		Params: nil,
	}, []interface{}{&priceByAsset[0]})
	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    RsETHPool,
		Target: config.RsethPool,
		Method: methodGetFeeBps,
		Params: nil,
	}, []interface{}{&feeBps})
	if wstEthSupported {
		getPoolStateRequest.AddCall(&ethrpc.Call{
			ABI:    WstethETHOracle,
			Target: config.WstethOracle,
			Method: methodGetRateWRSETH,
			Params: nil,
		}, []interface{}{&latestRoundData})
	}
	resp, err := getPoolStateRequest.TryAggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}

	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	if wstEthSupported {
		priceByAsset[2] = latestRoundData.Answer
	}
	poolExtra := PoolExtra{
		PriceByAsset:    map[string]*big.Int{},
		supportedTokens: make([]*entity.PoolToken, len(assets)),
	}
	for i, asset := range assets {
		assetAddress := strings.ToLower(asset.String())
		poolExtra.PriceByAsset[assetAddress] = priceByAsset[i]
		poolExtra.supportedTokens[i] = &entity.PoolToken{
			Address:   assetAddress,
			Decimals:  tokenDecimals[i],
			Swappable: true,
		}
		poolExtra.FeeBps = feeBps
	}
	return poolExtra, resp.BlockNumber.Uint64(), nil
}
