package virtual

const (
	Protocol = "virtual"

	methodIsLiquidityPool = "isLiquidityPool"
	methodBuyTax          = "totalBuyTaxBasisPoints"
	methodSellTax         = "totalSellTaxBasisPoints"
	methodProjectBuyTax   = "projectBuyTaxBasisPoints"
	methodProjectSellTax  = "projectSellTaxBasisPoints"
)

var factories = map[string]struct{}{
	"0x8909dc15e40173ff4699343b6eb8132c65e18ec6": {}, // Base
	"0x5c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f": {}, // Ethereum
	"0x8bceaa40b9acdfaedf85adf4ff01f5ad6517937f": {}, // Robinhood
}

var baseTokens = map[string]struct{}{
	"0x0b3e328455c4059eeb9e3f84b5543f74e24e7e1b": {}, // Base
	"0x44ff8620b8ca30902395a7bd3f2407e1a091bf73": {}, // Ethereum
	"0xc6911796042b15d7fa4f6cde69e245ddcd3d9c31": {}, // Robinhood
}

var projectTaxFactories = map[string]struct{}{
	"0x8bceaa40b9acdfaedf85adf4ff01f5ad6517937f": {}, // Robinhood
}
