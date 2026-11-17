package platypus

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// CloneState gives a typed copy whose asset map a swap on the clone does not reach. Without one, pathfinder-lib's
// DefaultClonePool falls back to go-clone's Slowly, which stores the clone's pointers without GC write barriers.
func TestCloneStateIsTypedAndIsolated(t *testing.T) {
	sim, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "platypus-base",
		Reserves: entity.PoolReserves{"318775844196", "397986108460", "464922144507443325081222", "801063044626", "834216051471"},
		Tokens:   []*entity.PoolToken{{Address: "0xc7198437980c041c805a1edcba50c1ce5db95118"}, {Address: "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664"}, {Address: "0xd586e7f844cea2f87f50152665bcbc2c279d8d70"}, {Address: "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e"}, {Address: "0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7"}},
		Extra:    "{\"priceOracle\":\"0x7b52f4b5c476e7afd09266c35274737cd0af746b\",\"oracleType\":\"Chainlink\",\"c1\":376927610599998308,\"haircutRate\":100000000000000,\"retentionRatio\":1000000000000000000,\"slippageParamK\":20000000000000,\"slippageParamN\":7,\"xThreshold\":329811659274998519,\"paused\":false,\"sAvaxRate\":null,\"assetByToken\":{\"0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7\":{\"address\":\"\",\"decimals\":6,\"cash\":834216051471,\"liability\":982413796476,\"underlyingToken\":\"0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7\",\"aggregateAccount\":\"0x1655e447b7281e014e54cf0c1ad976b006e2b3dc\"},\"0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664\":{\"address\":\"\",\"decimals\":6,\"cash\":397986108460,\"liability\":464687034571,\"underlyingToken\":\"0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664\",\"aggregateAccount\":\"0x1655e447b7281e014e54cf0c1ad976b006e2b3dc\"},\"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e\":{\"address\":\"\",\"decimals\":6,\"cash\":801063044626,\"liability\":825349085270,\"underlyingToken\":\"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e\",\"aggregateAccount\":\"0x1655e447b7281e014e54cf0c1ad976b006e2b3dc\"},\"0xc7198437980c041c805a1edcba50c1ce5db95118\":{\"address\":\"\",\"decimals\":6,\"cash\":318775844196,\"liability\":388315206569,\"underlyingToken\":\"0xc7198437980c041c805a1edcba50c1ce5db95118\",\"aggregateAccount\":\"0x1655e447b7281e014e54cf0c1ad976b006e2b3dc\"},\"0xd586e7f844cea2f87f50152665bcbc2c279d8d70\":{\"address\":\"\",\"decimals\":18,\"cash\":464922144507443325081222,\"liability\":113995414420528900845291,\"underlyingToken\":\"0xd586e7f844cea2f87f50152665bcbc2c279d8d70\",\"aggregateAccount\":\"0x1655e447b7281e014e54cf0c1ad976b006e2b3dc\"}}}",
	}, valueobject.ChainIDAvalancheCChain)
	require.NoError(t, err)

	cloned := sim.CloneState()
	require.NotNil(t, cloned, "CloneState must not leave the clone to go-clone")
	require.IsType(t, sim, cloned)
	testutil.TestCloneState(t, sim, pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xc7198437980c041c805a1edcba50c1ce5db95118", Amount: big.NewInt(10000)},
		TokenOut:      "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664",
	}, nil)
}
