package doppleramm

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	if err := cfg.withDefaults(); err != nil {
		return nil, err
	}
	return &PoolTracker{cfg: cfg, ethrpcClient: ethrpcClient}, nil
}

type marketReserves struct {
	ReserveX  *big.Int `abi:"reserveX"`
	ReserveY  *big.Int `abi:"reserveY"`
	Liquidity *big.Int `abi:"liquidity"`
}

type protocolFee struct {
	SwapFeeShareBps uint16         `abi:"swapFeeShareBps"`
	SwapBaseFeeBps  uint16         `abi:"swapBaseFeeBps"`
	Account         common.Address `abi:"account"`
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}
	marketID, ok := new(big.Int).SetString(staticExtra.MarketID, 0)
	if !ok {
		return p, ErrUnsupportedMarket
	}

	if staticExtra.Module != "" {
		return t.getLaunchState(ctx, p, staticExtra, marketID)
	}

	var reserves marketReserves
	var fee protocolFee
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: ammABI, Target: staticExtra.AMM, Method: "reserves", Params: []any{marketID}}, []any{&reserves}).
		AddCall(&ethrpc.Call{ABI: ammABI, Target: staticExtra.AMM, Method: "protocolFeeForMarket", Params: []any{marketID}}, []any{&fee}).
		TryBlockAndAggregate()
	if err != nil {
		return p, err
	}
	if len(resp.Result) != 2 || !resp.Result[0] || !resp.Result[1] {
		return p, ErrCallReverted
	}
	if reserves.ReserveX == nil || reserves.ReserveY == nil ||
		reserves.ReserveX.Sign() <= 0 || reserves.ReserveY.Sign() <= 0 {
		return p, ErrInsufficientLiquidity
	}

	// The LP fee and its recipient are fixed per market: read them from the config blob once.
	var prev Extra
	_ = json.Unmarshal([]byte(p.Extra), &prev)
	if prev.FeeRecipient == "" {
		if err = t.readSpotConfig(ctx, staticExtra.AMM, marketID, &prev); err != nil {
			return p, err
		}
	}

	extra, err := json.Marshal(Extra{TokenX: staticExtra.TokenX, BaseFeeBps: fee.SwapBaseFeeBps, FeeShareBps: fee.SwapFeeShareBps,
		LpFeeBps: prev.LpFeeBps, FeeRecipient: prev.FeeRecipient, LpFeeStays: prev.LpFeeStays})
	if err != nil {
		return p, err
	}
	p.Extra = string(extra)
	p.Reserves = entity.PoolReserves{reserves.ReserveX.String(), reserves.ReserveY.String()}
	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}
	p.Timestamp = time.Now().Unix()
	return p, nil
}

type marketConfig struct {
	TokenX        common.Address `abi:"tokenX"`
	TokenY        common.Address `abi:"tokenY"`
	Extension     common.Address `abi:"extension"`
	Unused3       *big.Int       `abi:"unused3"`
	Module        common.Address `abi:"module"`
	Unused5       [32]byte       `abi:"unused5"`
	ConfigPointer common.Address `abi:"configPointer"`
}

// A spot market's config blob (the code of marketConfig.configPointer) is a zero byte followed by nine
// words: tokenX, tokenY, extension, module, salt, 0, 0, LP fee recipient, LP fee in bps.
const (
	spotConfigWords     = 9
	spotConfigModuleIdx = 3
	spotConfigFeeToIdx  = 7
	spotConfigLpFeeIdx  = 8
)

// readSpotConfig fills the LP fee, its recipient and whether the recipient keeps the fee in the market.
// On chain a swap pays the protocol its base fee and 7.5% of the LP fee out of the input reserve. The
// rest of the LP fee stays in the reserve when the recipient is a contract and is paid out to it when
// it is an EOA (observed on every Base market; the AMM is unverified).
func (t *PoolTracker) readSpotConfig(ctx context.Context, amm string, marketID *big.Int, into *Extra) error {
	var cfg marketConfig
	if _, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: ammABI, Target: amm, Method: "marketConfig", Params: []any{marketID}}, []any{&cfg}).
		Call(); err != nil {
		return err
	}
	client := t.ethrpcClient.GetETHClient()
	blob, err := client.CodeAt(ctx, cfg.ConfigPointer, nil)
	if err != nil {
		return err
	}
	if len(blob) != 1+spotConfigWords*32 {
		return ErrUnsupportedMarket
	}
	word := func(i int) []byte { return blob[1+i*32 : 1+(i+1)*32] }
	lpFee := new(big.Int).SetBytes(word(spotConfigLpFeeIdx))
	if common.BytesToAddress(word(spotConfigModuleIdx)) != cfg.Module || !lpFee.IsUint64() || lpFee.Uint64() > bpsDenominator {
		return ErrUnsupportedMarket
	}
	recipient := common.BytesToAddress(word(spotConfigFeeToIdx))
	recipientCode, err := client.CodeAt(ctx, recipient, nil)
	if err != nil {
		return err
	}
	into.LpFeeBps = uint16(lpFee.Uint64())
	into.FeeRecipient = hexutil.Encode(recipient[:])
	into.LpFeeStays = len(recipientCode) > 0
	return nil
}
