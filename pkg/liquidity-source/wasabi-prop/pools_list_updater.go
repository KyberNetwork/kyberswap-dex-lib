package wasabiprop

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type (
	PoolsListUpdater struct {
		cfg          *Config
		ethrpcClient *ethrpc.Client
	}

	PoolsListUpdaterMetadata struct {
		Offset int `json:"offset"`
	}
)

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	log := logger.WithFields(logger.Fields{"dexId": u.cfg.DexID})
	log.Info("wasabi-prop: start get pools")

	metadata, err := u.getMetadata(metadataBytes)
	if err != nil {
		log.Warnf("wasabi-prop: getMetadata failed: %v", err)
	}

	// 1. Get listed tokens from factory
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	var listed []common.Address
	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: u.cfg.FactoryAddress,
		Method: "getListedTokens",
	}, []any{&listed})

	if _, err := req.TryAggregate(); err != nil {
		log.Errorf("wasabi-prop: getListedTokens failed: %v", err)
		return nil, metadataBytes, err
	}

	if metadata.Offset > len(listed) {
		metadata.Offset = 0
	}
	if metadata.Offset == len(listed) {
		return nil, metadataBytes, nil
	}

	tokens := listed[metadata.Offset:]

	// 2. Get pool address for each token
	req2 := u.ethrpcClient.NewRequest().SetContext(ctx)
	poolAddrs := make([]common.Address, len(tokens))
	for i, tok := range tokens {
		req2.AddCall(&ethrpc.Call{
			ABI:    factoryABI,
			Target: u.cfg.FactoryAddress,
			Method: "getPropPool",
			Params: []any{tok},
		}, []any{&poolAddrs[i]})
	}

	if _, err := req2.TryAggregate(); err != nil {
		log.Errorf("wasabi-prop: getPropPool calls failed: %v", err)
		return nil, metadataBytes, err
	}

	// 3. Find first valid pool and get quote token
	var firstPoolAddr string
	for _, addr := range poolAddrs {
		if addr != (common.Address{}) {
			firstPoolAddr = addr.Hex()
			break
		}
	}
	if firstPoolAddr == "" {
		log.Warn("wasabi-prop: no valid pools found")
		newMeta, _ := u.newMetadata(metadata.Offset + len(tokens))
		return nil, newMeta, nil
	}

	req3 := u.ethrpcClient.NewRequest().SetContext(ctx)
	var quoteToken common.Address
	req3.AddCall(&ethrpc.Call{
		ABI:    poolABI,
		Target: firstPoolAddr,
		Method: "getQuoteToken",
	}, []any{&quoteToken})

	if _, err := req3.TryAggregate(); err != nil {
		log.Errorf("wasabi-prop: getQuoteToken failed: %v", err)
		return nil, metadataBytes, err
	}

	// 4. Create pools
	staticExtraBytes, _ := json.Marshal(StaticExtra{
		RouterAddress: strings.ToLower(u.cfg.RouterAddress),
	})

	pools := make([]entity.Pool, 0, len(tokens))
	now := time.Now().Unix()

	for i, token := range tokens {
		if poolAddrs[i] == (common.Address{}) {
			continue
		}
		p := entity.Pool{
			Address:   strings.ToLower(poolAddrs[i].Hex()),
			Exchange:  u.cfg.DexID,
			Type:      DexType,
			Timestamp: now,
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(token.Hex()), Swappable: true},
				{Address: strings.ToLower(quoteToken.Hex()), Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		}
		pools = append(pools, p)
	}

	newMetadata, err := u.newMetadata(metadata.Offset + len(tokens))
	if err != nil {
		log.Warnf("wasabi-prop: newMetadata failed: %v", err)
		return pools, metadataBytes, nil
	}

	return pools, newMetadata, nil
}

func (u *PoolsListUpdater) getMetadata(metadataBytes []byte) (PoolsListUpdaterMetadata, error) {
	if len(metadataBytes) == 0 {
		return PoolsListUpdaterMetadata{}, nil
	}
	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return PoolsListUpdaterMetadata{}, err
	}
	return metadata, nil
}

func (u *PoolsListUpdater) newMetadata(offset int) ([]byte, error) {
	metadata := PoolsListUpdaterMetadata{Offset: offset}
	return json.Marshal(metadata)
}
