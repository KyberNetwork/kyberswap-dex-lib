package arberaden

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

type PoolsListUpdaterMetadata struct {
	Addresses []string `json:"addresses"`
}

type PoolState struct {
	Assets        []RPCAsset
	AssetSupplies []*big.Int
	Supply        *big.Int
	Fee           RPCFee
}

type RPCFee struct {
	Bond    uint16
	Debond  uint16
	Buy     uint16
	Sell    uint16
	Partner uint16
	Burn    uint16
}

type RPCAsset struct {
	Token           common.Address
	Weight          *big.Int
	BasePriceUSDX96 *big.Int
	C1              common.Address
	Q1              *big.Int
}

type Index struct {
	Index    common.Address `json:"index"`
	Verified bool           `json:"verified"`
}

func (d *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata PoolsListUpdaterMetadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}

	var indexes []Index
	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    indexManagerABI,
		Target: d.config.IndexManager,
		Method: "allIndexes",
		Params: nil,
	}, []interface{}{&indexes})

	_, err := req.Aggregate()
	if err != nil {
		return nil, metadataBytes, err
	}

	var newPoolAddresses []string
	for _, index := range indexes {
		if lo.Contains(metadata.Addresses, strings.ToLower(index.Index.Hex())) {
			continue
		}
		newPoolAddresses = append(newPoolAddresses, strings.ToLower(index.Index.Hex()))
	}

	if len(newPoolAddresses) == 0 {
		return nil, metadataBytes, nil
	}

	newPools, err := initPools(ctx, newPoolAddresses, d.config, d.ethrpcClient)
	if err != nil {
		return nil, metadataBytes, err
	}
	newPools, err = trackPools(ctx, newPools, d.ethrpcClient)
	if err != nil {
		return nil, metadataBytes, err
	}
	metadata.Addresses = append(metadata.Addresses, newPoolAddresses...)
	metadataBytes, err = json.Marshal(metadata)
	if err != nil {
		return nil, metadataBytes, err
	}

	return newPools, metadataBytes, nil
}

func initPools(ctx context.Context, addresses []string, cfg *Config, rpcClient *ethrpc.Client) ([]entity.Pool, error) {
	req := rpcClient.NewRequest().SetContext(ctx)
	poolStates := make([]PoolState, len(addresses))
	for i, address := range addresses {
		req.AddCall(&ethrpc.Call{
			ABI:    weightedIndexABI,
			Target: address,
			Method: "getAllAssets",
			Params: nil,
		}, []interface{}{&poolStates[i].Assets})
	}
	_, err := req.Aggregate()
	if err != nil {
		return nil, err
	}
	extras := make([]string, len(poolStates))
	for i, poolState := range poolStates {
		bytes, err := json.Marshal(Extra{
			Assets: lo.Map(poolState.Assets, func(asset RPCAsset, _ int) Asset {
				return Asset{
					Token:           strings.ToLower(asset.Token.Hex()),
					Weighting:       uint256.MustFromBig(asset.Weight),
					BasePriceUSDX96: uint256.MustFromBig(asset.BasePriceUSDX96),
					C1:              strings.ToLower(asset.C1.Hex()),
					Q1:              uint256.MustFromBig(asset.Q1),
				}
			}),
		})
		if err != nil {
			return nil, err
		}
		extras[i] = string(bytes)
	}
	if err != nil {
		return nil, err
	}
	return lo.Map(poolStates, func(poolState PoolState, i int) entity.Pool {
		return entity.Pool{
			Address:   addresses[i],
			Exchange:  cfg.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Tokens: append(
				[]*entity.PoolToken{
					{
						Address:   strings.ToLower(addresses[i]),
						Swappable: true,
					},
				},
				lo.Map(poolState.Assets, func(asset RPCAsset, _ int) *entity.PoolToken {
					return &entity.PoolToken{
						Address:   strings.ToLower(asset.Token.Hex()),
						Swappable: true,
					}
				})...,
			),
			Extra: extras[i],
		}
	}), nil
}

func trackPools(ctx context.Context, pools []entity.Pool, rpcClient *ethrpc.Client) ([]entity.Pool, error) {
	req := rpcClient.NewRequest().SetContext(ctx)
	poolStates := make([]PoolState, len(pools))
	for i, pool := range pools {
		poolStates[i].AssetSupplies = make([]*big.Int, len(pool.Tokens)-1)
		for j, asset := range pool.Tokens[1:] {
			req.AddCall(&ethrpc.Call{
				ABI:    weightedIndexABI,
				Target: asset.Address,
				Method: "balanceOf",
				Params: []interface{}{common.HexToAddress(pool.Address)},
			}, []interface{}{&poolStates[i].AssetSupplies[j]})
		}
		req.AddCall(&ethrpc.Call{
			ABI:    weightedIndexABI,
			Target: pool.Address,
			Method: "totalSupply",
			Params: nil,
		}, []interface{}{&poolStates[i].Supply})
		req.AddCall(&ethrpc.Call{
			ABI:    weightedIndexABI,
			Target: pool.Address,
			Method: "fees",
			Params: nil,
		}, []interface{}{&poolStates[i].Fee})
	}

	_, err := req.Aggregate()
	if err != nil {
		return nil, err
	}

	for i, pool := range pools {
		var extra Extra
		if err := json.Unmarshal([]byte(pool.Extra), &extra); err != nil {
			return nil, err
		}
		extra.AssetSupplies = lo.Map(poolStates[i].AssetSupplies, func(asset *big.Int, _ int) *uint256.Int {
			return uint256.MustFromBig(asset)
		})
		extra.Supply = uint256.MustFromBig(poolStates[i].Supply)
		extra.Fee = Fee{
			Bond:   uint256.NewInt(uint64(poolStates[i].Fee.Bond)),
			Debond: uint256.NewInt(uint64(poolStates[i].Fee.Debond)),
			Burn:   uint256.NewInt(uint64(poolStates[i].Fee.Burn)),
		}
		bytes, err := json.Marshal(extra)
		if err != nil {
			return nil, err
		}
		pools[i].Extra = string(bytes)
	}
	return pools, nil
}
