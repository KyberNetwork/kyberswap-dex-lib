package bancorv21

import (
	"embed"
	"encoding/json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

//go:embed sample_pool_data.txt
var sampleFile embed.FS

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	// Read the embedded file
	data, err := sampleFile.ReadFile("sample_pool_data.txt")
	if err != nil {
		panic(err)
	}

	// Unmarshal the JSON data into the Pool struct
	var pool entity.Pool
	err = json.Unmarshal(data, &pool)
	if err != nil {
		panic(err)
	}
	poolSim, err := NewPoolSimulator(pool)
	if err != nil {
		panic(err)
	}

	return []*PoolSimulator{
		poolSim,
	}
}
