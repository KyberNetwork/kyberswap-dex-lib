package aavev3

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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
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

	staticExtra := StaticExtra{}
	err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	if err != nil {
		logger.
			WithFields(logger.Fields{"pool_id": p.Address}).
			Error("failed to unmarshal staticExtra")
		return p, err
	}

	rpcData, liquidity, totalSupply, err := d.getPoolData(ctx, staticExtra.AavePoolAddress, p.Tokens[0].Address,
		p.Tokens[1].Address, overrides)
	if err != nil {
		logger.
			WithFields(logger.Fields{"pool_id": p.Address}).
			Error("failed to getPoolData")
		return p, err
	}

	newPool, err := d.updatePool(p, rpcData, liquidity, totalSupply)
	if err != nil {
		logger.
			WithFields(logger.Fields{"pool_id": p.Address}).
			Error("failed to updatePool")
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
) (RPCConfiguration, *big.Int, *big.Int, error) {
	var rpcData RPCConfiguration
	var liquidity *big.Int
	var totalSupply *big.Int

	req := d.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetConfiguration,
		Params: []any{common.HexToAddress(assetToken)},
	}, []any{&rpcData.Configuration}).AddCall(&ethrpc.Call{
		ABI:    abi.Erc20ABI,
		Target: assetToken,
		Method: abi.Erc20BalanceOfMethod,
		Params: []any{common.HexToAddress(aTokenAddress)},
	}, []any{&liquidity}).AddCall(&ethrpc.Call{
		ABI:    abi.Erc20ABI,
		Target: aTokenAddress,
		Method: abi.Erc20TotalSupplyMethod,
	}, []any{&totalSupply})

	resp, err := req.Aggregate()
	if err != nil {
		return RPCConfiguration{}, nil, nil, err
	}

	rpcData.BlockNumber = resp.BlockNumber.Uint64()

	return rpcData, liquidity, totalSupply, nil
}

func (d *PoolTracker) updatePool(pool entity.Pool, data RPCConfiguration, liquidity, totalSupply *big.Int) (entity.Pool,
	error) {
	extra := parseConfiguration(data.Configuration.Data.Data)
	extraBytes, err := json.Marshal(&extra)
	if err != nil {
		return entity.Pool{}, err
	}

	pool.Reserves = d.calculateReserves(extra, data.Configuration.Data.Data, liquidity, totalSupply,
		pool.Tokens[1].Decimals)

	pool.BlockNumber = data.BlockNumber
	pool.Timestamp = time.Now().Unix()
	pool.Extra = string(extraBytes)

	return pool, nil
}

func (d *PoolTracker) calculateReserves(extra Extra, configuration, liquidity, totalSupply *big.Int,
	decimals uint8) entity.PoolReserves {
	supplyCap := parseSupplyCap(configuration)

	// Calculate reserve[0] (aToken): supply cap - totalSupply
	// Supply cap is in whole tokens, need to scale by decimals
	var reserve0 *big.Int
	if supplyCap > 0 {
		// Scale supply cap by decimals
		scaledSupplyCap := new(big.Int).SetUint64(supplyCap)
		scaledSupplyCap.Mul(scaledSupplyCap, bignumber.TenPowInt(decimals))

		// Available supply = supply cap - total supply
		reserve0 = scaledSupplyCap.Sub(scaledSupplyCap, totalSupply)
		if reserve0.Sign() < 0 {
			reserve0 = bignumber.ZeroBI
		}
	} else {
		// No cap, use large default value
		reserve0 = bignumber.TenPowInt(1000)
	}

	// Handle Frozen vs Paused logic
	// Frozen: stops new supplies and borrows but allows withdrawals and repayments
	// Paused: more restrictive, blocks almost all interactions including withdrawals
	var reserve1 *big.Int
	if !extra.IsActive || extra.IsPaused {
		// Not active or paused: block all interactions
		reserve0 = bignumber.ZeroBI
		reserve1 = bignumber.ZeroBI
	} else if extra.IsFrozen {
		// Frozen: can withdraw (token[0] -> token[1]), cannot supply (token[1] -> token[0])
		// So reserve[0] (aToken) = 0 (cannot supply)
		reserve0 = bignumber.ZeroBI
		reserve1 = liquidity
	} else {
		// Active and not frozen: allow both supply and withdraw
		reserve1 = liquidity
	}

	return entity.PoolReserves{reserve0.String(), reserve1.String()}
}
