package dexv2

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-v2/abis"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

type onchainListMetadata struct {
	SeenPoolIDs []string `json:"seenPoolIds"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, client *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: client}
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata onchainListMetadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}
	var keys [2][]DexKey
	request := d.ethrpcClient.R().SetContext(ctx)
	for i, method := range []string{"getD3PermissionedDexes", "getD4PermissionedDexes"} {
		request.AddCall(&ethrpc.Call{ABI: abis.ResolverABI, Target: d.config.Resolver, Method: method}, []any{&keys[i]})
	}
	if _, err := request.Aggregate(); err != nil {
		return nil, metadataBytes, err
	}
	seen := make(map[string]bool, len(metadata.SeenPoolIDs))
	for _, id := range metadata.SeenPoolIDs {
		seen[id] = true
	}
	var pools []entity.Pool
	for i, list := range keys {
		for _, key := range list {
			p, err := d.poolFromKey(key, uint32(i+3))
			if err != nil {
				return nil, metadataBytes, err
			}
			if seen[p.Address] {
				continue
			}
			seen[p.Address] = true
			metadata.SeenPoolIDs = append(metadata.SeenPoolIDs, p.Address)
			pools = append(pools, p)
		}
	}
	updated, err := json.Marshal(metadata)
	if err != nil {
		return nil, metadataBytes, err
	}
	return pools, updated, nil
}

func (d *PoolsListUpdater) poolFromKey(key DexKey, dexType uint32) (entity.Pool, error) {
	// getDexPoolState's second argument is the same DexKey tuple used by abi.encode(dexKey).
	encoded, err := abis.ResolverABI.Methods["getDexPoolState"].Inputs[1:].Pack(key)
	if err != nil {
		return entity.Pool{}, err
	}
	static := StaticExtra{Dex: strings.ToLower(d.config.Dex), DexType: dexType,
		Fee: uint32(key.Fee.Uint64()), TickSpacing: uint32(key.TickSpacing.Uint64()), TrackLiquidityLimits: d.config.TrackLiquidityLimits}
	if key.Controller != (common.Address{}) {
		static.Controller = hexutil.Encode(key.Controller[:])
	}
	tokens := make([]*entity.PoolToken, 2)
	for i, token := range []common.Address{key.Token0, key.Token1} {
		address := hexutil.Encode(token[:])
		if valueobject.IsNative(address) {
			static.IsNative[i] = true
			address = valueobject.WrapNativeLower(address, d.config.ChainID)
		}
		tokens[i] = &entity.PoolToken{Address: address, Swappable: true}
	}
	data, err := json.Marshal(static)
	if err != nil {
		return entity.Pool{}, err
	}
	return entity.Pool{Address: encodeFluidDexV2PoolAddress(crypto.Keccak256Hash(encoded).Hex(), dexType),
		Exchange: d.config.DexID, Type: DexType, Tokens: tokens, Reserves: []string{"0", "0"},
		StaticExtra: string(data), Extra: "{}", Timestamp: time.Now().Unix()}, nil
}
