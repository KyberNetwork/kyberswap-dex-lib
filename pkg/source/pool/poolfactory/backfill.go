package poolfactory

import (
	"context"
	"errors"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// BackfillConfig configures a FilterLogsBackfiller.
type BackfillConfig struct {
	// FactoryAddresses scopes the FilterLogs call (one backfiller can cover a
	// decoder spanning several factory addresses).
	FactoryAddresses []common.Address
	// StartBlock is the fromBlock used on a cold start (empty metadataBytes).
	StartBlock uint64
	// MaxBlockRangePerScan caps a single eth_getLogs call's block span. Must
	// be > 0; there is no silent default, since a reasonable value depends on
	// the target chain/RPC provider's own getLogs window limits.
	MaxBlockRangePerScan uint64
}

type backfillMetadata struct {
	LastScannedBlock uint64 `json:"lastScannedBlock"`
	// Done is set once the scan has caught up to its target. It lets a
	// restarted process short-circuit before any RPC call, instead of
	// spending a GetBlockNumber round-trip just to re-derive what a prior
	// run already established.
	Done bool `json:"done"`
}

// FilterLogsBackfiller drives an IPoolFactoryDecoder against historical
// FilterLogs results, scanning forward in bounded windows from a persisted
// cursor until it catches up to a target block height fixed on the first
// call -- then reports done so the caller stops scheduling further calls.
type FilterLogsBackfiller struct {
	decoder      IPoolFactoryDecoder
	ethrpcClient *ethrpc.Client
	cfg          BackfillConfig
	logger       logger.Logger

	targetHead *uint64
}

var _ pool.IPoolsBackfiller = (*FilterLogsBackfiller)(nil)

func NewFilterLogsBackfiller(
	decoder IPoolFactoryDecoder, ethrpcClient *ethrpc.Client, cfg BackfillConfig,
) (*FilterLogsBackfiller, error) {
	if cfg.MaxBlockRangePerScan == 0 {
		return nil, errors.New("poolfactory: BackfillConfig.MaxBlockRangePerScan must be > 0")
	}
	return &FilterLogsBackfiller{
		decoder:      decoder,
		ethrpcClient: ethrpcClient,
		cfg:          cfg,
		logger:       logger.WithFields(logger.Fields{"component": "poolfactory.FilterLogsBackfiller"}),
	}, nil
}

func (u *FilterLogsBackfiller) Backfill(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, bool, error) {
	meta, err := u.parseMetadata(metadataBytes)
	if err != nil {
		return nil, metadataBytes, false, err
	}
	if meta.Done {
		return nil, metadataBytes, true, nil
	}

	if u.targetHead == nil {
		head, err := u.ethrpcClient.GetBlockNumber(ctx)
		if err != nil {
			return nil, metadataBytes, false, err
		}
		u.targetHead = &head
	}

	if meta.LastScannedBlock > *u.targetHead {
		return nil, metadataBytes, true, nil
	}
	fromBlock := meta.LastScannedBlock
	toBlock := min(fromBlock+u.cfg.MaxBlockRangePerScan-1, *u.targetHead)

	logs, err := u.ethrpcClient.GetETHClient().FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(fromBlock),
		ToBlock:   new(big.Int).SetUint64(toBlock),
		Addresses: u.cfg.FactoryAddresses,
		// No Topics filter: mirrors how the live path (EventDecoder.GetNewPoolsFromLogs)
		// also filters purely via IsEventSupported client-side, rather than requiring
		// decoders to additionally expose a topic list.
	})
	if err != nil {
		return nil, metadataBytes, false, err // cursor NOT advanced; caller retries the same window
	}

	pools := make([]entity.Pool, 0, len(logs))
	for _, l := range logs {
		if len(l.Topics) == 0 || !u.decoder.IsEventSupported(l.Topics[0]) {
			continue
		}
		p, err := u.decoder.DecodePoolCreated(l)
		if err != nil {
			u.logger.WithFields(logger.Fields{"error": err, "txHash": l.TxHash.Hex()}).
				Error("failed to decode pool-created log during backfill")
			continue // a single bad log must not stall the cursor forever
		}
		if p != nil {
			pools = append(pools, *p)
		}
	}

	done := toBlock >= *u.targetHead
	newMetadataBytes, err := json.Marshal(backfillMetadata{LastScannedBlock: toBlock + 1, Done: done})
	if err != nil {
		return nil, metadataBytes, false, err
	}

	u.logger.WithFields(logger.Fields{
		"newPools":  len(pools),
		"fromBlock": fromBlock,
		"toBlock":   toBlock,
		"done":      done,
	}).Info("finished backfill scan window")

	return pools, newMetadataBytes, done, nil
}

func (u *FilterLogsBackfiller) parseMetadata(metadataBytes []byte) (backfillMetadata, error) {
	if len(metadataBytes) == 0 {
		return backfillMetadata{LastScannedBlock: u.cfg.StartBlock}, nil
	}
	var meta backfillMetadata
	if err := json.Unmarshal(metadataBytes, &meta); err != nil {
		return backfillMetadata{}, err
	}
	return meta, nil
}
