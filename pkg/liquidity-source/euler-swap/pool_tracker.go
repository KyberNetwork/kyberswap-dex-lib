package eulerswap

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	bignumber "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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

	var staticExtra StaticExtra
	err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	if err != nil {
		logger.
			WithFields(logger.Fields{"pool_id": p.Address}).
			Error("failed to unmarshal staticExtra")
		return p, err
	}

	rpcData, blockNumber, err := d.getPoolData(ctx, p.Address, staticExtra.EulerAccount, staticExtra.Vault0, staticExtra.Vault1, overrides)
	if err != nil {
		logger.
			WithFields(logger.Fields{"pool_id": p.Address}).
			Error("failed to getPoolData")
		return p, err
	}

	newPool, err := d.updatePool(p, rpcData, blockNumber)
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
	eulerAccount,
	vault0, vault1 string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (TrackerData, *big.Int, error) {
	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	var (
		reserves ReserveRPC
		vaults   = make([]VaultRPC, 2)
	)

	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetReserves,
		Params: nil,
	}, []any{&reserves})

	for i, vaultAddress := range []string{vault0, vault1} {
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: vaultAddress,
			Method: vaultMethodCash,
			Params: nil,
		}, []any{&vaults[i].Cash})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: vaultAddress,
			Method: vaultMethodDebtOf,
			Params: []any{common.HexToAddress(eulerAccount)},
		}, []any{&vaults[i].Debt})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: vaultAddress,
			Method: vaultMethodMaxDeposit,
			Params: []any{common.HexToAddress(eulerAccount)},
		}, []any{&vaults[i].MaxDeposit})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: vaultAddress,
			Method: vaultMethodCaps,
			Params: nil,
		}, []any{&vaults[i].Caps})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: vaultAddress,
			Method: vaultMethodTotalBorrows,
			Params: nil,
		}, []any{&vaults[i].TotalBorrows})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: vaultAddress,
			Method: vaultMethodTotalAssets,
			Params: nil,
		}, []any{&vaults[i].TotalAssets})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: vaultAddress,
			Method: vaultMethodTotalSupply,
			Params: nil,
		}, []any{&vaults[i].TotalSupply})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: vaultAddress,
			Method: vaultMethodBalanceOf,
			Params: []any{common.HexToAddress(eulerAccount)},
		}, []any{&vaults[i].EulerAccountBalance})
	}

	resp, err := req.Aggregate()
	if err != nil {
		return TrackerData{}, nil, err
	}

	return TrackerData{
		Vaults:   vaults,
		Reserves: reserves,
	}, resp.BlockNumber, nil
}

func (d *PoolTracker) updatePool(pool entity.Pool, data TrackerData, blockNumber *big.Int) (entity.Pool, error) {
	var vaults = make([]Vault, len(data.Vaults))

	for i := range data.Vaults {
		totalAssets := uint256.MustFromBig(data.Vaults[i].TotalAssets)
		totalSupply := uint256.MustFromBig(data.Vaults[i].TotalSupply)
		eulerAccountShare := uint256.MustFromBig(data.Vaults[i].EulerAccountBalance)

		vaults[i] = Vault{
			Cash:               uint256.MustFromBig(data.Vaults[i].Cash),
			Debt:               uint256.MustFromBig(data.Vaults[i].Debt),
			MaxDeposit:         uint256.MustFromBig(data.Vaults[i].MaxDeposit),
			TotalBorrows:       uint256.MustFromBig(data.Vaults[i].TotalBorrows),
			EulerAccountAssets: convertToAssets(eulerAccountShare, totalAssets, totalSupply),
			MaxWithdraw:        decodeCap(uint256.NewInt(uint64(data.Vaults[i].Caps[1]))), // index 1 is borrowCap _ used as maxWithdraw
		}
	}

	extraBytes, err := json.Marshal(&Extra{
		Pause:  data.Reserves.Pause,
		Vaults: vaults,
	})
	if err != nil {
		return entity.Pool{}, err
	}

	pool.Reserves = entity.PoolReserves{
		data.Reserves.Reserve0.String(),
		data.Reserves.Reserve1.String(),
	}

	pool.BlockNumber = blockNumber.Uint64()
	pool.Timestamp = time.Now().Unix()
	pool.Extra = string(extraBytes)

	return pool, nil
}

func decodeCap(amountCap *uint256.Int) *uint256.Int {
	//   10 ** (amountCap & 63) * (amountCap >> 6) / 100
	if amountCap.Cmp(bignumber.ZeroBI) == 0 {
		return new(uint256.Int).Set(maxUint256)
	}

	powerBits := new(uint256.Int).And(amountCap, sixtyThree)
	tenToPower := new(uint256.Int).Exp(ten, powerBits)

	multiplier := new(uint256.Int).Rsh(amountCap, 6)

	amountCap = tenToPower.Mul(tenToPower, multiplier)

	return amountCap.Div(amountCap, hundred)
}

func convertToAssets(shares, totalAssets, totalSupply *uint256.Int) *uint256.Int {
	shares.MulDivOverflow(shares, totalAssets.Add(totalAssets, VIRTUAL_AMOUNT), totalSupply.Add(totalSupply, VIRTUAL_AMOUNT))
	return shares
}
