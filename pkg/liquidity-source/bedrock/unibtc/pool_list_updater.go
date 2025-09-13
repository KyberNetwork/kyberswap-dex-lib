package unibtc

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdater struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client

	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolListUpdater)

func NewPoolListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolListUpdater {
	return &PoolListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, nil, nil
	}
	u.hasInitialized = true

	pools := make([]entity.Pool, 0, len(u.cfg.Vaults))
	for vaultAddr, vaultCfg := range u.cfg.Vaults {
		extra := &PoolExtra{
			TokensPaused:     make([]bool, len(vaultCfg.Tokens)),
			TokensAllowed:    make([]bool, len(vaultCfg.Tokens)),
			Caps:             make([]*big.Int, len(vaultCfg.Tokens)),
			TokenUsedCaps:    make([]*big.Int, len(vaultCfg.Tokens)),
			ExchangeRateBase: bignumber.TenPowInt(10),
		}
		blockNumber, err := updateExtra(ctx, extra, vaultAddr, vaultCfg, u.ethrpcClient)
		if err != nil {
			return nil, nil, err
		}

		extraBytes, err := json.Marshal(extra)
		if err != nil {
			return nil, nil, err
		}
		pools = append(pools, entity.Pool{
			Address:   strings.ToLower(vaultAddr),
			Exchange:  string(valueobject.ExchangeBedrockUniBTC),
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves: lo.Map(vaultCfg.Tokens, func(token string, _ int) string {
				return reserves
			}),
			Tokens: lo.Map(vaultCfg.Tokens, func(token string, _ int) *entity.PoolToken {
				return &entity.PoolToken{
					Address:   strings.ToLower(token),
					Swappable: true,
				}
			}),
			BlockNumber: blockNumber,
			Extra:       string(extraBytes),
		})
	}

	return pools, nil, nil
}

func updateExtra(ctx context.Context, extra *PoolExtra, vaultAddr string, cfg VaultCfg, ethrpcClient *ethrpc.Client) (uint64, error) {
	req := ethrpcClient.NewRequest().SetContext(ctx)
	abi := lo.Ternary(cfg.Type == VaultTypeUniBTC, VaultUniBTCABI, VaultBrBTCABI)
	if cfg.Type == VaultTypeUniBTC {
		req.AddCall(&ethrpc.Call{
			ABI:    PausedABI,
			Target: vaultAddr,
			Method: "paused",
		}, []any{&(extra.Paused)})
	}
	for i, token := range cfg.Tokens {
		req.AddCall(&ethrpc.Call{
			ABI:    abi,
			Target: vaultAddr,
			Method: lo.Ternary(cfg.Type == VaultTypeUniBTC, "allowedTokenList", "allowedTokens"),
			Params: []any{common.HexToAddress(token)},
		}, []any{&(extra.TokensAllowed[i])})
		req.AddCall(&ethrpc.Call{
			ABI:    abi,
			Target: vaultAddr,
			Method: lo.Ternary(cfg.Type == VaultTypeUniBTC, "paused", "pausedTokens"),
			Params: []any{common.HexToAddress(token)},
		}, []any{&(extra.TokensPaused[i])})
		req.AddCall(&ethrpc.Call{
			ABI:    abi,
			Target: vaultAddr,
			Method: lo.Ternary(cfg.Type == VaultTypeUniBTC, "caps", "tokenCaps"),
			Params: []any{common.HexToAddress(token)},
		}, []any{&(extra.Caps[i])})
	}
	if cfg.Type == VaultTypeUniBTC {
		req.AddCall(&ethrpc.Call{
			ABI:    abi,
			Target: vaultAddr,
			Method: "supplyFeeder",
		}, []any{&(extra.SupplyFeeder)})
	}

	ignoreTokenUsedCaps := cfg.Type == VaultTypeUniBTC && extra.SupplyFeeder == (common.Address{})
	for i, token := range cfg.Tokens {
		if i == len(cfg.Tokens)-1 { //uniBTC
			extra.TokenUsedCaps[i] = big.NewInt(0)
			continue
		}
		_ = lo.TernaryF(ignoreTokenUsedCaps, func() bool { return true },
			func() bool {
				req.AddCall(lo.Ternary(cfg.Type == VaultTypeUniBTC, &ethrpc.Call{
					ABI:    TotalSupplyABI,
					Target: extra.SupplyFeeder.Hex(),
					Method: "totalSupply",
					Params: []any{common.HexToAddress(token)},
				}, &ethrpc.Call{
					ABI:    abi,
					Target: vaultAddr,
					Method: "tokenUsedCaps",
					Params: []any{common.HexToAddress(token)},
				}), []any{&(extra.TokenUsedCaps[i])})
				return true
			})
	}
	resp, err := req.Aggregate()
	if err != nil {
		return 0, err
	}
	return resp.BlockNumber.Uint64(), nil
}
