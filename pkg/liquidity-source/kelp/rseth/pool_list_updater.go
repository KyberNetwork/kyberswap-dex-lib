package rseth

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/kelp/common"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

type PoolListUpdater struct {
	ethrpcClient *ethrpc.Client

	hasInitialized bool
}

func NewPoolListUpdater(
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
	return &PoolListUpdater{
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}

	startTime := time.Now()
	u.hasInitialized = true

	extra, blockNumber, err := getExtra(ctx, u.ethrpcClient)
	if err != nil {
		return nil, nil, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return nil, nil, err
	}

	tokens := []*entity.PoolToken{
		{
			Address:   strings.ToLower(common.RSETH),
			Symbol:    "rsETH",
			Decimals:  18,
			Name:      "rsETH",
			Swappable: true,
		},
	}
	tokens = append(tokens, extra.supportedTokens...)
	reserves := make([]string, len(extra.supportedTokens)+1)
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
			Address:     strings.ToLower(common.LRTDepositPool),
			Exchange:    string(valueobject.ExchangeKelpRSETH),
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      tokens,
			BlockNumber: blockNumber,
			Extra:       string(extraBytes),
		},
	}, nil, nil
}

func getExtra(ctx context.Context, ethrpcClient *ethrpc.Client) (PoolExtra, uint64, error) {
	// Step 1:
	// - call LRTConfig.getSupportedAssetList to get supported assets
	// Step 2:
	// With each asset:
	// - call LRTConfig.depositLimitByAsset to get depositLimitByAsset
	// - call LRTDepositPool.getTotalAssetDeposits to get totalDepositByAsset
	// - call LRTOracle.getAssetPrice to get priceByAsset
	// Step 3:
	// - call LRTDepositPool.minAmountToDeposit to get minAmountToDeposit
	// - call LRTOracle.rsETHPrice to get RSETHPrice
	// Step 4:
	// - combine data from 3 steps above, remember to convert from ETH to WETH and build pool. The first token have to be rsETH

	var (
		assets             []gethcommon.Address
		minAmountToDeposit *big.Int
		rsETHPrice         *big.Int
	)

	// Get supportedAssetList, minAmountToDeposit & rsETHPrice
	getPoolStateRequest := ethrpcClient.NewRequest().SetContext(ctx)

	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    common.LRTConfigABI,
		Target: common.LRTConfig,
		Method: common.LRTConfigMethodGetSupportedAssetList,
		Params: []interface{}{},
	}, []interface{}{&assets})
	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    common.LRTDepositPoolABI,
		Target: common.LRTDepositPool,
		Method: common.LRTDepositPoolMethodMinAmountToDeposit,
		Params: []interface{}{},
	}, []interface{}{&minAmountToDeposit})
	getPoolStateRequest.AddCall(&ethrpc.Call{
		ABI:    common.LRTOracleABI,
		Target: common.LRTOracle,
		Method: common.LRTOracleMethodRSETHPrice,
		Params: []interface{}{},
	}, []interface{}{&rsETHPrice})

	_, err := getPoolStateRequest.TryAggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}

	// Get depositLimitByAsset, getTotalAssetDeposits & getAssetPrice for each asset.
	// Get token's decimals as well for building pool's tokens.
	var (
		depositLimitByAsset = make([]*big.Int, len(assets))
		totalDepositByAsset = make([]*big.Int, len(assets))
		priceByAsset        = make([]*big.Int, len(assets))
		tokenDecimals       = make([]uint8, len(assets))
	)

	getAssetStateRequest := ethrpcClient.NewRequest().SetContext(ctx)
	for i, asset := range assets {
		getAssetStateRequest.AddCall(&ethrpc.Call{
			ABI:    common.LRTConfigABI,
			Target: common.LRTConfig,
			Method: common.LRTConfigMethodDepositLimitByAsset,
			Params: []interface{}{asset},
		}, []interface{}{&depositLimitByAsset[i]})

		getAssetStateRequest.AddCall(&ethrpc.Call{
			ABI:    common.LRTDepositPoolABI,
			Target: common.LRTDepositPool,
			Method: common.LRTDepositPoolMethodGetTotalAssetDeposits,
			Params: []interface{}{asset},
		}, []interface{}{&totalDepositByAsset[i]})

		getAssetStateRequest.AddCall(&ethrpc.Call{
			ABI:    common.LRTOracleABI,
			Target: common.LRTOracle,
			Method: common.LRTOracleMethodGetAssetPrice,
			Params: []interface{}{asset},
		}, []interface{}{&priceByAsset[i]})

		assetAddress := strings.ToLower(asset.String())
		if assetAddress == common.ETH {
			assetAddress = common.WETH
		}
		getAssetStateRequest.AddCall(&ethrpc.Call{
			ABI:    common.Erc20ABI,
			Target: assetAddress,
			Method: common.Erc20MethodDecimals,
			Params: nil,
		}, []interface{}{&tokenDecimals[i]})
	}

	resp, err := getAssetStateRequest.TryAggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}
	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}

	// Merge data and return poolExtra
	poolExtra := PoolExtra{
		MinAmountToDeposit:  minAmountToDeposit,
		TotalDepositByAsset: map[string]*big.Int{},
		DepositLimitByAsset: map[string]*big.Int{},
		PriceByAsset:        map[string]*big.Int{},
		RSETHPrice:          rsETHPrice,

		supportedTokens: make([]*entity.PoolToken, len(assets)),
	}
	for i, asset := range assets {
		assetAddress := strings.ToLower(asset.String())
		if assetAddress == common.ETH {
			assetAddress = common.WETH
		}
		poolExtra.TotalDepositByAsset[assetAddress] = totalDepositByAsset[i]
		poolExtra.DepositLimitByAsset[assetAddress] = depositLimitByAsset[i]
		poolExtra.PriceByAsset[assetAddress] = priceByAsset[i]
		poolExtra.supportedTokens[i] = &entity.PoolToken{
			Address:   assetAddress,
			Decimals:  tokenDecimals[i],
			Swappable: true,
		}
	}

	return poolExtra, resp.BlockNumber.Uint64(), nil
}
