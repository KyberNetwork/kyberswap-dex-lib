package virtual

const (
	Protocol = "virtual"

	factoryBase     = "0x8909dc15e40173ff4699343b6eb8132c65e18ec6"
	factoryEthereum = "0x5c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f"

	methodIsLiquidityPool = "isLiquidityPool"
	methodBuyTax          = "totalBuyTaxBasisPoints"
	methodSellTax         = "totalSellTaxBasisPoints"
)

var baseTokens = map[string]struct{}{
	"0x0b3e328455c4059eeb9e3f84b5543f74e24e7e1b": {}, // Base
	"0x44ff8620b8ca30902395a7bd3f2407e1a091bf73": {}, // Ethereum
}
