package honey

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
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
	calls.AddCall(&ethrpc.Call{
		ABI:    honeyABI,
		Target: p.Address,
		Method: "numRegisteredAssets",
		Params: []interface{}{},
	}, []interface{}{&numRegisteredAssets})
	calls.AddCall(&ethrpc.Call{
		ABI:    honeyABI,
		Target: p.Address,
		Method: "isBasketModeEnabled",
		Params: []interface{}{true},
	}, []interface{}{&isBasketModeEnabledMint})
	calls.AddCall(&ethrpc.Call{
		ABI:    honeyABI,
		Target: p.Address,
		Method: "isBasketModeEnabled",
		Params: []interface{}{false},
	}, []interface{}{&isBasketModeEnabledRedeem})
	calls.AddCall(&ethrpc.Call{
		ABI:    honeyABI,
		Target: p.Address,
		Method: "forcedBasketMode",
		Params: []interface{}{},
	}, []interface{}{&forcedBasketMode})
	calls.AddCall(&ethrpc.Call{
		ABI:    honeyABI,
		Target: p.Address,
		Method: "polFeeCollectorFeeRate",
		Params: []interface{}{},
	}, []interface{}{&polFeeCollectorFeeRate})
	_, err = calls.Aggregate()
	if err != nil {
		return p, err
	}

	noAssets := int(numRegisteredAssets.Int64())
	registeredAssets := lo.Map(extra.RegisteredAssets, func(item string, _ int) common.Address {
		return common.HexToAddress(item)
	})
	hasNewAssets := false
	if noAssets != len(extra.RegisteredAssets) {
		hasNewAssets = true
	}
	if hasNewAssets {
		// new registered assets
		calls = t.ethrpcClient.NewRequest().SetContext(ctx)
		registeredAssets = make([]common.Address, noAssets)
		for i := 0; i < int(noAssets); i++ {
			calls.AddCall(&ethrpc.Call{
				ABI:    honeyABI,
				Target: p.Address,
				Method: "registeredAssets",
				Params: []interface{}{big.NewInt(int64(i))},
			}, []interface{}{&registeredAssets[i]})
		}
		_, err = calls.Aggregate()
		if err != nil {
			return p, err
		}
	}

	calls = t.ethrpcClient.NewRequest().SetContext(ctx)
	vaults := make([]common.Address, noAssets)
	assetsDecimals := extra.AssetsDecimals
	assetsName := make([]string, noAssets)
	assetsSymbol := make([]string, noAssets)
	if hasNewAssets {
		assetsDecimals = make([]uint8, noAssets)
		for i := 0; i < int(noAssets); i++ {
			calls.AddCall(&ethrpc.Call{
				ABI:    honeyABI,
				Target: p.Address,
				Method: "vaults",
				Params: []interface{}{registeredAssets[i]},
			}, []interface{}{&vaults[i]})
			calls.AddCall(&ethrpc.Call{
				ABI:    erc20ABI,
				Target: registeredAssets[i].Hex(),
				Method: "decimals",
				Params: []interface{}{},
			}, []interface{}{&assetsDecimals[i]})
			calls.AddCall(&ethrpc.Call{
				ABI:    erc20ABI,
				Target: registeredAssets[i].Hex(),
				Method: "name",
				Params: []interface{}{},
			}, []interface{}{&assetsName[i]})
			calls.AddCall(&ethrpc.Call{
				ABI:    erc20ABI,
				Target: registeredAssets[i].Hex(),
				Method: "symbol",
				Params: []interface{}{},
			}, []interface{}{&assetsSymbol[i]})
		}
	}

	isPegged := make([]bool, noAssets)
	isBadCollateral := make([]bool, noAssets)
	mintRates := make([]*big.Int, noAssets)
	redeemRates := make([]*big.Int, noAssets)
	for i := 0; i < int(noAssets); i++ {
		calls.AddCall(&ethrpc.Call{
			ABI:    honeyABI,
			Target: p.Address,
			Method: "isPegged",
			Params: []interface{}{registeredAssets[i]},
		}, []interface{}{&isPegged[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    honeyABI,
			Target: p.Address,
			Method: "isBadCollateralAsset",
			Params: []interface{}{registeredAssets[i]},
		}, []interface{}{&isBadCollateral[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    honeyABI,
			Target: p.Address,
			Method: "mintRates",
			Params: []interface{}{registeredAssets[i]},
		}, []interface{}{&mintRates[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    honeyABI,
			Target: p.Address,
			Method: "redeemRates",
			Params: []interface{}{registeredAssets[i]},
		}, []interface{}{&redeemRates[i]})

	}
	_, err = calls.Aggregate()
	if err != nil {
		return p, err
	}

	vaultsDecimals := extra.VaultsDecimals
	if hasNewAssets {
		calls = t.ethrpcClient.NewRequest().SetContext(ctx)
		vaultsDecimals = make([]uint8, noAssets)
		for i := 0; i < int(noAssets); i++ {
			calls.AddCall(&ethrpc.Call{
				ABI:    assetVaultABI,
				Target: vaults[i].Hex(),
				Method: "decimals",
				Params: []interface{}{},
			}, []interface{}{&vaultsDecimals[i]})
		}
		_, err = calls.Aggregate()
		if err != nil {
			return p, err
		}
	}
	extraBytes, err := json.Marshal(Extra{
		RegisteredAssets: lo.Map(registeredAssets, func(item common.Address, _ int) string {
			return strings.ToLower(item.Hex())
		}),
		ForceBasketMode:        forcedBasketMode,
		IsBasketEnabledMint:    isBasketModeEnabledMint,
		IsBasketEnabledRedeem:  isBasketModeEnabledRedeem,
		IsPegged:               isPegged,
		IsBadCollateral:        isBadCollateral,
		PolFeeCollectorFeeRate: uint256.MustFromBig(polFeeCollectorFeeRate),
		MintRates: lo.Map(mintRates, func(item *big.Int, _ int) *uint256.Int {
			return uint256.MustFromBig(item)
		}),
		RedeemRates: lo.Map(redeemRates, func(item *big.Int, _ int) *uint256.Int {
			return uint256.MustFromBig(item)
		}),
		VaultsDecimals: vaultsDecimals,
		AssetsDecimals: assetsDecimals,
	})
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	if hasNewAssets {
		for i := range registeredAssets {
			if _, ok := lo.Find(p.Tokens, func(item *entity.PoolToken) bool {
				return strings.EqualFold(item.Address, registeredAssets[i].Hex())
			}); !ok {
				p.Reserves = append(p.Reserves, defaultReserves)
				p.Tokens = append(p.Tokens, &entity.PoolToken{
					Address:   strings.ToLower(registeredAssets[i].Hex()),
					Name:      strings.ToLower(assetsName[i]),
					Symbol:    strings.ToLower(assetsSymbol[i]),
					Decimals:  assetsDecimals[i],
					Swappable: true,
				})
			}
		}
	}
	p.Timestamp = time.Now().Unix()
	logger.WithFields(logger.Fields{
		"exchange": p.Exchange,
		"address":  p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}
