package hyfiprop

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Hook prices HyFi pairs. HyFi is a v4 hook, but we execute through swapExactInDirect, which prices
// with the same code (HyFi._trade) and skips the PoolManager and its protocol fee. The pool holds no
// v4 liquidity. Each side of the book is an updater-pushed ladder of up to 68 ticks; a trade only
// moves the walk pointer (curTick, amountLeft) in slot0. The staleness fee grows per second since
// the book timestamp, so quotes use the current time. The output is paid from the contract's own
// token balances, which are shared across pairs.
type Hook struct {
	uniswapv4.Hook `json:"-"`

	Cfg       PairConfig     `json:"c"`
	Book      [2]Side        `json:"k"` // bid, ask
	Inventory [2]uint256.Int `json:"i"` // contract balance of currency0/1
}

// SwapInfo is what UpdateBalance needs to advance the walked side and move the inventory.
type SwapInfo struct {
	ZeroForOne    bool        `json:"z"`
	IsSellingBase bool        `json:"sb"`
	Cur           uint64      `json:"c"`
	Left          uint256.Int `json:"l"`
	AmountIn      uint256.Int `json:"in"`
	AmountOut     uint256.Int `json:"out"`
}

type pairConfigRPC struct {
	TickWidth    *big.Int `abi:"tickWidth"`
	BaseLiqUnit  *big.Int `abi:"baseLiqUnit"`
	FeePerSecond *big.Int `abi:"feePerSecond"`
	BaseIsC0     bool     `abi:"baseIsCurrency0"`
}

type bookSideRPC struct {
	Slot0 *big.Int `abi:"slot0"`
	WordA *big.Int `abi:"wordA"`
	WordB *big.Int `abi:"wordB"`
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4HyFiProp},
	}
	_ = param.HookExtra.Unmarshal(hook)
	return hook
}, HookAddresses...)

// TracksActivity: hX holds only chain state, so a changed hX bumps Timestamp/BlockNumber and the
// pool stays active while the book moves, even with no log on the pool.
func (h *Hook) TracksActivity() {}

// AllowEmptyTicks: the pool has no v4 liquidity; the whole input is consumed in BeforeSwap.
func (h *Hook) AllowEmptyTicks() bool { return true }

func (h *Hook) GetReserves(context.Context, *uniswapv4.HookParam) (entity.PoolReserves, error) {
	return entity.PoolReserves{h.Inventory[0].Dec(), h.Inventory[1].Dec()}, nil
}

// Track reads the config, both book sides and the contract balances in one multicall, so they
// come from a single block.
func (h *Hook) Track(ctx context.Context, param *uniswapv4.HookParam) (json.RawMessage, error) {
	if len(param.Pool.Tokens) != 2 {
		return nil, ErrUnexpectedResponse
	}
	var staticExtra uniswapv4.StaticExtra
	if err := json.Unmarshal([]byte(param.Pool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	} else if staticExtra.IsNative[0] || staticExtra.IsNative[1] {
		return nil, fmt.Errorf("%w: native currency", ErrUnexpectedResponse)
	}

	poolId := common.HexToHash(param.Pool.Address)
	hookAddr := hexutil.Encode(param.HookAddress[:])
	var cfg pairConfigRPC
	var books [2]bookSideRPC
	var bals [2]*big.Int
	req := param.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(param.BlockNumber).
		SetOverrides(param.Overrides).
		AddCall(&ethrpc.Call{ABI: hyfiABI, Target: hookAddr, Method: "pairConfig",
			Params: []any{poolId}}, []any{&cfg})
	for i := range books { // isSellingBase: bid first
		req.AddCall(&ethrpc.Call{ABI: hyfiABI, Target: hookAddr, Method: "getBookSideRaw",
			Params: []any{poolId, i == sideBid}}, []any{&books[i]})
		req.AddCall(&ethrpc.Call{ABI: hyfiABI, Target: param.Pool.Tokens[i].Address, Method: "balanceOf",
			Params: []any{param.HookAddress}}, []any{&bals[i]})
	}
	if _, err := req.Aggregate(); err != nil {
		return nil, fmt.Errorf("failed to track hyfi: %w", err)
	}

	if cfg.TickWidth == nil || cfg.BaseLiqUnit == nil || cfg.FeePerSecond == nil || bals[0] == nil || bals[1] == nil {
		return nil, ErrUnexpectedResponse
	} else if cfg.TickWidth.Sign() == 0 {
		return nil, ErrPairNotConfigured
	}
	h.Cfg.TickWidth.SetFromBig(cfg.TickWidth)
	h.Cfg.BaseLiqUnit.SetFromBig(cfg.BaseLiqUnit)
	h.Cfg.FeePerSecond = uint32(cfg.FeePerSecond.Uint64())
	h.Cfg.BaseIsC0 = cfg.BaseIsC0
	for i := range books {
		if books[i].Slot0 == nil || books[i].WordA == nil || books[i].WordB == nil {
			return nil, ErrUnexpectedResponse
		}
		h.Book[i].Slot0.SetFromBig(books[i].Slot0)
		h.Book[i].WordA.SetFromBig(books[i].WordA)
		h.Book[i].WordB.SetFromBig(books[i].WordB)
		h.Inventory[i].SetFromBig(bals[i])
	}
	return json.Marshal(h)
}

func (h *Hook) BeforeSwap(params *uniswapv4.BeforeSwapParams) (*uniswapv4.BeforeSwapResult, error) {
	if params.AmountSpecified == nil || params.AmountSpecified.Sign() <= 0 {
		return nil, ErrInvalidAmount
	}
	var amt uint256.Int
	if overflow := amt.SetFromBig(params.AmountSpecified); overflow {
		return nil, ErrInvalidAmount
	}
	now := unixNow()

	// quote prices an exact-in trade; the contract transfers the output from its balance.
	quote := func(amountIn *uint256.Int) (*SwapInfo, error) {
		res, err := priceExactIn(&h.Book, &h.Cfg, params.ZeroForOne, amountIn, now)
		if err != nil {
			return nil, err
		} else if res.amountOut.IsZero() {
			return nil, ErrZeroOutput
		} else if res.amountOut.Gt(&h.Inventory[outIndex(params.ZeroForOne)]) {
			return nil, ErrInsufficientInv
		}
		return &SwapInfo{ZeroForOne: params.ZeroForOne, IsSellingBase: res.isSellingBase, Cur: res.cur,
			Left: res.left, AmountIn: *amountIn, AmountOut: res.amountOut}, nil
	}

	if params.CalcOut {
		info, err := quote(&amt)
		if err != nil {
			return nil, err
		}
		return &uniswapv4.BeforeSwapResult{
			DeltaSpecified:   info.AmountIn.ToBig(),
			DeltaUnspecified: new(big.Int).Neg(info.AmountOut.ToBig()),
			Gas:              gasBeforeSwap,
			SwapInfo:         info,
		}, nil
	}

	// CalcIn reverses exact-in (see uniswapv4.BeforeSwapParams): search the min amountIn.
	amountIn, err := uniswapv4.MinAmountIn(params.AmountSpecified, nil, func(amountIn *big.Int) (*big.Int, error) {
		var in uint256.Int
		if in.SetFromBig(amountIn) {
			return nil, ErrOverflow
		}
		info, err := quote(&in)
		if err != nil {
			return nil, err
		}
		return info.AmountOut.ToBig(), nil
	})
	if err != nil {
		return nil, err
	}
	var in uint256.Int
	in.SetFromBig(amountIn)
	info, err := quote(&in)
	if err != nil {
		return nil, err
	}
	return &uniswapv4.BeforeSwapResult{
		DeltaSpecified:   new(big.Int).Neg(params.AmountSpecified),
		DeltaUnspecified: amountIn,
		Gas:              gasBeforeSwap,
		SwapInfo:         info,
	}, nil
}

// gasBeforeSwap offsets the PoolManager swap gas the v4 simulator adds on top, so the total is
// the direct swap's.
var gasBeforeSwap = gasDirect - uniswapv4.DefaultGas.BaseGas

// unixNow is the quote clock; tests replace it.
var unixNow = defaultUnixNow

func defaultUnixNow() uint64 { return uint64(time.Now().Unix()) }

// outIndex is the index of the currency the taker receives.
func outIndex(zeroForOne bool) int {
	if zeroForOne {
		return 1
	}
	return 0
}

func (h *Hook) CloneState() uniswapv4.Hook {
	cloned := *h // all state is held by value
	return &cloned
}

// UpdateBalance moves the walk pointer of the walked side and the contract balances. The fee
// stays in the contract.
func (h *Hook) UpdateBalance(swapInfo any) {
	info, ok := swapInfo.(*SwapInfo)
	if !ok || info == nil {
		return
	}
	side := &h.Book[sideAsk]
	if info.IsSellingBase {
		side = &h.Book[sideBid]
	}
	persistWalk(side, info.Cur, &info.Left)
	outIdx := outIndex(info.ZeroForOne)
	h.Inventory[1-outIdx].Add(&h.Inventory[1-outIdx], &info.AmountIn)
	h.Inventory[outIdx].Sub(&h.Inventory[outIdx], &info.AmountOut)
}

var _ uniswapv4.Hook = (*Hook)(nil)
