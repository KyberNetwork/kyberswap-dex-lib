package everlongcvamm

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

// Metadata is the listing cursor: the set of ALMs already listed, so venues added to
// the config later are picked up without relisting the rest.
type Metadata struct {
	Listed map[string]bool `json:"listed"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

// GetNewPools lists each configured CvammALM venue exactly once. The ALM is the venue —
// pool address = ALM address — and pins the stable to leg 0 (token0) and the volatile to
// leg 1, read from the contract rather than configured. Reserves are left to the tracker.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	metadata := Metadata{Listed: map[string]bool{}}
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
		if metadata.Listed == nil {
			metadata.Listed = map[string]bool{}
		}
	}

	var newALMs []ALMConfig
	for _, alm := range u.config.ALMs {
		if !metadata.Listed[strings.ToLower(alm.Address)] {
			newALMs = append(newALMs, alm)
		}
	}
	if len(newALMs) == 0 {
		return nil, metadataBytes, nil
	}

	tokens0 := make([]common.Address, len(newALMs))
	tokens1 := make([]common.Address, len(newALMs))
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i, alm := range newALMs {
		req.AddCall(&ethrpc.Call{
			ABI:    almABI,
			Target: alm.Address,
			Method: almMethodToken0,
		}, []any{&tokens0[i]})
		req.AddCall(&ethrpc.Call{
			ABI:    almABI,
			Target: alm.Address,
			Method: almMethodToken1,
		}, []any{&tokens1[i]})
	}
	resp, err := req.Aggregate()
	if err != nil {
		return nil, metadataBytes, err
	}
	var blockNumber uint64
	if resp.BlockNumber != nil {
		blockNumber = resp.BlockNumber.Uint64()
	}

	pools := make([]entity.Pool, 0, len(newALMs))
	for i, alm := range newALMs {
		if tokens0[i] == (common.Address{}) || tokens1[i] == (common.Address{}) {
			continue
		}
		staticExtra, err := json.Marshal(StaticExtra{
			Adapter:       strings.ToLower(alm.Adapter),
			Quoter:        strings.ToLower(alm.Quoter),
			GasStableIn:   alm.GasStableIn,
			GasVolatileIn: alm.GasVolatileIn,
		})
		if err != nil {
			return nil, metadataBytes, err
		}

		almAddress := strings.ToLower(alm.Address)
		pools = append(pools, entity.Pool{
			Address:   almAddress,
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(tokens0[i][:]), Swappable: true},
				{Address: hexutil.Encode(tokens1[i][:]), Swappable: true},
			},
			Reserves:    entity.PoolReserves{"0", "0"},
			StaticExtra: string(staticExtra),
			Extra:       "{}",
			BlockNumber: blockNumber,
		})
		metadata.Listed[almAddress] = true
	}

	newMetadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, metadataBytes, err
	}
	return pools, newMetadataBytes, nil
}
