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
	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	var staticExtra StaticExtra
	err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra)
	if err != nil {
		logger.
			WithFields(logger.Fields{"pool_id": p.Address}).
			Error("failed to unmarshal staticExtra")
		return p, err
	}

	vaults := []VaultInfo{
		{
			VaultAddress: staticExtra.Vault0,
			AssetAddress: p.Tokens[0].Address,
			QuoteAmount:  bignumber.TenPowInt(int64(p.Tokens[0].Decimals)),
		},
		{
			VaultAddress: staticExtra.Vault1,
			AssetAddress: p.Tokens[1].Address,
			QuoteAmount:  bignumber.TenPowInt(int64(p.Tokens[1].Decimals)),
		},
	}

	rpcData, blockNumber, err := d.getPoolData(ctx, p.Address, staticExtra.EulerAccount,
		staticExtra.EVC, vaults, overrides)
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
	evc string,
	vaultList []VaultInfo,
	overrides map[common.Address]gethclient.OverrideAccount,
) (TrackerData, *big.Int, error) {
	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	var (
		isOperatorAuthorized bool
		reserves             ReserveRPC
		vaults               = make([]VaultRPC, 2)
		oracles              = make([]common.Address, 2)
		unitOfAccounts       = make([]common.Address, 2)
		ltv                  = make([]uint16, 2)

		accountLiquidities = []AccountLiquidityRPC{
			{
				CollateralValue: big.NewInt(0),
				LiabilityValue:  big.NewInt(0),
			},
			{
				CollateralValue: big.NewInt(0),
				LiabilityValue:  big.NewInt(0),
			},
		}
	)

	req.AddCall(&ethrpc.Call{
		ABI:    evcABI,
		Target: evc,
		Method: evcMethodIsAccountOperatorAuthorized,
		Params: []any{common.HexToAddress(eulerAccount), common.HexToAddress(poolAddress)},
	}, []any{&isOperatorAuthorized})
	req.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: poolAddress,
		Method: poolMethodGetReserves,
		Params: nil,
	}, []any{&reserves})

	for i, v := range vaultList {
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodCash,
			Params: nil,
		}, []any{&vaults[i].Cash})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodDebtOf,
			Params: []any{common.HexToAddress(eulerAccount)},
		}, []any{&vaults[i].Debt})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodMaxDeposit,
			Params: []any{common.HexToAddress(eulerAccount)},
		}, []any{&vaults[i].MaxDeposit})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodCaps,
			Params: nil,
		}, []any{&vaults[i].Caps})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodTotalBorrows,
			Params: nil,
		}, []any{&vaults[i].TotalBorrows})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodTotalAssets,
			Params: nil,
		}, []any{&vaults[i].TotalAssets})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodTotalSupply,
			Params: nil,
		}, []any{&vaults[i].TotalSupply})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodBalanceOf,
			Params: []any{common.HexToAddress(eulerAccount)},
		}, []any{&vaults[i].EulerAccountBalance})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodAccountLiquidity,
			Params: []any{common.HexToAddress(eulerAccount), false},
		}, []any{&accountLiquidities[i]})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodOracle,
			Params: nil,
		}, []any{&oracles[i]})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodUnitOfAccount,
			Params: nil,
		}, []any{&unitOfAccounts[i]})
		req.AddCall(&ethrpc.Call{
			ABI:    vaultABI,
			Target: v.VaultAddress,
			Method: vaultMethodLTVBorrow,
			Params: []any{common.HexToAddress(vaultList[1-i].VaultAddress)},
		}, []any{&ltv[i]})
	}

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return TrackerData{}, nil, err
	}

	getQuotesCalls := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		getQuotesCalls.SetOverrides(overrides)
	}

	getQuotesCalls.SetBlockNumber(resp.BlockNumber)

	var (
		assetQuotes = make([][2]*big.Int, len(vaultList))
		shareQuotes = make([][2]*big.Int, len(vaultList))
	)

	for i, v := range vaultList {
		if oracles[i].Cmp(common.Address{}) == 0 {
			continue
		}

		getQuotesCalls.AddCall(&ethrpc.Call{
			ABI:    routerABI,
			Target: oracles[i].Hex(),
			Method: routerMethodGetQuotes,
			Params: []any{v.QuoteAmount, common.HexToAddress(v.AssetAddress), unitOfAccounts[i]},
		}, []any{&assetQuotes[i]})

		getQuotesCalls.AddCall(&ethrpc.Call{
			ABI:    routerABI,
			Target: oracles[i].Hex(),
			Method: routerMethodGetQuotes,
			Params: []any{v.QuoteAmount, common.HexToAddress(v.VaultAddress), unitOfAccounts[i]},
		}, []any{&shareQuotes[i]})
	}

	resp, err = getQuotesCalls.TryBlockAndAggregate()
	if err != nil {
		return TrackerData{}, nil, err
	}

	var (
		assetPrices = []*big.Int{big.NewInt(0), big.NewInt(0)}
		sharePrices = []*big.Int{big.NewInt(0), big.NewInt(0)}
	)
	for i, v := range vaultList {
		if assetQuotes[i][1] != nil {
			assetPrices[i].Div(assetQuotes[i][1], v.QuoteAmount) // second output is ask price
		}
		if shareQuotes[i][1] != nil {
			sharePrices[i].Div(shareQuotes[i][1], v.QuoteAmount) // second output is ask price
		}
	}

	return TrackerData{
		Vaults:               vaults,
		AssetPrices:          assetPrices,
		SharePrices:          sharePrices,
		AccountLiquidities:   accountLiquidities,
		LTV:                  ltv,
		Reserves:             reserves,
		IsOperatorAuthorized: isOperatorAuthorized,
	}, resp.BlockNumber, nil
}

func (d *PoolTracker) updatePool(pool entity.Pool, data TrackerData, blockNumber *big.Int) (entity.Pool, error) {
	var vaults = make([]Vault, len(data.Vaults))

	allBalancesZero := true

	for i := range data.Vaults {
		totalAssets := uint256.MustFromBig(data.Vaults[i].TotalAssets)
		totalSupply := uint256.MustFromBig(data.Vaults[i].TotalSupply)
		eulerAccountBalance := uint256.MustFromBig(data.Vaults[i].EulerAccountBalance)

		if !eulerAccountBalance.IsZero() {
			allBalancesZero = false
		}

		vaults[i] = Vault{
			Cash:               uint256.MustFromBig(data.Vaults[i].Cash),
			Debt:               uint256.MustFromBig(data.Vaults[i].Debt),
			MaxDeposit:         uint256.MustFromBig(data.Vaults[i].MaxDeposit),
			TotalBorrows:       uint256.MustFromBig(data.Vaults[i].TotalBorrows),
			EulerAccountAssets: convertToAssets(eulerAccountBalance, totalAssets, totalSupply),
			MaxWithdraw:        decodeCap(uint256.NewInt(uint64(data.Vaults[i].Caps[1]))), // index 1 is borrowCap _ used as maxWithdraw
			CollateralValue:    uint256.MustFromBig(data.AccountLiquidities[i].CollateralValue),
			LiabilityValue:     uint256.MustFromBig(data.AccountLiquidities[i].LiabilityValue),
			AssetPrice:         uint256.MustFromBig(data.AssetPrices[i]),
			SharePrice:         uint256.MustFromBig(data.SharePrices[i]),
			TotalAssets:        uint256.MustFromBig(data.Vaults[i].TotalAssets),
			TotalSupply:        uint256.MustFromBig(data.Vaults[i].TotalSupply),
			LTV:                uint256.NewInt(uint64(data.LTV[i])),
		}
	}

	reserve0 := data.Reserves.Reserve0.String()
	reserve1 := data.Reserves.Reserve1.String()
	status := data.Reserves.Status
	if !data.IsOperatorAuthorized || allBalancesZero {
		reserve0 = "0"
		reserve1 = "0"
		status = 2 // locked
	}

	extraBytes, err := json.Marshal(&Extra{
		Pause:  status,
		Vaults: vaults,
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

func convertToAssets(shares, totalAssets, totalSupply *uint256.Int) *uint256.Int {
	shares.MulDivOverflow(shares, totalAssets.Add(totalAssets, VIRTUAL_AMOUNT), totalSupply.Add(totalSupply, VIRTUAL_AMOUNT))
	return shares
}
