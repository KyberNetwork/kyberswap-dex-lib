package integral

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/suite"
)

type PoolListUpdaterTestSuite struct {
	suite.Suite
	client  *ethrpc.Client
	updater PoolListUpdater
}

func (ts *PoolListUpdaterTestSuite) SetupTest() {
	// Setup RPC server
	rpcClient := ethrpc.New("https://ethereum.kyberengineering.io")
	rpcClient.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))

	ts.client = rpcClient

	config := Config{
		DexID:          DexTypeIntegral,
		FactoryAddress: "0xC480b33eE5229DE3FbDFAD1D2DCD3F3BAD0C56c6",
		PoolPagingSize: 1000,
	}

	ts.updater = PoolListUpdater{
		config:       &config,
		ethrpcClient: ts.client,
	}
}

func (ts *PoolListUpdaterTestSuite) TestGetNewPools() {
	// get length of the pool list
	req := ts.client.NewRequest()
	var length *big.Int
	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: "0xC480b33eE5229DE3FbDFAD1D2DCD3F3BAD0C56c6",
		Method: factoryAllPairsLengthMethod,
		Params: nil,
	}, []interface{}{&length})
	_, err := req.TryAggregate()
	if err != nil {
		return
	}

	metadata := PoolListUpdaterMetadata{
		Offset: 0,
	}
	metadataBytes, _ := json.Marshal(metadata)
	pools, metadataRes, _ := ts.updater.GetNewPools(context.Background(), metadataBytes)

	for _, p := range pools {
		assert.NotNil(ts.Suite.T(), p.Address)
	}
	compare := length.Int64()
	if compare > length.Int64()-int64(metadata.Offset) {
		compare = length.Int64() - int64(metadata.Offset)
	}

	assert.Equal(ts.T(), compare, int64(len(pools)))
	var savedMetadataRes PoolListUpdaterMetadata
	err = json.Unmarshal(metadataRes, &savedMetadataRes)
	if err != nil {
		assert.Fail(ts.Suite.T(), "Error when unmarshal metadata after fetch")
	}
	assert.Equal(ts.T(), int(compare+int64(metadata.Offset)), savedMetadataRes.Offset)

}

func TestPoolListUpdaterTestSuite(t *testing.T) {
	t.Skip("Skipping testing in CI environment")
	suite.Run(t, new(PoolListUpdaterTestSuite))
}
