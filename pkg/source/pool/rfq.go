package pool

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// RFQParams is the params for firm quote operations such as calling firm-quote API
type RFQParams struct {
	NetworkID    valueobject.ChainID // blockchain network id
	RequestID    string              // request id from getRoute
	PoolID       string              // pool id
	Origin       string              // original address
	Sender       string              // swap tx origin
	Recipient    string              // fund recipient of swap tx
	RFQSender    string              // RFQ caller (executor)
	RFQRecipient string              // RFQ fund recipient (executor/next pool/recipient)
	Source       string              // source client
	TokenIn      string              // address of token swap from
	TokenOut     string              // address of token swap to
	SwapAmount   *big.Int            // amount of TokenIn to swap
	AmountOut    *big.Int            // amount of TokenOut received
	Slippage     int64               // tolerance (in bps) for RFQs that also aggregate dexes
	PoolExtra    any                 // extra pool metadata
	SwapInfo     any                 // swap info of the RFQ swap
	FeeInfo      any                 // generic fee info
}

func (r *RFQParams) GetOrigin() string {
	if r.Origin != "" {
		return r.Origin
	}
	return r.Sender
}

// RFQResult is the result for firm quote operations
type RFQResult struct {
	NewAmountOut *big.Int
	Extra        any
}

// RFQHandler is the default no-op RFQ handler
type RFQHandler struct{}

func (h *RFQHandler) RFQ(context.Context, []RFQParams) (*RFQResult, error) {
	return nil, nil
}

func (h *RFQHandler) BatchRFQ(context.Context, []RFQParams) ([]*RFQResult, error) {
	return nil, nil
}

func (h *RFQHandler) SupportBatch() bool {
	return false
}

// RFQSequentialBatcher knows how to batch RFQs sequentially
type RFQSequentialBatcher struct {
	IPoolSingleRFQ
}

func (h *RFQSequentialBatcher) BatchRFQ(ctx context.Context, paramsSlice []RFQParams) (results []*RFQResult,
	err error) {
	results = make([]*RFQResult, len(paramsSlice))
	for i, params := range paramsSlice {
		if results[i], err = h.RFQ(ctx, params); err != nil {
			return nil, err
		}
	}
	return results, nil
}

func (h *RFQSequentialBatcher) SupportBatch() bool {
	return true
}

// RFQWithPoolState knows how to load pool state for simulations before RFQ or BatchRFQ call
type RFQWithPoolState struct {
	IPoolRFQ
	IPoolManager
	ICustomFuncs
	DexId string
}

func (h *RFQWithPoolState) RFQ(ctx context.Context, params RFQParams) (*RFQResult, error) {
	ctx = h.GetAndClonePoolState(ctx, params.PoolID)
	return h.IPoolRFQ.RFQ(ctx, params)
}

func (h *RFQWithPoolState) BatchRFQ(ctx context.Context, paramsSlice []RFQParams) (results []*RFQResult, err error) {
	ctx = h.GetAndClonePoolState(ctx, lo.Map(paramsSlice, func(p RFQParams, _ int) string { return p.PoolID })...)
	return h.IPoolRFQ.BatchRFQ(ctx, paramsSlice)
}

func (h *RFQWithPoolState) GetAndClonePoolState(ctx context.Context, poolAddrs ...string) context.Context {
	poolState, _ := h.GetStateByPoolAddresses(ctx, poolAddrs, []string{h.DexId}, PoolManagerExtraData{})
	if poolState == nil {
		poolState = &FindRouteState{}
	}
	for addr, poolSim := range poolState.Pools {
		poolState.Pools[addr] = h.ClonePool(ctx, poolSim)
	}
	for dex, limit := range poolState.SwapLimit {
		poolState.SwapLimit[dex] = h.CloneSwapLimit(ctx, limit)
	}
	return context.WithValue(ctx, h, poolState)
}

func (h *RFQWithPoolState) PoolState(ctx context.Context) *FindRouteState {
	poolState, _ := ctx.Value(h).(*FindRouteState)
	return poolState
}

type IPoolManager interface {
	// GetStateByPoolAddresses return a map of address - pools and a map of dexType- swapLimit for
	GetStateByPoolAddresses(ctx context.Context, poolAddresses, dex []string,
		extraData PoolManagerExtraData) (*FindRouteState, error)
}

type PoolManagerExtraData struct {
	KyberLimitOrderAllowedSenders string
}

type FindRouteState struct {
	Pools                   map[string]IPoolSimulator // map PoolAddress-IPoolSimulator implementation
	SwapLimit               map[string]SwapLimit      // map dexType-SwapLimit
	StateRoot               common.Hash               // aevm state root
	PublishedPoolsStorageID string                    // last published pools
}
