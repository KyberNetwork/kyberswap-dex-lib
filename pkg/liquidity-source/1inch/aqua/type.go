package oneinchaqua

import (
	"github.com/holiman/uint256"
)

// StaticExtra is the immutable identity of a strategy: the order the router receives plus the registry key.
type StaticExtra struct {
	Maker        string `json:"maker"`
	App          string `json:"app"`
	StrategyHash string `json:"strategyHash"`
	// Launch is set when the app is the launch router, whose protocol fee pull reverts on a shortfall;
	// the lister sets it from its config.
	Launch bool `json:"launch,omitempty"`
	// Traits is the order's maker traits as a 0x hex uint256 and Data is order.data as 0x hex.
	Traits string `json:"traits"`
	Data   string `json:"data"`
}

// Extra is the mutable state of a strategy. Token order follows entity.Pool.Tokens.
type Extra struct {
	// RawBalance0/1 are the strategy's virtual balances (Aqua rawBalances) at RawBlock.
	RawBalance0 *uint256.Int `json:"rawBalance0"`
	RawBalance1 *uint256.Int `json:"rawBalance1"`
	RawBlock    uint64       `json:"rawBlock"`

	// WalletBalance and Allowance are the maker's balanceOf and allowance to Aqua at WalletBlock.
	// WalletBlock == 0 means the pool has no wallet snapshot yet.
	WalletBalance0 *uint256.Int `json:"walletBalance0"`
	WalletBalance1 *uint256.Int `json:"walletBalance1"`
	Allowance0     *uint256.Int `json:"allowance0"`
	Allowance1     *uint256.Int `json:"allowance1"`
	WalletBlock    uint64       `json:"walletBlock"`

	// Docked is tokensCount 0xff. It is permanent only once DockedFinal is set: until a refresh reads the
	// strategy docked at least `confirmations` blocks after DockedAtBlock (the block of the first docked
	// read), the pool is read again every refresh with a current Timestamp (so the indexer keeps it active
	// and refreshed), and a dock that a reorg removes goes back to live. A final dock has Timestamp 1.
	// Inactive is tokensCount other than 2 (a reorged, lagging-node or never shipped strategy, not
	// permanent): it is read again every refresh too.
	Docked        bool   `json:"docked,omitempty"`
	DockedAtBlock uint64 `json:"dockedAtBlock,omitempty"`
	DockedFinal   bool   `json:"dockedFinal,omitempty"`
	Inactive      bool   `json:"inactive,omitempty"`
}

// PoolMeta is what the encoder gets per swap: the order to send and the spender to approve.
type PoolMeta struct {
	BlockNumber     uint64 `json:"blockNumber"`
	ApprovalAddress string `json:"approvalAddress"`
	Maker           string `json:"maker"`
	Traits          string `json:"traits"`
	Data            string `json:"data"`
}

// SwapInfo is the result of CalcAmountOut that UpdateBalance consumes.
type SwapInfo struct {
	AmountIn     uint256.Int `json:"amountIn"`
	AmountOut    uint256.Int `json:"amountOut"`
	FeeIn        uint256.Int `json:"feeIn"`
	FeeCollected uint256.Int `json:"feeCollected"`
}

// state is the simulator's own copy of Extra: plain values, so a clone shares no integer with its source.
type state struct {
	Raw         [2]uint256.Int
	RawBlock    uint64
	Wallet      [2]uint256.Int
	Allowance   [2]uint256.Int
	WalletBlock uint64
	Docked      bool
	Inactive    bool
}

func (e *Extra) state() state {
	s := state{RawBlock: e.RawBlock, WalletBlock: e.WalletBlock, Docked: e.Docked, Inactive: e.Inactive}
	for i, src := range [...]struct{ raw, wallet, allowance *uint256.Int }{
		{e.RawBalance0, e.WalletBalance0, e.Allowance0}, {e.RawBalance1, e.WalletBalance1, e.Allowance1},
	} {
		if src.raw != nil {
			s.Raw[i] = *src.raw
		}
		if src.wallet != nil {
			s.Wallet[i] = *src.wallet
		}
		if src.allowance != nil {
			s.Allowance[i] = *src.allowance
		}
	}
	return s
}

// Metadata is the lister state: the cursor, and the confirmed head of the last full Portal sweep.
type Metadata struct {
	// LastScannedBlock is the last block whose strategies the lister is sure of: the log scan covered it, or
	// a full Portal sweep that started at or after it listed everything open.
	LastScannedBlock uint64 `json:"lastScannedBlock"`
	// LastPortalHead is H (head - confirmations, read before the sweep) of the last full Portal sweep, 0 if
	// none yet. The next full sweep moves the cursor up to it and no further, so a log fallback always
	// re-scans at least the blocks of one sweep interval. Metadata stored before it existed reads as 0.
	LastPortalHead uint64 `json:"lastPortalHead,omitempty"`
}
