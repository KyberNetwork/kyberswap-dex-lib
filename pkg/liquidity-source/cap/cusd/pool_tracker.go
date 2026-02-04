package cusd

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
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	_ pool.GetNewPoolStateParams) (entity.Pool, error) {
	return getPoolState(ctx, t.ethrpcClient, t.config, p)
}

func getPoolState(ctx context.Context, ethrpcClient *ethrpc.Client, cfg *Config, p entity.Pool) (entity.Pool, error) {
	logger.Infof("start getting new state of pool")
	defer func() {
		logger.Infof("finished getting new state of pool")
	}()

	var (
		whitelisted bool
		paused      bool
		capSupply   *big.Int
		assets      []common.Address
	)
	req := ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    capTokenABI,
		Target: p.Address,
		Method: capTokenWhitelistedMethod,
		Params: []any{common.HexToAddress(cfg.Executor)},
	}, []any{&whitelisted}).AddCall(&ethrpc.Call{
		ABI:    capTokenABI,
		Target: p.Address,
		Method: abi.Erc20TotalSupplyMethod,
	}, []any{&capSupply}).AddCall(&ethrpc.Call{
		ABI:    pausableUpgradeableABI,
		Target: p.Address,
		Method: pausablePausedMethod,
	}, []any{&paused}).AddCall(&ethrpc.Call{
		ABI:    capTokenABI,
		Target: p.Address,
		Method: capTokenAssetsMethod,
	}, []any{&assets})
	resp, err := req.Aggregate()
	if err != nil {
		logger.Errorf("failed to aggregate state: %v", err)
		return p, err
	}

	assetCount := len(assets)
	if assetCount == 0 {
		return p, nil
	}

	tokens := make([]*entity.PoolToken, 0, len(assets)+1)
	for _, asset := range assets {
		tokens = append(tokens, &entity.PoolToken{Address: strings.ToLower(asset.String()), Swappable: true})
	}
	tokens = append(tokens, &entity.PoolToken{Address: strings.ToLower(p.Address), Swappable: true})

	var (
		assetsPaused       = make([]bool, assetCount)
		prices             = make([]PriceResult, assetCount+1)
		vaultAssetSupplies = make([]*big.Int, assetCount)
		fees               = make([]*FeeDataResult, assetCount)
		availableBalances  = make([]*big.Int, assetCount)
	)

	req = ethrpcClient.R().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
	for i, token := range tokens {
		tokenAddress := common.HexToAddress(token.Address)
		req.AddCall(&ethrpc.Call{
			ABI:    oracleABI,
			Target: cfg.Oracle,
			Method: oracleGetPriceMethod,
			Params: []any{tokenAddress},
		}, []any{&prices[i]})

		if i < len(tokens)-1 {
			fees[i] = &FeeDataResult{}
			req.AddCall(&ethrpc.Call{
				ABI:    capTokenABI,
				Target: p.Address,
				Method: capTokenTotalSuppliesMethod,
				Params: []any{tokenAddress},
			}, []any{&vaultAssetSupplies[i]}).AddCall(&ethrpc.Call{
				ABI:    capTokenABI,
				Target: p.Address,
				Method: capTokenGetFeeDataMethod,
				Params: []any{tokenAddress},
			}, []any{&fees[i]}).AddCall(&ethrpc.Call{
				ABI:    capTokenABI,
				Target: p.Address,
				Method: capTokenPausedMethod,
				Params: []any{tokenAddress},
			}, []any{&assetsPaused[i]}).AddCall(&ethrpc.Call{
				ABI:    capTokenABI,
				Target: p.Address,
				Method: capTokenAvailableBalanceMethod,
				Params: []any{tokenAddress},
			}, []any{&availableBalances[i]})
		}
	}
	resp, err = req.Aggregate()
	if err != nil {
		logger.Errorf("failed to aggregate state: %v", err)
		return p, err
	}

	extraBytes, err := json.Marshal(Extra{
		Paused:       paused,
		AssetsPaused: assetsPaused,
		IsWhitelist:  whitelisted,
		CapSupply:    uint256.MustFromBig(capSupply),
		Prices: lo.Map(prices, func(item PriceResult, _ int) *uint256.Int {
			return uint256.MustFromBig(item.Price)
		}),
		VaultAssetSupplies: lo.Map(vaultAssetSupplies, func(item *big.Int, _ int) *uint256.Int {
			return uint256.MustFromBig(item)
		}),
		Fees: lo.Map(fees, func(item *FeeDataResult, _ int) *FeeData {
			return item.toFeeData()
		}),
		Assets: lo.Map(assets, func(item common.Address, index int) string {
			return strings.ToLower(item.String())
		}),
		AvailableBalances: lo.Map(availableBalances, func(item *big.Int, index int) *uint256.Int {
			return uint256.MustFromBig(item)
		}),
	})
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = lo.Map(append(availableBalances, bignum.TwoPow128), func(r *big.Int, index int) string { return r.String() })
	p.Tokens = tokens
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = resp.BlockNumber.Uint64()

	return p, nil
}
