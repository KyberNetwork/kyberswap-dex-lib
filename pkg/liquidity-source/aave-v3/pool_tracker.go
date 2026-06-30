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
	utilabi "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
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
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{"pool_id": p.Address}).Error("failed to unmarshal staticExtra")
		return p, err
	}

	rd := newRPCData()
	req := d.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides)
	addRPCCalls(func(c *ethrpc.Call, o []any) { req.AddCall(c, o) },
		staticExtra.AavePoolAddress, p.Tokens[0].Address, p.Tokens[1].Address, rd)

	resp, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{"pool_id": p.Address}).Error("failed to getPoolData")
		return p, err
	}

	return buildPoolState(p, rd, resp.BlockNumber)
}

type rpcData struct {
	configuration RPCConfiguration
	liquidity     *big.Int
	totalSupply   *big.Int
}

func newRPCData() *rpcData {
	return &rpcData{
		liquidity:   new(big.Int),
		totalSupply: new(big.Int),
	}
}

func addRPCCalls(addFn func(*ethrpc.Call, []any), poolAddress, aTokenAddress, assetToken string, d *rpcData) {
	addFn(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: PoolMethodGetConfiguration,
		Params: []any{common.HexToAddress(assetToken)},
	}, []any{&d.configuration.Configuration})
	addFn(&ethrpc.Call{
		ABI:    utilabi.Erc20ABI,
		Target: assetToken,
		Method: utilabi.Erc20BalanceOfMethod,
		Params: []any{common.HexToAddress(aTokenAddress)},
	}, []any{&d.liquidity})
	addFn(&ethrpc.Call{
		ABI:    utilabi.Erc20ABI,
		Target: aTokenAddress,
		Method: utilabi.Erc20TotalSupplyMethod,
	}, []any{&d.totalSupply})
}

func buildPoolState(p entity.Pool, d *rpcData, blockNumber *big.Int) (entity.Pool, error) {
	extra := ParseConfiguration(d.configuration.Configuration.Data.Data)
	extraBytes, err := json.Marshal(&extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Reserves = calculateReserves(extra, d.configuration.Configuration.Data.Data, d.liquidity, d.totalSupply,
		p.Tokens[1].Decimals)
	if blockNumber != nil {
		p.BlockNumber = blockNumber.Uint64()
	}
	p.Timestamp = time.Now().Unix()
	p.Extra = string(extraBytes)

	return p, nil
}

func calculateReserves(extra Extra, configuration, liquidity, totalSupply *big.Int, decimals uint8) entity.PoolReserves {
	supplyCap := ParseSupplyCap(configuration)

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
