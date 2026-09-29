package everlongflamm

import _ "embed"

// Minimal ABIs, extracted with jq from the c104-deploy @ 80abd43 forge artifacts (the view functions the lister and
// tracker call, plus Multicall3's tryBlockAndAggregate). Three entries are not in that tree, because the contracts
// they belong to are not: AdaptiveCurveIrm.rateAtTarget is morpho-blue-irm v1.0.0's `rateAtTarget(Id) returns
// (int256)` (selector 0x01977b57); `aggregator()` in AggregatorV3.json is Chainlink's EACAggregatorProxy getter,
// which resolves the proxy a PriceFeed token or the sequencer feed is configured with to the aggregator that emits
// its rounds; and the four `BASE_FEED_x` / `QUOTE_FEED_x` getters in MorphoOracle.json are
// MorphoChainlinkOracleV2's, which name the proxies a venue's market oracle prices from. The last two are read
// only to resolve the tracker's dependency set (pool_tracker.go GetDependencies); nothing priced is decoded from
// them.
var (
	//go:embed abis/FLAMM.json
	flammABIJson []byte
	//go:embed abis/EverlongHook.json
	hookABIJson []byte
	//go:embed abis/EverlongLeverageHook.json
	levHookABIJson []byte
	//go:embed abis/LeverageSpreadHook.json
	spreadHookABIJson []byte
	//go:embed abis/PriceFeed.json
	priceFeedABIJson []byte
	//go:embed abis/MMRouter.json
	routerABIJson []byte
	//go:embed abis/MorphoBlueAccount.json
	accountABIJson []byte
	//go:embed abis/FLAMMFactory.json
	factoryABIJson []byte
	//go:embed abis/MorphoBlue.json
	morphoABIJson []byte
	//go:embed abis/AdaptiveCurveIrm.json
	irmABIJson []byte
	//go:embed abis/AggregatorV3.json
	aggregatorABIJson []byte
	//go:embed abis/MorphoOracle.json
	oracleABIJson []byte
	//go:embed abis/Multicall3.json
	multicallABIJson []byte
)
