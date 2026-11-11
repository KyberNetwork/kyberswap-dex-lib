package daikonprop

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: cfg, ethrpcClient: ethrpcClient}, nil
}

// GetNewPoolState reads the market state and the custody vault, then the vault's balance and
// router allowance of both tokens at the same block.
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	if len(p.Tokens) != 2 {
		return p, ErrInvalidToken
	}
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var (
		vault common.Address
		st    marketState
	)
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: daikonABI, Target: staticExtra.Router, Method: methodVault}, []any{&vault}).
		AddCall(&ethrpc.Call{ABI: daikonABI, Target: staticExtra.Manager, Method: methodGetMarketState,
			Params: []any{common.HexToAddress(p.Address)}}, []any{&st}).
		Aggregate()
	if err != nil {
		return p, err
	}

	var bal, allow [2]*big.Int
	router := common.HexToAddress(staticExtra.Router)
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
	for i, tok := range p.Tokens {
		req.AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: tok.Address, Method: "balanceOf", Params: []any{vault}},
			[]any{&bal[i]})
		req.AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: tok.Address, Method: "allowance",
			Params: []any{vault, router}}, []any{&allow[i]})
	}
	if _, err = req.Aggregate(); err != nil {
		return p, err
	}

	extra := Extra{
		BaseDecimals:  uint8(st.BaseDecimals.Uint64()),
		QuoteDecimals: uint8(st.QuoteDecimals.Uint64()),
		ObservedAtMs:  st.ObservedAtMs.Uint64(),
		TTL:           st.TTL.Uint64(),
		Spread1Bps:    st.Spread1Bps.Uint64(),
		Spread2Bps:    st.Spread2Bps.Uint64(),
		Paused:        st.Paused || !st.ManagerActive,
	}
	for dst, src := range map[**uint256.Int]*big.Int{
		&extra.Price: st.Price, &extra.BuyC1: st.BuyC1Rem, &extra.BuyC2: st.BuyC2Rem,
		&extra.SellC1: st.SellC1Rem, &extra.SellC2: st.SellC2Rem,
	} {
		v, overflow := uint256.FromBig(src)
		if overflow {
			return p, ErrOverflow
		}
		*dst = v
	}
	vaultCan := func(i int) *uint256.Int { return big256.Min(big256.FromBig(bal[i]), big256.FromBig(allow[i])).Clone() }
	extra.VaultBase, extra.VaultQuote = vaultCan(0), vaultCan(1)

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{extra.VaultBase.Dec(), extra.VaultQuote.Dec()}
	p.BlockNumber = resp.BlockNumber.Uint64()
	p.Timestamp = time.Now().Unix()
	return p, nil
}
