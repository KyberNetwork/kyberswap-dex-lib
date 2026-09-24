package prop

import (
	"context"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

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
	return &PoolsListUpdater{cfg: cfg, ethrpcClient: ethrpcClient}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	log := logger.WithFields(logger.Fields{"dexId": u.cfg.DexID})
	log.Info("kipseli-prop: start get pools")

	metadata, err := u.getMetadata(metadataBytes)
	if err != nil {
		log.Warnf("kipseli-prop: getMetadata failed: %v", err)
	}

	quoteToken, listed, err := u.fetchListedTokens(ctx)
	if err != nil {
		log.Errorf("kipseli-prop: lens calls failed: %v", err)
		return nil, metadataBytes, err
	}

	tokens := make([]common.Address, 0, len(listed))
	for _, t := range listed {
		if !strings.EqualFold(t.Hex(), quoteToken.Hex()) {
			tokens = append(tokens, t)
		}
	}

	if metadata.Offset > len(tokens) {
		metadata.Offset = 0
	}
	if metadata.Offset == len(tokens) {
		return nil, metadataBytes, nil
	}
	tokens = tokens[metadata.Offset:]

	staticExtraBytes, _ := json.Marshal(StaticExtra{RouterAddress: strings.ToLower(u.cfg.RouterAddress)})

	pools := make([]entity.Pool, 0, len(tokens))
	now := time.Now().Unix()
	for _, token := range tokens {
		pools = append(pools, entity.Pool{
			Address:   syntheticPoolAddress(u.cfg.DexID, token, quoteToken),
			Exchange:  u.cfg.DexID,
			Type:      DexType,
			Timestamp: now,
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: hexutil.Encode(token[:]), Swappable: true},
				{Address: hexutil.Encode(quoteToken[:]), Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		})
	}

	newMetadata, err := u.newMetadata(metadata.Offset + len(tokens))
	if err != nil {
		log.Warnf("kipseli-prop: newMetadata failed: %v", err)
		return pools, metadataBytes, nil
	}

	log.Infof("kipseli-prop: finish get %d new pools", len(pools))
	return pools, newMetadata, nil
}

// fetchListedTokens speaks whichever Lens ABI this chain's venue exposes:
// the prop venue's getQuoteToken()/getListedTokens(), or the pamm venue's
// QUOTE_TOKEN()/getListedTokens().
func (u *PoolsListUpdater) fetchListedTokens(ctx context.Context) (common.Address, []common.Address, error) {
	quoteTokenMethod, abi := "getQuoteToken", lensABI
	if u.cfg.isPamm() {
		quoteTokenMethod, abi = "QUOTE_TOKEN", lensPammABI
	}

	var quoteToken common.Address
	var listed []common.Address
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: abi, Target: u.cfg.LensAddress, Method: quoteTokenMethod}, []any{&quoteToken}).
		AddCall(&ethrpc.Call{ABI: abi, Target: u.cfg.LensAddress, Method: "getListedTokens"}, []any{&listed}).
		TryAggregate(); err != nil {
		return common.Address{}, nil, err
	}
	return quoteToken, listed, nil
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
	return json.Marshal(PoolsListUpdaterMetadata{Offset: offset})
}

func syntheticPoolAddress(dexID string, token, quote common.Address) string {
	return strings.ToLower(dexID + "_" + token.Hex() + "_" + quote.Hex())
}
