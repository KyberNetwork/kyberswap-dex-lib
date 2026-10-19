package stableng

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
)

type (
	// PoolsListUpdaterMetadata extends the shared metadata with the pools already listed,
	// so each run only probes pools that appeared since.
	PoolsListUpdaterMetadata struct {
		shared.PoolListUpdaterMetadata
		ListedPools []string `json:",omitempty"`
	}

	StaticExtra struct {
		APrecision *uint256.Int
		// which coins are originally native (before being converted to wrapped)
		IsNativeCoins []bool
	}

	Extra struct {
		InitialA            *uint256.Int
		FutureA             *uint256.Int
		InitialATime        int64
		FutureATime         int64
		SwapFee             *uint256.Int
		AdminFee            *uint256.Int
		OffpegFeeMultiplier *uint256.Int

		RateMultipliers []uint256.Int `json:",omitempty"`
	}
)
