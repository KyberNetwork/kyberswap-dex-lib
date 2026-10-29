package ammalgam

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

const defaultNewPoolLimit = 100

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

type PoolsListUpdaterMetadata struct {
	Offset int `json:"offset"`
}

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: config, ethrpcClient: ethrpcClient}
}

func discoveryBatchSize(total, offset, limit int) int {
	if offset >= total {
		return 0
	}
	if limit <= 0 {
		limit = defaultNewPoolLimit
	}
	if remaining := total - offset; remaining < limit {
		return remaining
	}
	return limit
}

func validateFactoryMetadata(config *Config, beacon, implementation common.Address) error {
	if config == nil || beacon == (common.Address{}) || implementation == (common.Address{}) {
		return errors.New("invalid Ammalgam factory metadata")
	}
	if config.PairBeacon != "" &&
		(!validPairAddress(config.PairBeacon) || !strings.EqualFold(config.PairBeacon, beacon.Hex())) {
		return fmt.Errorf("ammalgam pair beacon mismatch: configured %s, on chain %s", config.PairBeacon, beacon.Hex())
	}
	if config.PairImplementation != "" &&
		(!validPairAddress(config.PairImplementation) || !strings.EqualFold(config.PairImplementation, implementation.Hex())) {
		return fmt.Errorf("ammalgam pair implementation mismatch: configured %s, on chain %s",
			config.PairImplementation, implementation.Hex())
	}
	return nil
}

func requireAllCallsSucceeded(response *ethrpc.Response, count int) error {
	if response == nil || len(response.Result) != count {
		return errors.New("incomplete Ammalgam RPC batch")
	}
	for i, ok := range response.Result {
		if !ok {
			return fmt.Errorf("ammalgam RPC call %d failed", i)
		}
	}
	return nil
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u == nil || u.config == nil || u.ethrpcClient == nil || !validPairAddress(u.config.FactoryAddress) {
		return nil, metadataBytes, errors.New("invalid Ammalgam discovery configuration")
	}
	var metadata PoolsListUpdaterMetadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, metadataBytes, fmt.Errorf("decode Ammalgam discovery metadata: %w", err)
		}
	}
	if metadata.Offset < 0 {
		return nil, metadataBytes, errors.New("negative Ammalgam discovery offset")
	}

	var total *big.Int
	var beacon common.Address
	initial, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: factoryABI, Target: u.config.FactoryAddress, Method: "allPairsLength"}, []any{&total}).
		AddCall(&ethrpc.Call{ABI: factoryABI, Target: u.config.FactoryAddress, Method: "pairBeacon"}, []any{&beacon}).
		TryBlockAndAggregate()
	if err != nil {
		return nil, metadataBytes, fmt.Errorf("read Ammalgam factory: %w", err)
	}
	if err := requireAllCallsSucceeded(initial, 2); err != nil {
		return nil, metadataBytes, err
	}
	if initial.BlockNumber == nil || !initial.BlockNumber.IsUint64() || total == nil || !total.IsInt64() ||
		total.Sign() < 0 || total.Int64() > int64(^uint(0)>>1) || !validPairAddress(beacon.Hex()) {
		return nil, metadataBytes, errors.New("invalid Ammalgam factory response")
	}
	blockNumber := initial.BlockNumber

	var implementation common.Address
	implementationResponse, err := u.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber).
		AddCall(&ethrpc.Call{ABI: beaconABI, Target: beacon.Hex(), Method: "implementation"}, []any{&implementation}).
		TryAggregate()
	if err != nil {
		return nil, metadataBytes, fmt.Errorf("read Ammalgam pair implementation: %w", err)
	}
	if err := requireAllCallsSucceeded(implementationResponse, 1); err != nil {
		return nil, metadataBytes, err
	}
	if err := validateFactoryMetadata(u.config, beacon, implementation); err != nil {
		return nil, metadataBytes, err
	}

	count := discoveryBatchSize(int(total.Int64()), metadata.Offset, u.config.NewPoolLimit)
	if count == 0 {
		return nil, metadataBytes, nil
	}
	pairs := make([]common.Address, count)
	req := u.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	for i := range pairs {
		req.AddCall(&ethrpc.Call{
			ABI: factoryABI, Target: u.config.FactoryAddress, Method: "allPairs",
			Params: []any{big.NewInt(int64(metadata.Offset + i))},
		}, []any{&pairs[i]})
	}
	pairResponse, err := req.TryAggregate()
	if err != nil {
		return nil, metadataBytes, fmt.Errorf("list Ammalgam pairs: %w", err)
	}
	if err := requireAllCallsSucceeded(pairResponse, count); err != nil {
		return nil, metadataBytes, err
	}

	tokens := make([]struct {
		TokenX common.Address
		TokenY common.Address
	}, count)
	req = u.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)
	for i, pair := range pairs {
		if !validPairAddress(pair.Hex()) {
			return nil, metadataBytes, fmt.Errorf("invalid Ammalgam pair at index %d", metadata.Offset+i)
		}
		req.AddCall(&ethrpc.Call{
			ABI: pairABI, Target: pair.Hex(), Method: "underlyingTokens",
		}, []any{&tokens[i]})
	}
	tokenResponse, err := req.TryAggregate()
	if err != nil {
		return nil, metadataBytes, fmt.Errorf("read Ammalgam pair tokens: %w", err)
	}
	if err := requireAllCallsSucceeded(tokenResponse, count); err != nil {
		return nil, metadataBytes, err
	}

	gas := u.config.Gas
	if gas <= 0 {
		gas = defaultGas
	}
	staticExtraBytes, err := json.Marshal(StaticExtra{
		FactoryAddress:     strings.ToLower(u.config.FactoryAddress),
		PairBeacon:         hexutil.Encode(beacon[:]),
		PairImplementation: hexutil.Encode(implementation[:]),
		Gas:                gas,
	})
	if err != nil {
		return nil, metadataBytes, err
	}
	extraBytes, err := json.Marshal(Extra{})
	if err != nil {
		return nil, metadataBytes, err
	}
	now := time.Now().Unix()
	pools := make([]entity.Pool, count)
	for i := range pairs {
		x, y := tokens[i].TokenX, tokens[i].TokenY
		if !validPairAddress(x.Hex()) || !validPairAddress(y.Hex()) || x == y {
			return nil, metadataBytes, fmt.Errorf("invalid Ammalgam tokens for pair %s", pairs[i].Hex())
		}
		pools[i] = entity.Pool{
			Address:     hexutil.Encode(pairs[i][:]),
			Exchange:    u.config.DexID,
			Type:        DexType,
			BlockNumber: blockNumber.Uint64(),
			Timestamp:   now,
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(x[:]), Swappable: true},
				{Address: hexutil.Encode(y[:]), Swappable: true},
			},
			Reserves:    entity.PoolReserves{"0", "0"},
			Extra:       string(extraBytes),
			StaticExtra: string(staticExtraBytes),
		}
	}
	nextMetadata, err := json.Marshal(PoolsListUpdaterMetadata{Offset: metadata.Offset + count})
	if err != nil {
		return nil, metadataBytes, err
	}
	return pools, nextMetadata, nil
}
