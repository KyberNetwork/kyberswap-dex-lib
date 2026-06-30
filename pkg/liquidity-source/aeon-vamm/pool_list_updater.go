package aeonvamm

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var factoryABI, _ = abi.JSON(strings.NewReader(`[
  {"name":"allPairsLength","type":"function","stateMutability":"view","inputs":[],"outputs":[{"type":"uint256"}]},
  {"name":"allPairs","type":"function","stateMutability":"view","inputs":[{"name":"","type":"uint256"}],"outputs":[{"type":"address"}]}
]`))

var pairABI, _ = abi.JSON(strings.NewReader(`[
  {"name":"token0","type":"function","stateMutability":"view","inputs":[],"outputs":[{"type":"address"}]},
  {"name":"token1","type":"function","stateMutability":"view","inputs":[],"outputs":[{"type":"address"}]},
  {"name":"getReserves","type":"function","stateMutability":"view","inputs":[],"outputs":[
    {"name":"reserve0","type":"uint112"},
    {"name":"reserve1","type":"uint112"},
    {"name":"blockTimestampLast","type":"uint32"}
  ]}
]`))

const batchSize = 100

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: config, ethrpcClient: ethrpcClient}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var meta PoolListUpdaterMetadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &meta); err != nil {
			return nil, metadataBytes, err
		}
	}

	// Get total pairs count
	var totalPairs *big.Int
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.config.FactoryAddress,
		Method: "allPairsLength",
		Params: nil,
	}, []interface{}{&totalPairs})
	if _, err := req.Call(); err != nil {
		return nil, metadataBytes, err
	}

	total := int(totalPairs.Int64())
	if meta.Offset >= total {
		return nil, metadataBytes, nil
	}

	end := meta.Offset + batchSize
	if end > total {
		end = total
	}

	// Fetch pair addresses
	pairAddresses := make([]common.Address, end-meta.Offset)
	addrReq := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i := meta.Offset; i < end; i++ {
		idx := i
		addrReq.AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: u.config.FactoryAddress,
			Method: "allPairs",
			Params: []interface{}{big.NewInt(int64(idx))},
		}, []interface{}{&pairAddresses[i-meta.Offset]})
	}
	if _, err := addrReq.TryBlockAndAggregate(); err != nil {
		return nil, metadataBytes, err
	}

	pools := make([]entity.Pool, 0, len(pairAddresses))
	for _, addr := range pairAddresses {
		if addr == (common.Address{}) {
			continue
		}

		var token0, token1 common.Address
		var reserve0, reserve1 *big.Int
		var blockTimestampLast uint32

		pairReq := u.ethrpcClient.NewRequest().SetContext(ctx)
		pairReq.AddCall(&ethrpc.Call{ABI: pairABI, Target: addr.Hex(), Method: "token0"}, []interface{}{&token0})
		pairReq.AddCall(&ethrpc.Call{ABI: pairABI, Target: addr.Hex(), Method: "token1"}, []interface{}{&token1})
		pairReq.AddCall(&ethrpc.Call{ABI: pairABI, Target: addr.Hex(), Method: "getReserves"}, []interface{}{&reserve0, &reserve1, &blockTimestampLast})
		if _, err := pairReq.TryBlockAndAggregate(); err != nil {
			continue
		}

		extra := Extra{
			Reserve0: reserve0,
			Reserve1: reserve1,
			Fee:      30, // default 0.3% bps; ideally read from pool.feeBps()
		}
		extraBytes, _ := json.Marshal(extra)

		r0 := "0"
		r1 := "0"
		if reserve0 != nil {
			r0 = reserve0.String()
		}
		if reserve1 != nil {
			r1 = reserve1.String()
		}

		pools = append(pools, entity.Pool{
			Address:   strings.ToLower(addr.Hex()),
			Exchange:  u.config.DexID,
			Type:      DexTypeAeonVAMM,
			Timestamp: time.Now().Unix(),
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(token0.Hex()), Swappable: true},
				{Address: strings.ToLower(token1.Hex()), Swappable: true},
			},
			Reserves: entity.PoolReserves{r0, r1},
			Extra:    string(extraBytes),
		})
	}

	meta.Offset = end
	newMetaBytes, _ := json.Marshal(meta)
	return pools, newMetaBytes, nil
}
