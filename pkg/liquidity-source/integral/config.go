package integral

type Config struct {
	DexID          string `json:"dexID"`
	FactoryAddress string `json:"factoryAddress"`
	PoolPagingSize int
	ChainID        uint
}
