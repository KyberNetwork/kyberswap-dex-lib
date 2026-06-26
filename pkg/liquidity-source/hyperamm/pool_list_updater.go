package hyperamm

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

// fieldHyperAMM is the bytes32-encoded ASCII name for the hyperAMM field
// in HyperAMMFactory.getPoolAddress.
var fieldHyperAMM = stringToBytes32("hyperAMM")

// fieldSwapFeeModule is the bytes32-encoded name for the swapFeeModule field.
var fieldSwapFeeModule = stringToBytes32("swapFeeModule")

// Metadata persists the cursor used by the lister across poll cycles.
type Metadata struct {
	// LastCheckedPoolID is the highest pool ID already processed.
	// On the next run we start from LastCheckedPoolID + 1.
	LastCheckedPoolID int64 `json:"lastCheckedPoolID"`
}

// PoolsListUpdater discovers HyperAMM pools by scanning the factory.
type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

// GetNewPools iterates pool IDs from the factory starting at the last cursor
// position.  It stops when getPoolAddress returns the zero address.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}

	var pools []entity.Pool
	poolID := metadata.LastCheckedPoolID

	for {
		// Resolve the per-pool HyperAMM address for this pool ID.
		poolIDBig := big.NewInt(poolID)
		var hyperAMM common.Address
		if _, err := u.ethrpcClient.NewRequest().
			SetContext(ctx).
			AddCall(&ethrpc.Call{
				ABI:    hyperAMMFactoryABI,
				Target: u.config.Factory,
				Method: "getPoolAddress",
				Params: []any{poolIDBig, fieldHyperAMM},
			}, []any{&hyperAMM}).
			Call(); err != nil {
			// If the call fails (e.g. pool ID doesn't exist), stop discovery.
			break
		}
		if hyperAMM == (common.Address{}) {
			break
		}

		// Resolve the SwapFeeModule for this pool.
		var swapFeeModule common.Address
		if _, err := u.ethrpcClient.NewRequest().
			SetContext(ctx).
			AddCall(&ethrpc.Call{
				ABI:    hyperAMMFactoryABI,
				Target: u.config.Factory,
				Method: "getPoolAddress",
				Params: []any{poolIDBig, fieldSwapFeeModule},
			}, []any{&swapFeeModule}).
			Call(); err != nil {
			logger.Errorf("hyperamm: failed to read swapFeeModule for pool %d: %v", poolID, err)
			break
		}

		// Read token0, token1, and isToken0Based from the HyperAMM contract.
		var (
			token0, token1 common.Address
		)
		if _, err := u.ethrpcClient.NewRequest().
			SetContext(ctx).
			AddCall(&ethrpc.Call{
				ABI:    hyperAMMABI,
				Target: hyperAMM.String(),
				Method: "token0",
			}, []any{&token0}).
			AddCall(&ethrpc.Call{
				ABI:    hyperAMMABI,
				Target: hyperAMM.String(),
				Method: "token1",
			}, []any{&token1}).
			Aggregate(); err != nil {
			logger.Errorf("hyperamm: failed to read pool metadata for pool %d: %v", poolID, err)
			break
		}

		staticExtraBytes, err := json.Marshal(StaticExtra{
			SwapFeeModule: hexutil.Encode(swapFeeModule[:]),
		})
		if err != nil {
			return nil, metadataBytes, err
		}

		pools = append(pools, entity.Pool{
			Address:     hexutil.Encode(hyperAMM[:]),
			Exchange:    u.config.DexId,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    []string{"0", "0"},
			StaticExtra: string(staticExtraBytes),
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(token0[:]), Swappable: true},
				{Address: hexutil.Encode(token1[:]), Swappable: true},
			},
		})

		poolID++
	}

	if len(pools) == 0 {
		return nil, metadataBytes, nil
	}

	newMetadata, err := json.Marshal(Metadata{LastCheckedPoolID: poolID})
	if err != nil {
		return nil, metadataBytes, err
	}

	logger.Infof("hyperamm: discovered %d new pool(s)", len(pools))
	return pools, newMetadata, nil
}

// stringToBytes32 encodes an ASCII string as a [32]byte (left-aligned, zero
// padded on the right), matching Solidity's bytes32 literal encoding.
func stringToBytes32(s string) [32]byte {
	var b [32]byte
	copy(b[:], s)
	return b
}
