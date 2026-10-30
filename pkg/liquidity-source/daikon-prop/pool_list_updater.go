package daikonprop

import (
	"context"
	"math/big"

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

// Metadata is the number of manager pairs already listed (pairs are append-only).
type Metadata struct {
	Listed int `json:"listed"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: cfg, ethrpcClient: ethrpcClient}
}

// marketState is PAMMManager.getMarketState's tuple. Only the ABI-named fields are decoded.
type marketState struct {
	Pair          common.Address
	BaseToken     common.Address
	QuoteToken    common.Address
	BaseDecimals  *big.Int
	QuoteDecimals *big.Int
	Price         *big.Int
	ObservedAtMs  *big.Int
	C1            *big.Int
	C2            *big.Int
	Spread1Bps    *big.Int
	Spread2Bps    *big.Int
	BuyC1Rem      *big.Int `abi:"buyC1Remaining"`
	BuyC2Rem      *big.Int `abi:"buyC2Remaining"`
	SellC1Rem     *big.Int `abi:"sellC1Remaining"`
	SellC2Rem     *big.Int `abi:"sellC2Remaining"`
	TTL           *big.Int `abi:"ttl"`
	Paused        bool
	ManagerActive bool
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var metadata Metadata
	if len(metadataBytes) != 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, err
		}
	}

	var manager common.Address
	var count *big.Int
	resp, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: daikonABI, Target: u.config.Router, Method: methodManager}, []any{&manager}).Aggregate()
	if err != nil {
		return nil, metadataBytes, err
	}
	if _, err = u.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(resp.BlockNumber).
		AddCall(&ethrpc.Call{ABI: daikonABI, Target: manager.Hex(), Method: methodPairCount}, []any{&count}).
		Aggregate(); err != nil {
		return nil, metadataBytes, err
	}
	total := int(count.Int64())
	if total <= metadata.Listed {
		return nil, metadataBytes, nil
	}

	pairs := make([]common.Address, total-metadata.Listed)
	req := u.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
	for i := range pairs {
		req.AddCall(&ethrpc.Call{ABI: daikonABI, Target: manager.Hex(), Method: methodAllPairs,
			Params: []any{big.NewInt(int64(metadata.Listed + i))}}, []any{&pairs[i]})
	}
	if _, err = req.Aggregate(); err != nil {
		return nil, metadataBytes, err
	}
	states := make([]marketState, len(pairs))
	req = u.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
	for i := range pairs {
		req.AddCall(&ethrpc.Call{ABI: daikonABI, Target: manager.Hex(), Method: methodGetMarketState,
			Params: []any{pairs[i]}}, []any{&states[i]})
	}
	if _, err = req.Aggregate(); err != nil {
		return nil, metadataBytes, err
	}

	staticExtra, err := json.Marshal(StaticExtra{
		Router:    hexutil.Encode(common.HexToAddress(u.config.Router).Bytes()),
		Manager:   hexutil.Encode(manager[:]),
		PartnerID: u.config.PartnerID,
		FeePpm:    min(u.config.FeePpm, MaxFeePpm),
	})
	if err != nil {
		return nil, metadataBytes, err
	}
	pools := make([]entity.Pool, len(pairs))
	for i, st := range states {
		pools[i] = entity.Pool{
			Address:  hexutil.Encode(pairs[i][:]),
			Exchange: u.config.DexID,
			Type:     DexType,
			Reserves: entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(st.BaseToken[:]), Swappable: true},
				{Address: hexutil.Encode(st.QuoteToken[:]), Swappable: true},
			},
			StaticExtra: string(staticExtra),
		}
	}
	newMetadata, err := json.Marshal(Metadata{Listed: total})
	if err != nil {
		return nil, metadataBytes, err
	}
	return pools, newMetadata, nil
}
