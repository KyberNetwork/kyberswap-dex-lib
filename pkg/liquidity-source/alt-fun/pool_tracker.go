package altfun

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

// rpcGetTokenInfoResult wraps the Bonding.TokenInfo tuple for ethrpc decoding.
// Field order in the anonymous struct must match the ABI tuple exactly.
// go-ethereum's copyAtomic sets the decoded tuple into Data (Field 0), then setStruct
// copies field-by-field. Urls is [3]string because go-ethereum decodes string[3] as [3]string.
type rpcGetTokenInfoResult struct {
	Data struct {
		Creator     common.Address
		Pair        common.Address
		LtAddress   common.Address
		Name        string
		Ticker      string
		Description string
		Image       string
		Urls        [3]string
		Lifecycle   uint8
	}
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{"address": p.Address}).Info("[alt-fun] getting pool state")

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	tokenAddr := common.HexToAddress(p.Address)

	// Phase 1: get Bonding.TokenInfo to confirm lifecycle and resolve addresses.
	var result rpcGetTokenInfoResult
	req1 := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req1.SetOverrides(overrides)
	}
	req1.AddCall(&ethrpc.Call{
		ABI:    bondingABI,
		Target: t.config.BondingAddress,
		Method: "getTokenInfo",
		Params: []any{tokenAddr},
	}, []any{&result})
	if _, err := req1.Aggregate(); err != nil {
		return p, err
	}

	tokenInfo := result.Data
	lifecycle := Lifecycle(tokenInfo.Lifecycle)
	if lifecycle != LifecycleCurve {
		// Pool is graduating or graduated — zero reserves so routing skips it.
		// The simulator also guards via ErrPoolGraduated, but zeroing here ensures
		// the pool store doesn't serve stale non-zero reserves to pool-selection logic.
		var extra Extra
		if len(p.Extra) > 0 {
			_ = json.Unmarshal([]byte(p.Extra), &extra)
		}
		extra.Lifecycle = lifecycle
		b, _ := json.Marshal(extra)
		p.Extra = string(b)
		p.Reserves = entity.PoolReserves{"0", "0"}
		p.Timestamp = time.Now().Unix()
		return p, nil
	}

	pairAddr := tokenInfo.Pair.Hex()

	// Phase 2: fetch only bonding-curve Pair state.
	// LT pricing (exchangeRate, fees, mintPaused) is owned by the bounce-tech base pool.
	type reserveResult struct {
		TokenReserve *big.Int
		AssetReserve *big.Int
	}

	var (
		k            = new(big.Int)
		tokenBalance = new(big.Int)

		reserves = reserveResult{
			TokenReserve: new(big.Int),
			AssetReserve: new(big.Int),
		}
	)

	req2 := t.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req2.SetOverrides(overrides)
	}
	req2.
		AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: pairAddr,
			Method: "getReserves",
		}, []any{&reserves}).
		AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: pairAddr,
			Method: "k",
		}, []any{&k}).
		AddCall(&ethrpc.Call{
			ABI:    pairABI,
			Target: pairAddr,
			Method: "tokenBalance",
		}, []any{&tokenBalance})

	resp, err := req2.Aggregate()
	if err != nil {
		return p, err
	}

	extra := Extra{
		ReserveToken: uint256.MustFromBig(reserves.TokenReserve),
		ReserveAsset: uint256.MustFromBig(reserves.AssetReserve),
		K:            uint256.MustFromBig(k),
		TokenBalance: uint256.MustFromBig(tokenBalance),
		Lifecycle:    LifecycleCurve,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		reserves.TokenReserve.String(),
		tokenBalance.String(),
	}

	return p, nil
}
