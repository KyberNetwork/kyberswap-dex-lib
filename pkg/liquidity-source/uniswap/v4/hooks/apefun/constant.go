package apefun

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Deployment is one ape.fun suite: the hook and the factory and quote asset it was deployed with (all immutable,
// LaunchFeeProtectionHook.suiteConfig()). Addresses repeat across chains (the Arc factory is the Robinhood custody
// contract), so every address is read together with its chain and Track refuses a pool on any other chain.
type Deployment struct {
	ChainID    valueobject.ChainID
	Factory    common.Address
	QuoteAsset common.Address
}

var (
	HookArc       = common.HexToAddress("0x73D1b3c74552481fF1Fb57E28f5f3b1BF481e0cc")
	HookRobinhood = common.HexToAddress("0x6a87584119A16ef1df36c17a7d7c55e090DA60Cc")

	Deployments = map[common.Address]Deployment{
		HookArc: {valueobject.ChainIDArc, common.HexToAddress("0x7561415fB5539eE0A971cF520b0D0EEaC722c277"),
			common.HexToAddress("0x3600000000000000000000000000000000000000")}, // USDC, 6 decimals
		HookRobinhood: {valueobject.ChainIDRobinhood, common.HexToAddress("0xEAd6a9b1c6bc048564955872d6af7E7F7dD4D9aB"),
			common.HexToAddress("0x0bd7d308f8e1639fab988df18a8011f41eacad73")}, // WETH
	}

	HookAddresses = []common.Address{HookArc, HookRobinhood}
)

// Mirrors of LaunchFeeProtectionHook._fees and LaunchRegistryFactory.
const (
	bps         = 10_000
	baseFeeBps  = 100   // gross * 100 / 10000 on every swap
	maxTakeBps  = 9_800 // the snipe rate is capped at 9800 - tax
	maxTaxBps   = 1_000 // register rejects creatorTaxBps above 10%
	stageTrades = 3     // Launch.stage once the creation transaction ends; never changes after
)

// clockMarginSeconds: how far the quoting clock may run ahead of the block timestamp the hook sees.
const clockMarginSeconds = 1

// snipeBps is the anti-snipe rate on a buy by elapsed seconds since launch: 99%, 6.18%, 0.19%, then 0.
var snipeBps = [...]int64{9_900, 618, 19}

// Hook gas from callTracer on mainnet swaps routed by a KyberSwap executor (Robinhood and Arc, October 2026). A buy
// pays its whole fee in beforeSwap (pending write, credits, PoolManager.mint), a sell splits it with afterSwap.
// Pools with a buyback vault also record a price observation in afterSwap.
const (
	gasBeforeSwapBuy  = 216_000
	gasBeforeSwapSell = 117_000
	gasAfterSwapBuy   = 5_000
	gasAfterSwapSell  = 86_000
	gasObservation    = 33_000
)
