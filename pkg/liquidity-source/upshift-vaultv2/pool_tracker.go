package upshiftvaultv2

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

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	startTime := time.Now()
	logger.WithFields(logger.Fields{"dex_id": t.config.DexID, "pool_id": p.Address}).Info("Start getting new pool state")

	extra, blockNumber, err := t.fetchExtra(ctx, p, overrides)
	if err != nil {
		logger.WithFields(logger.Fields{"dex_id": t.config.DexID, "pool_id": p.Address, "error": err}).
			Error("failed to get pool state")
		return p, err
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	a, l := extra.Reserves()
	p.Reserves = []string{a.String(), l.String()}
	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"dex_id": t.config.DexID, "pool_id": p.Address, "duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("Finished getting new pool state")
	return p, nil
}

// fetchExtra reads the vault in one multicall, then the whitelist valuation pinned to the same
// block. The valuation is an eth_call from the vault: it reads msg.sender's balances.
func (t *PoolTracker) fetchExtra(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (Extra, uint64, error) {
	var (
		paused                                                   bool
		sendersWL, assetsWL                                      common.Address
		externalAssets, maxDeposit, depositCap, sup, totalAssets *big.Int
	)
	vaultCall := func(method string) *ethrpc.Call {
		return &ethrpc.Call{ABI: vaultABI, Target: p.Address, Method: method}
	}
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).
		AddCall(vaultCall(vaultMethodDepositsPaused), []any{&paused}).
		AddCall(vaultCall(vaultMethodSendersWL), []any{&sendersWL}).
		AddCall(vaultCall(vaultMethodAssetsWL), []any{&assetsWL}).
		AddCall(vaultCall(vaultMethodExternalAssets), []any{&externalAssets}).
		AddCall(vaultCall(vaultMethodMaxDeposit), []any{&maxDeposit}).
		AddCall(vaultCall(vaultMethodDepositCap), []any{&depositCap}).
		AddCall(vaultCall(vaultMethodTotalAssets), []any{&totalAssets}).
		AddCall(&ethrpc.Call{ABI: erc20ABI, Target: p.Tokens[1].Address, Method: erc20MethodTotalSupply}, []any{&sup}).
		Aggregate()
	if err != nil {
		return Extra{}, 0, err
	}
	if resp.BlockNumber == nil {
		return Extra{}, 0, ErrInvalidExtra
	}

	var valuation *big.Int
	if _, err = t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).
		SetBlockNumber(resp.BlockNumber).SetFrom(common.HexToAddress(p.Address)).
		AddCall(&ethrpc.Call{
			ABI: assetsWhitelistABI, Target: hexutil.Encode(assetsWL[:]), Method: wlMethodValuation, Params: []any{externalAssets},
		}, []any{&valuation}).
		Call(); err != nil {
		return Extra{}, 0, err
	}

	return Extra{
		DepositsPaused: paused,
		SenderGated:    sendersWL != (common.Address{}),
		Supply:         big256.FromBig(sup),
		Valuation:      big256.FromBig(valuation),
		TotalAssets:    big256.FromBig(totalAssets),
		MaxDeposit:     big256.FromBig(maxDeposit),
		DepositCap:     big256.FromBig(depositCap),
	}, resp.BlockNumber.Uint64(), nil
}
