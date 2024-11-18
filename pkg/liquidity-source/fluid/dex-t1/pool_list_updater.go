package dexT1

import (
	"context"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config       Config
	ethrpcClient *ethrpc.Client
}

type Metadata struct {
	LastSyncPoolsLength int `json:"lastSyncPoolsLength"`
}

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       *config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.WithFields(logger.Fields{
		"dexType": DexType,
	}).Infof("Start updating pools list ...")
	defer func() {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
		}).Infof("Finish updating pools list.")
	}()

	allPools, err := u.getAllPools(ctx)
	if err != nil {
		return nil, nil, err
	}

	newMetadataBytes, err := json.Marshal(Metadata{
		LastSyncPoolsLength: len(allPools),
	})
	if err != nil {
		return nil, nil, err
	}

	var metadata Metadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, nil, err
		}
	}

	if metadata.LastSyncPoolsLength > 0 {
		// only handle new pools after last synced index
		allPools = allPools[metadata.LastSyncPoolsLength:]
	}

	pools := make([]entity.Pool, 0)

	for _, curPool := range allPools {
		token0Decimals, token1Decimals, err := u.readTokensDecimals(ctx, curPool.Token0Address, curPool.Token1Address)
		if err != nil {
			return nil, nil, err
		}

		staticExtraBytes, err := json.Marshal(&StaticExtra{
			DexReservesResolver: u.config.DexReservesResolver,
			HasNative: strings.EqualFold(curPool.Token0Address.Hex(), valueobject.EtherAddress) ||
				strings.EqualFold(curPool.Token1Address.Hex(), valueobject.EtherAddress),
		})
		if err != nil {
			return nil, nil, err
		}

		extra := PoolExtra{
			CollateralReserves: curPool.CollateralReserves,
			DebtReserves:       curPool.DebtReserves,
			DexLimits:          curPool.Limits,
		}

		extraBytes, err := json.Marshal(extra)
		if err != nil {
			logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("Error marshaling extra data")
			return nil, nil, err
		}

		pool := entity.Pool{
			Address:  curPool.PoolAddress.String(),
			Exchange: string(valueobject.ExchangeFluidDexT1),
			Type:     DexType,
			Reserves: entity.PoolReserves{
				getMaxReserves(
					token0Decimals,
					curPool.Limits.WithdrawableToken0,
					curPool.Limits.BorrowableToken0,
					curPool.CollateralReserves.Token0RealReserves,
					curPool.DebtReserves.Token0RealReserves).String(),
				getMaxReserves(
					token1Decimals,
					curPool.Limits.WithdrawableToken1,
					curPool.Limits.BorrowableToken1,
					curPool.CollateralReserves.Token1RealReserves,
					curPool.DebtReserves.Token1RealReserves).String(),
			},
			Tokens: []*entity.PoolToken{
				{
					Address:   valueobject.WrapETHLower(curPool.Token0Address.Hex(), u.config.ChainID),
					Weight:    1,
					Swappable: true,
					Decimals:  token0Decimals,
				},
				{
					Address:   valueobject.WrapETHLower(curPool.Token1Address.Hex(), u.config.ChainID),
					Weight:    1,
					Swappable: true,
					Decimals:  token1Decimals,
				},
			},
			SwapFee:     float64(curPool.Fee.Int64()) / float64(FeePercentPrecision),
			Extra:       string(extraBytes),
			StaticExtra: string(staticExtraBytes),
		}

		pools = append(pools, pool)
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getAllPools(ctx context.Context) ([]PoolWithReserves, error) {
	var pools []PoolWithReserves

	req := u.ethrpcClient.R().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    dexReservesResolverABI,
		Target: u.config.DexReservesResolver,
		Method: DRRMethodGetAllPoolsReservesAdjusted,
	}, []interface{}{&pools})

	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("Failed to get all pools reserves")
		return nil, err
	}

	return pools, nil
}

func (u *PoolsListUpdater) readTokensDecimals(ctx context.Context, token0 common.Address, token1 common.Address) (uint8, uint8, error) {
	var decimals0, decimals1 uint8

	req := u.ethrpcClient.R().SetContext(ctx)

	if strings.EqualFold(valueobject.EtherAddress, token0.String()) {
		decimals0 = 18
	} else {
		req.AddCall(&ethrpc.Call{
			ABI:    erc20,
			Target: token0.String(),
			Method: TokenMethodDecimals,
			Params: nil,
		}, []interface{}{&decimals0})
	}

	if strings.EqualFold(valueobject.EtherAddress, token1.String()) {
		decimals1 = 18
	} else {
		req.AddCall(&ethrpc.Call{
			ABI:    erc20,
			Target: token1.String(),
			Method: TokenMethodDecimals,
			Params: nil,
		}, []interface{}{&decimals1})
	}

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("can not read token info")
		return 0, 0, err
	}

	return decimals0, decimals1, nil
}
