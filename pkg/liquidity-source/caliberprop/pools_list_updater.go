package caliberprop

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	utileth "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, client *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: client}
}

func (l *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	address := l.config.Address
	var start uint16
	if len(metadataBytes) == 2 { // max 1000 pairs
		start = binary.BigEndian.Uint16(metadataBytes)
	}
	var pairIds []common.Hash
	if _, err := l.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    caliberABI,
		Target: address,
		Method: methodGetAllPairIds,
		Params: []any{big.NewInt(int64(start)), maxFetchPairCount},
	}, []any{&pairIds}).Call(); err != nil {
		return nil, nil, err
	}
	addr := common.HexToAddress(address)
	tokenPairs, err := l.fetchPairs(ctx, addr, pairIds)
	if err != nil {
		return nil, nil, err
	}

	staticExtra, err := json.Marshal(StaticExtra{Address: address})
	if err != nil {
		return nil, nil, err
	}
	pools := make([]entity.Pool, 0, len(pairIds))
	for i, pairId := range pairIds {
		tokenPair := tokenPairs[i]

		for j, xor := range addr {
			pairId[j] ^= xor
		}
		pairID := hexutil.Encode(pairId[:])
		token0 := hexutil.Encode(tokenPair[0][12:])
		token1 := hexutil.Encode(tokenPair[1][12:])

		pools = append(pools, entity.Pool{
			Address:  pairID,
			Exchange: l.config.DexID,
			Type:     DexType,
			Reserves: entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: token0, Swappable: true},
				{Address: token1, Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtra),
			Timestamp:   time.Now().Unix(),
		})
	}

	metadataBytes = binary.BigEndian.AppendUint16(metadataBytes[:0], start+uint16(len(pairIds)))
	return pools, metadataBytes, nil
}

func (l *PoolsListUpdater) fetchPairs(ctx context.Context, address common.Address,
	pairIds []common.Hash) ([][2]common.Hash, error) {
	reads := make([]utileth.StorageRead, 2*len(pairIds))
	var tmp uint256.Int
	for i, pairId := range pairIds {
		slotToken0 := crypto.Keccak256Hash(pairId[:], pairConfigBaseSlot[:])
		reads[2*i].Slot = slotToken0
		reads[2*i+1].Slot = tmp.SetBytes32(slotToken0[:]).AddUint64(&tmp, 1).Bytes32()
	}

	if err := utileth.BatchGetStorageAt(ctx, l.ethrpcClient.GetETHClient().Client(), address,
		reads, utileth.PatientBatchRetry); err != nil {
		return nil, fmt.Errorf("batch eth_getStorageAt: %w", err)
	}

	results := make([][2]common.Hash, len(pairIds))
	for i := range pairIds {
		results[i][0] = reads[2*i].Result
		results[i][1] = reads[2*i+1].Result
	}
	return results, nil
}
