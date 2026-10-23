package saturnusdat

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
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	extra, blockNumber, err := t.fetchExtra(ctx, p.Address, &staticExtra, overrides)
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

// fetchExtra reads all swap-path state in one multicall pinned to one block.
// USDat (the pool address) holds the MultiMint state; the facility holds the
// pause and approved-extension flags.
func (t *PoolTracker) fetchExtra(
	ctx context.Context,
	usdat string,
	staticExtra *StaticExtra,
	overrides map[common.Address]gethclient.OverrideAccount,
) (Extra, uint64, error) {
	var (
		assetCap, assetBalance, assetDecimals, extDecimals *big.Int
		allowed, usdatPaused, whitelistEnabled             bool
		facilityPaused, extensionApproved                  bool
		executorWhitelisted                                = true
	)
	asset := common.HexToAddress(staticExtra.Asset)

	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides)
	usdatCall := func(method string, params ...any) *ethrpc.Call {
		return &ethrpc.Call{ABI: usdatABI, Target: usdat, Method: method, Params: params}
	}
	facilityCall := func(method string, params ...any) *ethrpc.Call {
		return &ethrpc.Call{ABI: swapFacilityABI, Target: staticExtra.Facility, Method: method, Params: params}
	}
	req.
		AddCall(usdatCall(usdatMethodAssetCap, asset), []any{&assetCap}).
		AddCall(usdatCall(usdatMethodAssetBalanceOf, asset), []any{&assetBalance}).
		AddCall(usdatCall(usdatMethodIsAllowedAsset, asset), []any{&allowed}).
		AddCall(usdatCall(usdatMethodAssetDecimals, asset), []any{&assetDecimals}).
		AddCall(usdatCall(usdatMethodExtensionDecimals), []any{&extDecimals}).
		AddCall(usdatCall(usdatMethodPaused), []any{&usdatPaused}).
		AddCall(usdatCall(usdatMethodIsWhitelistEnabled), []any{&whitelistEnabled}).
		AddCall(facilityCall(facilityMethodPaused), []any{&facilityPaused}).
		AddCall(facilityCall(facilityMethodIsApprovedExtension, common.HexToAddress(usdat)), []any{&extensionApproved})
	if t.config.Executor != "" {
		req.AddCall(usdatCall(usdatMethodIsWhitelisted, common.HexToAddress(t.config.Executor)), []any{&executorWhitelisted})
	}

	resp, err := req.Aggregate()
	if err != nil {
		return Extra{}, 0, err
	}
	var blockNumber uint64
	if resp.BlockNumber != nil {
		blockNumber = resp.BlockNumber.Uint64()
	}

	return Extra{
		AssetCap:          big256.FromBig(assetCap),
		AssetBalance:      big256.FromBig(assetBalance),
		AssetDecimals:     uint8(assetDecimals.Uint64()),
		ExtensionDecimals: uint8(extDecimals.Uint64()),
		AssetAllowed:      allowed,
		FacilityPaused:    facilityPaused,
		USDatPaused:       usdatPaused,
		ExtensionApproved: extensionApproved,
		WhitelistEnabled:  whitelistEnabled,
		NotWhitelisted:    whitelistEnabled && !executorWhitelisted,
	}, blockNumber, nil
}
