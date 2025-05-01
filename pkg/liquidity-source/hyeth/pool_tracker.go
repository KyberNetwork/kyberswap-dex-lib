package hyeth

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
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

type issuanceSettings struct {
	MaxManagerFee       *big.Int
	ManagerIssueFee     *big.Int
	ManagerRedeemFee    *big.Int
	FeeRecipient        common.Address
	ManagerIssuanceHook common.Address
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

	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return p, err
	}

	var issuanceModule common.Address
	var issuanceSettings issuanceSettings
	var components, externalPositionModules []common.Address
	var totalSupply, totalAsset, componentHyethBalance, maxDeposit, maxRedeem, defaultPositionRealUnit, hyethTotalSupply *big.Int
	calls := t.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: p.Address,
		Method: "issuanceModule",
		Params: []interface{}{},
	}, []interface{}{&issuanceModule}).
		AddCall(&ethrpc.Call{
			ABI:    hyethABI,
			Target: hyethToken,
			Method: "getComponents",
			Params: []interface{}{},
		}, []interface{}{&components}).
		AddCall(&ethrpc.Call{
			ABI:    hyethABI,
			Target: hyethToken,
			Method: "totalSupply",
			Params: []interface{}{},
		}, []interface{}{&hyethTotalSupply})
	_, err := calls.Aggregate()
	if err != nil {
		return p, err
	}

	if len(components) != 1 {
		// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/develop/src/contracts/executor-helpers/ExecutorHelper9.sol#L473
		extraBytes, err := json.Marshal(Extra{
			IsDisabled: true,
		})

		if err != nil {
			return p, err
		}
		p.Extra = string(extraBytes)
		p.Timestamp = time.Now().Unix()
		logger.WithFields(logger.Fields{
			"exchange": p.Exchange,
			"address":  p.Address,
		}).Infof("[%s] Finish getting new state of pool", p.Type)

		return p, nil
	}

	calls = t.ethrpcClient.NewRequest().SetContext(ctx)
	calls.
		AddCall(&ethrpc.Call{
			ABI:    issuanceModuleABI,
			Target: issuanceModule.Hex(),
			Method: "issuanceSettings",
			Params: []interface{}{common.HexToAddress(hyethToken)},
		}, []interface{}{&issuanceSettings}).
		AddCall(&ethrpc.Call{
			ABI:    hyethABI,
			Target: hyethToken,
			Method: "getDefaultPositionRealUnit",
			Params: []interface{}{components[0]},
		}, []interface{}{&defaultPositionRealUnit}).
		AddCall(&ethrpc.Call{
			ABI:    hyethABI,
			Target: hyethToken,
			Method: "getExternalPositionModules",
			Params: []interface{}{components[0]},
		}, []interface{}{&externalPositionModules}).
		AddCall(&ethrpc.Call{
			ABI:    hyethComponent4626ABI,
			Target: components[0].Hex(),
			Method: "totalSupply",
		}, []interface{}{&totalSupply}).
		AddCall(&ethrpc.Call{
			ABI:    hyethComponent4626ABI,
			Target: components[0].Hex(),
			Method: "totalAssets",
		}, []interface{}{&totalAsset}).
		AddCall(&ethrpc.Call{
			ABI:    hyethComponent4626ABI,
			Target: components[0].Hex(),
			Method: "balanceOf",
			Params: []interface{}{common.HexToAddress(hyethToken)},
		}, []interface{}{&componentHyethBalance}).
		AddCall(&ethrpc.Call{
			ABI:    hyethComponent4626ABI,
			Target: components[0].Hex(),
			Method: "maxDeposit",
			Params: []interface{}{eth.AddressZero},
		}, []interface{}{&maxDeposit}).
		AddCall(&ethrpc.Call{
			ABI:    hyethComponent4626ABI,
			Target: components[0].Hex(),
			Method: "maxRedeem",
			Params: []interface{}{eth.AddressZero},
		}, []interface{}{&maxRedeem})

	_, err = calls.Aggregate()
	if err != nil {
		return p, err
	}

	externalPositionRealUnits := make([]*big.Int, len(externalPositionModules))
	calls = t.ethrpcClient.NewRequest().SetContext(ctx)
	for i, module := range externalPositionModules {
		calls.AddCall(&ethrpc.Call{
			ABI:    hyethABI,
			Target: hyethToken,
			Method: "getExternalPositionRealUnit",
			Params: []interface{}{components[0], module},
		}, []interface{}{&externalPositionRealUnits[i]})
	}

	_, err = calls.Aggregate()
	if err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(Extra{
		ManagerIssueFee:         uint256.MustFromBig(issuanceSettings.ManagerIssueFee),
		ManagerRedeemFee:        uint256.MustFromBig(issuanceSettings.ManagerRedeemFee),
		Component:               components[0],
		ComponentTotalSupply:    uint256.MustFromBig(totalSupply),
		ComponentTotalAsset:     uint256.MustFromBig(totalAsset),
		DefaultPositionRealUnit: uint256.MustFromBig(defaultPositionRealUnit),
		ComponentHyethBalance:   uint256.MustFromBig(componentHyethBalance),
		HyethTotalSupply:        uint256.MustFromBig(hyethTotalSupply),
		MaxDeposit:              lo.Ternary(maxDeposit != nil && maxDeposit.Sign() > 0, uint256.MustFromBig(maxDeposit), number.MaxU256),
		MaxRedeem:               lo.Ternary(maxRedeem != nil && maxRedeem.Sign() > 0, uint256.MustFromBig(maxRedeem), number.MaxU256),
		ExternalPositionRealUnits: lo.Map(externalPositionRealUnits, func(item *big.Int, _ int) *uint256.Int {
			return uint256.MustFromBig(item)
		}),
		IsDisabled: false,
	})

	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{totalSupply.String(), totalAsset.String()}
	p.Timestamp = time.Now().Unix()
	logger.WithFields(logger.Fields{
		"exchange": p.Exchange,
		"address":  p.Address,
	}).Infof("[%s] Finish getting new state of pool", p.Type)

	return p, nil
}
