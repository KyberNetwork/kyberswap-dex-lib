package upshiftusdf

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
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

	reserves := Reserves(&extra)
	p.Reserves = []string{reserves[0].String(), reserves[1].String()}
	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"dex_id": t.config.DexID, "pool_id": p.Address, "duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("Finished getting new pool state")
	return p, nil
}

// fetchExtra reads the vault in two calls pinned to one block: getSnapshot needs currentPeriod.
func (t *PoolTracker) fetchExtra(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (Extra, uint64, error) {
	var startOfYear, period, minDeposit, maxSupply, totalSupply *big.Int
	usdf := p.Tokens[1].Address

	vaultCall := func(method string, params ...any) *ethrpc.Call {
		return &ethrpc.Call{ABI: vaultABI, Target: p.Address, Method: method, Params: params}
	}
	tokenCall := func(method string) *ethrpc.Call {
		return &ethrpc.Call{ABI: receiptTokenABI, Target: usdf, Method: method}
	}
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).
		AddCall(vaultCall(vaultMethodStartOfYear), []any{&startOfYear}).
		AddCall(vaultCall(vaultMethodPeriod), []any{&period}).
		AddCall(vaultCall(vaultMethodMinDeposit), []any{&minDeposit}).
		AddCall(tokenCall(tokenMethodMaxSupply), []any{&maxSupply}).
		AddCall(tokenCall(tokenMethodTotalSupply), []any{&totalSupply}).
		Aggregate()
	if err != nil {
		return Extra{}, 0, err
	}
	if resp.BlockNumber == nil {
		return Extra{}, 0, ErrInvalidExtra
	}

	var snapshot struct {
		Apr            *big.Int `abi:"apr"`
		TokenPrice     *big.Int `abi:"tokenPrice"`
		TotalDeposited *big.Int `abi:"totalDeposited"`
		DailyInterest  *big.Int `abi:"dailyInterest"`
	}
	if _, err = t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).
		SetBlockNumber(resp.BlockNumber).
		AddCall(vaultCall(vaultMethodSnapshot, period), []any{&snapshot}).
		Aggregate(); err != nil {
		return Extra{}, 0, err
	}

	return Extra{
		StartOfYear:   startOfYear.Uint64(),
		CurrentPeriod: period.Uint64(),
		Apr:           big256.FromBig(snapshot.Apr),
		TokenPrice:    big256.FromBig(snapshot.TokenPrice),
		MinDeposit:    big256.FromBig(minDeposit),
		MaxSupply:     big256.FromBig(maxSupply),
		TotalSupply:   big256.FromBig(totalSupply),
	}, resp.BlockNumber.Uint64(), nil
}
