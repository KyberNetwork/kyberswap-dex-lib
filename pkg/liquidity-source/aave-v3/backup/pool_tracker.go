package backup

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	aavev3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/aave-v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	utilabi "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config       *aavev3.Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterBackupFactoryCE(aavev3.DexType, NewPoolTracker)

func NewPoolTracker(
	config *aavev3.Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, params, nil)
}

func (d *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (d *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	staticExtra := aavev3.StaticExtra{}
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{"pool_id": p.Address}).Error("failed to unmarshal staticExtra")
		return p, err
	}

	rpcData, liquidity, totalSupply, err := d.getPoolData(ctx, staticExtra.AavePoolAddress, p.Tokens[0].Address,
		p.Tokens[1].Address, overrides)
	if err != nil {
		logger.WithFields(logger.Fields{"pool_id": p.Address}).Error("failed to getPoolData")
		return p, err
	}

	newPool, err := d.updatePool(p, rpcData, liquidity, totalSupply)
	if err != nil {
		logger.WithFields(logger.Fields{"pool_id": p.Address}).Error("failed to updatePool")
		return p, err
	}

	return newPool, nil
}

func (d *PoolTracker) getPoolData(
	ctx context.Context,
	poolAddress,
	aTokenAddress,
	assetToken string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (aavev3.RPCConfiguration, *big.Int, *big.Int, error) {
	var rpcData aavev3.RPCConfiguration
	var liquidity *big.Int
	var totalSupply *big.Int

	req := d.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).AddCall(&ethrpc.Call{
		ABI:    *aavev3.PoolABI,
		Target: poolAddress,
		Method: aavev3.PoolMethodGetConfiguration,
		Params: []any{common.HexToAddress(assetToken)},
	}, []any{&rpcData.Configuration}).AddCall(&ethrpc.Call{
		ABI:    utilabi.Erc20ABI,
		Target: assetToken,
		Method: utilabi.Erc20BalanceOfMethod,
		Params: []any{common.HexToAddress(aTokenAddress)},
	}, []any{&liquidity}).AddCall(&ethrpc.Call{
		ABI:    utilabi.Erc20ABI,
		Target: aTokenAddress,
		Method: utilabi.Erc20TotalSupplyMethod,
	}, []any{&totalSupply})

	resp, err := req.Aggregate()
	if err != nil {
		return aavev3.RPCConfiguration{}, nil, nil, err
	}

	rpcData.BlockNumber = resp.BlockNumber.Uint64()

	return rpcData, liquidity, totalSupply, nil
}

func (d *PoolTracker) updatePool(p entity.Pool, data aavev3.RPCConfiguration, liquidity, totalSupply *big.Int) (entity.Pool, error) {
	extra := aavev3.ParseConfiguration(data.Configuration.Data.Data)
	extraBytes, err := json.Marshal(&extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Reserves = calculateReserves(extra, data.Configuration.Data.Data, liquidity, totalSupply, p.Tokens[1].Decimals)
	p.BlockNumber = data.BlockNumber
	p.Timestamp = time.Now().Unix()
	p.Extra = string(extraBytes)

	return p, nil
}

func calculateReserves(extra aavev3.Extra, configuration, liquidity, totalSupply *big.Int, decimals uint8) entity.PoolReserves {
	supplyCap := aavev3.ParseSupplyCap(configuration)

	var reserve0 *big.Int
	if supplyCap > 0 {
		scaledSupplyCap := new(big.Int).SetUint64(supplyCap)
		scaledSupplyCap.Mul(scaledSupplyCap, bignumber.TenPowInt(decimals))
		reserve0 = scaledSupplyCap.Sub(scaledSupplyCap, totalSupply)
		if reserve0.Sign() < 0 {
			reserve0 = bignumber.ZeroBI
		}
	} else {
		reserve0 = bignumber.TenPowInt(1000)
	}

	var reserve1 *big.Int
	if !extra.IsActive || extra.IsPaused {
		reserve0 = bignumber.ZeroBI
		reserve1 = bignumber.ZeroBI
	} else if extra.IsFrozen {
		reserve0 = bignumber.ZeroBI
		reserve1 = liquidity
	} else {
		reserve1 = liquidity
	}

	return entity.PoolReserves{reserve0.String(), reserve1.String()}
}
