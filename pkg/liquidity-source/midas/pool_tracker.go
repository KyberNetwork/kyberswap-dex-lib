package midas

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	config.MTokens = lo.MapEntries(config.MTokens, func(k string, v MTokenConfig) (string, MTokenConfig) {
		return strings.ToLower(k), v
	})

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
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("start getting new state of pool")
	defer func() {
		logger.WithFields(logger.Fields{
			"address": p.Address,
		}).Infof("finished getting new state of pool")
	}()

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var (
		depositVaultState    VaultStateResponse
		redemptionVaultState VaultStateResponse
		tokenRate            *big.Int
		mTokenRate           *big.Int
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	currentDayNumber := time.Now().Unix() / oneDayInSecond
	mToken := p.Tokens[0].Address
	token := p.Tokens[1].Address
	depositVault := t.config.MTokens[mToken].DepositVault
	redemptionVault := t.config.MTokens[p.Address].RedemptionVault

	req.SetContext(ctx).
		SetRequireSuccess(false).
		AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: staticExtra.DataFeed.String(),
			Method: dataFeedGetDataInBase18Method,
		}, []any{&tokenRate}).
		AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: staticExtra.MTokenDataFeed.String(),
			Method: dataFeedGetDataInBase18Method,
		}, []any{&mTokenRate})

	if staticExtra.CanDeposit {
		req = t.addVaultCalls(req, token, depositVault, &depositVaultState, true, currentDayNumber)
	}
	if staticExtra.CanRedeem {
		req = t.addVaultCalls(req, token, redemptionVault, &redemptionVaultState, false, currentDayNumber)

		switch staticExtra.RedemptionVaultType {
		case RedemptionVaultWithSwapper:
			req.AddCall(&ethrpc.Call{
				ABI:    abi.Erc20ABI,
				Target: token,
				Method: abi.Erc20BalanceOfMethod,
				Params: []any{staticExtra.RedemptionVault},
			}, []any{&redemptionVaultState.TokenBalance})
		default:
		}
	}

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}

	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}

	extra := Extra{
		DepositVault:    depositVaultState.ToVaultState(token, mTokenRate, tokenRate),
		RedemptionVault: redemptionVaultState.ToRedemptionVaultState(token, mTokenRate, tokenRate),
		TokenRate:       uint256.MustFromBig(tokenRate),
		MTokenRate:      uint256.MustFromBig(mTokenRate),
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{
		redemptionVaultState.InstantDailyLimit.String(),
		depositVaultState.InstantDailyLimit.String(),
	}
	p.Timestamp = time.Now().Unix()

	return p, nil
}

func (t *PoolTracker) addVaultCalls(req *ethrpc.Request, token, vault string, vaultState *VaultStateResponse,
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
	}, []any{&vaultState.TokenConfig})

	return req
}
