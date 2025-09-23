package midas

import (
	"context"
	"errors"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
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

	redemptionVaultToType map[string]string
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

	redemptionVaultToType := make(map[string]string)

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
	lg := logger.WithFields(logger.Fields{"pool": p.Address})

	lg.Infof("start getting new state of pool")
	defer func() {
		lg.Infof("finished getting new state of pool")
	}()

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var (
		extraBytes []byte
	)

	currentDayNumber := time.Now().Unix() / oneDayInSecond
	token := p.Tokens[1].Address

	if staticExtra.IsDepositVault {
		depositVaultState, err := t.getVaultState(ctx, token, p.Address, true, currentDayNumber)
		if err != nil {
			return p, err
		}
		if extraBytes, err = json.Marshal(depositVaultState); err != nil {
			return p, err
		}
	} else {
		switch staticExtra.VaultType {
		case redemptionVault:
			redemptionVaultState, err := t.getVaultState(ctx, token, p.Address, false, currentDayNumber)
			if err != nil {
				return p, err
			}
			if extraBytes, err = json.Marshal(redemptionVaultState); err != nil {
				return p, err
			}
		case redemptionVaultSwapper:
			redemptionVaultState, err := t.getRedemptionVaultWithSwapperState(ctx, token, p.Address, currentDayNumber)
			if err != nil {
				return p, err
			}
			if extraBytes, err = json.Marshal(redemptionVaultState); err != nil {
				return p, err
			}
		case redemptionVaultUstb:
			redemptionVaultState, err := t.getRedemptionVaultWithUstbState(ctx, token, p.Address, currentDayNumber)
			if err != nil {
				return p, err
			}
			if extraBytes, err = json.Marshal(redemptionVaultState); err != nil {
				return p, err
			}
		default:
		}
	}

	p.Extra = string(extraBytes)

	p.Reserves = entity.PoolReserves{
		"1000000000000000000", "10000000000000000000000",
	}

	p.Timestamp = time.Now().Unix()

	return p, nil
}

func getVaultStateCalls(req *ethrpc.Request, token, vault string, vaultState *VaultStateResponse,
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
	}, []any{&vaultState.MTokenDataFeed})

	return req
}

func (t *PoolTracker) getVaultState(ctx context.Context, token, vault string, isDeposit bool, currentDayNumber int64) (*VaultState, error) {
	var vaultStateResponse VaultStateResponse
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req = getVaultStateCalls(req, token, vault, &vaultStateResponse, isDeposit, currentDayNumber)

	resp, err := req.Aggregate()
	if err != nil {
		return nil, err
	}

	var mTokenRate, tokenRate *big.Int
	if _, err := t.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		SetRequireSuccess(false).
		SetBlockNumber(resp.BlockNumber).
		AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: vaultStateResponse.TokenConfig.DataFeed.String(),
			Method: dataFeedGetDataInBase18Method,
		}, []any{&tokenRate}).
		AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: vaultStateResponse.MTokenDataFeed.String(),
			Method: dataFeedGetDataInBase18Method,
		}, []any{&mTokenRate}).
		Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate data feed calls")
		return nil, err
	}

	return vaultStateResponse.ToVaultState(token, mTokenRate, tokenRate), nil
}

func addRedemptionVaultStateCalls(req *ethrpc.Request, token, vault string, vaultState *VaultStateResponse,
	currentDayNumber int64, vaultType string) *ethrpc.Request {
	req = getVaultStateCalls(req, token, vault, vaultState, false, currentDayNumber)
	req.AddCall(&ethrpc.Call{
		ABI:    abi.Erc20ABI,
		Target: token,
		Method: abi.Erc20BalanceOfMethod,
		Params: []any{common.HexToAddress(vault)},
	}, []any{&vaultState.TokenBalance})

	switch vaultType {
	case redemptionVaultSwapper:
		req.AddCall(&ethrpc.Call{
			ABI:    RedemptionVaultABI,
			Target: vault,
			Method: redemptionVaultSwapperMTbillRedemptionVaultMethod,
		}, []any{&vaultState.MTbillRedemptionVault})
	case redemptionVaultUstb:
		req.AddCall(&ethrpc.Call{
			ABI:    redemptionVaultWithUstbABI,
			Target: vault,
			Method: redemptionVaultUstbUstbRedemptionMethod,
		}, []any{&vaultState.UstbRedemption})
	default:
	}

	return req
}

func (t *PoolTracker) getRedemptionVaultWithUstbState(
	ctx context.Context,
	token string,
	redemptionVault string,
	currentDayNumber int64,
) (*RedemptionVaultWithUstbState, error) {
	var vaultResponse VaultStateResponse

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req = getVaultStateCalls(req, token, redemptionVault, &vaultResponse, false, currentDayNumber)
	req = addRedemptionVaultStateCalls(req, token, redemptionVault, &vaultResponse, currentDayNumber, redemptionVaultUstb)
	resp, err := req.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate redemption vault with ustb %v", redemptionVault)
		return nil, err
	}

	blockNumber := resp.BlockNumber
	ustbRedemption := vaultResponse.UstbRedemption.String()

	var superstateToken common.Address
	if _, err = t.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    redemptionABI,
			Target: ustbRedemption,
			Method: redemptionSuperstateTokenMethod,
		}, []any{&superstateToken}).
		Call(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get superstate token, vault %v, ustbRedemption %v", redemptionVault, ustbRedemption)
		return nil, err
	}

	var (
		usdc           common.Address
		redemptionFee  *big.Int
		ustbBalance    *big.Int
		chainlinkPrice struct {
			IsBadData bool
			UpdatedAt *big.Int
			Price     *big.Int
		}
		chainLinkFeedPrecision   *big.Int
		superstateTokenPrecision *big.Int
	)

	var mTokenRate, tokenRate *big.Int
	if _, err = t.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		SetRequireSuccess(false).
		SetBlockNumber(blockNumber).
		AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: vaultResponse.TokenConfig.DataFeed.String(),
			Method: dataFeedGetDataInBase18Method,
		}, []any{&tokenRate}).
		AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: vaultResponse.MTokenDataFeed.String(),
			Method: dataFeedGetDataInBase18Method,
		}, []any{&mTokenRate}).
		AddCall(&ethrpc.Call{
			ABI:    redemptionABI,
			Target: ustbRedemption,
			Method: redemptionUsdcMethod,
		}, []any{&usdc}).
		AddCall(&ethrpc.Call{
			ABI:    redemptionABI,
			Target: ustbRedemption,
			Method: redemptionRedemptionFeeMethod,
		}, []any{&redemptionFee}).
		AddCall(&ethrpc.Call{
			ABI:    abi.Erc20ABI,
			Target: superstateToken.String(),
			Method: abi.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(redemptionVault)},
		}, []any{&ustbBalance}).
		AddCall(&ethrpc.Call{
			ABI:    redemptionABI,
			Target: ustbRedemption,
			Method: redemptionGetChainlinkPriceMethod,
		}, []any{&chainlinkPrice}).
		AddCall(&ethrpc.Call{
			ABI:    redemptionABI,
			Target: ustbRedemption,
			Method: redemptionChainlinkFeedPrecisionMethod,
		}, []any{&chainLinkFeedPrecision}).
		AddCall(&ethrpc.Call{
			ABI:    redemptionABI,
			Target: ustbRedemption,
			Method: redemptionSuperstateTokenPrecisionMethod,
		}, []any{&superstateTokenPrecision}).
		TryBlockAndAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get redemption state for %v", ustbRedemption)
		return nil, err
	}

	return &RedemptionVaultWithUstbState{
		VaultState: *vaultResponse.ToVaultState(token, mTokenRate, tokenRate),
		UstbRedemptionState: &RedemptionState{
			SuperstateToken: superstateToken,
			USDC:            usdc,
			RedemptionFee:   uint256.MustFromBig(redemptionFee),
			UstbBalance:     uint256.MustFromBig(ustbBalance),
			ChainlinkPrice: &ChainlinkPrice{
				IsBadData: chainlinkPrice.IsBadData,
				UpdatedAt: uint256.MustFromBig(chainlinkPrice.UpdatedAt),
				Price:     uint256.MustFromBig(chainlinkPrice.Price),
			},
			ChainLinkFeedPrecision:   uint256.MustFromBig(chainLinkFeedPrecision),
			SuperstateTokenPrecision: uint256.MustFromBig(superstateTokenPrecision),
		},
	}, nil
}

func (t *PoolTracker) getRedemptionVaultWithSwapperState(
	ctx context.Context, token string, redemptionVault string, currentDayNumber int64,
) (*RedemptionVaultWithSwapperState, error) {
	var vaultResponse VaultStateResponse

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req = getVaultStateCalls(req, token, redemptionVault, &vaultResponse, false, currentDayNumber)
	req = addRedemptionVaultStateCalls(req, token, redemptionVault, &vaultResponse, currentDayNumber, redemptionVaultSwapper)
	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, err
	}

	var mTokenRate, tokenRate *big.Int
	if _, err = t.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		SetRequireSuccess(false).
		SetBlockNumber(resp.BlockNumber).
		AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: vaultResponse.TokenConfig.DataFeed.String(),
			Method: dataFeedGetDataInBase18Method,
		}, []any{&tokenRate}).
		AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: vaultResponse.MTokenDataFeed.String(),
			Method: dataFeedGetDataInBase18Method,
		}, []any{&mTokenRate}).
		Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate data feed calls")
		return nil, err
	}

	var mTbillRedemptionVaultState *RedemptionVaultWithUstbState
	mTbillRedemptionVault := vaultResponse.MTbillRedemptionVault.String()

	if !strings.EqualFold(mTbillRedemptionVault, dummyAddress) &&
		!strings.EqualFold(mTbillRedemptionVault, eth.AddressZero.String()) {

		rVaultType, ok := t.redemptionVaultToType[strings.ToLower(mTbillRedemptionVault)]
		if !ok {
			logger.Warnf("unknown redemption vault type %v", mTbillRedemptionVault)
		} else if rVaultType == redemptionVaultUstb {
			mTbillRedemptionVaultState, err = t.getRedemptionVaultWithUstbState(ctx, token, mTbillRedemptionVault, currentDayNumber)
			if err != nil {
				logger.WithFields(logger.Fields{
					"error": err,
				}).Warnf("failed to get redemption vault state for %v, base vault %v", mTbillRedemptionVault, redemptionVault)
			}
		}
	}

	return &RedemptionVaultWithSwapperState{
		VaultState:            *vaultResponse.ToVaultState(token, mTokenRate, tokenRate),
		MTbillRedemptionVault: mTbillRedemptionVaultState,
	}, nil
}
