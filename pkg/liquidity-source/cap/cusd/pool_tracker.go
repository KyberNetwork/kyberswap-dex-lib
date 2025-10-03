package cusd

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
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
}

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	_ pool.GetNewPoolStateParams) (entity.Pool, error) {
	logger.Infof("start getting new state of pool")
	defer func() {
		logger.Infof("finished getting new state of pool")
	}()

	var (
		whitelisted        bool
		capSupply          *big.Int
		prices             = make([]PriceResult, len(p.Tokens))
		vaultAssetSupplies = make([]*big.Int, len(p.Tokens)-1)
		fees               = make([]FeeDataResult, len(p.Tokens)-1)
	)
	req := t.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    capTokenABI,
			Target: p.Address,
			Method: capTokenWhitelistedMethod,
			Params: []any{t.config.Executor},
		}, []any{&whitelisted}).
		AddCall(&ethrpc.Call{
			ABI:    capTokenABI,
			Target: p.Address,
			Method: abi.Erc20TotalSupplyMethod,
		}, []any{&capSupply})

	for i, token := range p.Tokens {
		req.AddCall(&ethrpc.Call{
			ABI:    oracleABI,
			Target: t.config.Oracle,
			Method: oracleGetPriceMethod,
			Params: []any{token},
		}, []any{&prices[i]})

		if i < len(p.Tokens)-1 {
			req.AddCall(&ethrpc.Call{
				ABI:    oracleABI,
				Target: p.Address,
				Method: capTokenTotalSuppliesMethod,
				Params: []any{token},
			}, []any{&vaultAssetSupplies[i]}).AddCall(&ethrpc.Call{
				ABI:    oracleABI,
				Target: p.Address,
				Method: capTokenGetFeeDataMethod,
				Params: []any{token},
			}, []any{&fees[i]})
		}
	}

	extraBytes, err := json.Marshal(Extra{
		IsWhitelist: whitelisted,
		CapSupply:   uint256.MustFromBig(capSupply),
		Prices: lo.Map(prices, func(item PriceResult, _ int) *uint256.Int {
			return uint256.MustFromBig(item.Price)
		}),
		VaultAssetSupplies: lo.Map(vaultAssetSupplies, func(item *big.Int, _ int) *uint256.Int {
			return uint256.MustFromBig(item)
		}),
		Fees: lo.Map(fees, func(item FeeDataResult, _ int) *FeeData {
			return item.toFeeData()
		}),
	})
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)

	p.Timestamp = time.Now().Unix()

	return p, nil
}
