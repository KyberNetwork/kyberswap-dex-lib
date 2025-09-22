package eulerswap

import (
	"context"
	"math/big"
	"slices"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

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
	l := logger.WithFields(logger.Fields{"pool_id": p.Address})
	l.Info("Started getting new pool state")
	defer l.Info("Finished getting new pool state")

	var staticExtra StaticExtra
	err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	if err != nil {
		l.Error("failed to unmarshal staticExtra")
		return p, err
	}

	vaults := []VaultInfo{
		{
			VaultAddress: staticExtra.Vault0,
			AssetAddress: p.Tokens[0].Address,
			QuoteAmount:  bignumber.TenPowInt(p.Tokens[0].Decimals),
		},
		{
			VaultAddress: staticExtra.Vault1,
			AssetAddress: p.Tokens[1].Address,
			QuoteAmount:  bignumber.TenPowInt(p.Tokens[1].Decimals),
		},
	}

	rpcData, blockNumber, err := d.getPoolData(ctx, p.Address, staticExtra.EulerAccount,
		staticExtra.EVC, vaults, overrides)
	if err != nil {
		l.Error("failed to getPoolData")
		return p, err
	}

	newPool, err := d.updatePool(p, rpcData, blockNumber)
	if err != nil {
		l.Error("failed to updatePool")
		return p, err
	}

	return newPool, nil
}

func (d *PoolTracker) getPoolData(
	ctx context.Context,
	poolAddress,
	eulerAccount,
	evc string,
	vaultList []VaultInfo,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*TrackerData, *big.Int, error) {
	var (
		data           TrackerData
		vaults         = make([]VaultRPC, 2)
		totalAssets    [2]*big.Int
		totalSupplies  [2]*big.Int
		oracles        [3]common.Address // last element is for controller vault
		unitOfAccounts [3]common.Address // last element is for controller vault
		collaterals    []common.Address
		controllers    []common.Address
	)
	data.Vaults = vaults

	eulerAcctAddr := common.HexToAddress(eulerAccount)
	req := d.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).AddCall(&ethrpc.Call{
		ABI:    evcABI,
		Target: evc,
		Method: evcMethodIsAccountOperatorAuthorized,
		Params: []any{eulerAcctAddr, common.HexToAddress(poolAddress)},
	}, []any{&data.IsOperatorAuthorized}).AddCall(&ethrpc.Call{
		ABI:    evcABI,
		Target: evc,
		Method: evcMethodGetCollaterals,
		Params: []any{eulerAcctAddr},
	}, []any{&collaterals}).AddCall(&ethrpc.Call{
		ABI:    evcABI,
		Target: evc,
		Method: evcMethodGetControllers,
		Params: []any{eulerAcctAddr},
	}, []any{&controllers}).AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetReserves,
	}, []any{&data.Reserves})

	for i, v := range vaultList {
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodCash,
		}, []any{&vaults[i].Cash}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodDebtOf,
			Params: []any{eulerAcctAddr},
		}, []any{&vaults[i].Debt}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodMaxDeposit,
			Params: []any{eulerAcctAddr},
		}, []any{&vaults[i].MaxDeposit}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodCaps,
		}, []any{&vaults[i].Caps}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodTotalBorrows,
		}, []any{&vaults[i].TotalBorrows}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodTotalAssets,
		}, []any{&totalAssets[i]}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodTotalSupply,
		}, []any{&totalSupplies[i]}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodBalanceOf,
			Params: []any{eulerAcctAddr},
		}, []any{&vaults[i].EulerAccountBalance}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodOracle,
		}, []any{&oracles[i]}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodUnitOfAccount,
		}, []any{&unitOfAccounts[i]})
	}

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, nil, errors.WithMessage(err, "1st rt")
	}

	var (
		controllerDecimals       uint8
		controllerAsset          common.Address
		collatTotalAssets        = make([]*big.Int, len(collaterals))
		collatTotalSupplies      = make([]*big.Int, len(collaterals))
		collateralDecimals       = make([]uint8, len(collaterals))
		collateralAssets         = make([]common.Address, len(collaterals))
		collateralOracles        = make([]common.Address, len(collaterals))
		collateralUnitOfAccounts = make([]common.Address, len(collaterals))
	)
	req = d.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).SetBlockNumber(resp.BlockNumber)
	if len(controllers) > 0 {
		vaults = append(vaults, VaultRPC{})
		data.Vaults = vaults
		data.Controller = hexutil.Encode(controllers[0][:])
		vaultList = append(vaultList, VaultInfo{
			VaultAddress: data.Controller,
		})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: data.Controller,
			Method: vaultMethodDebtOf,
			Params: []any{eulerAcctAddr},
		}, []any{&vaults[2].Debt}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: data.Controller,
			Method: vaultMethodDecimals,
		}, []any{&controllerDecimals}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: data.Controller,
			Method: vaultMethodAsset,
		}, []any{&controllerAsset}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: data.Controller,
			Method: vaultMethodOracle,
		}, []any{&oracles[2]}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: data.Controller,
			Method: vaultMethodUnitOfAccount,
		}, []any{&unitOfAccounts[2]})
	}

	data.CollatAmts = make([]*big.Int, len(collaterals))
	vaultAddrs := lo.Map(vaultList, func(vaultInfo VaultInfo, _ int) common.Address {
		return common.HexToAddress(vaultInfo.VaultAddress)
	})
	for i, collateral := range collaterals {
		if idx := slices.Index(vaultAddrs, collateral); idx >= 0 && idx < 2 {
			continue
		}
		collateralStr := hexutil.Encode(collateral[:])
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: collateralStr,
			Method: vaultMethodBalanceOf,
			Params: []any{eulerAcctAddr},
		}, []any{&data.CollatAmts[i]}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: collateralStr,
			Method: vaultMethodTotalAssets,
		}, []any{&collatTotalAssets[i]}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: collateralStr,
			Method: vaultMethodTotalSupply,
		}, []any{&collatTotalSupplies[i]}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: collateralStr,
			Method: vaultMethodDecimals,
		}, []any{&collateralDecimals[i]}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: collateralStr,
			Method: vaultMethodAsset,
		}, []any{&collateralAssets[i]}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: collateralStr,
			Method: vaultMethodOracle,
		}, []any{&collateralOracles[i]}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: collateralStr,
			Method: vaultMethodUnitOfAccount,
		}, []any{&collateralUnitOfAccounts[i]})
	}

	if len(req.Calls) > 0 {
		if _, err = req.TryAggregate(); err != nil {
			return nil, nil, errors.WithMessage(err, "2nd rt")
		}
	}

	if data.Controller != "" {
		vaultList[2].AssetAddress = hexutil.Encode(controllerAsset[:])
		vaultList[2].QuoteAmount = bignumber.TenPowInt(controllerDecimals)
	}

	req = d.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).SetBlockNumber(resp.BlockNumber)
	collatQuoteAmts := make([]*big.Int, len(collaterals))
	data.CollatPrices = make([][3][2]*big.Int, len(collaterals))
	data.CollatLtvs = make([][3]uint16, len(collaterals))

	for i := range vaultList {
		oracleStr := hexutil.Encode(oracles[i][:])
		for j, otherV := range vaultList {
			req.AddCall(&ethrpc.Call{
				ABI:    routerABI,
				Target: oracleStr,
				Method: routerMethodGetQuotes,
				Params: []any{otherV.QuoteAmount, common.HexToAddress(otherV.AssetAddress), unitOfAccounts[j]},
			}, []any{&data.VaultPrices[j][i]})
			if vaultList[i].VaultAddress != vaultList[j].VaultAddress {
				req.AddCall(&ethrpc.Call{
					ABI:    vaultABI,
					Target: vaultList[i].VaultAddress,
					Method: vaultMethodLTVBorrow,
					Params: []any{vaultAddrs[j]},
				}, []any{&data.VaultLtvs[j][i]})
			}
		}
		for j, collateral := range collaterals {
			if idx := slices.Index(vaultAddrs, collateral); idx >= 0 {
				continue
			}
			collatQuoteAmts[j] = bignumber.TenPowInt(collateralDecimals[j])
			req.AddCall(&ethrpc.Call{
				ABI:    routerABI,
				Target: oracleStr,
				Method: routerMethodGetQuotes,
				Params: []any{collatQuoteAmts[j], collateralAssets[j], collateralUnitOfAccounts[j]},
			}, []any{&data.CollatPrices[j][i]}).AddCall(&ethrpc.Call{
				ABI:    vaultABI,
				Target: vaultList[i].VaultAddress,
				Method: vaultMethodLTVBorrow,
				Params: []any{collateral},
			}, []any{&data.CollatLtvs[j][i]})
		}
	}

	if _, err = req.TryAggregate(); err != nil {
		return nil, nil, errors.WithMessage(err, "3rd rt")
	}

	for i, v := range vaults {
		if i < 2 {
			v.EulerAccountBalance = convertToAssets(v.EulerAccountBalance, totalAssets[i], totalSupplies[i])
		}
		for j, otherV := range vaultList {
			for _, q := range data.VaultPrices[j][i] {
				q.Div(q, otherV.QuoteAmount)
			}
		}
		for j, collateral := range collaterals {
			if idx := slices.Index(vaultAddrs, collateral); idx >= 0 {
				data.CollatPrices[j][i] = data.VaultPrices[idx][i]
				continue
			}
			for _, q := range data.CollatPrices[j][i] {
				q.Div(q, collatQuoteAmts[j])
			}
		}
	}
	for i, collateral := range collaterals {
		if idx := slices.Index(vaultAddrs, collateral); idx >= 0 && idx < 2 {
			data.CollatAmts[i] = vaults[idx].EulerAccountBalance
			continue
		}
		data.CollatAmts[i] = convertToAssets(data.CollatAmts[i], collatTotalAssets[i], collatTotalSupplies[i])
	}

	return &data, resp.BlockNumber, nil
}

func (d *PoolTracker) updatePool(pool entity.Pool, data *TrackerData, blockNumber *big.Int) (entity.Pool, error) {
	var vaults [3]*Vault
	for i := range data.Vaults {
		vaults[i] = &Vault{
			Cash:               uint256.MustFromBig(data.Vaults[i].Cash),
			Debt:               uint256.MustFromBig(data.Vaults[i].Debt),
			MaxDeposit:         uint256.MustFromBig(data.Vaults[i].MaxDeposit),
			MaxWithdraw:        decodeCap(uint256.NewInt(uint64(data.Vaults[i].Caps[1]))), // index 1 is borrowCap _ used as maxWithdraw
			TotalBorrows:       uint256.MustFromBig(data.Vaults[i].TotalBorrows),
			EulerAccountAssets: uint256.MustFromBig(data.Vaults[i].EulerAccountBalance),
			DebtPrice:          uint256.MustFromBig(data.VaultPrices[i][i][1]),
			ValuePrices: lo.Map(data.CollatPrices,
				func(p [3][2]*big.Int, _ int) *uint256.Int { return uint256.MustFromBig(p[i][0]) }),
			VaultValuePrices: [2]*uint256.Int(lo.Map(data.VaultPrices[:2],
				func(p [3][2]*big.Int, _ int) *uint256.Int { return uint256.MustFromBig(p[i][0]) })),
			LTVs:      lo.Map(data.CollatLtvs, func(l [3]uint16, _ int) uint64 { return uint64(l[i]) }),
			VaultLTVs: [2]uint64(lo.Map(data.VaultLtvs[:2], func(l [3]uint16, _ int) uint64 { return uint64(l[i]) })),
		}
	}

	reserve0 := data.Reserves.Reserve0.String()
	reserve1 := data.Reserves.Reserve1.String()
	status := data.Reserves.Status
	if !data.IsOperatorAuthorized {
		reserve0 = "0"
		reserve1 = "0"
		status = 2 // locked
	}

	extraBytes, err := json.Marshal(&Extra{
		Pause:           status,
		Vaults:          vaults,
		ControllerVault: data.Controller,
		Collaterals:     big256.MustFromBigs(data.CollatAmts),
	})
	if err != nil {
		return entity.Pool{}, err
	}

	pool.Reserves = entity.PoolReserves{reserve0, reserve1}

	pool.BlockNumber = blockNumber.Uint64()
	pool.Timestamp = time.Now().Unix()
	pool.Extra = string(extraBytes)

	return pool, nil
}

func decodeCap(amountCap *uint256.Int) *uint256.Int {
	//   10 ** (amountCap & 63) * (amountCap >> 6) / 100
	if amountCap.IsZero() {
		return new(uint256.Int).Set(big256.UMax)
	}

	var powerBits, tenToPower, multiplier uint256.Int
	powerBits.And(amountCap, sixtyThree)
	tenToPower.Exp(big256.U10, &powerBits)
	multiplier.Rsh(amountCap, 6)

	amountCap.Mul(&tenToPower, &multiplier)
	return amountCap.Div(amountCap, big256.U100)
}

func convertToAssets(shares, totalAssets, totalSupply *big.Int) *big.Int {
	return shares.Mul(shares, totalAssets.Add(totalAssets, VirtualAmount)).
		Div(shares, totalSupply.Add(totalSupply, VirtualAmount))
}
