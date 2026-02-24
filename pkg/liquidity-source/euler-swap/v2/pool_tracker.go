package v2

import (
	"context"
	"math/big"
	"strings"
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
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/v2/hooks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	config       *shared.Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	config *shared.Config,
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
	l := logger.WithFields(logger.Fields{
		"poolAddress": p.Address,
		"handler":     "EulerSwapV2.PoolTracker",
	})

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		l.Error("failed to unmarshal staticExtra")
		return p, err
	}

	vaults := []shared.VaultInfo{
		{
			VaultAddress: staticExtra.SupplyVault0,
			AssetAddress: p.Tokens[0].Address,
			QuoteAmount:  bignumber.TenPowInt(p.Tokens[0].Decimals),
		},
		{
			VaultAddress: staticExtra.SupplyVault1,
			AssetAddress: p.Tokens[1].Address,
			QuoteAmount:  bignumber.TenPowInt(p.Tokens[1].Decimals),
		},
	}
	if staticExtra.BorrowVault0 != "" && staticExtra.BorrowVault0 != staticExtra.SupplyVault0 && staticExtra.BorrowVault0 != valueobject.ZeroAddress {
		vaults = append(vaults, shared.VaultInfo{
			VaultAddress: staticExtra.BorrowVault0,
			AssetAddress: p.Tokens[0].Address,
			QuoteAmount:  bignumber.TenPowInt(p.Tokens[0].Decimals),
		})
	}
	if staticExtra.BorrowVault1 != "" && staticExtra.BorrowVault1 != staticExtra.SupplyVault1 && staticExtra.BorrowVault1 != valueobject.ZeroAddress {
		vaults = append(vaults, shared.VaultInfo{
			VaultAddress: staticExtra.BorrowVault1,
			AssetAddress: p.Tokens[1].Address,
			QuoteAmount:  bignumber.TenPowInt(p.Tokens[1].Decimals),
		})
	}

	rpcData, blockNumber, err := d.getPoolData(ctx, p.Address, staticExtra.EulerAccount,
		staticExtra.EVC, vaults, overrides)
	if err != nil {
		l.Error("failed to getPoolData")
		return p, err
	}

	newPool, err := d.updatePool(ctx, p, rpcData, blockNumber)
	if err != nil {
		l.Error("failed to updatePool")
		return p, err
	}

	return newPool, nil
}

func (d *PoolTracker) getPoolData(
	ctx context.Context,
	poolAddress,
	eulerAccountAddr string,
	evcAddr string,
	vaultList []shared.VaultInfo,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*TrackerData, uint64, error) {
	uniqueVaultAddresses := lo.Map(lo.UniqBy(vaultList, func(v shared.VaultInfo) string {
		return strings.ToLower(v.VaultAddress)
	}), func(v shared.VaultInfo, _ int) string {
		return v.VaultAddress
	})

	var (
		data           TrackerData
		vaults         = make([]shared.VaultRPC, len(uniqueVaultAddresses))
		oracles        = make([]common.Address, len(uniqueVaultAddresses))
		unitOfAccounts = make([]common.Address, len(uniqueVaultAddresses))
		collaterals    []common.Address
		controllers    []common.Address
	)

	eulerAcctAddr := common.HexToAddress(eulerAccountAddr)
	req := d.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: shared.PoolMethodIsInstalled,
		Params: nil,
	}, []any{&data.IsOperatorAuthorized}).AddCall(&ethrpc.Call{
		ABI:    evcABI,
		Target: evcAddr,
		Method: shared.EvcMethodGetCollaterals,
		Params: []any{eulerAcctAddr},
	}, []any{&collaterals}).AddCall(&ethrpc.Call{
		ABI:    evcABI,
		Target: evcAddr,
		Method: shared.EvcMethodGetControllers,
		Params: []any{eulerAcctAddr},
	}, []any{&controllers}).AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: shared.PoolMethodGetReserves,
	}, []any{&data.Reserves}).AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: shared.PoolMethodGetDynamicParams,
	}, []any{&data.DynamicParams})

	for i, addr := range uniqueVaultAddresses {
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: addr,
			Method: shared.VaultMethodCash,
		}, []any{&vaults[i].Cash}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: addr,
			Method: shared.VaultMethodDebtOf,
			Params: []any{eulerAcctAddr},
		}, []any{&vaults[i].Debt}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: addr,
			Method: shared.VaultMethodMaxDeposit,
			Params: []any{eulerAcctAddr},
		}, []any{&vaults[i].MaxDeposit}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: addr,
			Method: shared.VaultMethodCaps,
		}, []any{&vaults[i].Caps}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: addr,
			Method: shared.VaultMethodTotalBorrows,
		}, []any{&vaults[i].TotalBorrows}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: addr,
			Method: shared.VaultMethodTotalAssets,
		}, []any{&vaults[i].TotalAssets}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: addr,
			Method: shared.VaultMethodTotalSupply,
		}, []any{&vaults[i].TotalSupply}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: addr,
			Method: shared.VaultMethodBalanceOf,
			Params: []any{eulerAcctAddr},
		}, []any{&vaults[i].EulerAccountBalance}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: addr,
			Method: shared.VaultMethodOracle,
		}, []any{&oracles[i]}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: addr,
			Method: shared.VaultMethodUnitOfAccount,
		}, []any{&unitOfAccounts[i]}).AddCall(&ethrpc.Call{
			ABI:    evcABI,
			Target: evcAddr,
			Method: shared.EvcMethodIsControllerEnabled,
			Params: []any{eulerAcctAddr, common.HexToAddress(addr)},
		}, []any{&vaults[i].IsControllerEnabled})
	}

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, 0, errors.WithMessage(err, "1st rt")
	}

	var (
		controllerDecimals  uint8
		controllerAsset     common.Address
		collatTotalAssets   = make([]*big.Int, len(collaterals))
		collatTotalSupplies = make([]*big.Int, len(collaterals))
		collateralDecimals  = make([]uint8, len(collaterals))
		collateralAssets    = make([]common.Address, len(collaterals))
	)

	if len(controllers) > 0 {
		data.Controller = hexutil.Encode(controllers[0][:])
		found := false
		lowerController := strings.ToLower(data.Controller)
		for _, addr := range uniqueVaultAddresses {
			if strings.ToLower(addr) == lowerController {
				found = true
				break
			}
		}

		if !found {
			vaults = append(vaults, shared.VaultRPC{})
			oracles = append(oracles, common.Address{})
			unitOfAccounts = append(unitOfAccounts, common.Address{})
			uniqueVaultAddresses = append(uniqueVaultAddresses, data.Controller)

			idx := len(vaults) - 1
			req = d.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).SetBlockNumber(resp.BlockNumber)
			req.AddCall(&ethrpc.Call{
				ABI:    vaultABI,
				Target: data.Controller,
				Method: shared.VaultMethodDebtOf,
				Params: []any{eulerAcctAddr},
			}, []any{&vaults[idx].Debt}).AddCall(&ethrpc.Call{
				ABI:    vaultABI,
				Target: data.Controller,
				Method: shared.VaultMethodDecimals,
			}, []any{&controllerDecimals}).AddCall(&ethrpc.Call{
				ABI:    vaultABI,
				Target: data.Controller,
				Method: shared.VaultMethodAsset,
			}, []any{&controllerAsset}).AddCall(&ethrpc.Call{
				ABI:    vaultABI,
				Target: data.Controller,
				Method: shared.VaultMethodOracle,
			}, []any{&oracles[idx]}).AddCall(&ethrpc.Call{
				ABI:    vaultABI,
				Target: data.Controller,
				Method: shared.VaultMethodUnitOfAccount,
			}, []any{&unitOfAccounts[idx]})

			if resp, err = req.TryBlockAndAggregate(); err != nil {
				return nil, 0, err
			}
			vaults[idx].EulerAccountBalance = big.NewInt(0)
		}
	}

	data.UniqueVaultAddresses = uniqueVaultAddresses
	data.Vaults = vaults

	data.CollatAmounts = make([]*big.Int, len(collaterals))
	fullVaultList := lo.Map(uniqueVaultAddresses, func(addr string, _ int) shared.VaultInfo {
		if v, ok := lo.Find(vaultList, func(v shared.VaultInfo) bool {
			return strings.EqualFold(v.VaultAddress, addr)
		}); ok {
			return v
		}
		return shared.VaultInfo{
			VaultAddress: addr,
			AssetAddress: hexutil.Encode(controllerAsset[:]),
			QuoteAmount:  bignumber.TenPowInt(controllerDecimals),
		}
	})

	fullVaultAddrs := lo.Map(fullVaultList, func(v shared.VaultInfo, _ int) common.Address {
		return common.HexToAddress(v.VaultAddress)
	})

	fullVaultAddrMap := make(map[common.Address]int, len(fullVaultAddrs))
	for idx, addr := range fullVaultAddrs {
		fullVaultAddrMap[addr] = idx
	}

	req = d.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).SetBlockNumber(resp.BlockNumber)
	for i, collateral := range collaterals {
		if _, ok := fullVaultAddrMap[collateral]; ok {
			continue
		}
		collateralStr := hexutil.Encode(collateral[:])
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: collateralStr,
			Method: shared.VaultMethodBalanceOf,
			Params: []any{eulerAcctAddr},
		}, []any{&data.CollatAmounts[i]}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: collateralStr,
			Method: shared.VaultMethodTotalAssets,
		}, []any{&collatTotalAssets[i]}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: collateralStr,
			Method: shared.VaultMethodTotalSupply,
		}, []any{&collatTotalSupplies[i]}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: collateralStr,
			Method: shared.VaultMethodDecimals,
		}, []any{&collateralDecimals[i]}).AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: collateralStr,
			Method: shared.VaultMethodAsset,
		}, []any{&collateralAssets[i]})
	}

	if len(req.Calls) > 0 {
		if _, err = req.TryAggregate(); err != nil {
			return nil, 0, errors.WithMessage(err, "2nd rt")
		}
	}

	req = d.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).SetBlockNumber(resp.BlockNumber)
	collatQuoteAmts := make([]*big.Int, len(collaterals))
	data.CollatPrices = make([][][2]*big.Int, len(collaterals))
	data.CollatLtvs = make([][]uint16, len(collaterals))

	data.VaultPrices = lo.Times(len(fullVaultList), func(_ int) [][2]*big.Int {
		return make([][2]*big.Int, len(uniqueVaultAddresses))
	})
	data.VaultLtvs = lo.Times(len(fullVaultList), func(_ int) []uint16 {
		return make([]uint16, len(uniqueVaultAddresses))
	})

	for i := range uniqueVaultAddresses {
		oracleStr := hexutil.Encode(oracles[i][:])
		if valueobject.IsZeroAddress(oracles[i]) {
			continue
		}

		for j, otherV := range fullVaultList {
			req.AddCall(&ethrpc.Call{
				ABI:    routerABI,
				Target: oracleStr,
				Method: shared.RouterMethodGetQuotes,
				Params: []any{otherV.QuoteAmount, common.HexToAddress(otherV.AssetAddress), unitOfAccounts[i]},
			}, []any{&data.VaultPrices[j][i]})

			if uniqueVaultAddresses[i] != otherV.VaultAddress {
				req.AddCall(&ethrpc.Call{
					ABI:    vaultABI,
					Target: uniqueVaultAddresses[i],
					Method: shared.VaultMethodLTVBorrow,
					Params: []any{fullVaultAddrs[j]},
				}, []any{&data.VaultLtvs[j][i]})
			}
		}
		for j, collateral := range collaterals {
			if len(data.CollatPrices[j]) <= i {
				for len(data.CollatPrices[j]) <= i {
					data.CollatPrices[j] = append(data.CollatPrices[j], [2]*big.Int{})
				}
			}
			if len(data.CollatLtvs[j]) <= i {
				for len(data.CollatLtvs[j]) <= i {
					data.CollatLtvs[j] = append(data.CollatLtvs[j], 0)
				}
			}

			if idx, ok := fullVaultAddrMap[collateral]; ok {
				data.CollatPrices[j][i] = data.VaultPrices[idx][i]
				data.CollatLtvs[j][i] = data.VaultLtvs[idx][i]
				continue
			}
			collatQuoteAmts[j] = bignumber.TenPowInt(collateralDecimals[j])
			req.AddCall(&ethrpc.Call{
				ABI:    routerABI,
				Target: oracleStr,
				Method: shared.RouterMethodGetQuotes,
				Params: []any{collatQuoteAmts[j], collateralAssets[j], unitOfAccounts[i]},
			}, []any{&data.CollatPrices[j][i]}).AddCall(&ethrpc.Call{
				ABI:    vaultABI,
				Target: uniqueVaultAddresses[i],
				Method: shared.VaultMethodLTVBorrow,
				Params: []any{collateral},
			}, []any{&data.CollatLtvs[j][i]})
		}
	}

	if _, err = req.TryAggregate(); err != nil {
		return nil, 0, errors.WithMessage(err, "3rd rt")
	}

	for i := range uniqueVaultAddresses {
		for j, v := range fullVaultList {
			for k, q := range data.VaultPrices[j][i] {
				if q != nil {
					data.VaultPrices[j][i][k] = new(big.Int).Div(q, v.QuoteAmount)
				} else {
					data.VaultPrices[j][i][k] = big.NewInt(0)
				}
			}
		}
		for j, collatAddr := range collaterals {
			if idx, ok := fullVaultAddrMap[collatAddr]; ok {
				data.CollatPrices[j][i] = data.VaultPrices[idx][i]
				continue
			}
			for k, q := range data.CollatPrices[j][i] {
				if q != nil {
					data.CollatPrices[j][i][k] = new(big.Int).Div(q, collatQuoteAmts[j])
				} else {
					data.CollatPrices[j][i][k] = big.NewInt(0)
				}
			}
		}
	}

	for i := range vaults {
		vaults[i].EulerAccountBalance = shared.ConvertToAssets(vaults[i].EulerAccountBalance, vaults[i].TotalAssets, vaults[i].TotalSupply)
	}
	for i, collateral := range collaterals {
		if idx, ok := fullVaultAddrMap[collateral]; ok {
			data.CollatAmounts[i] = vaults[idx].EulerAccountBalance
			data.CollatLtvs[i] = data.VaultLtvs[idx]
		} else {
			data.CollatAmounts[i] = shared.ConvertToAssets(data.CollatAmounts[i], collatTotalAssets[i], collatTotalSupplies[i])
		}
	}

	return &data, resp.BlockNumber.Uint64(), nil
}

func (d *PoolTracker) updatePool(
	ctx context.Context,
	pool entity.Pool,
	data *TrackerData,
	blockNumber uint64,
) (entity.Pool, error) {

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(pool.StaticExtra), &staticExtra); err != nil {
		return entity.Pool{}, err
	}

	uniqueVaultAddrMap := make(map[string]int, len(data.UniqueVaultAddresses))
	for idx, addr := range data.UniqueVaultAddresses {
		uniqueVaultAddrMap[strings.ToLower(addr)] = idx
	}

	idx0 := uniqueVaultAddrMap[strings.ToLower(staticExtra.SupplyVault0)]
	idx1 := uniqueVaultAddrMap[strings.ToLower(staticExtra.SupplyVault1)]

	vaultMap := make(map[string]*shared.VaultState, len(data.Vaults))
	for idx, v := range data.Vaults {
		addr := strings.ToLower(data.UniqueVaultAddresses[idx])
		state := &shared.VaultState{
			Cash:                uint256.MustFromBig(v.Cash),
			Debt:                uint256.MustFromBig(v.Debt),
			MaxDeposit:          uint256.MustFromBig(v.MaxDeposit),
			BorrowCap:           shared.DecodeCap(uint256.NewInt(uint64(v.Caps[1]))),
			TotalBorrows:        uint256.MustFromBig(v.TotalBorrows),
			EulerAccountAssets:  uint256.MustFromBig(v.EulerAccountBalance),
			IsControllerEnabled: v.IsControllerEnabled,
		}

		if idx < len(data.VaultPrices) {
			state.DebtPrice = uint256.MustFromBig(data.VaultPrices[idx][idx][1])
			state.ValuePrices = lo.Map(data.CollatPrices, func(p [][2]*big.Int, _ int) *uint256.Int {
				return uint256.MustFromBig(p[idx][0])
			})
			state.VaultValuePrices = [2]*uint256.Int{
				uint256.MustFromBig(data.VaultPrices[idx0][idx][0]),
				uint256.MustFromBig(data.VaultPrices[idx1][idx][0]),
			}
			state.LTVs = lo.Map(data.CollatLtvs, func(l []uint16, _ int) uint64 { return uint64(l[idx]) })
			state.VaultLTVs = [2]uint64{
				uint64(data.VaultLtvs[idx0][idx]),
				uint64(data.VaultLtvs[idx1][idx]),
			}
		}
		vaultMap[addr] = state
	}

	reserve0 := data.Reserves.Reserve0.String()
	reserve1 := data.Reserves.Reserve1.String()
	status := data.Reserves.Status
	if !data.IsOperatorAuthorized {
		reserve0 = "0"
		reserve1 = "0"
		status = 2 // locked
	}
	pool.Reserves = entity.PoolReserves{reserve0, reserve1}

	dParams := DynamicParams{
		EquilibriumReserve0: uint256.MustFromBig(data.DynamicParams.Data.EquilibriumReserve0),
		EquilibriumReserve1: uint256.MustFromBig(data.DynamicParams.Data.EquilibriumReserve1),
		MinReserve0:         uint256.MustFromBig(data.DynamicParams.Data.MinReserve0),
		MinReserve1:         uint256.MustFromBig(data.DynamicParams.Data.MinReserve1),
		PriceX:              uint256.MustFromBig(data.DynamicParams.Data.PriceX),
		PriceY:              uint256.MustFromBig(data.DynamicParams.Data.PriceY),
		ConcentrationX:      uint256.NewInt(data.DynamicParams.Data.ConcentrationX),
		ConcentrationY:      uint256.NewInt(data.DynamicParams.Data.ConcentrationY),
		Fee0:                uint256.NewInt(data.DynamicParams.Data.Fee0),
		Fee1:                uint256.NewInt(data.DynamicParams.Data.Fee1),
		Expiration:          data.DynamicParams.Data.Expiration.Uint64(),
		SwapHookedOps:       data.DynamicParams.Data.SwapHookedOperations,
		SwapHook:            hexutil.Encode(data.DynamicParams.Data.SwapHook[:]),
	}

	var hookExtra string
	if dParams.SwapHookedOps != 0 && dParams.SwapHook != "" {
		hookAddr := common.HexToAddress(dParams.SwapHook)
		hook := hooks.GetHook(hookAddr, &hooks.HookParam{
			Pool:        &pool,
			HookAddress: hookAddr,
		})
		if hook != nil {
			var err error
			hookExtra, err = hook.Track(ctx, &hooks.HookParam{
				RpcClient:   d.ethrpcClient,
				Pool:        &pool,
				HookAddress: hookAddr,
				BlockNumber: big.NewInt(int64(blockNumber)),
			})
			if err != nil {
				return entity.Pool{}, err
			}
		}
	}

	extraBytes, err := json.Marshal(&Extra{
		Pause:           status,
		SupplyVault:     [2]*shared.VaultState{vaultMap[strings.ToLower(staticExtra.SupplyVault0)], vaultMap[strings.ToLower(staticExtra.SupplyVault1)]},
		BorrowVault:     [3]*shared.VaultState{vaultMap[strings.ToLower(staticExtra.BorrowVault0)], vaultMap[strings.ToLower(staticExtra.BorrowVault1)], vaultMap[strings.ToLower(data.Controller)]},
		ControllerVault: data.Controller,
		Collaterals: lo.Map(data.CollatAmounts, func(v *big.Int, _ int) *uint256.Int {
			return uint256.MustFromBig(v)
		}),
		DynamicParams: dParams,
		HookExtra:     hookExtra,
	})
	if err != nil {
		return entity.Pool{}, err
	}

	pool.Extra = string(extraBytes)
	pool.Timestamp = time.Now().Unix()
	pool.BlockNumber = blockNumber

	return pool, nil
}
