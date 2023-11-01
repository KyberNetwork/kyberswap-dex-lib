package smardex

type Config struct {
	DexID          string `json:"dexId"`
	FactoryAddress string
	PoolPagingSize int
	ChainID        uint
}
