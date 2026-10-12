package weighted

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}

	// PoolsListUpdaterMetadata persists how many pools we have already emitted per lowercased factory
	// address, so each GetNewPools call only initializes pools a factory has added since.
	PoolsListUpdaterMetadata struct {
		Offsets map[string]int `json:"offsets,omitempty"`
		// Offset is the legacy single-factory count, migrated to Offsets[Factories[0]].
		Offset int `json:"offset,omitempty"`
	}
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

// GetNewPools discovers Range Pools on-chain from the RangePoolFactories (Range is not indexed by any
// subgraph — custom, non-canonical Vault) via getPoolCount + getPoolsInRange. It reads
// getRangePoolImmutableData for each newly seen pool to build its tokens + StaticExtra, and advances the
// per-factory offsets so subsequent calls are incremental.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	startTime := time.Now()
	l := log.Ctx(ctx).With().Str("dex", DexType).Str("dexID", u.config.DexID).Logger()
	l.Info().Msg("Started getting new pools")

	if len(u.config.Factories) == 0 {
		return nil, metadataBytes, errors.New("no factories configured")
	}
	metadata, err := u.getMetadata(metadataBytes)
	if err != nil {
		l.Warn().Err(err).Msg("getMetadata failed")
	}

	factoryPools, offsets, err := shared.ListFactoryPools(ctx, u.ethrpcClient, u.config.Factories,
		metadata.Offsets, u.config.NewPoolLimit)
	if err != nil {
		l.Error().Err(err).Msg("ListFactoryPools failed")
		return nil, metadataBytes, err
	}
	var addrs []common.Address
	var factoryOf []string
	for i, pools := range factoryPools {
		addrs = append(addrs, pools...)
		factoryOf = append(factoryOf, slices.Repeat([]string{u.config.Factories[i]}, len(pools))...)
	}

	pools, err := u.initPools(ctx, addrs, factoryOf)
	if err != nil {
		l.Error().Err(err).Msg("initPools failed")
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := json.Marshal(PoolsListUpdaterMetadata{Offsets: offsets})
	if err != nil {
		return nil, metadataBytes, err
	}

	l.Info().
		Int("poolsLen", len(pools)).
		Int64("durationMs", time.Since(startTime).Milliseconds()).
		Msg("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getMetadata(metadataBytes []byte) (m PoolsListUpdaterMetadata, err error) {
	if len(metadataBytes) == 0 {
		return m, nil
	}
	if err = json.Unmarshal(metadataBytes, &m); err != nil {
		return PoolsListUpdaterMetadata{}, err
	}
	if len(m.Offsets) == 0 && m.Offset > 0 {
		m.Offsets = map[string]int{strings.ToLower(u.config.Factories[0]): m.Offset}
	}
	return m, nil
}

// initPools reads getRangePoolImmutableData for each pool (tokens, scaling factors,
// normalized weights) and assembles the entity.Pool. Dynamic state (balances, fees,
// virtual balances, liveness) is left for the tracker; reserves start at "0" so the
// pool is treated as paused until first tracked.
func (u *PoolsListUpdater) initPools(ctx context.Context, addrs []common.Address, factoryOf []string) ([]entity.Pool, error) {
	if len(addrs) == 0 {
		return nil, nil
	}

	immutables := make([]rangePoolImmutableDataResult, len(addrs))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, addr := range addrs {
		req.AddCall(&ethrpc.Call{
			ABI:    rangePoolABI,
			Target: hexutil.Encode(addr[:]),
			Method: poolMethodGetImmutableData,
		}, []any{&immutables[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	// Empty Extra placeholder (the tracker overwrites it with real state).
	emptyExtraBytes, err := json.Marshal(Extra{Extra: &shared.Extra{}})
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(addrs))
	for i, addr := range addrs {
		imm := immutables[i].Data

		tokens := make([]*entity.PoolToken, len(imm.Tokens))
		reserves := make(entity.PoolReserves, len(imm.Tokens))
		for j, t := range imm.Tokens {
			tokens[j] = &entity.PoolToken{Address: hexutil.Encode(t[:]), Swappable: true}
			reserves[j] = "0"
		}

		staticExtra := StaticExtra{
			StaticExtra: shared.StaticExtra{
				Hook: hexutil.Encode(HookAddress(u.config.ChainID).Bytes()),
				// Range has no ERC4626 buffers, but base.PoolSimulator.GetMetaInfo indexes
				// bufferTokens[tokenIndex], so it must be a len(tokens) slice of empty
				// strings (not nil) or GetMetaInfo panics.
				BufferTokens: make([]string, len(imm.Tokens)),
			},
			NormalizedWeights:     shared.FromBigs(imm.NormalizedWeights),
			DecimalScalingFactors: shared.FromBigs(imm.DecimalScalingFactors),
			FactoryAddress:        factoryOf[i],
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			return nil, err
		}

		pools = append(pools, entity.Pool{
			Address:     hexutil.Encode(addr[:]),
			Exchange:    u.config.DexID,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      tokens,
			Extra:       string(emptyExtraBytes),
			StaticExtra: string(staticExtraBytes),
		})
	}
	return pools, nil
}
