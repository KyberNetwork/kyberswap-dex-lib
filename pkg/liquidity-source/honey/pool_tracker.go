package honey

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	_ map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[%s] Start getting new state of pool", p.Type)

	extra := Extra{}
	err := json.Unmarshal([]byte(p.Extra), &extra)
	if err != nil {
		return entity.Pool{}, err
	}

	calls := t.ethrpcClient.NewRequest().SetContext(ctx)

	var numRegisteredAssets *big.Int
	var isBasketModeEnabledMint bool
	var isBasketModeEnabledRedeem bool
	var forcedBasketMode bool
	var polFeeCollectorFeeRate *big.Int
	if _, err = calls.AddCall(&ethrpc.Call{
		ABI:    honeyABI,
		Target: p.Address,
		Method: "numRegisteredAssets",
	}, []any{&numRegisteredAssets}).AddCall(&ethrpc.Call{
		ABI:    honeyABI,
		Target: p.Address,
		Method: "isBasketModeEnabled",
		Params: []any{true},
	}, []any{&isBasketModeEnabledMint}).AddCall(&ethrpc.Call{
		ABI:    honeyABI,
		Target: p.Address,
		Method: "isBasketModeEnabled",
		Params: []any{false},
	}, []any{&isBasketModeEnabledRedeem}).AddCall(&ethrpc.Call{
		ABI:    honeyABI,
		Target: p.Address,
		Method: "forcedBasketMode",
	}, []any{&forcedBasketMode}).AddCall(&ethrpc.Call{
		ABI:    honeyABI,
		Target: p.Address,
		Method: "polFeeCollectorFeeRate",
	}, []any{&polFeeCollectorFeeRate}).Aggregate(); err != nil {
		return p, err
	}

	noAssets := int(numRegisteredAssets.Int64())
	registeredAssets := lo.Map(extra.RegisteredAssets,
		func(item string, _ int) common.Address { return common.HexToAddress(item) })
	vaults := lo.Map(extra.Vaults, func(item string, _ int) common.Address { return common.HexToAddress(item) })
	assetsDecimals := extra.AssetsDecimals
	vaultsDecimals := extra.VaultsDecimals
	hasNewAssets := noAssets != len(extra.RegisteredAssets)
	if hasNewAssets { // new registered assets
		registeredAssets = make([]common.Address, noAssets)
		vaults = make([]common.Address, noAssets)
		calls = t.ethrpcClient.NewRequest().SetContext(ctx)
		for i := range noAssets {
			calls.AddCall(&ethrpc.Call{
				ABI:    honeyABI,
				Target: p.Address,
				Method: "registeredAssets",
				Params: []any{big.NewInt(int64(i))},
			}, []any{&registeredAssets[i]})
		}
		if _, err = calls.Aggregate(); err != nil {
			return p, err
		}

		assetsDecimals = make([]uint8, noAssets)
		vaultsDecimals = make([]uint8, noAssets)
		calls = t.ethrpcClient.NewRequest().SetContext(ctx)
		for i := range noAssets {
			calls.AddCall(&ethrpc.Call{
				ABI:    honeyABI,
				Target: p.Address,
				Method: "vaults",
				Params: []any{registeredAssets[i]},
			}, []any{&vaults[i]}).AddCall(&ethrpc.Call{
				ABI:    erc20ABI,
				Target: registeredAssets[i].Hex(),
				Method: "decimals",
				Params: []interface{}{},
			}, []interface{}{&assetsDecimals[i]})
		}
		if _, err = calls.Aggregate(); err != nil {
			return p, err
		}
	}

	isPegged := make([]bool, noAssets)
	isBadCollateral := make([]bool, noAssets)
	mintRates := make([]*big.Int, noAssets)
	redeemRates := make([]*big.Int, noAssets)
	vaultsMaxRedeems := make([]*big.Int, noAssets)
	poolAddress := common.HexToAddress(p.Address)
	for i := range noAssets {
		if hasNewAssets {
			calls.AddCall(&ethrpc.Call{
				ABI:    assetVaultABI,
				Target: vaults[i].Hex(),
				Method: "decimals",
			}, []any{&vaultsDecimals[i]})
		}
		calls.AddCall(&ethrpc.Call{
			ABI:    honeyABI,
			Target: p.Address,
			Method: "isPegged",
			Params: []any{registeredAssets[i]},
		}, []any{&isPegged[i]}).AddCall(&ethrpc.Call{
			ABI:    honeyABI,
			Target: p.Address,
			Method: "isBadCollateralAsset",
			Params: []any{registeredAssets[i]},
		}, []any{&isBadCollateral[i]}).AddCall(&ethrpc.Call{
			ABI:    honeyABI,
			Target: p.Address,
			Method: "mintRates",
			Params: []any{registeredAssets[i]},
		}, []any{&mintRates[i]}).AddCall(&ethrpc.Call{
			ABI:    honeyABI,
			Target: p.Address,
			Method: "redeemRates",
			Params: []any{registeredAssets[i]},
		}, []any{&redeemRates[i]}).AddCall(&ethrpc.Call{
			ABI:    assetVaultABI,
			Target: vaults[i].Hex(),
			Method: "maxRedeem",
			Params: []any{poolAddress},
		}, []any{&vaultsMaxRedeems[i]})
	}
	if _, err = calls.Aggregate(); err != nil {
		return p, err
	}

	extra = Extra{
		RegisteredAssets: lo.Map(registeredAssets, func(item common.Address, _ int) string {
			return hexutil.Encode(item[:])
		}),
		ForceBasketMode:        forcedBasketMode,
		IsBasketEnabledMint:    isBasketModeEnabledMint,
		IsBasketEnabledRedeem:  isBasketModeEnabledRedeem,
		IsPegged:               isPegged,
		IsBadCollateral:        isBadCollateral,
		PolFeeCollectorFeeRate: uint256.MustFromBig(polFeeCollectorFeeRate),
		MintRates: lo.Map(mintRates,
			func(item *big.Int, _ int) *uint256.Int { return uint256.MustFromBig(item) }),
		RedeemRates: lo.Map(redeemRates,
			func(item *big.Int, _ int) *uint256.Int { return uint256.MustFromBig(item) }),
		Vaults: lo.Map(vaults,
			func(item common.Address, _ int) string { return hexutil.Encode(item[:]) }),
		VaultsDecimals: vaultsDecimals,
		AssetsDecimals: assetsDecimals,
		VaultsMaxRedeems: lo.Map(vaultsMaxRedeems,
			func(item *big.Int, _ int) *uint256.Int { return uint256.MustFromBig(item) }),
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Reserves = p.Reserves[:min(1, len(p.Reserves))]
	p.Tokens = p.Tokens[:min(1, len(p.Tokens))]
	var shares, exp uint256.Int
	for i := range registeredAssets {
		shares.MulDivOverflow(extra.VaultsMaxRedeems[i], extra.RedeemRates[i], U1e18)
		if vaultsDecimals[i] >= assetsDecimals[i] {
			shares.Div(&shares, exp.Exp(U10, exp.SetUint64(uint64(vaultsDecimals[i]-assetsDecimals[i]))))
		} else {
			shares.Mul(&shares, exp.Exp(U10, exp.SetUint64(uint64(assetsDecimals[i]-vaultsDecimals[i]))))
		}
		p.Reserves = append(p.Reserves, shares.String())
		p.Tokens = append(p.Tokens, &entity.PoolToken{
			Address:   hexutil.Encode(registeredAssets[i][:]),
			Swappable: true,
		})
	}
	p.Timestamp = time.Now().Unix()
	logger.WithFields(logger.Fields{
		"exchange": p.Exchange,
		"address":  p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}
