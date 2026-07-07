package metricpropamm

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/go-resty/resty/v2"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/axima"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// poolABI exposes the Uniswap-V4-style external storage read used to check the
// pool's pause state on-chain. _setPause is packed in storage slot 0; its low
// byte is the pause level (0 = active).
var poolABI, _ = abi.JSON(strings.NewReader(
	`[{"inputs":[{"type":"bytes32"}],"name":"extsload","outputs":[{"type":"bytes32"}],"stateMutability":"view","type":"function"}]`))

type bidAsk struct {
	Bid                  string      `json:"bidAdj"`
	Ask                  string      `json:"askAdj"`
	TotalToken0Available string      `json:"totalToken0Available"`
	TotalToken1Available string      `json:"totalToken1Available"`
	ServerTs             int64       `json:"serverTs"`
	Depth                axima.Depth `json:"depth"`
}

type PoolTracker struct {
	config       *axima.Config
	client       *resty.Client
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(config *axima.Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	client := resty.NewWithClient(http.DefaultClient).
		SetBaseURL(config.HTTPConfig.BaseURL).
		SetTimeout(config.HTTPConfig.Timeout.Duration).
		SetRetryCount(config.HTTPConfig.RetryCount)
	if config.HTTPConfig.APIKey != "" {
		client = client.SetAuthToken(config.HTTPConfig.APIKey)
	}
	return &PoolTracker{config: config, client: client, ethrpcClient: ethrpcClient}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context, p entity.Pool, _ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context, p entity.Pool, _ poolpkg.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p)
}

func (t *PoolTracker) getNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	poolAddr := strings.ToLower(p.Address)

	unswappable := false
	var staticExtra axima.StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err == nil && staticExtra.SwapWhitelistingEnabled {
		unswappable = true
	} else if t.ethrpcClient != nil {
		var slot0 [32]byte
		if _, err := t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: p.Address,
			Method: "extsload",
			Params: []any{[32]byte{}}, // storage slot 0; low byte = pause level (0 = active)
		}, []any{&slot0}).Call(); err != nil {
			logger.WithFields(logger.Fields{"dexType": DexType, "pool": p.Address}).
				Warnf("failed to read on-chain pause state: %v", err)
		} else if slot0[31] != 0 {
			unswappable = true
		}
	}

	var ba bidAsk
	var fetchErr error
	if !unswappable {
		res, err := t.client.R().
			SetContext(ctx).
			SetResult(&ba).
			Get(fmt.Sprintf("/public/v1/evm/%d/%s/bid_ask", t.config.ChainID, poolAddr))
		if err != nil {
			fetchErr = err
		} else if res.IsError() {
			fetchErr = fmt.Errorf("bid_ask API error: %s", res.String())
		}
	}
	if unswappable || fetchErr != nil {
		if fetchErr != nil {
			logger.WithFields(logger.Fields{"dexType": DexType, "pool": poolAddr}).
				Warnf("failed to fetch bid/ask: %v", fetchErr)
		}
		unavailable, _ := json.Marshal(axima.Extra{QuoteAvailable: false, MaxAge: t.config.MaxAge, IsV2: true})
		p.Extra = string(unavailable)
		return p, nil
	}

	extra, err := json.Marshal(axima.Extra{
		InitBid:        bignumber.NewBig(ba.Bid),
		InitAsk:        bignumber.NewBig(ba.Ask),
		QuoteAvailable: true,
		MaxAge:         t.config.MaxAge,
		IsV2:           true,
		Bids:           axima.ConvertBins(ba.Depth.Bids),
		Asks:           axima.ConvertBins(ba.Depth.Asks),
	})
	if err != nil {
		return entity.Pool{}, err
	}

	p.Reserves = []string{ba.TotalToken0Available, ba.TotalToken1Available}
	p.Extra = string(extra)
	p.Timestamp = time.Now().Unix()
	return p, nil
}
