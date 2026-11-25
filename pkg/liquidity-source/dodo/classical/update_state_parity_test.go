package classical

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// The adapter path (CalcAmtOut, UpdateState without a fee) must reach the same state as UpdateBalance with the fee.
func TestUpdateStateParity(t *testing.T) {
	var ent entity.Pool
	require.NoError(t, json.Unmarshal([]byte(`{"address":"0xe4b2dfc82977dd2dce7e8d37895a6a8f50cbb4fb","swapFee":10000000000000,"exchange":"dodo-classical","type":"dodo-classical","timestamp":1716521335,"reserves":["1444873953831","578850766374"],"tokens":[{"address":"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9","name":"Tether USD","symbol":"USDT","decimals":6,"weight":50,"swappable":true},{"address":"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8","name":"USD Coin (Arb1)","symbol":"USDC","decimals":6,"weight":50,"swappable":true}],"extra":"{\"B\":\"1444873953831\",\"Q\":\"578850766374\",\"B0\":\"978121462386\",\"Q0\":\"1045528008085\",\"rStatus\":2,\"oraclePrice\":\"1000000000000000000\",\"k\":\"200000000000000\",\"mtFeeRate\":\"10000000000000\",\"lpFeeRate\":\"0\",\"tradeAllowed\":true,\"sellingAllowed\":true,\"buyingAllowed\":true,\"swappable\":true}","staticExtra":"{\"poolId\":\"0xe4b2dfc82977dd2dce7e8d37895a6a8f50cbb4fb\",\"lpToken\":\"0x82b423848cdd98740fb57f961fa692739f991633\",\"type\":\"CLASSICAL\",\"tokens\":[\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\"],\"dodoV1SellHelper\":\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\"}"}`), &ent))
	testutil.TestUpdateStateParity(t, func() any {
		sim, err := NewPoolSimulator(ent)
		require.NoError(t, err)
		return sim
	}, []string{"1000", "1000000", "1000000000", "1000000000000000", "1000000000000000000"})
}
