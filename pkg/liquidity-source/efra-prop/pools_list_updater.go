package efraprop

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
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{cfg: cfg, ethrpcClient: ethrpcClient}
}

// GetNewPools lists one pool per configured token against the venue's quote token. The venue
// has no token enumeration view (the maker announces a listing with a TokenSet event and
// tokenConfigs(token).enabled only answers for a known token), so the token set lives in
// config and the full list is returned on every poll: the caller skips pools it already has.
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var usdc common.Address
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: venueABI, Target: u.cfg.Venue, Method: "USDC"}, []any{&usdc}).
		Aggregate(); err != nil {
		return nil, metadataBytes, err
	}
	if usdc == (common.Address{}) {
		return nil, metadataBytes, ErrNoUSDC
	}

	pools, err := u.buildPools(usdc, time.Now().Unix())
	if err != nil {
		return nil, metadataBytes, err
	}
	return pools, metadataBytes, nil
}

// buildPools turns the configured tokens into pool records. Discovery sets no reserves: the
// tracker fills them from the venue's balances on its first refresh.
func (u *PoolsListUpdater) buildPools(usdc common.Address, now int64) ([]entity.Pool, error) {
	staticExtraBytes, err := json.Marshal(StaticExtra{
		Venue: strings.ToLower(u.cfg.Venue),
		Taker: strings.ToLower(u.cfg.dest()),
	})
	if err != nil {
		return nil, err
	}

	quote := hexutil.Encode(usdc[:])
	seen := make(map[string]struct{}, len(u.cfg.Tokens))
	pools := make([]entity.Pool, 0, len(u.cfg.Tokens))
	for _, token := range u.cfg.Tokens {
		addr := strings.ToLower(token)
		if !common.IsHexAddress(addr) {
			return nil, ErrInvalidToken
		}
		tokenAddr := hexutil.Encode(common.HexToAddress(addr).Bytes())
		if tokenAddr == quote {
			continue
		}
		if _, ok := seen[tokenAddr]; ok {
			continue
		}
		seen[tokenAddr] = struct{}{}

		pools = append(pools, entity.Pool{
			Address:   syntheticPoolAddress(u.cfg.DexID, tokenAddr, quote),
			Exchange:  u.cfg.DexID,
			Type:      DexType,
			Timestamp: now,
			Reserves:  entity.PoolReserves{"0", "0"},
			Tokens: []*entity.PoolToken{
				{Address: tokenAddr, Swappable: true},
				{Address: quote, Swappable: true},
			},
			Extra:       "{}",
			StaticExtra: string(staticExtraBytes),
		})
	}
	return pools, nil
}

// syntheticPoolAddress keys a market by its pair: every market of this venue shares one
// contract address, so the contract address cannot be the pool id.
func syntheticPoolAddress(dexID, token, quote string) string {
	return strings.ToLower(dexID) + "_" + token + "_" + quote
}
