package dexT1

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config       Config
	ethrpcClient *ethrpc.Client
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

	staticExtraBytes, err := json.Marshal(&StaticExtra{
		DexReservesResolver: u.config.DexReservesResolver,
	})
	if err != nil {
		return nil, nil, err
	}

	allPools, err := u.getAllPools(ctx)

	if err != nil {
		return nil, nil, err
	}

	pools := make([]entity.Pool, 0)

	for _, curPool := range allPools {

		token0Decimals, token1Decimals, err := u.readTokensDecimals(ctx, curPool.Token0Address, curPool.Token1Address)
		if err != nil {
			return nil, nil, err
		}

		if curPool.CollateralReserves.Token0RealReserves == nil ||
			curPool.CollateralReserves.Token1RealReserves == nil ||
			curPool.CollateralReserves.Token0RealReserves.Cmp(bignumber.ZeroBI) != 0 ||
			curPool.CollateralReserves.Token1RealReserves.Cmp(bignumber.ZeroBI) != 0 ||
			curPool.DebtReserves.Token0RealReserves == nil ||
			curPool.DebtReserves.Token1RealReserves == nil ||
			curPool.DebtReserves.Token0RealReserves.Cmp(bignumber.ZeroBI) != 0 ||
			curPool.DebtReserves.Token1RealReserves.Cmp(bignumber.ZeroBI) != 0 {
			logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("Error reserves are nil / 0")
			return nil, nil, errors.New("pool reserves are nil / 0")
		}

		extra := PoolExtra{
			CollateralReserves: curPool.CollateralReserves,
			DebtReserves:       curPool.DebtReserves,
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
				new(big.Int).Add(curPool.CollateralReserves.Token0RealReserves, curPool.DebtReserves.Token0RealReserves).String(),
				new(big.Int).Add(curPool.CollateralReserves.Token1RealReserves, curPool.DebtReserves.Token1RealReserves).String(),
			},
			Tokens: []*entity.PoolToken{
				{
					Address:   curPool.Token0Address.String(),
					Weight:    1,
					Swappable: true,
					Decimals:  token0Decimals,
				},
				{
					Address:   curPool.Token1Address.String(),
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

	return pools, metadataBytes, nil
}

func (u *PoolsListUpdater) getAllPools(ctx context.Context) ([]PoolWithReserves, error) {
	var pools []PoolWithReserves

	req := u.ethrpcClient.R().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    dexReservesResolverABI,
		Target: u.config.DexReservesResolver,
		Method: DRRMethodGetAllPoolsReserves,
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

	if strings.EqualFold(NativeETH, token0.String()) {
		decimals0 = 18
	} else {
		req.AddCall(&ethrpc.Call{
			ABI:    erc20,
			Target: token0.String(),
			Method: TokenMethodDecimals,
			Params: nil,
		}, []interface{}{&decimals0})
	}

	if strings.EqualFold(NativeETH, token1.String()) {
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
