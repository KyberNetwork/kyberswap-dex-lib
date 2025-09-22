package midas

import (
	"context"
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

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	_ pool.GetNewPoolStateParams) (entity.Pool, error) {
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
		mTokenRate           *big.Int
		tokenRate            *big.Int
	)

	currentDayNumber := time.Now().Unix() / oneDayInSecond
	token := p.Tokens[1].Address

	req := t.ethrpcClient.
		NewRequest().SetContext(ctx).
		SetRequireSuccess(false).
		AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: staticExtra.DataFeed,
			Method: dataFeedGetDataInBase18Method,
		}, []any{&tokenRate}).
		AddCall(&ethrpc.Call{
			ABI:    dataFeedABI,
			Target: staticExtra.MTokenDataFeed,
			Method: dataFeedGetDataInBase18Method,
		}, []any{&mTokenRate})

	if staticExtra.CanDeposit {
		req = t.addVaultCalls(req, token, staticExtra.DepositVault, &depositVaultState, true, currentDayNumber)
	}
	if staticExtra.CanRedeem {
		req = t.addVaultCalls(req, token, staticExtra.RedemptionVault, &redemptionVaultState, false, currentDayNumber)
	}

	resp, err := req.TryAggregate()
	if err != nil {
		logger.Errorf("failed to aggregate new pool state")
		return p, err
	}

	blockNumber := resp.BlockNumber

	skipInitExtra := false
	if staticExtra.CanRedeem {
		req = t.addVaultCalls(req, token, staticExtra.RedemptionVault, &redemptionVaultState, false, currentDayNumber)
		switch staticExtra.RedemptionVaultType {
		case redemptionVaultSwapper:
			logger.Infof("pools %v", p.Address)
			var (
				mTbillRedemptionVault common.Address
				tokenOutBalance       *big.Int
			)
			_, err = t.ethrpcClient.
				NewRequest().
				SetContext(ctx).
				SetBlockNumber(blockNumber).
				AddCall(&ethrpc.Call{
					ABI:    RedemptionVaultABI,
					Target: staticExtra.RedemptionVault,
					Method: redemptionVaultSwapperMTbillRedemptionVaultMethod,
				}, []any{&mTbillRedemptionVault}).
				AddCall(&ethrpc.Call{
					ABI:    abi.Erc20ABI,
					Target: token,
					Method: abi.Erc20BalanceOfMethod,
					Params: []any{common.HexToAddress(staticExtra.RedemptionVault)},
				}, []any{&tokenOutBalance}).
				TryAggregate()
			if err != nil {
				return p, err
			}

			var (
				mTbillRedemptionVaultState VaultStateResponse
				ustbRedemption             common.Address

				superstateToken common.Address
			)

			req = t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
			req = t.addVaultCalls(req, token, mTbillRedemptionVault.String(), &mTbillRedemptionVaultState, false, currentDayNumber)
			req.AddCall(&ethrpc.Call{
				ABI:    redemptionVaultWithUstbABI,
				Target: mTbillRedemptionVault.String(),
				Method: redemptionVaultUstbUstbRedemptionMethod,
			}, []any{&ustbRedemption})
			_, err = req.TryAggregate()
			if err != nil {
				return p, err
			}

			if _, err = t.ethrpcClient.
				NewRequest().
				SetContext(ctx).
				SetBlockNumber(blockNumber).
				AddCall(&ethrpc.Call{
					ABI:    redemptionABI,
					Target: ustbRedemption.String(),
					Method: redemptionSuperstateTokenMethod,
				}, []any{&superstateToken}).
				Call(); err != nil {
				return p, err
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
			req = t.ethrpcClient.
				NewRequest().
				SetContext(ctx).
				SetBlockNumber(blockNumber).
				AddCall(&ethrpc.Call{
					ABI:    redemptionABI,
					Target: ustbRedemption.String(),
					Method: redemptionUsdcMethod,
				}, []any{&usdc}).
				AddCall(&ethrpc.Call{
					ABI:    redemptionABI,
					Target: ustbRedemption.String(),
					Method: redemptionRedemptionFeeMethod,
				}, []any{&redemptionFee}).
				AddCall(&ethrpc.Call{
					ABI:    abi.Erc20ABI,
					Target: superstateToken.String(),
					Method: abi.Erc20BalanceOfMethod,
					Params: []any{mTbillRedemptionVault},
				}, []any{&ustbBalance}).
				AddCall(&ethrpc.Call{
					ABI:    redemptionABI,
					Target: ustbRedemption.String(),
					Method: redemptionGetChainlinkPriceMethod,
				}, []any{&chainlinkPrice}).
				AddCall(&ethrpc.Call{
					ABI:    redemptionABI,
					Target: ustbRedemption.String(),
					Method: redemptionChainlinkFeedPrecisionMethod,
				}, []any{&chainLinkFeedPrecision}).
				AddCall(&ethrpc.Call{
					ABI:    redemptionABI,
					Target: ustbRedemption.String(),
					Method: redemptionSuperstateTokenPrecisionMethod,
				}, []any{&superstateTokenPrecision})
			_, err = req.TryAggregate()
			if err != nil {
				return p, err
			}

			extra := Extra[RedemptionVaultWithSwapperState]{
				DepositVault: depositVaultState.ToVaultState(token, mTokenRate, tokenRate),
				RedemptionVault: &RedemptionVaultWithSwapperState{
					VaultState:   *redemptionVaultState.ToVaultState(token, mTokenRate, tokenRate),
					TokenBalance: uint256.MustFromBig(tokenOutBalance),
					MTbillRedemptionVault: &RedemptionVaultWithUSTBState{
						VaultState:      *mTbillRedemptionVaultState.ToVaultState(token, mTokenRate, tokenRate),
						SuperstateToken: superstateToken,
						USDC:            usdc,
						TokenOutBalance: uint256.MustFromBig(tokenOutBalance),
						RedemptionFee:   uint256.MustFromBig(redemptionFee),
						USTBBalance:     uint256.MustFromBig(ustbBalance),
						ChainlinkPrice: &ChainlinkPrice{
							IsBadData: chainlinkPrice.IsBadData,
							UpdatedAt: uint256.MustFromBig(chainlinkPrice.UpdatedAt),
							Price:     uint256.MustFromBig(chainlinkPrice.Price),
						},
						ChainLinkFeedPrecision:   uint256.MustFromBig(chainLinkFeedPrecision),
						SuperstateTokenPrecision: uint256.MustFromBig(superstateTokenPrecision),
					},
				},
			}

			extraBytes, err := json.Marshal(extra)
			if err != nil {
				return p, err
			}
			p.Extra = string(extraBytes)

			skipInitExtra = true
		default:
		}
	}

	if !skipInitExtra {
		extra := Extra[VaultState]{
			DepositVault:    depositVaultState.ToVaultState(token, mTokenRate, tokenRate),
			RedemptionVault: redemptionVaultState.ToVaultState(token, mTokenRate, tokenRate),
		}

		extraBytes, err := json.Marshal(extra)
		if err != nil {
			return p, err
		}
		p.Extra = string(extraBytes)
	}

	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}

	depositInstantDailyLimit := uint256.MustFromBig(depositVaultState.InstantDailyLimit)
	redeemInstantDailyLimit := uint256.MustFromBig(redemptionVaultState.InstantDailyLimit)
	p.Reserves = entity.PoolReserves{
		convertFromBase18(depositInstantDailyLimit, p.Tokens[0].Decimals).String(),
		convertFromBase18(redeemInstantDailyLimit, p.Tokens[1].Decimals).String(),
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
