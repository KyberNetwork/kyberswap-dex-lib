package dto

type IndexPoolsCommand struct {
	UsePoolAddresses bool

	PoolAddresses []string

	// pools with timestamp older than this will be ignored (default to 0 to index all pools)
	IgnorePoolsBeforeTimestamp int64
}

type GetChunkPoolCommand struct {
	ChunkSize int

	UsePoolAddresses bool
	PoolAddresses    []string

	AddressChunkIndex int

	Cursor uint64

	IsLastCommand bool
}

func NewGetChunkPoolCommand(chunkSize int, usePoolAddresses bool, poolAddresses []string) GetChunkPoolCommand {
	return GetChunkPoolCommand{
		ChunkSize:         chunkSize,
		UsePoolAddresses:  usePoolAddresses,
		PoolAddresses:     poolAddresses,
		AddressChunkIndex: 0,
		Cursor:            0,
		IsLastCommand:     false,
	}
}
