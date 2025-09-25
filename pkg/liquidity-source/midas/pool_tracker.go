package midas

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client

	redemptionVaultToType map[string]VaultType
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	configByte, ok := bytesByPath[config.ConfigPath]
	if !ok {
		return nil, errors.New("misconfigured config path")
	}

	var mTokenConfigs map[string]MTokenConfig
	if err := json.Unmarshal(configByte, &mTokenConfigs); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to unmarshal config")
		return nil, err
	}

	redemptionVaultToType := make(map[string]VaultType)

	for _, cfg := range mTokenConfigs {
		redemptionVaultToType[strings.ToLower(cfg.RedemptionVault)] = cfg.RedemptionVaultType
	}

	return &PoolTracker{
		config:                config,
		ethrpcClient:          ethrpcClient,
		redemptionVaultToType: redemptionVaultToType,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	_ pool.GetNewPoolStateParams) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	lg := logger.WithFields(logger.Fields{
		"pool": p.Address,
	})

	lg.Infof("start getting new state of pool")
	defer func() {
		lg.Infof("finished getting new state of pool")
	}()

	currentDayNumber := time.Now().Unix() / oneDayInSecond
	mToken := p.Tokens[0].Address
	token := p.Tokens[1].Address
	vault := strings.ToLower(staticExtra.Vault)

	var (
		vaultState *VaultStateRpcResult
		err        error
	)
	if staticExtra.IsDv {
		vaultState, err = t.getDvState(ctx, vault, mToken, token, currentDayNumber)
	} else {
		rvCfg, ok := rvConfigs[vault]
		if !ok {
			lg.Errorf("failed to find redemption vault config")
			return p, nil
		}
		vaultState, err = t.getRvState(ctx, rvCfg, token, currentDayNumber)
	}
	if err != nil {
		return p, nil
	}

	extraBytes, err := json.Marshal(vaultState.ToVaultState(staticExtra.VaultType, token))
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)

	p.Reserves = entity.PoolReserves{
		convertFromBase18(uint256.MustFromBig(vaultState.InstantDailyLimit), p.Tokens[0].Decimals).String(),
		convertFromBase18(uint256.MustFromBig(vaultState.TokenConfig.Allowance), p.Tokens[1].Decimals).String(),
	}

	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (t *PoolTracker) initVaultCalls(req *ethrpc.Request, vault, token string, vaultState *VaultStateRpcResult,
	isDepositVault bool, currentDayNumber int64) *ethrpc.Request {

	vaultAbi := lo.Ternary(isDepositVault, DepositVaultABI, RedemptionVaultABI)
	fnSelector := lo.Ternary(isDepositVault, depositInstantSelector, redeemInstantSelector)

	req.AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vaultGetPaymentTokensMethod,
	}, []any{&vaultState.PaymentTokens}).AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vaultPausedMethod,
	}, []any{&vaultState.Paused}).AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vaultFnPausedMethod,
		Params: []any{fnSelector},
	}, []any{&vaultState.FnPaused}).AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vaultInstantDailyLimitMethod,
	}, []any{&vaultState.InstantDailyLimit}).AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vaultDailyLimitsMethod,
		Params: []any{big.NewInt(currentDayNumber)},
	}, []any{&vaultState.DailyLimits}).AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vaultInstantFeeMethod,
	}, []any{&vaultState.InstantFee}).AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vaultMinAmountMethod,
	}, []any{&vaultState.MinAmount}).AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vaultTokensConfigMethod,
		Params: []any{common.HexToAddress(token)},
	}, []any{&vaultState.TokenConfig}).AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vaultMTokenDataFeedMethod,
	}, []any{&vaultState.MTokenDataFeed}).AddCall(&ethrpc.Call{
		ABI:    vaultAbi,
		Target: vault,
		Method: vaultWaivedFeeRestrictionMethod,
		Params: []any{common.HexToAddress(t.config.Executor)},
	}, []any{&vaultState.WaivedFeeRestriction})

	return req
}

func (t *PoolTracker) getDvState(ctx context.Context, vault, mToken, token string,
	currentDayNumber int64) (*VaultStateRpcResult, error) {
	var vaultStateResult VaultStateRpcResult
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req = t.initVaultCalls(req, vault, token, &vaultStateResult, true, currentDayNumber)
	req.AddCall(&ethrpc.Call{
		ABI:    DepositVaultABI,
		Target: vault,
		Method: dvMinMTokenAmountForFirstDepositMethod,
	}, []any{&vaultStateResult.MinMTokenAmountForFirstDeposit}).AddCall(&ethrpc.Call{
		ABI:    DepositVaultABI,
		Target: vault,
		Method: dvTotalMintedMethod,
		Params: []any{common.HexToAddress(t.config.Executor)},
	}, []any{&vaultStateResult.TotalMinted}).AddCall(&ethrpc.Call{
		ABI:    DepositVaultABI,
		Target: vault,
		Method: dvMaxSupplyCapMethod,
	}, []any{&vaultStateResult.MaxSupplyCap}).AddCall(&ethrpc.Call{
		ABI:    abi.Erc20ABI,
		Target: mToken,
		Method: abi.Erc20TotalSupplyMethod,
	}, []any{&vaultStateResult.MTokenTotalSupply})

	resp, err := req.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate dv state, mToken %v, vault %v", mToken, vault)
		return nil, err
	}

	if _, err = t.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		SetRequireSuccess(false).
		SetBlockNumber(resp.BlockNumber).
		AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: vaultStateResult.TokenConfig.DataFeed.String(),
			Method: dataFeedGetDataInBase18Method,
		}, []any{&vaultStateResult.TokenRate}).
		AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: vaultStateResult.MTokenDataFeed.String(),
			Method: dataFeedGetDataInBase18Method,
		}, []any{&vaultStateResult.MTokenRate}).
		Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate data feed calls")
		return nil, err
	}

	return &vaultStateResult, nil
}

func (t *PoolTracker) getRvState(ctx context.Context, vaultCfg rvConfig, token string,
	currentDayNumber int64) (*VaultStateRpcResult, error) {
	var vaultStateResult VaultStateRpcResult
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req = t.initVaultCalls(req, vaultCfg.Address, token, &vaultStateResult, true, currentDayNumber)
	req.AddCall(&ethrpc.Call{
		ABI:    abi.Erc20ABI,
		Target: token,
		Method: abi.Erc20BalanceOfMethod,
		Params: []any{common.HexToAddress(vaultCfg.Address)},
	}, []any{&vaultStateResult.TokenBalance})

	switch vaultCfg.RvType {
	case redemptionVault:
	case redemptionVaultSwapper:
		vaultStateResult.SwapperVaultType = rvConfigs[vaultCfg.MTbillRedemptionVault].RvType
		req.AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: vaultCfg.MToken,
			Method: abi.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(vaultCfg.LiquidityProvider)},
		}, []any{&vaultStateResult.MToken1Balance}).AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: rvConfigs[vaultCfg.MTbillRedemptionVault].MToken,
			Method: abi.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(vaultCfg.LiquidityProvider)},
		}, []any{&vaultStateResult.MToken2Balance})
	case redemptionVaultUstb:
		req.AddCall(&ethrpc.Call{
			ABI:    redemptionABI,
			Target: vaultCfg.UstbRedemption,
			Method: redemptionUsdcMethod,
		}, []any{&vaultStateResult.Redemption.Usdc}).
			AddCall(&ethrpc.Call{
				ABI:    redemptionABI,
				Target: vaultCfg.UstbRedemption,
				Method: redemptionRedemptionFeeMethod,
			}, []any{&vaultStateResult.Redemption.RedemptionFee}).
			AddCall(&ethrpc.Call{
				ABI:    abi.Erc20ABI,
				Target: vaultCfg.SuperstateToken,
				Method: abi.Erc20BalanceOfMethod,
				Params: []any{common.HexToAddress(vaultCfg.Address)},
			}, []any{&vaultStateResult.Redemption.UstbBalance}).
			AddCall(&ethrpc.Call{
				ABI:    redemptionABI,
				Target: vaultCfg.UstbRedemption,
				Method: redemptionGetChainlinkPriceMethod,
			}, []any{&vaultStateResult.Redemption.ChainlinkPrice}).
			AddCall(&ethrpc.Call{
				ABI:    redemptionABI,
				Target: vaultCfg.UstbRedemption,
				Method: redemptionChainlinkFeedPrecisionMethod,
			}, []any{&vaultStateResult.Redemption.ChainLinkFeedPrecision}).
			AddCall(&ethrpc.Call{
				ABI:    redemptionABI,
				Target: vaultCfg.UstbRedemption,
				Method: redemptionSuperstateTokenPrecisionMethod,
			}, []any{&vaultStateResult.Redemption.SuperstateTokenPrecision})
	default:
		return nil, ErrNotSupported
	}

	resp, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate rv state, type %v, vault %v", vaultCfg.RvType, vaultCfg.Address)
		return nil, err
	}

	if _, err = t.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		SetRequireSuccess(false).
		SetBlockNumber(resp.BlockNumber).
		AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: vaultStateResult.TokenConfig.DataFeed.String(),
			Method: dataFeedGetDataInBase18Method,
		}, []any{&vaultStateResult.TokenRate}).
		AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: vaultStateResult.MTokenDataFeed.String(),
			Method: dataFeedGetDataInBase18Method,
		}, []any{&vaultStateResult.MTokenRate}).
		Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate data feed calls")
		return nil, err
	}

	if vaultCfg.RvType == redemptionVaultSwapper {
		vaultStateResult.MTbillRedemptionVault, err = t.getRvState(ctx, rvConfigs[vaultCfg.MTbillRedemptionVault],
			token, currentDayNumber)
		if err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Warnf("failed to aggregate mTbillRedemptionVault for rv swapper")
		}
	}

	return &vaultStateResult, nil
}
