package doppler

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type DHook struct { // scheduled
	uniswapv4.Hook `json:"-"`
	IDHook         `json:"-"`
	Asset          common.Address  `json:"a"`
	DHook          common.Address  `json:"h"`
	HookExtra      json.RawMessage `json:"e"`
}

var _ = uniswapv4.RegisterHooksFactory(func(param *uniswapv4.HookParam) uniswapv4.Hook {
	var dHook DHook
	if param.HookExtra != "" {
		_ = json.Unmarshal([]byte(param.HookExtra), &dHook)
	}
	dHook.Hook = &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Doppler}
	if f := DHooks[dHook.DHook]; f != nil {
		dHook.IDHook = f(dHook.HookExtra)
	}
	return &dHook
}, InitializerAddresses...)

type PoolState struct {
	Numeraire                     common.Address
	TotalTokensOnBondingCurve     *big.Int
	DopplerHook                   common.Address
	GraduationDopplerHookCalldata []byte
	Status                        uint8
	PoolKey                       uniswapv4.PoolKey
	FarTick                       int32
}

func (h *DHook) Track(ctx context.Context, param *uniswapv4.HookParam) (string, error) {
	var poolState, poolState2 PoolState

	req := param.RpcClient.NewRequest().SetContext(ctx)
	paramHook := hexutil.Encode(param.HookAddress[:])
	var err error
	if h.Asset != valueobject.AddrZero {
		_, err = req.AddCall(&ethrpc.Call{
			ABI:    poolStateABI,
			Target: paramHook,
			Method: "getState",
			Params: []any{h.Asset},
		}, []any{&poolState}).Call()
	} else {
		_, err = req.AddCall(&ethrpc.Call{
			ABI:    poolStateABI,
			Target: paramHook,
			Method: "getState",
			Params: []any{common.HexToAddress(param.Pool.Tokens[0].Address)},
		}, []any{&poolState}).AddCall(&ethrpc.Call{
			ABI:    poolStateABI,
			Target: paramHook,
			Method: "getState",
			Params: []any{common.HexToAddress(param.Pool.Tokens[1].Address)},
		}, []any{&poolState2}).TryAggregate()
	}
	if err != nil {
		return "", err
	}

	if h.Asset == valueobject.AddrZero {
		if poolState.Numeraire == valueobject.AddrZero {
			poolState = poolState2
			h.Asset = poolState.PoolKey.Currency1
		} else {
			h.Asset = poolState.PoolKey.Currency0
		}
	}
	if h.DHook != poolState.DopplerHook {
		h.DHook = poolState.DopplerHook
		if f := DHooks[h.DHook]; f != nil {
			h.IDHook = f(h.HookExtra)
		}
	}

	if h.IDHook != nil {
		if h.HookExtra, err = h.IDHook.Track(ctx, param, h); err != nil {
			return "", err
		}
	}

	extraBytes, _ := json.Marshal(h)
	return string(extraBytes), nil
}

func (h *DHook) AfterSwap(params *uniswapv4.AfterSwapParams) (*uniswapv4.AfterSwapResult, error) {
	if h.IDHook != nil {
		return h.IDHook.AfterSwap(params, h)
	}
	return h.Hook.AfterSwap(params)
}

type IDHook interface {
	Track(ctx context.Context, param *uniswapv4.HookParam, dExtra *DHook) (json.RawMessage, error)
	AfterSwap(params *uniswapv4.AfterSwapParams, dExtra *DHook) (*uniswapv4.AfterSwapResult, error)
}

var MaxSwapFee = big.NewInt(1e6)

type RehypeDHook struct {
	CustomFee *big.Int `json:"f"`
}

func NewRehypeDHook(dExtra json.RawMessage) IDHook {
	var hook RehypeDHook
	_ = json.Unmarshal(dExtra, &hook)
	return &hook
}

type HookFees struct {
	Fees0             *big.Int
	Fees1             *big.Int
	BeneficiaryFees0  *big.Int
	BeneficiaryFees1  *big.Int
	AirlockOwnerFees0 *big.Int
	AirlockOwnerFees1 *big.Int
	CustomFee         *big.Int
}

func (h *RehypeDHook) Track(ctx context.Context, param *uniswapv4.HookParam, dHook *DHook) (json.RawMessage, error) {
	var hookFees HookFees
	if _, err := param.RpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    rehypeDopplerHookABI,
		Target: hexutil.Encode(dHook.DHook[:]),
		Method: "getHookFees",
		Params: []any{common.HexToHash(param.Pool.Address)},
	}, []any{&hookFees}).Call(); err != nil {
		return nil, err
	}

	h.CustomFee = hookFees.CustomFee
	return json.Marshal(h)
}

func (h *RehypeDHook) AfterSwap(params *uniswapv4.AfterSwapParams, _ *DHook) (*uniswapv4.AfterSwapResult, error) {
	var hookFee big.Int
	return &uniswapv4.AfterSwapResult{
		HookFee: hookFee.Mul(lo.Ternary(params.ExactIn, params.AmountOut, params.AmountIn), h.CustomFee).
			Div(&hookFee, MaxSwapFee),
		Gas: 198835,
	}, nil
}
