package gblin

import (
	"context"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config         *Config
	ethrpcClient   *ethrpc.Client
	hasInitialized bool
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

// GetNewPools lists the single GBLIN vault. The vault is its own share token and the only mint entry point for
// WETH, so the pool is the vault itself with tokens [WETH, GBLIN].
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	if u.hasInitialized {
		return nil, metadataBytes, nil
	}

	var sequencerFeed common.Address
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    lensABI,
		Target: u.config.Lens,
		Method: lensMethodSequencerFeed,
		Params: []any{common.HexToAddress(u.config.Vault)},
	}, []any{&sequencerFeed}).Call(); err != nil {
		logger.WithFields(logger.Fields{"dexId": u.config.DexID, "error": err}).Error("failed to read sequencer feed")
		return nil, metadataBytes, err
	}

	staticExtra := StaticExtra{Lens: strings.ToLower(u.config.Lens)}
	if sequencerFeed != (common.Address{}) {
		staticExtra.SequencerFeed = hexutil.Encode(sequencerFeed[:])
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return nil, metadataBytes, err
	}

	vault := strings.ToLower(u.config.Vault)
	p := entity.Pool{
		Address:  vault,
		Exchange: u.config.DexID,
		Type:     DexType,
		Reserves: entity.PoolReserves{reserveZero, reserveZero},
		Tokens: []*entity.PoolToken{
			{Address: valueobject.WrappedNativeMap[u.config.ChainID], Swappable: true},
			{Address: vault, Swappable: true},
		},
		StaticExtra: string(staticExtraBytes),
	}
	p.Tokens[0].Address = strings.ToLower(p.Tokens[0].Address)

	u.hasInitialized = true
	return []entity.Pool{p}, metadataBytes, nil
}
