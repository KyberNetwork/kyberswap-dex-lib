package limitorder

type ChainID uint

type tokenPair struct {
	Token0 string `json:"token0"`
	Token1 string `json:"token1"`
}

type Extra struct {
	SellOrders []*order
	BuyOrders  []*order
}
