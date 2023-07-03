package platypus

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolSimulator_CalcAmountOut_Base(t *testing.T) {
	// test data from https://snowtrace.io/address/0xbe52548488992cc76ffa1b42f3a58f646864df45#readProxyContract
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"0xc7198437980c041c805a1edcba50c1ce5db95118", 10000, "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664", 10001},
		{"0xc7198437980c041c805a1edcba50c1ce5db95118", 10, "0xd586e7f844cea2f87f50152665bcbc2c279d8d70", 10005786957061},
		{"0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7", 10, "0xd586e7f844cea2f87f50152665bcbc2c279d8d70", 10004178653696},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "platypus-base",
		Reserves: entity.PoolReserves{"318775844196", "397986108460", "464922144507443325081222", "801063044626", "834216051471"},
		Tokens:   []*entity.PoolToken{{Address: "0xc7198437980c041c805a1edcba50c1ce5db95118"}, {Address: "0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664"}, {Address: "0xd586e7f844cea2f87f50152665bcbc2c279d8d70"}, {Address: "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e"}, {Address: "0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7"}},
		Extra:    "{\"priceOracle\":\"0x7b52f4b5c476e7afd09266c35274737cd0af746b\",\"oracleType\":\"Chainlink\",\"c1\":376927610599998308,\"haircutRate\":100000000000000,\"retentionRatio\":1000000000000000000,\"slippageParamK\":20000000000000,\"slippageParamN\":7,\"xThreshold\":329811659274998519,\"paused\":false,\"sAvaxRate\":null,\"assetByToken\":{\"0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7\":{\"address\":\"\",\"decimals\":6,\"cash\":834216051471,\"liability\":982413796476,\"underlyingToken\":\"0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7\",\"aggregateAccount\":\"0x1655e447b7281e014e54cf0c1ad976b006e2b3dc\"},\"0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664\":{\"address\":\"\",\"decimals\":6,\"cash\":397986108460,\"liability\":464687034571,\"underlyingToken\":\"0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664\",\"aggregateAccount\":\"0x1655e447b7281e014e54cf0c1ad976b006e2b3dc\"},\"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e\":{\"address\":\"\",\"decimals\":6,\"cash\":801063044626,\"liability\":825349085270,\"underlyingToken\":\"0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e\",\"aggregateAccount\":\"0x1655e447b7281e014e54cf0c1ad976b006e2b3dc\"},\"0xc7198437980c041c805a1edcba50c1ce5db95118\":{\"address\":\"\",\"decimals\":6,\"cash\":318775844196,\"liability\":388315206569,\"underlyingToken\":\"0xc7198437980c041c805a1edcba50c1ce5db95118\",\"aggregateAccount\":\"0x1655e447b7281e014e54cf0c1ad976b006e2b3dc\"},\"0xd586e7f844cea2f87f50152665bcbc2c279d8d70\":{\"address\":\"\",\"decimals\":18,\"cash\":464922144507443325081222,\"liability\":113995414420528900845291,\"underlyingToken\":\"0xd586e7f844cea2f87f50152665bcbc2c279d8d70\",\"aggregateAccount\":\"0x1655e447b7281e014e54cf0c1ad976b006e2b3dc\"}}}",
	}, valueobject.ChainIDAvalancheCChain)
	require.Nil(t, err)

	assert.Equal(t, []string{"0xa7d7079b0fead91f3e65f86e8915cb59c1a4c664", "0xd586e7f844cea2f87f50152665bcbc2c279d8d70", "0xb97ef9ef8734c71904d8002f8b6bc66dd9c48a6e", "0x9702230a8ea53601f5cd2dc00fdbc13d4df4a8c7"}, p.CanSwapTo("0xc7198437980c041c805a1edcba50c1ce5db95118"))
	assert.Equal(t, 0, len(p.CanSwapTo("X")))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := p.CalcAmountOut(pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}, tc.out)
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
		})
	}
}

func TestPoolSimulator_CalcAmountOut_SAvax(t *testing.T) {
	// test data from https://snowtrace.io/address/0x4658ea7e9960d6158a261104aaa160cc953bb6ba#readProxyContract
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7", 10000, "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be", 9161},
		{"0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be", 10000, "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7", 10912},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "platypus-avax",
		Reserves: entity.PoolReserves{"363418522270035285223312", "838204096894457556759791"},
		Tokens:   []*entity.PoolToken{{Address: "0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7"}, {Address: "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be"}},
		Extra:    "{\"priceOracle\":\"0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be\",\"oracleType\":\"StakedAvax\",\"c1\":376927610599998308,\"haircutRate\":200000000000000,\"retentionRatio\":1000000000000000000,\"slippageParamK\":20000000000000,\"slippageParamN\":7,\"xThreshold\":329811659274998519,\"paused\":false,\"sAvaxRate\":1091359589234183301,\"assetByToken\":{\"0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be\":{\"address\":\"\",\"decimals\":18,\"cash\":838204096894457556759791,\"liability\":691613251026561379195920,\"underlyingToken\":\"0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be\",\"aggregateAccount\":\"0xc7100b7dba6154d43a4e50a1b68c3235e459c294\"},\"0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7\":{\"address\":\"\",\"decimals\":18,\"cash\":363418522270035285223312,\"liability\":512315100112742529413540,\"underlyingToken\":\"0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7\",\"aggregateAccount\":\"0xc7100b7dba6154d43a4e50a1b68c3235e459c294\"}}}",
	}, valueobject.ChainIDAvalancheCChain)
	require.Nil(t, err)

	assert.Equal(t, []string{"0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be"}, p.CanSwapTo("0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7"))
	assert.Equal(t, 0, len(p.CanSwapTo("X")))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := p.CalcAmountOut(pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}, tc.out)
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
		})
	}
}

func TestPoolSimulator_DiffAggAccount(t *testing.T) {
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Reserves: entity.PoolReserves{"1", "1"},
		Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
		Extra:    "{\"priceOracle\":\"OracleAddress\",\"oracleType\":\"Chainlink\",\"c1\":376927610599998308,\"haircutRate\":100000000000000,\"retentionRatio\":1000000000000000000,\"slippageParamK\":20000000000000,\"slippageParamN\":7,\"xThreshold\":329811659274998519,\"paused\":false,\"sAvaxRate\":null,\"assetByToken\":{\"B\":{\"address\":\"\",\"decimals\":6,\"cash\":393825691073,\"liability\":464687034571,\"underlyingToken\":\"B\",\"aggregateAccount\":\"AggAcc\"},\"A\":{\"address\":\"\",\"decimals\":6,\"cash\":321752815149,\"liability\":388315206569,\"underlyingToken\":\"A\",\"aggregateAccount\":\"AggAccXXXX\"}}}",
	}, valueobject.ChainIDAvalancheCChain)
	require.Nil(t, err)

	_, err = p.CalcAmountOut(pool.TokenAmount{Token: "A", Amount: big.NewInt(1)}, "B")
	assert.Equal(t, ErrDiffAggAcc, err)
}
