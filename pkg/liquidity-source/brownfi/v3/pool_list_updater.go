package brownfiv3

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog/log"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type (
	PoolsListUpdater struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}

	PoolsListUpdaterMetadata struct {
		Offset int `json:"offset"`
	}
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	startTime := time.Now()
	l := log.Ctx(ctx).With().Str("dex", DexType).Logger()
	l.Info().Msg("Started getting new pools")

	allPairsLength, err := u.getAllPairsLength(ctx)
	if err != nil {
		l.Error().Err(err).Msg("getAllPairsLength failed")
		return nil, metadataBytes, err
	}

	offset, err := u.getOffset(metadataBytes)
	if err != nil {
		l.Warn().Err(err).Msg("getOffset failed")
	}

	batchSize := u.getBatchSize(allPairsLength, u.config.NewPoolLimit, offset)

	pairAddresses, err := u.listPairAddresses(ctx, offset, batchSize)
	if err != nil {
		l.Error().Err(err).Msg("listPairAddresses failed")
		return nil, metadataBytes, err
	}

	pools, err := u.initPools(ctx, pairAddresses)
	if err != nil {
		l.Error().Err(err).Msg("initPools failed")
		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(offset + batchSize)
	if err != nil {
		l.Error().Err(err).Msg("newMetadata failed")
		return nil, metadataBytes, err
	}

	l.Info().Int("pools_len", len(pools)).Int("offset", offset).
		Int64("duration_ms", time.Since(startTime).Milliseconds()).Msg("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getAllPairsLength(ctx context.Context) (int, error) {
	var length *big.Int
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    brownFiV3FactoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryMethodAllPairsLength,
	}, []any{&length})
	if _, err := req.Call(); err != nil {
		return 0, err
	}
	return int(length.Int64()), nil
}

func (u *PoolsListUpdater) getOffset(metadataBytes []byte) (int, error) {
	if len(metadataBytes) == 0 {
		return 0, nil
	}
	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return 0, err
	}
	return metadata.Offset, nil
}

func (u *PoolsListUpdater) listPairAddresses(ctx context.Context, offset, batchSize int) ([]common.Address, error) {
	result := make([]common.Address, batchSize)
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i := range batchSize {
		req.AddCall(&ethrpc.Call{
			ABI:    brownFiV3FactoryABI,
			Target: u.config.FactoryAddress,
			Method: factoryMethodAllPairs,
			Params: []any{big.NewInt(int64(offset + i))},
		}, []any{&result[i]})
	}
	resp, err := req.TryAggregate()
	if err != nil {
		return nil, err
	}
	var pairs []common.Address
	for i, ok := range resp.Result {
		if ok {
			pairs = append(pairs, result[i])
		}
	}
	return pairs, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, pairAddresses []common.Address) ([]entity.Pool, error) {
	token0List, token1List, err := u.listPairTokens(ctx, pairAddresses)
	if err != nil {
		return nil, err
	}
	pools := make([]entity.Pool, 0, len(pairAddresses))
	for i, pairAddress := range pairAddresses {
		pools = append(pools, entity.Pool{
			Address:   hexutil.Encode(pairAddress[:]),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(token0List[i][:]), Swappable: true},
				{Address: hexutil.Encode(token1List[i][:]), Swappable: true},
			},
			Extra: "{}",
		})
	}
	return pools, nil
}

func (u *PoolsListUpdater) listPairTokens(ctx context.Context, pairAddresses []common.Address) ([]common.Address, []common.Address, error) {
	token0s := make([]common.Address, len(pairAddresses))
	token1s := make([]common.Address, len(pairAddresses))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, addr := range pairAddresses {
		req.AddCall(&ethrpc.Call{
			ABI: brownFiV3PairABI, Target: addr.Hex(), Method: pairMethodToken0,
		}, []any{&token0s[i]})
		req.AddCall(&ethrpc.Call{
			ABI: brownFiV3PairABI, Target: addr.Hex(), Method: pairMethodToken1,
		}, []any{&token1s[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, nil, err
	}
	return token0s, token1s, nil
}

func (u *PoolsListUpdater) newMetadata(newOffset int) ([]byte, error) {
	return json.Marshal(PoolsListUpdaterMetadata{Offset: newOffset})
}

func (u *PoolsListUpdater) getBatchSize(length, limit, offset int) int {
	if length <= 0 || offset >= length {
		return 0
	}
	if limit <= 0 {
		limit = length
	}
	return min(length-offset, limit)
}
