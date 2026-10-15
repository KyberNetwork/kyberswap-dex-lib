package saturn

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

const executorAddress = "0x8f10b468b06c6fd214b65f87778827f7d113f996"

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
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
	extra, blockNumber, err := getExtra(ctx, t.ethrpcClient, overrides)
	if err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = blockNumber

	return p, nil
}

func getExtra(
	ctx context.Context,
	client *ethrpc.Client,
	overrides map[common.Address]gethclient.OverrideAccount,
) (PoolExtra, uint64, error) {
	var (
		swapFacilityPaused  bool
		extensionApproved   bool
		usdatPaused         bool
		assetAllowed        bool
		assetCap            *big.Int
		assetBalance        *big.Int
		assetDecimals       uint8
		whitelistEnabled    bool
		executorWhitelisted bool
	)

	calls := client.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}

	calls.AddCall(&ethrpc.Call{ABI: swapFacilityABI, Target: SwapFacilityAddress, Method: swapFacilityMethodPaused}, []any{&swapFacilityPaused})
	calls.AddCall(&ethrpc.Call{
		ABI:    swapFacilityABI,
		Target: SwapFacilityAddress,
		Method: swapFacilityMethodIsApprovedExtension,
		Params: []any{common.HexToAddress(USDatAddress)},
	}, []any{&extensionApproved})

	calls.AddCall(&ethrpc.Call{ABI: usdatABI, Target: USDatAddress, Method: usdatMethodPaused}, []any{&usdatPaused})
	calls.AddCall(&ethrpc.Call{
		ABI:    usdatABI,
		Target: USDatAddress,
		Method: usdatMethodIsAllowedAsset,
		Params: []any{common.HexToAddress(USDCAddress)},
	}, []any{&assetAllowed})
	calls.AddCall(&ethrpc.Call{
		ABI:    usdatABI,
		Target: USDatAddress,
		Method: usdatMethodAssetCap,
		Params: []any{common.HexToAddress(USDCAddress)},
	}, []any{&assetCap})
	calls.AddCall(&ethrpc.Call{
		ABI:    usdatABI,
		Target: USDatAddress,
		Method: usdatMethodAssetBalanceOf,
		Params: []any{common.HexToAddress(USDCAddress)},
	}, []any{&assetBalance})
	calls.AddCall(&ethrpc.Call{
		ABI:    usdatABI,
		Target: USDatAddress,
		Method: usdatMethodAssetDecimals,
		Params: []any{common.HexToAddress(USDCAddress)},
	}, []any{&assetDecimals})
	calls.AddCall(&ethrpc.Call{ABI: usdatABI, Target: USDatAddress, Method: usdatMethodIsWhitelistEnabled}, []any{&whitelistEnabled})
	calls.AddCall(&ethrpc.Call{
		ABI:    usdatABI,
		Target: USDatAddress,
		Method: usdatMethodIsWhitelisted,
		Params: []any{common.HexToAddress(executorAddress)},
	}, []any{&executorWhitelisted})

	resp, err := calls.Aggregate()
	if err != nil {
		return PoolExtra{}, 0, err
	}

	if resp.BlockNumber == nil {
		resp.BlockNumber = big.NewInt(0)
	}
	if assetCap == nil {
		assetCap = big.NewInt(0)
	}
	if assetBalance == nil {
		assetBalance = big.NewInt(0)
	}

	return PoolExtra{
		SwapFacilityPaused:  swapFacilityPaused,
		USDatPaused:         usdatPaused,
		ExtensionApproved:   extensionApproved,
		AssetAllowed:        assetAllowed,
		AssetCap:            uint256.MustFromBig(assetCap),
		AssetBalance:        uint256.MustFromBig(assetBalance),
		AssetDecimals:       assetDecimals,
		WhitelistEnabled:    whitelistEnabled,
		ExecutorWhitelisted: executorWhitelisted,
	}, resp.BlockNumber.Uint64(), nil
}
