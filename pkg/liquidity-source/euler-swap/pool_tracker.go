package eulerswap

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type (
	PoolTracker struct {
		config       *Config
		ethrpcClient *ethrpc.Client
		logDecoder   uniswapv2.ILogDecoder
	}
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		logDecoder:   uniswapv2.NewLogDecoder(),
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

	rpcData, blockNumber, err := d.getPoolData(ctx, p.Address, overrides)
	if err != nil {
		return p, err
	}

	return d.updatePool(p, rpcData, blockNumber)
}

func (d *PoolTracker) getPoolData(
	ctx context.Context,
	poolAddress string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (RPCData, *big.Int, error) {
	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	var (
		reserves            ReserveRPC
		eulerAccount        common.Address
		vault0              common.Address
		vault1              common.Address
		equilibriumReserve0 *big.Int
		equilibriumReserve1 *big.Int
		priceX              *big.Int
		priceY              *big.Int
		concentrationX      *big.Int
		concentrationY      *big.Int
		feeMultiplier       *big.Int
	)

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetReserves,
		Params: nil,
	}, []any{&reserves})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodEulerAccount,
		Params: nil,
	}, []any{&eulerAccount})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodVault0,
		Params: nil,
	}, []any{&vault0})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodVault1,
		Params: nil,
	}, []any{&vault1})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodVault0,
		Params: nil,
	}, []any{&vault0})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodEquilibriumReserve0,
		Params: nil,
	}, []any{&equilibriumReserve0})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodEquilibriumReserve1,
		Params: nil,
	}, []any{&equilibriumReserve1})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodPriceX,
		Params: nil,
	}, []any{&priceX})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodPriceY,
		Params: nil,
	}, []any{&priceY})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodConcentrationX,
		Params: nil,
	}, []any{&concentrationX})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodConcentrationY,
		Params: nil,
	}, []any{&concentrationY})

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodFeeMultiplier,
		Params: nil,
	}, []any{&feeMultiplier})

	resp, err := req.Aggregate()
	if err != nil {
		return RPCData{}, nil, err
	}

	vaults, err := d.getVaultsData(ctx, vault0, vault1, eulerAccount, resp.BlockNumber, overrides)
	if err != nil {
		return RPCData{}, nil, err
	}

	_ = vaults

	return RPCData{}, nil, nil
}

func (d *PoolTracker) getVaultsData(
	ctx context.Context,
	vault0, vault1, eulerAccount common.Address,
	blockNumber *big.Int,
	overrides map[common.Address]gethclient.OverrideAccount,
) ([]VaultRPC, error) {
	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}
	if blockNumber != nil {
		req.SetBlockNumber(blockNumber)
	}

	var vaults = make([]VaultRPC, 2)

	for i, vaultAddress := range []common.Address{vault0, vault1} {
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: vaultAddress.Hex(),
			Method: vaultMethodCash,
			Params: nil,
		}, []any{&vaults[i].Cash})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: vaultAddress.Hex(),
			Method: vaultMethodDebtOf,
			Params: []any{eulerAccount},
		}, []any{&vaults[i].Debt})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: vaultAddress.Hex(),
			Method: vaultMethodMaxDeposit,
			Params: []any{eulerAccount},
		}, []any{&vaults[i].MaxDeposit})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: vaultAddress.Hex(),
			Method: vaultMethodMaxWithdraw,
			Params: []any{eulerAccount},
		}, []any{&vaults[i].MaxWithdraw})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: vaultAddress.Hex(),
			Method: vaultMethodTotalBorrows,
			Params: nil,
		}, []any{&vaults[i].TotalBorrows})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: vaultAddress.Hex(),
			Method: vaultMethodBalanceOf,
			Params: []any{eulerAccount},
		}, []any{&vaults[i].Balance})
	}

	resp, err := req.Aggregate()
	if err != nil {
		return nil, err
	}

	asset0, asset1, err := d.getVaultAssets(ctx, vault0, vault1, vaults[0].Balance, vaults[1].Balance, resp.BlockNumber, overrides)
	if err != nil {
		return nil, err
	}

	_, _ = asset0, asset1

	return vaults, nil
}

func (d *PoolTracker) getVaultAssets(
	ctx context.Context,
	vault0, vault1 common.Address,
	balance0, balance1 *big.Int,
	blockNumber *big.Int,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*big.Int, *big.Int, error) {

	var (
		asset0, asset1 *big.Int
	)

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}
	if blockNumber != nil {
		req.SetBlockNumber(blockNumber)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    vaultABI,
		Target: vault0.Hex(),
		Method: vaultMethodConvertToAssets,
		Params: []any{balance0},
	}, []any{&asset0})

	req.AddCall(&ethrpc.Call{
		ABI:    vaultABI,
		Target: vault1.Hex(),
		Method: vaultMethodConvertToAssets,
		Params: []any{balance1},
	}, []any{&asset1})

	_, err := req.Aggregate()
	if err != nil {
		return nil, nil, err
	}

	return asset0, asset1, nil
}

func (d *PoolTracker) updatePool(pool entity.Pool, rpcData RPCData, blockNumber *big.Int) (entity.Pool, error) {
	// extra, err := json.Marshal(&originalReserves)
	// if err != nil {
	// 	return entity.Pool{}, err
	// }

	// pool.Reserves = entity.PoolReserves{
	// 	fwReserves.Reserve0.String(),
	// 	fwReserves.Reserve1.String(),
	// 	"1",
	// 	"1",
	// }

	// pool.BlockNumber = blockNumber.Uint64()
	// pool.Timestamp = time.Now().Unix()
	// pool.Extra = string(extra)

	return pool, nil
}
