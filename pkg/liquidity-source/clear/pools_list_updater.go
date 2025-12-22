package clear

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolsListUpdater struct {
	config        *Config
	graphqlClient *graphqlpkg.Client
}

var _ = poollist.RegisterFactoryCG(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	graphqlClient *graphqlpkg.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:        cfg,
		graphqlClient: graphqlClient,
	}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, nil, err
		}
	}

	if metadata.LastCreatedAtTimestamp == nil {
		metadata.LastCreatedAtTimestamp = big.NewInt(0)
	}

	vaults, err := d.getVaultsList(ctx, d.config.NewPoolLimit)
	if err != nil {
		return nil, nil, err
	}

	logger.Infof("[Clear] got %v vaults from GraphQL", len(vaults))

	pools := make([]entity.Pool, 0)

	for _, vault := range vaults {
		// Skip vaults with less than 2 tokens
		if len(vault.Tokens) < 2 {
			continue
		}

		// Create a pool for each token pair in the vault
		for i := 0; i < len(vault.Tokens); i++ {
			for j := i + 1; j < len(vault.Tokens); j++ {
				tokenA := vault.Tokens[i]
				tokenB := vault.Tokens[j]

				tokenADecimals, err := kutils.Atou[uint8](tokenA.Decimals)
				if err != nil {
					tokenADecimals = defaultTokenDecimals
				}

				tokenBDecimals, err := kutils.Atou[uint8](tokenB.Decimals)
				if err != nil {
					tokenBDecimals = defaultTokenDecimals
				}

				tokens := []*entity.PoolToken{
					{
						Address:   strings.ToLower(tokenA.Address),
						Symbol:    tokenA.Symbol,
						Decimals:  tokenADecimals,
						Swappable: true,
					},
					{
						Address:   strings.ToLower(tokenB.Address),
						Symbol:    tokenB.Symbol,
						Decimals:  tokenBDecimals,
						Swappable: true,
					},
				}

				// Collect all token addresses for the vault
				allTokens := make([]string, len(vault.Tokens))
				for k, t := range vault.Tokens {
					allTokens[k] = strings.ToLower(t.Address)
				}

				staticExtra := StaticExtra{
					VaultAddress: strings.ToLower(vault.Address),
					SwapAddress:  strings.ToLower(d.config.SwapAddress),
					Tokens:       allTokens,
				}

				staticExtraBytes, err := json.Marshal(staticExtra)
				if err != nil {
					logger.WithFields(logger.Fields{
						"error": err,
					}).Errorf("[Clear] failed to marshal static extra data")
					continue
				}

				// Pool address is a combination of vault and token pair
				// Format: vault_tokenA_tokenB
				poolAddress := fmt.Sprintf("%s_%s_%s",
					strings.ToLower(vault.Address),
					strings.ToLower(tokenA.Address),
					strings.ToLower(tokenB.Address),
				)

				newPool := entity.Pool{
					Address:     poolAddress,
					Exchange:    d.config.DexID,
					Type:        DexType,
					Reserves:    entity.PoolReserves{zeroString, zeroString},
					Tokens:      tokens,
					StaticExtra: string(staticExtraBytes),
				}

				pools = append(pools, newPool)
			}
		}
	}

	newMetadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, nil, err
	}

	return pools, newMetadataBytes, nil
}

func (d *PoolsListUpdater) getVaultsList(ctx context.Context, limit int) ([]GraphQLVault, error) {
	query := fmt.Sprintf(`{
		clearVaults(first: %d) {
			id
			address
			tokens {
				address
				symbol
				decimals
			}
		}
	}`, limit)

	req := graphqlpkg.NewRequest(query)

	var response GraphQLResponse
	if err := d.graphqlClient.Run(ctx, req, &response); err != nil {
		logger.Errorf("[Clear] failed to query GraphQL, err %v", err)
		return nil, err
	}

	return response.ClearVaults, nil
}
