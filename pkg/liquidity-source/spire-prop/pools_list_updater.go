package spireprop

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
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	cfg    *Config
	client *ethrpc.Client
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(cfg *Config, client *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{cfg, client}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadata []byte) ([]entity.Pool, []byte, error) {
	if !validAddress(u.cfg.Entrypoint) {
		return nil, metadata, ErrInvalidState
	}
	known := map[string]bool{}
	if len(metadata) != 0 {
		if err := json.Unmarshal(metadata, &known); err != nil {
			return nil, metadata, err
		}
	}
	var curve, custody, quote common.Address
	req := u.client.NewRequest().SetContext(ctx)
	for i, method := range []string{"curveBook", "custodian", "quoteToken"} {
		req.AddCall(&ethrpc.Call{ABI: entrypointABI, Target: u.cfg.Entrypoint, Method: method}, []any{[]*common.Address{&curve, &custody, &quote}[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, metadata, err
	}
	if curve == (common.Address{}) || custody == (common.Address{}) || quote == (common.Address{}) {
		return nil, metadata, ErrInvalidState
	}
	extra, err := json.Marshal(StaticExtra{Entrypoint: strings.ToLower(u.cfg.Entrypoint), CurveBook: hexutil.Encode(curve[:]), Custodian: hexutil.Encode(custody[:])})
	if err != nil {
		return nil, metadata, err
	}
	pools := make([]entity.Pool, 0, len(u.cfg.Bases))
	for _, token := range u.cfg.Bases {
		if !validAddress(token) || common.HexToAddress(token) == quote {
			return nil, metadata, ErrInvalidToken
		}
		base := common.HexToAddress(token)
		// Unique per entrypoint/base while every pair can share custodian limits.
		hash := crypto.Keccak256(common.HexToAddress(u.cfg.Entrypoint).Bytes(), base.Bytes())
		address := hexutil.Encode(hash[12:])
		if known[address] {
			continue
		}
		pools = append(pools, entity.Pool{Address: address, Exchange: u.cfg.DexID, Type: DexType, Timestamp: time.Now().Unix(),
			Tokens:   []*entity.PoolToken{{Address: hexutil.Encode(base[:]), Swappable: true}, {Address: hexutil.Encode(quote[:]), Swappable: true}},
			Reserves: entity.PoolReserves{"0", "0"}, Extra: "{}", StaticExtra: string(extra)})
		known[address] = true
	}
	next, err := json.Marshal(known)
	return pools, next, err
}
